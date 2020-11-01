package telegramIn

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/zelenin/go-tdlib/client"
)

const (
	DEFAULT_BUFFER_LENGHT  = 1000
	DEFAULT_CHATS_DATABASE = "chats.db"
	DEFAULT_DATABASE_DIR   = "database"
	DEFAULT_FILE_MAX_SIZE  = "10m"
	DEFAULT_FILES_DIR      = "files"
	DEFAULT_LOG_LEVEL      = 0
)

var (
	ERROR_CHAT_UNKNOWN     = errors.New("chat unknown: %s, %s")
	ERROR_CHAT_JOIN_ERROR  = errors.New("join to chat error: %s, %s")
	ERROR_DOWNLOAD_TIMEOUT = errors.New("download timeout: %s")
	ERROR_SAVE_CHATS_ERROR = errors.New("cannot save chats: %s")
)

func authorizePlugin(p *Plugin, clientAuthorizer *clientAuthorizer) {
	showMessage := func(m string) {
		log.WithFields(log.Fields{
			"flow":    p.Flow,
			"file":    p.File,
			"plugin":  p.Name,
			"type":    p.Type,
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
		for i := 0; i < p.Timeout; i++ {

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
		NewVerbosityLevel: int32(p.LogLevel),
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

func loadChats(p *Plugin) (*map[string]int64, error) {
	temp := make(map[string]int64)

	m, err := core.PluginLoadData(filepath.Join(p.PluginDir, p.Flow, p.Type, p.Name), DEFAULT_CHATS_DATABASE)
	if err != nil {
		return &temp, err
	} else {
		for k, v := range m {
			temp[k] = v.(int64)
		}
	}

	return &temp, nil
}

func receiveFiles(p *Plugin) {
	listener := p.TdlibClient.GetListener()
	defer listener.Close()

	tempDirMatcher := regexp.MustCompile(filepath.Join(DEFAULT_FILES_DIR, "temp"))

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
}

func receiveMessages(p *Plugin) {
	listener := p.TdlibClient.GetListener()
	defer listener.Close()

	// Loop over updates.
	for update := range listener.Updates {

		// We need updates with type "UpdateNewMessage".
		switch update.(type) {
		case *client.UpdateNewMessage:
			newMessage := update.(*client.UpdateNewMessage)
			messageChatId := newMessage.Message.ChatId
			messageContent := newMessage.Message.Content
			messageTime := time.Unix(int64(newMessage.Message.Date), 0).UTC()

			// Process only specific chats.
			if chatUserName, ok := (*p.ChatsById)[messageChatId]; ok {
				var u, _ = uuid.NewRandom()

				// Process only text messages for now.
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
							FLOW:       p.Flow,
							PLUGIN:     p.Name,
							SOURCE:     chatUserName,
							TIME:       messageTime,
							TIMEFORMAT: messageTime.In(p.TimeZone).Format(p.TimeFormat),
							UUID:       u,

							TELEGRAM: core.TelegramData{
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

					if int64(photoFile.Photo.Size) < p.FileMaxSize {
						localFile, err := downloadFile(p, photoFile.Photo.Remote.Id)
						if err == nil {
							media = append(media, localFile)
						}
					}

					// Send data to channel.
					if len(p.DataChannel) < DEFAULT_BUFFER_LENGHT {
						p.DataChannel <- &core.DataItem{
							FLOW:       p.Flow,
							PLUGIN:     p.Name,
							SOURCE:     chatUserName,
							TIME:       messageTime,
							TIMEFORMAT: messageTime.In(p.TimeZone).Format(p.TimeFormat),
							UUID:       u,

							TELEGRAM: core.TelegramData{
								MEDIA: media,
								TEXT:  caption.Text,
							},
						}
					}

				case *client.MessageVideo:
					media := make([]string, 0)
					caption := messageContent.(*client.MessageVideo).Caption
					video := messageContent.(*client.MessageVideo).Video

					if int64(video.Video.Size) < p.FileMaxSize {
						localFile, err := downloadFile(p, video.Video.Remote.Id)
						if err == nil {
							media = append(media, localFile)
						}
					}

					// Send data to channel.
					if len(p.DataChannel) < DEFAULT_BUFFER_LENGHT {
						p.DataChannel <- &core.DataItem{
							FLOW:       p.Flow,
							PLUGIN:     p.Name,
							SOURCE:     chatUserName,
							TIME:       messageTime,
							TIMEFORMAT: messageTime.In(p.TimeZone).Format(p.TimeFormat),
							UUID:       u,

							TELEGRAM: core.TelegramData{
								MEDIA: media,
								TEXT:  caption.Text,
							},
						}
					}

				}
			}
		}
	}
}

func saveChats(p *Plugin, chats *map[string]int64) error {
	temp := make(map[string]interface{}, 0)

	for k, v := range *chats {
		temp[k] = v
	}

	err := core.PluginSaveData(filepath.Join(p.PluginDir, p.Flow, p.Type, p.Name), DEFAULT_CHATS_DATABASE, temp)

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

	Hash string
	Flow string

	File string
	Name string
	Type string

	PluginDir string
	StateDir  string
	TempDir   string
	TgBaseDir string

	ExpireAction        []string
	ExpireActionDelay   int64
	ExpireActionTimeout int
	ExpireInterval      int64
	ExpireLast          int64
	Force               bool

	ApiId       int
	ApiHash     string
	ChatsById   *map[int64]string
	ChatsByName *map[string]int64
	FileMaxSize int64
	Input       []string
	LogLevel    int
	Timeout     int
	TimeFormat  string
	TimeZone    *time.Location

	FileChannel chan int32
	DataChannel chan *core.DataItem

	TdlibClient *client.Client
	TdlibParams *client.TdlibParameters
}

func (p *Plugin) Recv() ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)
	currentTime := time.Now().UTC()

	// Exit if there is no data in data channel.
	if len(p.DataChannel) == 0 {
		return temp, nil
	}

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}

	// Delete irrelevant/obsolete sources.
	for source := range flowStates {
		if !core.IsValueInSlice(source, &p.Input) {
			delete(flowStates, source)
		}
	}

	// Count fetched data from source.
	sourceStat := make(map[string]int32)

	// Save channel length.
	// It will be recalculated in loop, if use directly.
	length := len(p.DataChannel)

	for i := 1; i <= length; i++ {
		var lastTime time.Time
		item := <-p.DataChannel

		// Check if we work with source first time.
		if v, ok := flowStates[item.SOURCE]; ok {
			lastTime = v.(time.Time)
		} else {
			lastTime = currentTime
		}
		newLastTime := lastTime

		// Append to results if data is new.
		if item.TIME.Unix() > newLastTime.Unix() || p.Force {
			newLastTime = item.TIME
			temp = append(temp, item)
		}

		flowStates[item.SOURCE] = newLastTime
		sourceStat[item.SOURCE] += 1
	}

	for _, source := range p.Input {
		log.WithFields(log.Fields{
			"hash":   p.Hash,
			"flow":   p.Flow,
			"file":   p.File,
			"plugin": p.Name,
			"type":   p.Type,
			"source": source,
			"data":   fmt.Sprintf("last update: %s, fetched data: %d", flowStates[source], sourceStat[source]),
		}).Debug(core.LOG_PLUGIN_STAT)
	}

	// Save updated flow states.
	if err := p.SaveState(flowStates); err != nil {
		return temp, err
	}

	// Check every source for expiration.
	sourcesExpired := false

	// Check if any source is expired.
	for source, sourceTime := range flowStates {
		if (currentTime.Unix() - sourceTime.(time.Time).Unix()) > p.ExpireInterval {
			sourcesExpired = true

			// Execute command if expire delay exceeded.
			// ExpireLast keeps last execution timestamp.
			if (currentTime.Unix() - p.ExpireLast) > p.ExpireActionDelay {
				p.ExpireLast = currentTime.Unix()

				// Execute command with args.
				// We don't worry about command return code.
				if len(p.ExpireAction) > 0 {
					cmd := p.ExpireAction[0]
					args := []string{p.Flow, source, fmt.Sprintf("%v", sourceTime.(time.Time).Unix())}
					args = append(args, p.ExpireAction[1:]...)

					output, err := core.ExecWithTimeout(cmd, args, p.ExpireActionTimeout)

					log.WithFields(log.Fields{
						"hash":   p.Hash,
						"flow":   p.Flow,
						"file":   p.File,
						"plugin": p.Name,
						"type":   p.Type,
						"source": source,
						"data": fmt.Sprintf(
							"expire_action: command: %s, arguments: %v, output: %s, error: %v",
							cmd, args, output, err),
					}).Debug(core.LOG_PLUGIN_STAT)
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
	return p.File
}

func (p *Plugin) GetInput() []string {
	return p.Input
}

func (p *Plugin) GetName() string {
	return p.Name
}

func (p *Plugin) GetType() string {
	return p.Type
}

func (p *Plugin) LoadState() (map[string]interface{}, error) {
	p.m.Lock()
	defer p.m.Unlock()

	return core.PluginLoadData(p.StateDir, p.Flow)
}

func (p *Plugin) SaveState(data map[string]interface{}) error {
	p.m.Lock()
	defer p.m.Unlock()

	return core.PluginSaveData(p.StateDir, p.Flow, data)
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Hash: pluginConfig.Hash,
		Flow: pluginConfig.Flow,

		File: pluginConfig.File,
		Name: "telegram",
		Type: "input",

		PluginDir: pluginConfig.Config.GetString(core.VIPER_DEFAULT_PLUGIN_DATA),
		StateDir:  pluginConfig.Config.GetString(core.VIPER_DEFAULT_PLUGIN_STATE),
		TempDir:   pluginConfig.Config.GetString(core.VIPER_DEFAULT_PLUGIN_TEMP),

		ExpireLast: 0,
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
		"force":                 -1,
		"template":              -1,
		"timeout":               -1,
		"time_format":           -1,
		"time_zone":             -1,

		"api_id":        1,
		"api_hash":      1,
		"cred":          -1,
		"file_max_size": -1,
		"input":         1,
		"log_level":     -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	showParam := func(p string, v interface{}) {
		log.WithFields(log.Fields{
			"hash":   plugin.Hash,
			"flow":   plugin.Flow,
			"file":   plugin.File,
			"plugin": plugin.Name,
			"type":   plugin.Type,
			"value":  fmt.Sprintf("%s: %v", p, v),
		}).Debug(core.LOG_SET_VALUE)
	}

	// -----------------------------------------------------------------------------------------------------------------
	// cred: get from config and/or set user specified values (higher priority).
	cred, _ := core.IsString((*pluginConfig.Params)["cred"])

	// api_id.
	setApiID := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["api_id"] = 0
			plugin.ApiId = v
		}
	}
	setApiID(pluginConfig.Config.GetInt(fmt.Sprintf("%s.api_id", cred)))
	setApiID((*pluginConfig.Params)["api_id"])

	// api_hash.
	setApiHash := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["api_hash"] = 0
			plugin.ApiHash = v
		}
	}
	setApiHash(pluginConfig.Config.GetString(fmt.Sprintf("%s.api_hash", cred)))
	setApiHash((*pluginConfig.Params)["api_hash"])

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.Params)["template"])

	// expire_action.
	setExpireAction := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["expire_action"] = 0
			plugin.ExpireAction = v
		}
	}
	setExpireAction(pluginConfig.Config.GetStringSlice(core.VIPER_DEFAULT_EXPIRE_ACTION))
	setExpireAction(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.expire_action", template)))
	setExpireAction((*pluginConfig.Params)["expire_action"])
	showParam("expire_action", plugin.ExpireAction)

	// expire_action_delay.
	setExpireActionDelay := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["expire_action_delay"] = 0
			plugin.ExpireActionDelay = v
		}
	}
	setExpireActionDelay(pluginConfig.Config.GetString(core.VIPER_DEFAULT_EXPIRE_ACTION_DELAY))
	setExpireActionDelay(pluginConfig.Config.GetString(fmt.Sprintf("%s.expire_action_delay", template)))
	setExpireActionDelay((*pluginConfig.Params)["expire_action_delay"])
	showParam("expire_action_delay", plugin.ExpireActionDelay)

	// expire_action_timeout.
	setExpireActionTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["expire_action_timeout"] = 0
			plugin.ExpireActionTimeout = v
		}
	}
	setExpireActionTimeout(pluginConfig.Config.GetInt(core.VIPER_DEFAULT_EXPIRE_ACTION_TIMEOUT))
	setExpireActionTimeout(pluginConfig.Config.GetString(fmt.Sprintf("%s.expire_action_timeout", template)))
	setExpireActionTimeout((*pluginConfig.Params)["expire_action_timeout"])
	showParam("expire_action_timeout", plugin.ExpireActionTimeout)

	// expire_interval.
	setExpireInterval := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["expire_interval"] = 0
			plugin.ExpireInterval = v
		}
	}
	setExpireInterval(pluginConfig.Config.GetString(core.VIPER_DEFAULT_EXPIRE_INTERVAL))
	setExpireInterval(pluginConfig.Config.GetString(fmt.Sprintf("%s.expire_interval", template)))
	setExpireInterval((*pluginConfig.Params)["expire_interval"])
	showParam("expire_interval", plugin.ExpireInterval)

	// file_max_size.
	setFileMaxSize := func(p interface{}) {
		if v, b := core.IsSize(p); b {
			availableParams["file_max_size"] = 0
			plugin.FileMaxSize = v
		}
	}
	setFileMaxSize(DEFAULT_FILE_MAX_SIZE)
	setFileMaxSize(pluginConfig.Config.GetString(fmt.Sprintf("%s.file_max_size", template)))
	setFileMaxSize((*pluginConfig.Params)["file_max_size"])
	showParam("file_max_size", plugin.FileMaxSize)

	// force.
	setForce := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["force"] = 0
			plugin.Force = v
		}
	}
	setForce(core.DEFAULT_FORCE_INPUT)
	setForce(pluginConfig.Config.GetString(fmt.Sprintf("%s.force", template)))
	setForce((*pluginConfig.Params)["force"])
	showParam("force", plugin.Force)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.Input = core.ExtractConfigVariableIntoArray(pluginConfig.Config, v)
		}
	}
	setInput(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.input", template)))
	setInput((*pluginConfig.Params)["input"])
	showParam("input", plugin.Input)

	// log_level.
	setLogLevel := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["log_level"] = 0
			plugin.LogLevel = v
		}
	}
	setLogLevel(DEFAULT_LOG_LEVEL)
	setLogLevel(pluginConfig.Config.GetInt(fmt.Sprintf("%s.log_level", template)))
	setLogLevel((*pluginConfig.Params)["log_level"])

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.Timeout = v
		}
	}
	setTimeout(pluginConfig.Config.GetInt(core.VIPER_DEFAULT_PLUGIN_TIMEOUT))
	setTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.Params)["timeout"])
	showParam("timeout", plugin.Timeout)

	// time_format.
	setTimeFormat := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["time_format"] = 0
			plugin.TimeFormat = v
		}
	}
	setTimeFormat(pluginConfig.Config.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
	setTimeFormat(pluginConfig.Config.GetString(fmt.Sprintf("%s.time_format", template)))
	setTimeFormat((*pluginConfig.Params)["time_format"])
	showParam("time_format", plugin.TimeFormat)

	// time_zone.
	setTimeZone := func(p interface{}) {
		if v, b := core.IsTimeZone(p); b {
			availableParams["time_zone"] = 0
			plugin.TimeZone = v
		}
	}
	setTimeZone(pluginConfig.Config.GetString(core.VIPER_DEFAULT_TIME_ZONE))
	setTimeZone(pluginConfig.Config.GetString(fmt.Sprintf("%s.time_zone", template)))
	setTimeZone((*pluginConfig.Params)["time_zone"])
	showParam("time_zone", plugin.TimeZone)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Telegram.

	plugin.TgBaseDir = filepath.Join(plugin.PluginDir, plugin.Flow, plugin.Type, plugin.Name)

	plugin.TdlibParams = &client.TdlibParameters{
		ApiHash:                plugin.ApiHash,
		ApiId:                  int32(plugin.ApiId),
		ApplicationVersion:     core.APP_VERSION,
		DatabaseDirectory:      filepath.Join(plugin.TgBaseDir, DEFAULT_DATABASE_DIR),
		DeviceModel:            core.APP_NAME,
		EnableStorageOptimizer: true,
		FilesDirectory:         filepath.Join(plugin.TgBaseDir, DEFAULT_FILES_DIR),
		IgnoreFileNames:        true,
		SystemLanguageCode:     "en",
		SystemVersion:          core.APP_VERSION,
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
	if err != nil {
		return &Plugin{}, err
	}

	// Check if we known ids of all provided chats.
	for _, chatName := range plugin.Input {

		if id, ok := (*chatsByName)[chatName]; !ok {
			chatId, err := getChatId(&plugin, chatName)

			if err == nil {
				// Add found chat to chats databases.
				(*chatsByName)[chatName] = chatId
				chatsById[chatId] = chatName

				// Join to chat for updates receiving.
				err = joinToChat(&plugin, chatName, chatId)
				if err != nil {
					return &Plugin{}, err
				}

			} else {
				return &Plugin{}, fmt.Errorf(ERROR_CHAT_UNKNOWN.Error(), chatName, err)
			}

		} else {
			chatsById[id] = chatName
		}
	}

	// Save all chats mappings for further usage.
	err = saveChats(&plugin, chatsByName)
	if err != nil {
		return &Plugin{}, fmt.Errorf(ERROR_SAVE_CHATS_ERROR.Error(), err)
	}

	// Set plugin data.
	plugin.ChatsById = &chatsById
	plugin.ChatsByName = chatsByName

	// Get messages and files in background.
	plugin.FileChannel = make(chan int32, DEFAULT_BUFFER_LENGHT)
	plugin.DataChannel = make(chan *core.DataItem, DEFAULT_BUFFER_LENGHT)

	go receiveFiles(&plugin)
	go receiveMessages(&plugin)

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
