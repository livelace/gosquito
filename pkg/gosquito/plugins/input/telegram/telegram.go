package telegramIn

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/zelenin/go-tdlib/client"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	DEFAULT_BUFFER_LENGHT = 1000
	DEFAULT_CHATS_DB      = "chats"
	DEFAULT_DATABASE_DIR  = "database"
	DEFAULT_FILES_DIR     = "files"
	DEFAULT_FILE_MAX_SIZE = "10m"
	DEFAULT_LOG_LEVEL     = 0
	DEFAULT_MATCH_TTL     = "1d"
	DEFAULT_USERS_DB      = "users"
	MAX_INSTANCE_PER_APP  = 1
)

var (
	ERROR_CHAT_UNKNOWN     = errors.New("chat unknown: %s, %s")
	ERROR_CHAT_JOIN_ERROR  = errors.New("join to chat error: %s, %s")
	ERROR_DOWNLOAD_TIMEOUT = errors.New("download timeout: %s")
	ERROR_LOAD_USERS_ERROR = errors.New("cannot load users: %s")
	ERROR_SAVE_CHATS_ERROR = errors.New("cannot save chats: %s")
)

func authorizePlugin(p *Plugin, clientAuthorizer *clientAuthorizer) {
	showMessage := func(m string) {
		log.WithFields(log.Fields{
			"flow":    p.Flow.FlowName,
			"file":    p.Flow.FlowFile,
			"plugin":  p.PluginName,
			"type":    p.PluginType,
			"message": m,
		}).Warnf(core.LOG_PLUGIN_INIT)
	}

	for {
		select {
		case state, ok := <-clientAuthorizer.State:
			if !ok {
				return
			}

			switch state.AuthorizationStateType() {
			case client.TypeAuthorizationStateWaitPhoneNumber:
				var phone string
				showMessage("type your phone number")
				_, _ = fmt.Scan(&phone)

				clientAuthorizer.PhoneNumber <- phone

			case client.TypeAuthorizationStateWaitCode:
				var code string
				showMessage("type sent code")
				_, _ = fmt.Scan(&code)

				clientAuthorizer.Code <- code

			case client.TypeAuthorizationStateWaitPassword:
				var password string
				showMessage("type your password")
				_, _ = fmt.Scan(&password)

				clientAuthorizer.Password <- password

			case client.TypeAuthorizationStateReady:
				return
			}
		}
	}
}

func downloadFile(p *Plugin, remoteId string) (string, error) {
	remoteFileReq := client.GetRemoteFileRequest{RemoteFileId: remoteId}
	remoteFile, err := p.TdlibClient.GetRemoteFile(&remoteFileReq)
	if err != nil {
		return "", err
	}

	downloadFileReq := client.DownloadFileRequest{FileId: remoteFile.Id, Priority: 1}
	downloadFile, err := p.TdlibClient.DownloadFile(&downloadFileReq)
	if err != nil {
		return "", err
	}

	// Check if file already downloaded.
	// Or wait for file id from receiveFiles listener.
	if downloadFile.Local.Path == "" {

		// Read files ids from file channel.
		// Return error if timeout is happened.
		for i := 0; i < p.OptionTimeout; i++ {

			for id := range p.FileChannel {
				if id == downloadFile.Id {
					file, _ := p.TdlibClient.GetFile(&client.GetFileRequest{FileId: id})
					return file.Local.Path, nil
				}
			}

			time.Sleep(1 * time.Second)

		}
		return "", fmt.Errorf(ERROR_DOWNLOAD_TIMEOUT.Error(), remoteId)
	}

	return downloadFile.Local.Path, nil
}

func getClient(p *Plugin) (*client.Client, error) {
	authorizer := client.ClientAuthorizer()
	go authorizePlugin(p, (*clientAuthorizer)(authorizer))

	authorizer.TdlibParameters <- p.TdlibParams

	verbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: int32(p.OptionLogLevel),
	})

	return client.NewClient(authorizer, verbosity)
}

func getChatId(p *Plugin, name string) (int64, error) {
	chat, err := p.TdlibClient.SearchPublicChat(&client.SearchPublicChatRequest{Username: name})
	if err != nil {
		return 0, err
	} else {
		return chat.Id, nil
	}
}

