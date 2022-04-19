package telegramIn

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/livelace/go-tdlib/client"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
)

const (
	PLUGIN_NAME = "telegram"

	DEFAULT_ADS_ID            = "ads"
	DEFAULT_ADS_ENABLE        = true
	DEFAULT_ADS_PERIOD        = "5m"
	DEFAULT_BUFFER_LENGHT     = 100000
	DEFAULT_CHATS_DATA        = "chats.data"
	DEFAULT_DATABASE_DIR      = "database"
	DEFAULT_FETCH_TIMEOUT     = "1h"
	DEFAULT_FILES_DIR         = "files"
	DEFAULT_FILE_MAX_SIZE     = "10m"
	DEFAULT_LOG_LEVEL         = 0
	DEFAULT_MATCH_TTL         = "1d"
	DEFAULT_ORIGINAL_FILENAME = true
	DEFAULT_PROXY_ENABLE      = false
	DEFAULT_PROXY_PORT        = 9050
	DEFAULT_PROXY_SERVER      = "127.0.0.1"
	DEFAULT_PROXY_TYPE        = "socks"
	DEFAULT_SESSION_TTL       = 366
	DEFAULT_SHOW_CHAT         = false
	DEFAULT_SHOW_USER         = false
	DEFAULT_STATUS_ENABLE     = true
	DEFAULT_STATUS_PERIOD     = "5m"
	DEFAULT_STORAGE_OPTIMIZE  = true
	DEFAULT_STORAGE_PERIOD    = "1h"
	DEFAULT_USERS_DATA        = "users.data"
	MAX_INSTANCE_PER_APP      = 1
)

var (
	ERROR_CHAT_COMMON_ERROR  = errors.New("chat error: %s, %s")
	ERROR_CHAT_JOIN_ERROR    = errors.New("join to chat error: %s, %s")
	ERROR_FETCH_ERROR        = errors.New("fetch error: %s")
	ERROR_FETCH_TIMEOUT      = errors.New("fetch timeout: %s")
	ERROR_FILE_SIZE_EXCEEDED = errors.New("file size exceeded: %v (%v > %v)")
	ERROR_LOAD_USERS_ERROR   = errors.New("cannot load users: %s")
	ERROR_PROXY_TYPE_UNKNOWN = errors.New("proxy type unknown: %s")
	ERROR_SAVE_CHATS_ERROR   = errors.New("cannot save chats: %s")
)

type clientAuthorizer struct {
	TdlibParameters chan *client.TdlibParameters
	PhoneNumber     chan string
	Code            chan string
	State           chan client.AuthorizationState
	Password        chan string
}

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