func joinToChat(p *Plugin, name string, id int64) error {
	_, err := p.TdlibClient.JoinChat(&client.JoinChatRequest{ChatId: id})
	if err != nil {
		return fmt.Errorf(ERROR_CHAT_JOIN_ERROR.Error(), name, err)
	}

	return nil
}

func loadChats(p *Plugin) (map[string]int64, error) {
	data := make(map[string]int64, 0)

	temp, err := core.PluginLoadData(filepath.Join(p.PluginDataDir, DEFAULT_CHATS_DB))
	if err != nil {
		return data, fmt.Errorf("cannot load chats: %s", err)
	}

	for k, v := range temp {
		chat := fmt.Sprintf("%s", k)
		chatId, err := strconv.ParseInt(fmt.Sprintf("%s", v), 10, 64)
		if err != nil {
			return data, fmt.Errorf("cannot get chat id: %s, %s", chat, err)
		}

		data[chat] = chatId
	}

	return data, nil
}

func loadUsers(p *Plugin) (map[int32][]string, error) {
	data := make(map[int32][]string, 0)

	temp, err := core.PluginLoadData(filepath.Join(p.PluginDataDir, DEFAULT_USERS_DB))
	if err != nil {
		return data, fmt.Errorf("cannot load users: %s", err)
	}

	for k, v := range temp {
		userId, err := strconv.ParseInt(fmt.Sprintf("%s", k), 10, 32)
		if err != nil {
			return data, fmt.Errorf("cannot get user id: %s", err)
		}

		userData, ok := v.([]string)
		if !ok {
			return data, fmt.Errorf("cannot parse user data: %d", userId)
		}

		data[int32(userId)] = userData
	}

	return data, nil
}

func receiveFiles(p *Plugin) {
	tempDirMatcher := regexp.MustCompile(filepath.Join(DEFAULT_FILES_DIR, "temp"))

	// Loop till the app end.
	for {
		listener := p.TdlibClient.GetListener()

		// Wait for new files, be sure they are not in temp, send file.id into channel.
		for update := range listener.Updates {
			switch update.(type) {
			case *client.UpdateFile:
				newFile := update.(*client.UpdateFile).File
				if newFile.Local.Path != "" && newFile.Size > 0 && !tempDirMatcher.MatchString(newFile.Local.Path) {
					p.FileChannel <- newFile.Id
				}
			}
		}

		listener.Close()
		time.Sleep(1 * time.Second)
	}
}

func receiveMessages(p *Plugin) {
	// Loop till the app end.
	for {
		listener := p.TdlibClient.GetListener()

		// Loop over message events.
		for update := range listener.Updates {
			switch update.(type) {

			// Log all updated users profiles. Passive data gathering.
			case *client.UpdateUser:
				user := update.(*client.UpdateUser).User
				p.UsersById[user.Id] = []string{
					user.Username, user.Type.UserTypeType(),
					user.FirstName, user.LastName, user.PhoneNumber,
				}

			case *client.UpdateNewMessage:
				newMessage := update.(*client.UpdateNewMessage)
				messageChatId := newMessage.Message.ChatId
				messageContent := newMessage.Message.Content
				messageTime := time.Unix(int64(newMessage.Message.Date), 0).UTC()

				messageSenderId := int32(-1)
				switch messageSender := newMessage.Message.Sender.(type) {
				case *client.MessageSenderChat:
					messageSenderId = int32(messageSender.ClientId)
				case *client.MessageSenderUser:
					messageSenderId = int32(messageSender.ClientId)
				}

				messageUserId := fmt.Sprintf("%v", messageSenderId)
				messageUserName := ""
				messageUserType := ""
				messageUserFirstName := ""
				messageUserLastName := ""
				messageUserPhoneNumber := ""

				if v, ok := p.UsersById[messageSenderId]; ok {
					messageUserName = v[0]
					messageUserType = v[1]
					messageUserFirstName = v[2]
					messageUserLastName = v[3]
					messageUserPhoneNumber = v[4]
				}

				// Process only specified chats.
				if chatName, ok := p.ChatsById[messageChatId]; ok {
					var u, _ = uuid.NewRandom()

					switch messageContent.(type) {
					case *client.MessageText:
						var textURL string
						formattedText := messageContent.(*client.MessageText).Text

						// Search for text URL.
						for _, entity := range formattedText.Entities {
							switch entity.Type.(type) {
							case *client.TextEntityTypeTextUrl:
								textURL = entity.Type.(*client.TextEntityTypeTextUrl).Url
							}
						}

						// Send data to channel.
						if len(p.DataChannel) < DEFAULT_BUFFER_LENGHT {
							p.DataChannel <- &core.DataItem{
								FLOW:       p.Flow.FlowName,
								PLUGIN:     p.PluginName,
								SOURCE:     chatName,
								TIME:       messageTime,
								TIMEFORMAT: messageTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
								UUID:       u,

								TELEGRAM: core.TelegramData{
									USERID:   messageUserId,
									USERNAME: messageUserName,
									USERTYPE: messageUserType,

									FIRSTNAME: messageUserFirstName,
									LASTNAME:  messageUserLastName,
									PHONE:     messageUserPhoneNumber,

									TEXT: formattedText.Text,
									URL:  textURL,
								},
							}
						}

					case *client.MessagePhoto:
						media := make([]string, 0)
						caption := messageContent.(*client.MessagePhoto).Caption
						photo := messageContent.(*client.MessagePhoto).Photo

						photoFile := photo.Sizes[len(photo.Sizes)-1]

						if int64(photoFile.Photo.Size) < p.OptionFileMaxSize {
							localFile, err := downloadFile(p, photoFile.Photo.Remote.Id)
							if err == nil {
								media = append(media, localFile)
							}
						}

						// Send data to channel.
						if len(p.DataChannel) < DEFAULT_BUFFER_LENGHT {
							p.DataChannel <- &core.DataItem{
								FLOW:       p.Flow.FlowName,
								PLUGIN:     p.PluginName,
								SOURCE:     chatName,
								TIME:       messageTime,
								TIMEFORMAT: messageTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
								UUID:       u,

								TELEGRAM: core.TelegramData{
									USERID:   messageUserId,
									USERNAME: messageUserName,
									USERTYPE: messageUserType,

									FIRSTNAME: messageUserFirstName,
									LASTNAME:  messageUserLastName,
									PHONE:     messageUserPhoneNumber,

									MEDIA: media,
									TEXT:  caption.Text,
								},
							}
						}

					case *client.MessageVideo:
						media := make([]string, 0)
						caption := messageContent.(*client.MessageVideo).Caption
						video := messageContent.(*client.MessageVideo).Video

						if int64(video.Video.Size) < p.OptionFileMaxSize {
							localFile, err := downloadFile(p, video.Video.Remote.Id)
							if err == nil {
								media = append(media, localFile)
							}
						}

						// Send data to channel.
						if len(p.DataChannel) < DEFAULT_BUFFER_LENGHT {
							p.DataChannel <- &core.DataItem{
								FLOW:       p.Flow.FlowName,
								PLUGIN:     p.PluginName,
								SOURCE:     chatName,
								TIME:       messageTime,
								TIMEFORMAT: messageTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
								UUID:       u,

								TELEGRAM: core.TelegramData{
									USERID:   messageUserId,
									USERNAME: messageUserName,
									USERTYPE: messageUserType,

									FIRSTNAME: messageUserFirstName,
									LASTNAME:  messageUserLastName,
									PHONE:     messageUserPhoneNumber,

									MEDIA: media,
									TEXT:  caption.Text,
								},
							}
						}
					}

				} else {
					log.WithFields(log.Fields{
						"hash":   p.Flow.FlowHash,
						"flow":   p.Flow.FlowName,
						"file":   p.Flow.FlowFile,
						"plugin": p.PluginName,
						"type":   p.PluginType,
						"data":   fmt.Sprintf("chat id is unknown, messages excluded: %v", messageChatId),
					}).Debug(core.LOG_PLUGIN_DATA)
				}
			}

			// Save users between updates receiving.
			_ = saveUsers(p)
		}

		listener.Close()
		time.Sleep(1 * time.Second)
	}
}

func saveChats(p *Plugin) error {
	temp := make(map[interface{}]interface{}, 0)

	for k, v := range p.ChatsByName {
		temp[k] = v
	}

	err := core.PluginSaveData(filepath.Join(p.PluginDataDir, DEFAULT_CHATS_DB), &temp)
	if err != nil {
		return fmt.Errorf("cannot save chats: %s", err)
	}

	return err
}