func downloadFile(p *Plugin, remoteId string, originalFileName string) (string, error) {
	localFile := ""

	core.LogInputPlugin(p.LogFields, "fetch", fmt.Sprintf("begin: %v -> %v", remoteId, originalFileName))

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

	// 1. File downloading might be in progress. Just wait for it.
	// 2. File might be already downloaded.
	if downloadFile.Local.Path == "" {

		// 1. Read files IDs from file channel.
		// 2. Return error if timeout is happened.
		for i := 0; i < p.OptionFetchTimeout; i++ {
			for id := range p.FileChannel {
				if id == downloadFile.Id {
					if f, err := p.TdlibClient.GetFile(&client.GetFileRequest{FileId: id}); err == nil {
						localFile = f.Local.Path
						goto stopWaiting
					} else {
						return "", fmt.Errorf(ERROR_FETCH_ERROR.Error(), err)
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
		return "", fmt.Errorf(ERROR_FETCH_TIMEOUT.Error(), remoteId)

	} else {
		localFile = downloadFile.Local.Path
	}

stopWaiting:
	core.LogInputPlugin(p.LogFields, "fetch", fmt.Sprintf("end: %v -> %v", remoteId, localFile))

	return localFile, nil
}

func getClient(p *Plugin) (*client.Client, error) {
	authorizer := client.ClientAuthorizer()
	go authorizePlugin(p, (*clientAuthorizer)(authorizer))

	authorizer.TdlibParameters <- p.TdlibParams

	verbosity := client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: int32(p.OptionLogLevel),
	})

	if p.OptionProxyEnable {
		proxyRequest := client.AddProxyRequest{
			Server: p.OptionProxyServer,
			Port:   int32(p.OptionProxyPort),
			Enable: p.OptionProxyEnable,
		}

		switch p.OptionProxyType {
		case "socks":
			proxyRequest.Type = &client.ProxyTypeSocks5{
				Username: p.OptionProxyUsername,
				Password: p.OptionProxyPassword,
			}
		default:
			proxyRequest.Type = &client.ProxyTypeHttp{
				Username: p.OptionProxyUsername,
				Password: p.OptionProxyPassword,
			}
		}

		proxy := client.WithProxy(&proxyRequest)

		return client.NewClient(authorizer, verbosity, proxy)

	} else {
		c, err := client.NewClient(authorizer, verbosity)

		if err == nil {
			if proxies, err := c.GetProxies(); err == nil {
				for _, v := range proxies.Proxies {
					c.RemoveProxy(&client.RemoveProxyRequest{ProxyId: v.Id})
				}
			}
			c.DisableProxy()
		}

		return c, err
	}
}

func getPrivateChatId(p *Plugin, name string) (int64, error) {
	chatInfo, chatInfoErr := p.TdlibClient.CheckChatInviteLink(&client.CheckChatInviteLinkRequest{InviteLink: name})
	chat, err := p.TdlibClient.JoinChatByInviteLink(&client.JoinChatByInviteLinkRequest{InviteLink: name})

	if err != nil && err.Error() == "400 USER_ALREADY_PARTICIPANT" && chatInfoErr == nil {
		return chatInfo.ChatId, nil

	} else if err != nil {
		return 0, err

	} else {
		return chat.Id, nil
	}
}

func getPublicChatId(p *Plugin, name string) (int64, error) {
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

	err := core.PluginLoadData(filepath.Join(p.PluginDataDir, DEFAULT_CHATS_DATA), &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

func loadUsers(p *Plugin) (map[int64][]string, error) {
	data := make(map[int64][]string, 0)

	err := core.PluginLoadData(filepath.Join(p.PluginDataDir, DEFAULT_USERS_DATA), &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

func receiveAds(p *Plugin) {
	for {
		for chatId := range p.ChatsById {
			chatName := p.ChatsById[chatId]
			sponsoredMessage, err :=
				p.TdlibClient.GetChatSponsoredMessage(&client.GetChatSponsoredMessageRequest{ChatId: chatId})

			if err == nil {
				var u, _ = uuid.NewRandom()
				messageTime := time.Now().UTC()
				messageContent := sponsoredMessage.Content

				switch sponsoredMessage.Content.(type) {
				case *client.MessageText:
					textURLs := make([]string, 0)
					formattedText := messageContent.(*client.MessageText).Text

					// Search for text MESSAGETEXTURL.
					for _, entity := range formattedText.Entities {
						switch entity.Type.(type) {
						case *client.TextEntityTypeTextUrl:
							textURLs = append(textURLs, entity.Type.(*client.TextEntityTypeTextUrl).Url)
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

							TELEGRAM: core.Telegram{
								USERID:   DEFAULT_ADS_ID,
								USERNAME: DEFAULT_ADS_ID,
								USERTYPE: DEFAULT_ADS_ID,

								USERFIRSTNAME: DEFAULT_ADS_ID,
								USERLASTNAME:  DEFAULT_ADS_ID,
								USERPHONE:     DEFAULT_ADS_ID,

								MESSAGETEXT:    formattedText.Text,
								MESSAGETEXTURL: textURLs,
							},
						}
					}
				}
			}
		}

		time.Sleep(time.Duration(p.OptionAdsPeriod) * time.Second)
	}
}

func receiveFiles(p *Plugin) {
	listener := p.TdlibClient.GetListener()

	for {
		select {
		case update := <-listener.Updates:
			switch update.(type) {
			case *client.UpdateFile:
				newFile := update.(*client.UpdateFile).File
				if newFile.Local.IsDownloadingCompleted || !newFile.Local.CanBeDownloaded {
					p.FileChannel <- newFile.Id
				}
			}
		}
	}
}

func receiveMessages(p *Plugin) {
	listener := p.TdlibClient.GetListener()

	for {
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
				// Map message attributes to vars.
				message := update.(*client.UpdateNewMessage)
				messageChatId := message.Message.ChatId
				messageChatTitle := ""
				messageChatType := ""
				messageContent := message.Message.Content
				messageFileName := ""
				messageFileSize := int32(0)
				messageId := message.Message.Id
				messageMedia := make([]string, 0)
				messageSenderId := int64(-1)
				messageText := ""
				messageTextURLs := make([]string, 0)
				messageTime := time.Unix(int64(message.Message.Date), 0).UTC()
				messageType := messageContent.MessageContentType()
				messageURL := ""
				sendMessage := false
				warnings := make([]string, 0)

				if v, err := p.TdlibClient.GetChat(&client.GetChatRequest{ChatId: messageChatId}); err == nil {
					messageChatTitle = v.Title
					messageChatType = v.GetType()
				}

				if v, err := p.TdlibClient.GetMessageLink(&client.GetMessageLinkRequest{ChatId: messageChatId, MessageId: messageId}); err == nil {
					messageURL = v.Link
				}

				switch messageSender := message.Message.SenderId.(type) {
				case *client.MessageSenderChat:
				case *client.MessageSenderUser:
					messageSenderId = int64(messageSender.ClientId)
				}

				// Try to map user attributes from internal database (UpdateUser event).
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
					case *client.MessageAudio:
						sendMessage = true
						audio := messageContent.(*client.MessageAudio).Audio
						messageText = messageContent.(*client.MessageAudio).Caption.Text

						if int64(audio.Audio.Size) < p.OptionFileMaxSize {
							localFile, err := downloadFile(p, audio.Audio.Remote.Id, audio.FileName)
							if err == nil {
								messageMedia = append(messageMedia, localFile)
							}
						} else {
							messageFileName = audio.FileName
							messageFileSize = audio.Audio.Size
						}

					case *client.MessageDocument:
						sendMessage = true
						document := messageContent.(*client.MessageDocument).Document
						messageText = messageContent.(*client.MessageDocument).Caption.Text

						if int64(document.Document.Size) < p.OptionFileMaxSize {
							localFile, err := downloadFile(p, document.Document.Remote.Id, document.FileName)
							if err == nil {
								messageMedia = append(messageMedia, localFile)
							}
						} else {
							messageFileName = document.FileName
							messageFileSize = document.Document.Size
						}

					case *client.MessageText:
						sendMessage = true
						formattedText := messageContent.(*client.MessageText).Text
						messageText = formattedText.Text

						for _, entity := range formattedText.Entities {
							switch entity.Type.(type) {
							case *client.TextEntityTypeTextUrl:
								messageTextURLs = append(messageTextURLs, entity.Type.(*client.TextEntityTypeTextUrl).Url)
							}
						}

					case *client.MessagePhoto:
						sendMessage = true
						photo := messageContent.(*client.MessagePhoto).Photo
						photoFile := photo.Sizes[len(photo.Sizes)-1]
						messageText = messageContent.(*client.MessagePhoto).Caption.Text

						if int64(photoFile.Photo.Size) < p.OptionFileMaxSize {
							localFile, err := downloadFile(p, photoFile.Photo.Remote.Id, "")
							if err == nil {
								messageMedia = append(messageMedia, localFile)
							}
						} else {
							messageFileName = "phone"
							messageFileSize = photoFile.Photo.Size
						}

					case *client.MessageVideo:
						sendMessage = true
						messageText = messageContent.(*client.MessageVideo).Caption.Text
						video := messageContent.(*client.MessageVideo).Video

						if int64(video.Video.Size) < p.OptionFileMaxSize {
							localFile, err := downloadFile(p, video.Video.Remote.Id, video.FileName)
							if err == nil {
								messageMedia = append(messageMedia, localFile)
							}
						} else {
							messageFileName = video.FileName
							messageFileSize = video.Video.Size
						}

					case *client.MessageVoiceNote:
						sendMessage = true
						messageText = messageContent.(*client.MessageVoiceNote).Caption.Text
						note := messageContent.(*client.MessageVoiceNote).VoiceNote

						if int64(note.Voice.Size) < p.OptionFileMaxSize {
							localFile, err := downloadFile(p, note.Voice.Remote.Id, "")
							if err == nil {
								messageMedia = append(messageMedia, localFile)
							}
						} else {
							messageFileName = "voice note"
							messageFileSize = note.Voice.Size
						}

					case *client.MessageVideoNote:
						sendMessage = true
						note := messageContent.(*client.MessageVideoNote).VideoNote

						if int64(note.Video.Size) < p.OptionFileMaxSize {
							localFile, err := downloadFile(p, note.Video.Remote.Id, "")
							if err == nil {
								messageMedia = append(messageMedia, localFile)
							}
						} else {
							messageFileName = "video note"
							messageFileSize = note.Video.Size
						}
					}

					// Warnings.
					if messageFileSize > 0 {
						warnings = append(warnings, fmt.Sprintf(ERROR_FILE_SIZE_EXCEEDED.Error(),
							messageFileName, core.BytesToSize(int64(messageFileSize)), core.BytesToSize(p.OptionFileMaxSize)))

						core.LogInputPlugin(p.LogFields, "", fmt.Sprintf(ERROR_FILE_SIZE_EXCEEDED.Error(),
							messageFileName, core.BytesToSize(int64(messageFileSize)), core.BytesToSize(p.OptionFileMaxSize)))
					}

					// Send data to channel.
					if sendMessage && len(p.DataChannel) < DEFAULT_BUFFER_LENGHT {
						p.DataChannel <- &core.DataItem{
							FLOW:       p.Flow.FlowName,
							PLUGIN:     p.PluginName,
							SOURCE:     chatName,
							TIME:       messageTime,
							TIMEFORMAT: messageTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
							UUID:       u,

							TELEGRAM: core.Telegram{
								CHATID:    fmt.Sprintf("%v", messageChatId),
								CHATTITLE: messageChatTitle,
								CHATTYPE:  messageChatType,

								USERID:        messageUserId,
								USERNAME:      messageUserName,
								USERTYPE:      messageUserType,
								USERFIRSTNAME: messageUserFirstName,
								USERLASTNAME:  messageUserLastName,
								USERPHONE:     messageUserPhoneNumber,

								MESSAGEID:       fmt.Sprintf("%v", messageId),
								MESSAGEMEDIA:    messageMedia,
								MESSAGESENDERID: fmt.Sprintf("%v", messageSenderId),
								MESSAGETYPE:     messageType,
								MESSAGETEXT:     messageText,
								MESSAGETEXTURL:  messageTextURLs,
								MESSAGEURL:      messageURL,

								WARNINGS: warnings,
							},
						}
					}

				} else {
					core.LogInputPlugin(p.LogFields, "",
						fmt.Sprintf("chat filtered, message excluded: %v", messageChatId))
				}
			}

			// Save users between updates receiving.
			_ = saveUsers(p)
		}

		time.Sleep(1 * time.Second)
	}
}

func receiveState(p *Plugin) {
	listener := p.TdlibClient.GetListener()

	for {
		for update := range listener.Updates {
			switch update.(type) {
			case *client.UpdateConnectionState:
				switch update.(*client.UpdateConnectionState).State.ConnectionStateType() {
				case "connectionStateConnecting":
					p.ConnectionState = "connecting"
				case "connectionStateConnectingToProxy":
					p.ConnectionState = "connecting to proxy"
				case "connectionStateReady":
					p.ConnectionState = "ready"
				case "connectionStateUpdating":
					p.ConnectionState = "updating"
				case "connectionStateWaitingForNetwork":
					p.ConnectionState = "waiting for network"
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func saveChats(p *Plugin) error {
	return core.PluginSaveData(filepath.Join(p.PluginDataDir, DEFAULT_CHATS_DATA), p.ChatsByName)
}

func saveUsers(p *Plugin) error {
	return core.PluginSaveData(filepath.Join(p.PluginDataDir, DEFAULT_USERS_DATA), p.UsersById)
}

func showStatus(p *Plugin) {
	for {
		session, sessionError := p.TdlibClient.GetActiveSessions()
		storage, storageError := p.TdlibClient.GetStorageStatisticsFast()

		if sessionError != nil || storageError != nil {
			core.LogInputPlugin(p.LogFields, "status", fmt.Errorf("session error: %v, storage error: %v", sessionError, storageError))
		} else {
			for _, s := range session.Sessions {
				if s.IsCurrent {
					msg := "database size: %v, files amount: %v, files size: %v, geo: %v, ip: %v, last active: %v, login date: %v, proxy: %v, saved chats: %v, saved users: %v, state: %v"
					info := fmt.Sprintf(msg, core.BytesToSize(storage.DatabaseSize), storage.FileCount,
						core.BytesToSize(storage.FilesSize), strings.ToLower(s.Country),
						s.Ip, time.Unix(int64(s.LastActiveDate), 0), time.Unix(int64(s.LogInDate), 0),
						p.OptionProxyEnable, len(p.ChatsById), len(p.UsersById), p.ConnectionState)
					core.LogInputPlugin(p.LogFields, "status", info)
				}
			}
		}

		time.Sleep(time.Duration(p.OptionStatusPeriod) * time.Second)
	}
}

func storageOptimize(p *Plugin) {
	for {
		p.m.Lock()
		_, err := p.TdlibClient.OptimizeStorage(&client.OptimizeStorageRequest{})
		p.m.Unlock()

		if err != nil {
			core.LogInputPlugin(p.LogFields, "storage", fmt.Errorf("error: %v", err))
		}

		time.Sleep(time.Duration(p.OptionStoragePeriod) * time.Second)
	}
}

type Plugin struct {
	m sync.Mutex

	Flow *core.Flow

	LogFields log.Fields

	PluginName string
	PluginType string

	PluginDataDir string
	PluginTempDir string

	FileChannel chan int32
	DataChannel chan *core.DataItem

	ChatsById   map[int64]string
	ChatsByName map[string]int64
	UsersById   map[int64][]string

	TdlibClient *client.Client
	TdlibParams *client.TdlibParameters

	ConnectionState string

	OptionAdsEnable           bool
	OptionAdsPeriod           int64
	OptionApiHash             string
	OptionApiId               int
	OptionAppVersion          string
	OptionDeviceModel         string
	OptionExpireAction        []string
	OptionExpireActionDelay   int64
	OptionExpireActionTimeout int
	OptionExpireInterval      int64
	OptionExpireLast          int64
	OptionFetchTimeout        int
	OptionFileMaxSize         int64
	OptionFilePath            string
	OptionForce               bool
	OptionForceCount          int
	OptionIgnoreFileName      bool
	OptionInput               []string
	OptionLogLevel            int
	OptionMatchSignature      []string
	OptionMatchTTL            time.Duration
	OptionOriginalFileName    bool
	OptionProxyEnable         bool
	OptionProxyPort           int
	OptionProxyServer         string
	OptionProxyUsername       string
	OptionProxyPassword       string
	OptionProxyType           string
	OptionSessionTTL          int
	OptionShowChat            bool
	OptionShowUser            bool
	OptionStatusEnable        bool
	OptionStatusPeriod        int64
	OptionStorageOptimize     bool
	OptionStoragePeriod       int64
	OptionTimeFormat          string
	OptionTimeZone            *time.Location
	OptionTimeout             int
}

func (p *Plugin) FlowLog(message interface{}) {
	f := make(map[string]interface{}, len(p.LogFields))

	for k, v := range p.LogFields {
		f[k] = v
	}

	_, ok := message.(error)

	if ok {
		f["error"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Warn(core.LOG_FLOW_WARN)
	} else {
		f["data"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Debug(core.LOG_FLOW_STAT)
	}
}

func (p *Plugin) GetInput() []string {
	return p.OptionInput
}

func (p *Plugin) GetName() string {
	return p.PluginName
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

func (p *Plugin) Receive() ([]*core.DataItem, error) {
	currentTime := time.Now().UTC()
	temp := make([]*core.DataItem, 0)
	p.LogFields["run"] = p.Flow.GetRunID()

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}
	core.LogInputPlugin(p.LogFields, "all", fmt.Sprintf("states loaded: %d", len(flowStates)))

	// Source stat.
	sourceNewStat := make(map[string]int32)
	sourceReceivedStat := make(map[string]int32)

	// Fixate channel length (channel changes length size in the loop).
	length := len(p.DataChannel)

	// Process only specific amount of messages from every source if force = true.
	var start = 0
	var end = length - 1

	if p.OptionForce {
		if length > p.OptionForceCount {
			end = start + p.OptionForceCount - 1
		}
	}

	for i := start; i <= end; i++ {
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
			for _, v := range p.OptionMatchSignature {
				switch v {
				case "messagetext":
					itemSignature += item.TELEGRAM.MESSAGETEXT
					break
				case "messageurl":
					itemSignature += item.TELEGRAM.MESSAGEURL
					break
				case "source":
					itemSignature += item.SOURCE
					break
				case "time":
					itemSignature += item.TIME.String()
					break
				}
			}

			// set default value for signature if user provided wrong values.
			if len(itemSignature) == 0 {
				itemSignature += item.TELEGRAM.MESSAGETEXT + item.TIME.String()
			}

			itemSignatureHash = core.HashString(&itemSignature)

			if _, ok := flowStates[itemSignatureHash]; !ok || p.OptionForce {
				// save item signature hash to state.
				flowStates[itemSignatureHash] = currentTime

				// update source timestamp.
				if item.TIME.Unix() > sourceLastTime.Unix() {
					sourceLastTime = item.TIME
				}

				itemNew = true
			}

		} else {
			// item time can be the same (multiple files in single message).
			if item.TIME.Unix() >= sourceLastTime.Unix() || p.OptionForce {
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
		core.LogInputPlugin(p.LogFields, source, fmt.Sprintf("last update: %v, received data: %d, new data: %d",
			flowStates[source], sourceReceivedStat[source], sourceNewStat[source]))
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

					core.LogInputPlugin(p.LogFields, source, fmt.Sprintf(
						"expire_action: command: %s, arguments: %v, output: %s, error: %v",
						cmd, args, output, err))
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

func (p *Plugin) SaveState(data map[string]time.Time) error {
	p.m.Lock()
	defer p.m.Unlock()

	return core.PluginSaveState(p.Flow.FlowStateDir, &data, p.OptionMatchTTL)
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Flow: pluginConfig.Flow,
		LogFields: log.Fields{
			"hash":   pluginConfig.Flow.FlowHash,
			"run":    pluginConfig.Flow.GetRunID(),
			"flow":   pluginConfig.Flow.FlowName,
			"file":   pluginConfig.Flow.FlowFile,
			"plugin": PLUGIN_NAME,
			"type":   pluginConfig.PluginType,
		},
		PluginName:       PLUGIN_NAME,
		PluginType:       pluginConfig.PluginType,
		PluginDataDir:    filepath.Join(pluginConfig.Flow.FlowDataDir, pluginConfig.PluginType, PLUGIN_NAME),
		PluginTempDir:    filepath.Join(pluginConfig.Flow.FlowTempDir, pluginConfig.PluginType, PLUGIN_NAME),
		ConnectionState:  "unknown",
		OptionExpireLast: 0,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"cred":                  -1,
		"expire_action":         -1,
		"expire_action_timeout": -1,
		"expire_delay":          -1,
		"expire_interval":       -1,
		"force":                 -1,
		"force_count":           -1,
		"template":              -1,
		"timeout":               -1,
		"time_format":           -1,
		"time_zone":             -1,

		"ads_enable":        -1,
		"ads_period":        -1,
		"api_id":            1,
		"api_hash":          1,
		"app_version":       -1,
		"device_model":      -1,
		"fetch_timeout":     -1,
		"file_max_size":     -1,
		"file_path":         -1,
		"input":             1,
		"log_level":         -1,
		"match_signature":   -1,
		"match_ttl":         -1,
		"proxy_enable":      -1,
		"proxy_port":        -1,
		"proxy_server":      -1,
		"proxy_type":        -1,
		"show_chat":         -1,
		"show_user":         -1,
		"status_enable":     -1,
		"status_period":     -1,
		"storage_optimize":  -1,
		"storage_period":    -1,
		"original_filename": -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	cred, _ := core.IsString((*pluginConfig.PluginParams)["cred"])
	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// -----------------------------------------------------------------------------------------------------------------

	// proxy_username.
	setProxyUsername := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["proxy_username"] = 0
			plugin.OptionProxyUsername = v
		}
	}
	setProxyUsername(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.proxy_userrname", cred)))
	setProxyUsername((*pluginConfig.PluginParams)["proxy_username"])

	// proxy_password.
	setProxyPassword := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["proxy_password"] = 0
			plugin.OptionProxyPassword = v
		}
	}
	setProxyPassword(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.proxy_password", cred)))
	setProxyPassword((*pluginConfig.PluginParams)["proxy_password"])

	// -----------------------------------------------------------------------------------------------------------------

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

	// ads_enable.
	setAdsEnable := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["ads_enable"] = 0
			plugin.OptionAdsEnable = v
		}
	}
	setAdsEnable(DEFAULT_ADS_ENABLE)
	setAdsEnable(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.ads_enable", template)))
	setAdsEnable((*pluginConfig.PluginParams)["ads_enable"])
	core.ShowPluginParam(plugin.LogFields, "ads_enable", plugin.OptionAdsEnable)

	// ads_period.
	setAdsPeriod := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["ads_period"] = 0
			plugin.OptionAdsPeriod = v
		}
	}
	setAdsPeriod(DEFAULT_ADS_PERIOD)
	setAdsPeriod(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.ads_period", template)))
	setAdsPeriod((*pluginConfig.PluginParams)["ads_period"])
	core.ShowPluginParam(plugin.LogFields, "ads_period", plugin.OptionAdsPeriod)

	// app_version.
	setAppVersion := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["app_version"] = 0
			plugin.OptionAppVersion = v
		}
	}
	setAppVersion(core.APP_VERSION)
	setAppVersion(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.app_version", template)))
	setAppVersion((*pluginConfig.PluginParams)["app_version"])
	core.ShowPluginParam(plugin.LogFields, "app_version", plugin.OptionAppVersion)

	// device_model.
	setDeviceModel := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["device_model"] = 0
			plugin.OptionDeviceModel = v
		}
	}
	setDeviceModel(core.APP_NAME)
	setDeviceModel(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.device_model", template)))
	setDeviceModel((*pluginConfig.PluginParams)["device_model"])
	core.ShowPluginParam(plugin.LogFields, "device_model", plugin.OptionDeviceModel)

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
	core.ShowPluginParam(plugin.LogFields, "expire_action", plugin.OptionExpireAction)

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
	core.ShowPluginParam(plugin.LogFields, "expire_action_delay", plugin.OptionExpireActionDelay)

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
	core.ShowPluginParam(plugin.LogFields, "expire_action_timeout", plugin.OptionExpireActionTimeout)

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
	core.ShowPluginParam(plugin.LogFields, "expire_interval", plugin.OptionExpireInterval)

	// fetch_timeout.
	setFetchTimeout := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["fetch_timeout"] = 0
			plugin.OptionFetchTimeout = int(v)
		}
	}
	setFetchTimeout(DEFAULT_FETCH_TIMEOUT)
	setFetchTimeout(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.fetch_timeout", template)))
	setFetchTimeout((*pluginConfig.PluginParams)["fetch_timeout"])
	core.ShowPluginParam(plugin.LogFields, "fetch_timeout", plugin.OptionFetchTimeout)

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
	core.ShowPluginParam(plugin.LogFields, "file_max_size", plugin.OptionFileMaxSize)

	// file_path.
	setFilePath := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["file_path"] = 0
			plugin.OptionFilePath = v
		}
	}
	setFilePath(filepath.Join(plugin.PluginDataDir, DEFAULT_FILES_DIR))
	setFilePath(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_path", template)))
	setFilePath((*pluginConfig.PluginParams)["file_path"])
	core.ShowPluginParam(plugin.LogFields, "file_path", plugin.OptionFilePath)

	// force.
	setForce := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["force"] = 0
			plugin.OptionForce = v
		}
	}
	setForce(core.DEFAULT_FORCE_INPUT)
	setForce(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.force", template)))
	setForce((*pluginConfig.PluginParams)["force"])
	core.ShowPluginParam(plugin.LogFields, "force", plugin.OptionForce)

	// force_count.
	setForceCount := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["force_count"] = 0
			plugin.OptionForceCount = v
		}
	}
	setForceCount(core.DEFAULT_FORCE_COUNT)
	setForceCount(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.force_count", template)))
	setForceCount((*pluginConfig.PluginParams)["force_count"])
	core.ShowPluginParam(plugin.LogFields, "force_count", plugin.OptionForceCount)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.OptionInput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.input", template)))
	setInput((*pluginConfig.PluginParams)["input"])
	core.ShowPluginParam(plugin.LogFields, "input", plugin.OptionInput)

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
	core.ShowPluginParam(plugin.LogFields, "log_level", plugin.OptionLogLevel)

	// match_signature.
	setMatchSignature := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["match_signature"] = 0
			plugin.OptionMatchSignature = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setMatchSignature(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.match_signature", template)))
	setMatchSignature((*pluginConfig.PluginParams)["match_signature"])
	core.ShowPluginParam(plugin.LogFields, "match_signature", plugin.OptionMatchSignature)

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
	core.ShowPluginParam(plugin.LogFields, "match_ttl", plugin.OptionMatchTTL)

	// original_filename.
	setOriginalFileName := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["original_filename"] = 0
			plugin.OptionOriginalFileName = v
		}
	}
	setOriginalFileName(DEFAULT_ORIGINAL_FILENAME)
	setOriginalFileName(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.original_filename", template)))
	setOriginalFileName((*pluginConfig.PluginParams)["original_filename"])
	core.ShowPluginParam(plugin.LogFields, "original_filename", plugin.OptionOriginalFileName)

	if plugin.OptionOriginalFileName {
		plugin.OptionIgnoreFileName = false
	} else {
		plugin.OptionIgnoreFileName = true
	}

	// proxy_enable.
	setProxyEnable := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["proxy_enable"] = 0
			plugin.OptionProxyEnable = v
		}
	}
	setProxyEnable(DEFAULT_PROXY_ENABLE)
	setProxyEnable(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.proxy_enable", template)))
	setProxyEnable((*pluginConfig.PluginParams)["proxy_enable"])
	core.ShowPluginParam(plugin.LogFields, "proxy_enable", plugin.OptionProxyEnable)

	// proxy_port.
	setProxyPort := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["proxy_port"] = 0
			plugin.OptionProxyPort = v
		}
	}
	setProxyPort(DEFAULT_PROXY_PORT)
	setProxyPort(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.proxy_port", template)))
	setProxyPort((*pluginConfig.PluginParams)["proxy_port"])
	core.ShowPluginParam(plugin.LogFields, "proxy_port", plugin.OptionProxyPort)

	// proxy_server.
	setProxyServer := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["proxy_server"] = 0
			plugin.OptionProxyServer = v
		}
	}
	setProxyServer(DEFAULT_PROXY_SERVER)
	setProxyServer(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.proxy_server", template)))
	setProxyServer((*pluginConfig.PluginParams)["proxy_server"])
	core.ShowPluginParam(plugin.LogFields, "proxy_server", plugin.OptionProxyServer)

	// proxy_type.
	setProxyType := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["proxy_type"] = 0
			plugin.OptionProxyType = v
		}
	}
	setProxyType(DEFAULT_PROXY_TYPE)
	setProxyType(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.proxy_type", template)))
	setProxyType((*pluginConfig.PluginParams)["proxy_type"])
	core.ShowPluginParam(plugin.LogFields, "proxy_type", plugin.OptionProxyType)

	// show_chat.
	setShowChat := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["show_chat"] = 0
			plugin.OptionShowChat = v
		}
	}
	setShowChat(DEFAULT_SHOW_CHAT)
	setShowChat(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.show_chat", template)))
	setShowChat((*pluginConfig.PluginParams)["show_chat"])
	core.ShowPluginParam(plugin.LogFields, "show_chat", plugin.OptionShowChat)

	// show_user.
	setShowUser := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["show_user"] = 0
			plugin.OptionShowUser = v
		}
	}
	setShowUser(DEFAULT_SHOW_USER)
	setShowUser(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.show_user", template)))
	setShowUser((*pluginConfig.PluginParams)["show_user"])
	core.ShowPluginParam(plugin.LogFields, "show_user", plugin.OptionShowUser)

	// status_enable.
	setStatusEnable := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["status_enable"] = 0
			plugin.OptionStatusEnable = v
		}
	}
	setStatusEnable(DEFAULT_STATUS_ENABLE)
	setStatusEnable(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.status_enable", template)))
	setStatusEnable((*pluginConfig.PluginParams)["status_enable"])
	core.ShowPluginParam(plugin.LogFields, "status_enable", plugin.OptionStatusEnable)

	// status_period.
	setStatusPeriod := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["status_period"] = 0
			plugin.OptionStatusPeriod = v
		}
	}
	setStatusPeriod(DEFAULT_STATUS_PERIOD)
	setStatusPeriod(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.status_period", template)))
	setStatusPeriod((*pluginConfig.PluginParams)["status_period"])
	core.ShowPluginParam(plugin.LogFields, "status_period", plugin.OptionStatusPeriod)

	// storage_optimize.
	setStorageOptimize := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["storage_optimize"] = 0
			plugin.OptionStorageOptimize = v
		}
	}
	setStorageOptimize(DEFAULT_STORAGE_OPTIMIZE)
	setStorageOptimize(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.storage_optimize", template)))
	setStorageOptimize((*pluginConfig.PluginParams)["storage_optimize"])
	core.ShowPluginParam(plugin.LogFields, "storage_optimize", plugin.OptionStorageOptimize)

	// storage_period.
	setStoragePeriod := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["storage_period"] = 0
			plugin.OptionStoragePeriod = v
		}
	}
	setStoragePeriod(DEFAULT_STATUS_PERIOD)
	setStoragePeriod(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.storage_period", template)))
	setStoragePeriod((*pluginConfig.PluginParams)["storage_period"])
	core.ShowPluginParam(plugin.LogFields, "storage_period", plugin.OptionStoragePeriod)

	// TODO: Do we really need timeout for telegram event model ?
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
	core.ShowPluginParam(plugin.LogFields, "timeout", plugin.OptionTimeout)

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
	core.ShowPluginParam(plugin.LogFields, "time_format", plugin.OptionTimeFormat)

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
	core.ShowPluginParam(plugin.LogFields, "time_zone", plugin.OptionTimeZone)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if plugin.OptionProxyType != "socks" && plugin.OptionProxyType != "http" {
		return &Plugin{}, fmt.Errorf(ERROR_PROXY_TYPE_UNKNOWN.Error(), plugin.OptionProxyType)
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Telegram.

	plugin.TdlibParams = &client.TdlibParameters{
		ApiHash:                plugin.OptionApiHash,
		ApiId:                  int32(plugin.OptionApiId),
		ApplicationVersion:     plugin.OptionAppVersion,
		DatabaseDirectory:      filepath.Join(plugin.PluginDataDir, DEFAULT_DATABASE_DIR),
		DeviceModel:            plugin.OptionDeviceModel,
		EnableStorageOptimizer: plugin.OptionStorageOptimize,
		FilesDirectory:         plugin.OptionFilePath,
		IgnoreFileNames:        plugin.OptionIgnoreFileName,
		SystemLanguageCode:     "en",
		SystemVersion:          plugin.Flow.FlowName,
		UseChatInfoDatabase:    true,
		UseFileDatabase:        true,
		UseMessageDatabase:     true,
		UseSecretChats:         true,
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

	// 1. Check if we know IDs for all specified chats.
	// 2. We keep chats/user IDs due Telegram API limits.
	// 3. We may be banned for 24 hours if limits were reached (~200 requests might be enough).
	for _, chatName := range plugin.OptionInput {
		var chatId int64
		var err error

		if id, ok := chatsByName[chatName]; !ok {
			if strings.Contains(chatName, "t.me/+") {
				chatId, err = getPrivateChatId(&plugin, chatName)
			} else {
				chatId, err = getPublicChatId(&plugin, chatName)
			}

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
				// Don't accept any joining errors.
				return &Plugin{}, fmt.Errorf(ERROR_CHAT_COMMON_ERROR.Error(), chatName, err)
			}

		} else {
			// Always join to chat.
			err = joinToChat(&plugin, chatName, id)

			// Try to rejoin to chat if something wrong.
			if err != nil {
				if strings.Contains(chatName, "t.me/+") {
					chatId, err = getPrivateChatId(&plugin, chatName)
				} else {
					chatId, err = getPublicChatId(&plugin, chatName)
				}

				if err == nil {
					chatsByName[chatName] = chatId
					chatsById[chatId] = chatName
				} else {
					return &Plugin{}, err
				}

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

	// Get messages and files in background.
	plugin.FileChannel = make(chan int32, DEFAULT_BUFFER_LENGHT)
	plugin.DataChannel = make(chan *core.DataItem, DEFAULT_BUFFER_LENGHT)

	// Show chats.
	if plugin.OptionShowChat {
		core.LogInputPlugin(plugin.LogFields, "chats records", len(plugin.ChatsByName))
		for chatName, chatId := range plugin.ChatsByName {
			core.LogInputPlugin(plugin.LogFields, "chat",
				fmt.Sprintf("%v, %v", chatName, chatId))
		}
	}

	// Show users.
	if plugin.OptionShowUser {
		core.LogInputPlugin(plugin.LogFields, "users records", len(plugin.UsersById))
		for userId, userData := range plugin.UsersById {
			core.LogInputPlugin(plugin.LogFields, "user",
				fmt.Sprintf("%v, %v, %v, %v, %v, %v", userData[0], userId,
					userData[1], userData[2], userData[3], userData[4]))
		}
	}

	// Run main threads.
	go receiveFiles(&plugin)

	go receiveMessages(&plugin)

	go receiveState(&plugin)

	if plugin.OptionAdsEnable {
		go receiveAds(&plugin)
	}

	if plugin.OptionStatusEnable {
		go showStatus(&plugin)
	}

	if plugin.OptionStorageOptimize {
		go storageOptimize(&plugin)
	}
	// -------------------------------------------------------------------------

	return &plugin, nil
}