func saveUsers(p *Plugin) error {
	temp := make(map[interface{}]interface{}, 0)

	for k, v := range p.UsersById {
		temp[k] = v
	}

	err := core.PluginSaveData(filepath.Join(p.PluginDataDir, DEFAULT_USERS_DB), &temp)
	if err != nil {
		return fmt.Errorf("cannot save users: %s", err)
	}

	return err
}

type clientAuthorizer struct {
	TdlibParameters chan *client.TdlibParameters
	PhoneNumber     chan string
	Code            chan string
	State           chan client.AuthorizationState
	Password        chan string
}

type Plugin struct {
	m sync.Mutex

	Flow *core.Flow

	PluginName string
	PluginType string

	PluginDataDir string
	PluginTempDir string

	FileChannel chan int32
	DataChannel chan *core.DataItem

	ChatsById   map[int64]string
	ChatsByName map[string]int64
	UsersById   map[int32][]string

	TdlibClient *client.Client
	TdlibParams *client.TdlibParameters

	OptionApiHash             string
	OptionApiId               int
	OptionExpireAction        []string
	OptionExpireActionDelay   int64
	OptionExpireActionTimeout int
	OptionExpireInterval      int64
	OptionExpireLast          int64
	OptionFileMaxSize         int64
	OptionInput               []string
	OptionLogLevel            int
	OptionMatchSignature      []string
	OptionMatchTTL            time.Duration
	OptionTimeFormat          string
	OptionTimeZone            *time.Location
	OptionTimeout             int
}

func (p *Plugin) Recv() ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)
	currentTime := time.Now().UTC()

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}

	// Source stat.
	sourceNewStat := make(map[string]int32)
	sourceReceivedStat := make(map[string]int32)

	// Fixate channel length (channel changes length size in the loop).
	length := len(p.DataChannel)

	for i := 1; i <= length; i++ {
		var itemNew = false
		var itemSignature string
		var itemSignatureHash string
		var sourceLastTime time.Time

		item := <-p.DataChannel

		// Check if we work with source first time.
		if v, ok := flowStates[item.SOURCE]; ok {
			sourceLastTime = v
		} else {
			sourceLastTime = time.Unix(0, 0)
		}

		// Process only new items. Two methods:
		// 1. Match item by user provided signature.
		// 2. Compare item timestamp with source timestamp.
		if len(p.OptionMatchSignature) > 0 {
			itemSignature = item.SOURCE

			for _, v := range p.OptionMatchSignature {
				switch v {
				case "firstname":
					itemSignature += item.TELEGRAM.FIRSTNAME
					break
				case "lastname":
					itemSignature += item.TELEGRAM.LASTNAME
					break
				case "phone":
					itemSignature += item.TELEGRAM.PHONE
					break
				case "text":
					itemSignature += item.TELEGRAM.TEXT
					break
				case "time":
					itemSignature += item.TIME.String()
					break
				case "url":
					itemSignature += item.TELEGRAM.URL
					break
				case "username":
					itemSignature += item.TELEGRAM.USERNAME
					break
				case "usertype":
					itemSignature += item.TELEGRAM.USERTYPE
					break
				}
			}

			// set default value for signature if user provided wrong values.
			if itemSignature == item.SOURCE {
				itemSignature += item.TELEGRAM.TEXT + item.TIME.String()
			}

			itemSignatureHash = core.HashString(&itemSignature)

			if _, ok := flowStates[itemSignatureHash]; !ok {
				// save item signature hash to state.
				flowStates[itemSignatureHash] = currentTime

				// update source timestamp.
				if item.TIME.Unix() > sourceLastTime.Unix() {
					sourceLastTime = item.TIME
				}

				itemNew = true
			}

		} else {
			if item.TIME.Unix() > sourceLastTime.Unix() {
				sourceLastTime = item.TIME
				itemNew = true
			}
		}

		// Add item to result.
		if itemNew {
			temp = append(temp, item)
			sourceNewStat[item.SOURCE] += 1
		}

		flowStates[item.SOURCE] = sourceLastTime
		sourceReceivedStat[item.SOURCE] += 1
	}

	for _, source := range p.OptionInput {
		log.WithFields(log.Fields{
			"hash":   p.Flow.FlowHash,
			"flow":   p.Flow.FlowName,
			"file":   p.Flow.FlowFile,
			"plugin": p.PluginName,
			"type":   p.PluginType,
			"source": source,
			"data": fmt.Sprintf("last update: %v, received data: %d, new data: %d",
				flowStates[source], sourceReceivedStat[source], sourceNewStat[source]),
		}).Debug(core.LOG_PLUGIN_DATA)
	}

	// Save updated flow states.
	if err := p.SaveState(flowStates); err != nil {
		return temp, err
	}

	// Check every source for expiration.
	sourcesExpired := false

	// Check if any source is expired.
	for source, sourceTime := range flowStates {
		if (currentTime.Unix() - sourceTime.Unix()) > p.OptionExpireInterval {
			sourcesExpired = true

			// Execute command if expire delay exceeded.
			// ExpireLast keeps last execution timestamp.
			if (currentTime.Unix() - p.OptionExpireLast) > p.OptionExpireActionDelay {
				p.OptionExpireLast = currentTime.Unix()

				// Execute command with args.
				// We don't worry about command return code.
				if len(p.OptionExpireAction) > 0 {
					cmd := p.OptionExpireAction[0]
					args := []string{p.Flow.FlowName, source, fmt.Sprintf("%v", sourceTime.Unix())}
					args = append(args, p.OptionExpireAction[1:]...)

					output, err := core.ExecWithTimeout(cmd, args, p.OptionExpireActionTimeout)

					log.WithFields(log.Fields{
						"hash":   p.Flow.FlowHash,
						"flow":   p.Flow.FlowName,
						"file":   p.Flow.FlowFile,
						"plugin": p.PluginName,
						"type":   p.PluginType,
						"source": source,
						"data": fmt.Sprintf(
							"expire_action: command: %s, arguments: %v, output: %s, error: %v",
							cmd, args, output, err),
					}).Debug(core.LOG_PLUGIN_DATA)
				}
			}
		}
	}

	// Inform about expiration.
	if sourcesExpired {
		return temp, core.ERROR_FLOW_EXPIRE
	}

	return temp, nil
}

func (p *Plugin) GetFile() string {
	return p.Flow.FlowFile
}

func (p *Plugin) GetInput() []string {
	return p.OptionInput
}

func (p *Plugin) GetName() string {
	return p.PluginName
}

func (p *Plugin) GetType() string {
	return p.PluginType
}

func (p *Plugin) LoadState() (map[string]time.Time, error) {
	p.m.Lock()
	defer p.m.Unlock()

	data := make(map[string]time.Time, 0)

	if err := core.PluginLoadState(p.Flow.FlowStateDir, &data); err != nil {
		return data, err
	}

	return data, nil
}

func (p *Plugin) SaveState(data map[string]time.Time) error {
	p.m.Lock()
	defer p.m.Unlock()

	return core.PluginSaveState(p.Flow.FlowStateDir, &data, p.OptionMatchTTL)
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Flow:             pluginConfig.Flow,
		PluginName:       "telegram",
		PluginType:       "input",
		PluginDataDir:    filepath.Join(pluginConfig.Flow.FlowDataDir, "input", "telegram"),
		PluginTempDir:    filepath.Join(pluginConfig.Flow.FlowTempDir, "input", "telegram"),
		OptionExpireLast: 0,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"expire_action":         -1,
		"expire_action_timeout": -1,
		"expire_delay":          -1,
		"expire_interval":       -1,
		"template":              -1,
		"timeout":               -1,
		"time_format":           -1,
		"time_zone":             -1,

		"api_id":          1,
		"api_hash":        1,
		"cred":            -1,
		"file_max_size":   -1,
		"input":           1,
		"log_level":       -1,
		"match_signature": -1,
		"match_ttl":       -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	showParam := func(p string, v interface{}) {
		log.WithFields(log.Fields{
			"hash":   plugin.Flow.FlowHash,
			"flow":   plugin.Flow.FlowName,
			"file":   plugin.Flow.FlowFile,
			"plugin": plugin.PluginName,
			"type":   plugin.PluginType,
			"value":  fmt.Sprintf("%s: %v", p, v),
		}).Debug(core.LOG_SET_VALUE)
	}

	// -----------------------------------------------------------------------------------------------------------------
	// cred: get from config and/or set user specified values (higher priority).
	cred, _ := core.IsString((*pluginConfig.PluginParams)["cred"])

	// api_id.
	setApiID := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["api_id"] = 0
			plugin.OptionApiId = v
		}
	}
	setApiID(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.api_id", cred)))
	setApiID((*pluginConfig.PluginParams)["api_id"])

	// api_hash.
	setApiHash := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["api_hash"] = 0
			plugin.OptionApiHash = v
		}
	}
	setApiHash(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.api_hash", cred)))
	setApiHash((*pluginConfig.PluginParams)["api_hash"])

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// expire_action.
	setExpireAction := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["expire_action"] = 0
			plugin.OptionExpireAction = v
		}
	}
	setExpireAction(pluginConfig.AppConfig.GetStringSlice(core.VIPER_DEFAULT_EXPIRE_ACTION))
	setExpireAction(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.expire_action", template)))
	setExpireAction((*pluginConfig.PluginParams)["expire_action"])
	showParam("expire_action", plugin.OptionExpireAction)

	// expire_action_delay.
	setExpireActionDelay := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["expire_action_delay"] = 0
			plugin.OptionExpireActionDelay = v
		}
	}
	setExpireActionDelay(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_EXPIRE_ACTION_DELAY))
	setExpireActionDelay(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.expire_action_delay", template)))
	setExpireActionDelay((*pluginConfig.PluginParams)["expire_action_delay"])
	showParam("expire_action_delay", plugin.OptionExpireActionDelay)

	// expire_action_timeout.
	setExpireActionTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["expire_action_timeout"] = 0
			plugin.OptionExpireActionTimeout = v
		}
	}
	setExpireActionTimeout(pluginConfig.AppConfig.GetInt(core.VIPER_DEFAULT_EXPIRE_ACTION_TIMEOUT))
	setExpireActionTimeout(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.expire_action_timeout", template)))
	setExpireActionTimeout((*pluginConfig.PluginParams)["expire_action_timeout"])
	showParam("expire_action_timeout", plugin.OptionExpireActionTimeout)

	// expire_interval.
	setExpireInterval := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["expire_interval"] = 0
			plugin.OptionExpireInterval = v
		}
	}
	setExpireInterval(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_EXPIRE_INTERVAL))
	setExpireInterval(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.expire_interval", template)))
	setExpireInterval((*pluginConfig.PluginParams)["expire_interval"])
	showParam("expire_interval", plugin.OptionExpireInterval)

	// file_max_size.
	setFileMaxSize := func(p interface{}) {
		if v, b := core.IsSize(p); b {
			availableParams["file_max_size"] = 0
			plugin.OptionFileMaxSize = v
		}
	}
	setFileMaxSize(DEFAULT_FILE_MAX_SIZE)
	setFileMaxSize(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_max_size", template)))
	setFileMaxSize((*pluginConfig.PluginParams)["file_max_size"])
	showParam("file_max_size", plugin.OptionFileMaxSize)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.OptionInput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.input", template)))
	setInput((*pluginConfig.PluginParams)["input"])
	showParam("input", plugin.OptionInput)

	// log_level.
	setLogLevel := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["log_level"] = 0
			plugin.OptionLogLevel = v
		}
	}
	setLogLevel(DEFAULT_LOG_LEVEL)
	setLogLevel(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.log_level", template)))
	setLogLevel((*pluginConfig.PluginParams)["log_level"])

	// match_signature.
	setMatchSignature := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["match_signature"] = 0
			plugin.OptionMatchSignature = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setMatchSignature(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.match_signature", template)))
	setMatchSignature((*pluginConfig.PluginParams)["match_signature"])
	showParam("match_signature", plugin.OptionMatchSignature)

	// match_ttl.
	setMatchTTL := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["match_ttl"] = 0
			plugin.OptionMatchTTL = time.Duration(v) * time.Second
		}
	}
	setMatchTTL(DEFAULT_MATCH_TTL)
	setMatchTTL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.match_ttl", template)))
	setMatchTTL((*pluginConfig.PluginParams)["match_ttl"])
	showParam("match_ttl", plugin.OptionMatchTTL)

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.OptionTimeout = v
		}
	}
	setTimeout(pluginConfig.AppConfig.GetInt(core.VIPER_DEFAULT_PLUGIN_TIMEOUT))
	setTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.PluginParams)["timeout"])
	showParam("timeout", plugin.OptionTimeout)

	// time_format.
	setTimeFormat := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["time_format"] = 0
			plugin.OptionTimeFormat = v
		}
	}
	setTimeFormat(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
	setTimeFormat(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format", template)))
	setTimeFormat((*pluginConfig.PluginParams)["time_format"])
	showParam("time_format", plugin.OptionTimeFormat)

	// time_zone.
	setTimeZone := func(p interface{}) {
		if v, b := core.IsTimeZone(p); b {
			availableParams["time_zone"] = 0
			plugin.OptionTimeZone = v
		}
	}
	setTimeZone(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
	setTimeZone(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone", template)))
	setTimeZone((*pluginConfig.PluginParams)["time_zone"])
	showParam("time_zone", plugin.OptionTimeZone)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Telegram.

	plugin.TdlibParams = &client.TdlibParameters{
		ApiHash:                plugin.OptionApiHash,
		ApiId:                  int32(plugin.OptionApiId),
		ApplicationVersion:     core.APP_VERSION,
		DatabaseDirectory:      filepath.Join(plugin.PluginDataDir, DEFAULT_DATABASE_DIR),
		DeviceModel:            core.APP_NAME,
		EnableStorageOptimizer: true,
		FilesDirectory:         filepath.Join(plugin.PluginDataDir, DEFAULT_FILES_DIR),
		IgnoreFileNames:        true,
		SystemLanguageCode:     "en",
		SystemVersion:          plugin.Flow.FlowName,
		UseChatInfoDatabase:    true,
		UseFileDatabase:        true,
		UseMessageDatabase:     true,
		UseSecretChats:         false,
		UseTestDc:              false,
	}

	// Create client.
	tdlibClient, err := getClient(&plugin)
	if err != nil {
		return &Plugin{}, err
	} else {
		plugin.TdlibClient = tdlibClient
	}

	// Load already known chats ID mappings by their username (not available in API).
	// interfax_ru = -1001019826615
	chatsById := make(map[int64]string, 0)
	chatsByName, err := loadChats(&plugin)
	for i, v := range chatsByName {
		fmt.Printf("AAAAAAAAAAAAAAA: %s -> %s\n", i, v)
	}

	if err != nil {
		return &Plugin{}, err
	}

	// Check if we know ids for all specified chats.
	// We keep chats/user IDs due Telegram API limits.
	// We may be banned for 24 hours if limits were reached (~200 requests may be enough).
	for _, chatName := range plugin.OptionInput {

		if id, ok := chatsByName[chatName]; !ok {
			chatId, err := getChatId(&plugin, chatName)

			if err == nil {
				// Add found chat to chat database.
				chatsByName[chatName] = chatId
				chatsById[chatId] = chatName

				// Always join to chat.
				err = joinToChat(&plugin, chatName, chatId)
				if err != nil {
					return &Plugin{}, err
				}

			} else {
				// We are not tolerate to unknown chats (they might be closed).
				return &Plugin{}, fmt.Errorf(ERROR_CHAT_UNKNOWN.Error(), chatName, err)
			}

		} else {
			// Always join to chat.
			err = joinToChat(&plugin, chatName, id)

			// Recheck chat ID (it might be changed "silently").
			if err != nil {
				chatId, _ := getChatId(&plugin, chatName)
				err = joinToChat(&plugin, chatName, chatId)

				if err != nil {
					return &Plugin{}, err
				}

				chatsById[chatId] = chatName
			} else {
				chatsById[id] = chatName
			}
		}
	}

	// Set plugin data.
	plugin.ChatsById = chatsById
	plugin.ChatsByName = chatsByName

	// Save chats.
	if err := saveChats(&plugin); err != nil {
		return &Plugin{}, fmt.Errorf(ERROR_SAVE_CHATS_ERROR.Error(), err)
	}

	// Load users.
	// TODO: Users ids are mutable ?
	if users, err := loadUsers(&plugin); err == nil {
		plugin.UsersById = users
	} else {
		return &Plugin{}, fmt.Errorf(ERROR_LOAD_USERS_ERROR.Error(), err)
	}

	//showParam("chats amount", len(plugin.ChatsByName))
	//showParam("users amount", len(plugin.UsersById))

	// Get messages and files in background.
	plugin.FileChannel = make(chan int32, DEFAULT_BUFFER_LENGHT)
	plugin.DataChannel = make(chan *core.DataItem, DEFAULT_BUFFER_LENGHT)

	go receiveFiles(&plugin)
	go receiveMessages(&plugin)

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
