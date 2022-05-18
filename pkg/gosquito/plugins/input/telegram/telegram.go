package telegramIn

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/livelace/go-tdlib/client"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	_ "github.com/mattn/go-sqlite3"
)

const (
	PLUGIN_NAME = "telegram"

	DEFAULT_ADS_ENABLE       = true
	DEFAULT_ADS_ID           = "ads"
	DEFAULT_ADS_PERIOD       = "5m"
	DEFAULT_CHANNEL_SIZE     = 10000
	DEFAULT_CHATS_DB         = "chats.sqlite"
	DEFAULT_CHAT_LOG         = false
	DEFAULT_DATABASE_DIR     = "database"
	DEFAULT_FETCH_ALL        = true
	DEFAULT_FETCH_MAX_SIZE   = "10m"
	DEFAULT_FETCH_METADATA   = false
	DEFAULT_FETCH_OTHER      = false
	DEFAULT_FETCH_TIMEOUT    = "1h"
	DEFAULT_FILE_DIR         = "files"
	DEFAULT_FILE_ORIG_NAME   = true
	DEFAULT_INCLUDE_ALL      = true
	DEFAULT_INCLUDE_OTHER    = false
	DEFAULT_LOG_LEVEL        = 0
	DEFAULT_MATCH_TTL        = "1d"
	DEFAULT_POOL_SIZE        = 100000
	DEFAULT_PROXY_ENABLE     = false
	DEFAULT_PROXY_PORT       = 9050
	DEFAULT_PROXY_SERVER     = "127.0.0.1"
	DEFAULT_PROXY_TYPE       = "socks"
	DEFAULT_SESSION_TTL      = 366
	DEFAULT_SHOW_CHAT        = false
	DEFAULT_SHOW_USER        = false
	DEFAULT_STATUS_ENABLE    = true
	DEFAULT_STATUS_PERIOD    = "1h"
	DEFAULT_STORAGE_OPTIMIZE = true
	DEFAULT_STORAGE_PERIOD   = "1h"
	DEFAULT_USERS_DB         = "users.sqlite"
	DEFAULT_USER_LOG         = true

	MAX_INSTANCE_PER_APP = 1

	SQL_FIND_CHAT = `
      SELECT * FROM chats WHERE name=?
    `

	SQL_COUNT_USER = `
      SELECT count(DISTINCT id) FROM users
    `

	SQL_FIND_USER = `
      SELECT * FROM users WHERE id=? ORDER BY version DESC LIMIT 1
    `

	SQL_UPDATE_CHAT = `
      INSERT INTO chats (id, name, type, title, 
        client_data, has_protected_content,
        last_inbox_id, last_outbox_id, message_ttl,
        unread_count, timestamp
      ) 
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(id) DO UPDATE SET
        name=?, type=?, title=?, 
        client_data=?, has_protected_content=?,
        last_inbox_id=?, last_outbox_id=?, message_ttl=?,
        unread_count=?, timestamp=?
    `

	SQL_UPDATE_USER = `
      INSERT INTO users (id, version, username, type, lang, 
        first_name, last_name, phone_number, status, 
        is_accessible, is_contact, is_fake, is_mutual_contact, 
        is_scam, is_support, is_verified, restriction_reason, 
        timestamp
      ) 
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	SQL_CHATS_SCHEMA = `
      CREATE TABLE IF NOT EXISTS chats (
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL,
        type TEXT NOT NULL,
        title TEXT,
        client_data TEXT,
        has_protected_content INTEGER NOT NULL,
        last_inbox_id INTEGER NOT NULL,
        last_outbox_id INTEGER NOT NULL,
        message_ttl INTEGER NOT NULL,
        unread_count INTEGER NOT NULL,
        timestamp TEXT NOT NULL,
        UNIQUE(id, name)
      )
    `

	SQL_USERS_SCHEMA = `
      CREATE TABLE IF NOT EXISTS users (
        id INTEGER NOT NULL,
        version INTEGER NOT NULL,
        username TEXT,
        type TEXT NOT NULL,
        lang TEXT,
        first_name TEXT,
        last_name TEXT,
        phone_number TEXT,
        status TEXT NOT NULL,
        is_accessible INTEGER NOT NULL,
        is_contact INTEGER NOT NULL,
        is_fake INTEGER NOT NULL,
        is_mutual_contact INTEGER NOT NULL,
        is_scam INTEGER NOT NULL,
        is_support INTEGER NOT NULL,
        is_verified INTEGER NOT NULL,
        restriction_reason TEXT,
        timestamp TEXT NOT NULL,
        UNIQUE(id, version)
      )
    `
)

var (
	ERROR_CHAT_COMMON_ERROR     = errors.New("chat error: %v, %v")
	ERROR_CHAT_GET_ERROR        = errors.New("cannot get chat: %v, %v")
	ERROR_CHAT_JOIN_ERROR       = errors.New("join chat error: %d, %v, %v")
	ERROR_FETCH_ERROR           = errors.New("fetch error: %v")
	ERROR_FETCH_TIMEOUT         = errors.New("fetch timeout: %v")
	ERROR_FILE_SIZE_EXCEEDED    = errors.New("file size exceeded: %v (%v > %v)")
	ERROR_LOAD_USERS_ERROR      = errors.New("cannot load users: %v")
	ERROR_NO_CHATS              = errors.New("no chats!")
	ERROR_PROXY_TYPE_UNKNOWN    = errors.New("proxy type unknown: %v")
	ERROR_SAVE_CHATS_ERROR      = errors.New("cannot save chats: %v")
	ERROR_SQL_BEGIN_TRANSACTION = errors.New("cannot start transaction: %v, %v")
	ERROR_SQL_DB_OPTION         = errors.New("chat or user database not set: %v, %v")
	ERROR_SQL_EXEC_ERROR        = errors.New("cannot execute query: %v, %v")
	ERROR_SQL_INIT_DB           = errors.New("cannot init database: %v, %v")
	ERROR_SQL_PREPARE_ERROR     = errors.New("cannot prepare query: %v, %v")
	ERROR_STATUS_ERROR          = errors.New("session error: %v, storage error: %v")
	ERROR_USER_UPDATE_ERROR     = errors.New("cannot save user: %v")
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

func countUsers(p *Plugin) int {
	count := 0
	stmt, _ := p.UsersDbClient.Prepare(SQL_COUNT_USER)
	defer stmt.Close()
	stmt.QueryRow().Scan(&count)
	return count
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
			if len(p.FileChannel) > 0 {
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
			}
			time.Sleep(1 * time.Second)
		}
		p.TdlibClient.CancelDownloadFile(&client.CancelDownloadFileRequest{FileId: remoteFile.Id})

		return "", fmt.Errorf(ERROR_FETCH_TIMEOUT.Error(), remoteId)

	} else {
		localFile = downloadFile.Local.Path
	}

stopWaiting:
	core.LogInputPlugin(p.LogFields, "fetch", fmt.Sprintf("end: %v -> %v", remoteId, localFile))

	return localFile, nil
}

func getChat(p *Plugin, chatName string) core.Telegram {
	d := core.Telegram{}

	stmt, _ := p.ChatsDbClient.Prepare(SQL_FIND_CHAT)
	defer stmt.Close()
	stmt.QueryRow(chatName).Scan(&d.CHATID, &d.CHATNAME,
		&d.CHATTYPE, &d.CHATTITLE, &d.CHATCLIENTDATA,
		&d.CHATPROTECTEDCONTENT, &d.CHATLASTINBOXID,
		&d.CHATLASTOUTBOXID, &d.CHATMESSAGETTL, &d.CHATUNREADCOUNT,
		&d.CHATTIMESTAMP,
	)

	return d
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
		return 0, fmt.Errorf(ERROR_CHAT_GET_ERROR.Error(), name, err)

	} else {
		return chat.Id, nil
	}
}

func getPublicChatId(p *Plugin, name string) (int64, error) {
	chat, err := p.TdlibClient.SearchPublicChat(&client.SearchPublicChatRequest{Username: name})
	if err != nil {
		return 0, fmt.Errorf(ERROR_CHAT_GET_ERROR.Error(), name, err)
	} else {
		return chat.Id, nil
	}
}

func getUser(p *Plugin, userId int64) core.Telegram {
	d := core.Telegram{}

	stmt, _ := p.UsersDbClient.Prepare(SQL_FIND_USER)
	defer stmt.Close()
	stmt.QueryRow(userId).Scan(&d.USERID, &d.USERVERSION, &d.USERNAME,
		&d.USERTYPE, &d.USERLANG, &d.USERFIRSTNAME, &d.USERLASTNAME,
		&d.USERPHONE, &d.USERSTATUS, &d.USERACCESSIBLE, &d.USERCONTACT,
		&d.USERFAKE, &d.USERMUTUALCONTACT, &d.USERSCAM, &d.USERSUPPORT,
		&d.USERVERIFIED, &d.USERRESTRICTION, &d.USERTIMESTAMP)

	return d
}

func initChatsDb(p *Plugin) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", p.OptionChatDatabase)
	_, err = db.Exec(SQL_CHATS_SCHEMA)
	if err != nil {
		return db, fmt.Errorf(ERROR_SQL_INIT_DB.Error(), p.OptionChatDatabase, err)
	}
	return db, err
}

func initUsersDb(p *Plugin) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", p.OptionUserDatabase)
	_, err = db.Exec(SQL_USERS_SCHEMA)

	if err != nil {
		return db, fmt.Errorf(ERROR_SQL_INIT_DB.Error(), p.OptionUserDatabase, err)
	}
	return db, err
}

func joinToChat(p *Plugin, chatId int64, chatName string) error {
	_, err := p.TdlibClient.JoinChat(&client.JoinChatRequest{ChatId: chatId})
	if err != nil {
		return fmt.Errorf(ERROR_CHAT_JOIN_ERROR.Error(), chatId, chatName, err)
	}
	return nil
}

func receiveAds(p *Plugin) {
	for {
		for chatId, chatData := range p.ChatsCache {
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
					p.DataChannel <- &core.DataItem{
						FLOW:       p.Flow.FlowName,
						PLUGIN:     p.PluginName,
						SOURCE:     chatData.CHATNAME,
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

		time.Sleep(time.Duration(p.OptionAdsPeriod) * time.Second)
	}
}

func receiveFiles(p *Plugin) {
	for {
		if len(p.FileListener.Updates) > 0 {
			update := <-p.FileListener.Updates

			switch update.(type) {
			case *client.UpdateFile:
				newFile := update.(*client.UpdateFile).File
				if newFile.Local.IsDownloadingCompleted || !newFile.Local.CanBeDownloaded {
					p.FileChannel <- newFile.Id
				}
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func receiveUpdates(p *Plugin) {
	for {
		if len(p.UpdateListener.Updates) > 0 {
			update := <-p.UpdateListener.Updates

			switch update.(type) {

			// Connection state.
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

				// Messages.
			case *client.UpdateNewMessage:
				dataItem := core.DataItem{}

				message := update.(*client.UpdateNewMessage)
				messageChat, _ := p.TdlibClient.GetChat(&client.GetChatRequest{ChatId: message.Message.ChatId})
				messageContent := message.Message.Content
				messageFileName := ""
				messageFileSize := int32(0)
				messageId := message.Message.Id
				messageSenderId := int64(-1)
				messageTimestamp := message.Message.Date
				messageTextURLs := make([]string, 0)
				messageTime := time.Unix(int64(message.Message.Date), 0).UTC()
				messageType := messageContent.MessageContentType()
				messageURL := ""

				userData := core.Telegram{}

				validMessage := false

				// Get message url.
				if v, err := p.TdlibClient.GetMessageLink(&client.GetMessageLinkRequest{ChatId: messageChat.Id, MessageId: messageId}); err == nil {
					messageURL = v.Link
				}

				// Get sender id, saved user.
				switch messageSender := message.Message.SenderId.(type) {
				case *client.MessageSenderChat:
					messageSenderId = int64(messageSender.ChatId)
				case *client.MessageSenderUser:
					messageSenderId = int64(messageSender.UserId)
					userData = getUser(p, messageSenderId)
				}

                // Save message chat.
				if p.OptionChatLog {
					// Just try to update chat. Chat can be already there with different name (unique error).
					err := updateChat(p, messageChat.Id, messageChat.Title)
					if err != nil {
						core.LogInputPlugin(p.LogFields, "chat",
							fmt.Sprintf("cannnot log chat: %v, %v, %v, %v",
								messageChat.Id, messageChat.Type.ChatTypeType(), messageChat.Title, err))
					}
				}

				// Process only specified chats.
				if chatData, ok := p.ChatsCache[messageChat.Id]; ok {
					var u, _ = uuid.NewRandom()

					dataItem = core.DataItem{
						FLOW:       p.Flow.FlowName,
						PLUGIN:     p.PluginName,
						SOURCE:     chatData.CHATNAME,
						TIME:       messageTime,
						TIMEFORMAT: messageTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
						UUID:       u,

						TELEGRAM: core.Telegram{
							CHATID:               chatData.CHATID,
							CHATNAME:             chatData.CHATNAME,
							CHATTYPE:             chatData.CHATTYPE,
							CHATTITLE:            chatData.CHATTITLE,
							CHATCLIENTDATA:       chatData.CHATCLIENTDATA,
							CHATPROTECTEDCONTENT: chatData.CHATPROTECTEDCONTENT,
							CHATLASTINBOXID:      chatData.CHATLASTINBOXID,
							CHATLASTOUTBOXID:     chatData.CHATLASTOUTBOXID,
							CHATMESSAGETTL:       chatData.CHATMESSAGETTL,
							CHATUNREADCOUNT:      chatData.CHATUNREADCOUNT,
							CHATTIMESTAMP:        chatData.CHATTIMESTAMP,

							MESSAGEID:        fmt.Sprintf("%v", messageId),
							MESSAGEMEDIA:     make([]string, 0),
							MESSAGESENDERID:  fmt.Sprintf("%v", messageSenderId),
							MESSAGETYPE:      messageType,
							MESSAGETEXT:      "",
							MESSAGETEXTURL:   messageTextURLs,
							MESSAGETIMESTAMP: fmt.Sprintf("%v", messageTimestamp),
							MESSAGEURL:       messageURL,

							USERID:            userData.USERID,
							USERVERSION:       userData.USERVERSION,
							USERNAME:          userData.USERNAME,
							USERTYPE:          userData.USERTYPE,
							USERLANG:          userData.USERLANG,
							USERFIRSTNAME:     userData.USERFIRSTNAME,
							USERLASTNAME:      userData.USERLASTNAME,
							USERPHONE:         userData.USERPHONE,
							USERSTATUS:        userData.USERSTATUS,
							USERACCESSIBLE:    userData.USERACCESSIBLE,
							USERCONTACT:       userData.USERCONTACT,
							USERFAKE:          userData.USERFAKE,
							USERMUTUALCONTACT: userData.USERMUTUALCONTACT,
							USERSCAM:          userData.USERSCAM,
							USERSUPPORT:       userData.USERSUPPORT,
							USERVERIFIED:      userData.USERVERIFIED,
							USERRESTRICTION:   userData.USERRESTRICTION,
							USERTIMESTAMP:     userData.USERTIMESTAMP,
						},

                        WARNINGS: make([]string, 0),
					}

					switch messageContent.(type) {
					case *client.MessageAudio:
						if p.OptionProcessAll || p.OptionProcessAudio {
							audio := messageContent.(*client.MessageAudio).Audio
							dataItem.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessageAudio).Caption.Text

							if (p.OptionFetchAll || p.OptionFetchAudio) && int64(audio.Audio.Size) < p.OptionFetchMaxSize {
								localFile, err := downloadFile(p, audio.Audio.Remote.Id, audio.FileName)

								if err == nil && p.OptionFetchMetadata {
									writeMetadata(p, localFile, &dataItem.TELEGRAM)
								} else if err == nil {
									dataItem.TELEGRAM.MESSAGEMEDIA = append(dataItem.TELEGRAM.MESSAGEMEDIA, localFile)
								}
							}

							if (p.OptionFetchAll || p.OptionFetchAudio) && int64(audio.Audio.Size) > p.OptionFetchMaxSize {
								messageFileName = audio.FileName
								messageFileSize = audio.Audio.Size
							}

							validMessage = true
						}

					case *client.MessageDocument:
						if p.OptionProcessAll || p.OptionProcessDocument {
							document := messageContent.(*client.MessageDocument).Document
							dataItem.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessageDocument).Caption.Text

							if (p.OptionFetchAll || p.OptionFetchDocument) && int64(document.Document.Size) < p.OptionFetchMaxSize {
								localFile, err := downloadFile(p, document.Document.Remote.Id, document.FileName)

								if err == nil && p.OptionFetchMetadata {
									writeMetadata(p, localFile, &dataItem.TELEGRAM)
								} else if err == nil {
									dataItem.TELEGRAM.MESSAGEMEDIA = append(dataItem.TELEGRAM.MESSAGEMEDIA, localFile)
								}
							}

							if (p.OptionFetchAll || p.OptionFetchDocument) && int64(document.Document.Size) > p.OptionFetchMaxSize {
								messageFileName = document.FileName
								messageFileSize = document.Document.Size
							}

							validMessage = true
						}

					case *client.MessagePhoto:
						if p.OptionProcessAll || p.OptionProcessPhoto {
							photo := messageContent.(*client.MessagePhoto).Photo
							photoFile := photo.Sizes[len(photo.Sizes)-1]
							dataItem.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessagePhoto).Caption.Text

							if (p.OptionFetchAll || p.OptionFetchPhoto) && int64(photoFile.Photo.Size) < p.OptionFetchMaxSize {
								localFile, err := downloadFile(p, photoFile.Photo.Remote.Id, "")

								if err == nil && p.OptionFetchMetadata {
									writeMetadata(p, localFile, &dataItem.TELEGRAM)
								} else if err == nil {
									dataItem.TELEGRAM.MESSAGEMEDIA = append(dataItem.TELEGRAM.MESSAGEMEDIA, localFile)
								}
							}

							if (p.OptionFetchAll || p.OptionFetchPhoto) && int64(photoFile.Photo.Size) > p.OptionFetchMaxSize {
								messageFileName = "photo"
								messageFileSize = photoFile.Photo.Size
							}

							validMessage = true
						}

					case *client.MessageText:
						if p.OptionProcessAll || p.OptionProcessText {
							formattedText := messageContent.(*client.MessageText).Text
							dataItem.TELEGRAM.MESSAGETEXT = formattedText.Text

							for _, entity := range formattedText.Entities {
								switch entity.Type.(type) {
								case *client.TextEntityTypeTextUrl:
									messageTextURLs = append(messageTextURLs, entity.Type.(*client.TextEntityTypeTextUrl).Url)
								}
							}

							validMessage = true
						}

					case *client.MessageVideo:
						if p.OptionProcessAll || p.OptionProcessVideo {
							dataItem.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessageVideo).Caption.Text
							video := messageContent.(*client.MessageVideo).Video

							if (p.OptionFetchAll || p.OptionFetchVideo) && int64(video.Video.Size) < p.OptionFetchMaxSize {
								localFile, err := downloadFile(p, video.Video.Remote.Id, video.FileName)

								if err == nil && p.OptionFetchMetadata {
									writeMetadata(p, localFile, &dataItem.TELEGRAM)
								} else if err == nil {
									dataItem.TELEGRAM.MESSAGEMEDIA = append(dataItem.TELEGRAM.MESSAGEMEDIA, localFile)
								}
							}

							if (p.OptionFetchAll || p.OptionFetchVideo) && int64(video.Video.Size) > p.OptionFetchMaxSize {
								messageFileName = video.FileName
								messageFileSize = video.Video.Size
							}

							validMessage = true
						}

					case *client.MessageVideoNote:
						if p.OptionProcessAll || p.OptionProcessVideoNote {
							note := messageContent.(*client.MessageVideoNote).VideoNote

							if (p.OptionFetchAll || p.OptionFetchVideoNote) && int64(note.Video.Size) < p.OptionFetchMaxSize {
								localFile, err := downloadFile(p, note.Video.Remote.Id, "")

								if err == nil && p.OptionFetchMetadata {
									writeMetadata(p, localFile, &dataItem.TELEGRAM)
								} else if err == nil {
									dataItem.TELEGRAM.MESSAGEMEDIA = append(dataItem.TELEGRAM.MESSAGEMEDIA, localFile)
								}
							}

							if (p.OptionFetchAll || p.OptionFetchVideoNote) && int64(note.Video.Size) > p.OptionFetchMaxSize {
								messageFileName = "video_note"
								messageFileSize = note.Video.Size
							}

							validMessage = true
						}

					case *client.MessageVoiceNote:
						if p.OptionProcessAll || p.OptionProcessVoiceNote {
							dataItem.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessageVoiceNote).Caption.Text
							note := messageContent.(*client.MessageVoiceNote).VoiceNote

							if (p.OptionFetchAll || p.OptionFetchVoiceNote) && int64(note.Voice.Size) < p.OptionFetchMaxSize {
								localFile, err := downloadFile(p, note.Voice.Remote.Id, "")

								if err == nil && p.OptionFetchMetadata {
									writeMetadata(p, localFile, &dataItem.TELEGRAM)
								} else if err == nil {
									dataItem.TELEGRAM.MESSAGEMEDIA = append(dataItem.TELEGRAM.MESSAGEMEDIA, localFile)
								}
							}

							if (p.OptionFetchAll || p.OptionFetchVoiceNote) && int64(note.Voice.Size) > p.OptionFetchMaxSize {
								messageFileName = "voice_note"
								messageFileSize = note.Voice.Size
							}

							validMessage = true
						}
					}

					// Warnings.
					if messageFileSize > 0 {
						dataItem.WARNINGS = append(dataItem.WARNINGS, fmt.Sprintf(ERROR_FILE_SIZE_EXCEEDED.Error(),
							messageFileName, core.BytesToSize(int64(messageFileSize)), core.BytesToSize(p.OptionFetchMaxSize)))

						core.LogInputPlugin(p.LogFields, "", fmt.Sprintf(ERROR_FILE_SIZE_EXCEEDED.Error(),
							messageFileName, core.BytesToSize(int64(messageFileSize)), core.BytesToSize(p.OptionFetchMaxSize)))
					}

					// Send data to channel.
					if validMessage {
						p.DataChannel <- &dataItem
					}

				} else {
					core.LogInputPlugin(p.LogFields, "chat",
						fmt.Sprintf("filtered: %v, %v, %v", messageChat.Id, messageChat.Type.ChatTypeType(), messageChat.Title))

				}

			// Users.
			case *client.UpdateUser:
				if p.OptionUserLog {
					user := update.(*client.UpdateUser).User
					isNew, isChanged, version, err := updateUser(p, user)

					if err == nil {
						if isNew {
							core.LogInputPlugin(p.LogFields, "user",
								fmt.Sprintf("new: %v, version: %v, username: %v", user.Id, version, user.Username))
						} else {
							core.LogInputPlugin(p.LogFields, "user",
								fmt.Sprintf("old: %v, version: %v, username: %v", user.Id, version, user.Username))
						}

						if isChanged {
							core.LogInputPlugin(p.LogFields, "user",
								fmt.Sprintf("changed: %v, version: %v, username: %v", user.Id, version, user.Username))
						}
					} else {
						core.LogInputPlugin(p.LogFields, "user",
							fmt.Errorf(ERROR_USER_UPDATE_ERROR.Error(), err))
					}
				}
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func showStatus(p *Plugin) {
	for {
		session, sessionError := p.TdlibClient.GetActiveSessions()
		storage, storageError := p.TdlibClient.GetStorageStatisticsFast()

		if sessionError != nil || storageError != nil {
			core.LogInputPlugin(p.LogFields, "status",
				fmt.Errorf(ERROR_STATUS_ERROR.Error(), sessionError, storageError))
		} else {
			for _, s := range session.Sessions {
				if s.IsCurrent {
					m := []string{
						"database size: %v,",
						"files amount: %v,",
						"files size: %v,",
						"geo: %v,",
						"ip: %v,",
						"last active: %v,",
						"last state: %v,",
						"login date: %v,",
						"pool size: %v,",
						"proxy: %v,",
						"saved chats: %v,",
						"saved users: %v",
					}
					info := fmt.Sprintf(strings.Join(m, " "),
						core.BytesToSize(storage.DatabaseSize), storage.FileCount,
						core.BytesToSize(storage.FilesSize), strings.ToLower(s.Country),
						s.Ip, time.Unix(int64(s.LastActiveDate), 0),
						p.ConnectionState, time.Unix(int64(s.LogInDate), 0),
						len(p.UpdateListener.Updates), p.OptionProxyEnable,
						len(p.ChatsCache), countUsers(p),
					)

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

func updateChat(p *Plugin, chatId int64, chatName string) error {
	currentTime := time.Now().UTC().Format(time.RFC3339)
	tx, err := p.ChatsDbClient.Begin()
	if err != nil {
		return fmt.Errorf(ERROR_SQL_BEGIN_TRANSACTION.Error(), chatName, err)
	}

	chat, err := p.TdlibClient.GetChat(&client.GetChatRequest{ChatId: chatId})
	if err != nil {
		return fmt.Errorf(ERROR_CHAT_GET_ERROR.Error(), chatName, err)
	}

	stmt, err := p.ChatsDbClient.Prepare(SQL_UPDATE_CHAT)
	if err != nil {
		return fmt.Errorf(ERROR_SQL_PREPARE_ERROR.Error(), chatName, err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(chat.Id,
		chatName, chat.Type.ChatTypeType(),
		chat.Title, chat.ClientData, chat.HasProtectedContent,
		chat.LastReadInboxMessageId, chat.LastReadOutboxMessageId,
		chat.MessageTtl, chat.UnreadCount, currentTime,

		chatName, chat.Type.ChatTypeType(),
		chat.Title, chat.ClientData, chat.HasProtectedContent,
		chat.LastReadInboxMessageId, chat.LastReadOutboxMessageId,
		chat.MessageTtl, chat.UnreadCount, currentTime,
	)

	if err != nil {
		return fmt.Errorf(ERROR_SQL_EXEC_ERROR.Error(), chatName, err)
	}

	return tx.Commit()
}

func updateUser(p *Plugin, user *client.User) (bool, bool, int, error) {
	currentTime := time.Now().UTC().Format(time.RFC3339)
	oldUser := getUser(p, user.Id)
	userVersion := 0

	isNew := false
	isChanged := false

	if oldUser.USERID == "" {
		isNew = true
	} else {
		if user.Username != oldUser.USERNAME || user.Type.UserTypeType() != oldUser.USERTYPE ||
			user.LanguageCode != oldUser.USERLANG || user.FirstName != oldUser.USERFIRSTNAME ||
			user.LastName != oldUser.USERLASTNAME || user.PhoneNumber != oldUser.USERPHONE {
			isChanged = true
		}

		if b, s := core.IsBool(oldUser.USERACCESSIBLE); s && user.HaveAccess != b {
			isChanged = true
		}
		if b, s := core.IsBool(oldUser.USERCONTACT); s && user.IsContact != b {
			isChanged = true
		}
		if b, s := core.IsBool(oldUser.USERFAKE); s && user.IsFake != b {
			isChanged = true
		}
		if b, s := core.IsBool(oldUser.USERMUTUALCONTACT); s && user.IsMutualContact != b {
			isChanged = true
		}
		if b, s := core.IsBool(oldUser.USERSCAM); s && user.IsScam != b {
			isChanged = true
		}
		if b, s := core.IsBool(oldUser.USERSUPPORT); s && user.IsSupport != b {
			isChanged = true
		}
		if b, s := core.IsBool(oldUser.USERVERIFIED); s && user.IsVerified != b {
			isChanged = true
		}
		if user.RestrictionReason != oldUser.USERRESTRICTION {
			isChanged = true
		}

		oldVersion, _ := strconv.ParseInt(oldUser.USERVERSION, 10, 32)
		if isChanged {
			userVersion = int(oldVersion) + 1
		} else {
			userVersion = int(oldVersion)
		}
	}

	if isNew || isChanged {
		tx, err := p.UsersDbClient.Begin()
		if err != nil {
			return isNew, isChanged, userVersion, fmt.Errorf(ERROR_SQL_BEGIN_TRANSACTION.Error(), user.Username, err)
		}

		stmt, err := p.UsersDbClient.Prepare(SQL_UPDATE_USER)
		if err != nil {
			return isNew, isChanged, userVersion, fmt.Errorf(ERROR_SQL_PREPARE_ERROR.Error(), user.Username, err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(user.Id, userVersion, user.Username,
			user.Type.UserTypeType(), user.LanguageCode,
			user.FirstName, user.LastName, user.PhoneNumber,
			user.Status.UserStatusType(), user.HaveAccess,
			user.IsContact, user.IsFake, user.IsMutualContact,
			user.IsScam, user.IsSupport, user.IsVerified,
			user.RestrictionReason, currentTime,
		)

		if err != nil {
			return isNew, isChanged, userVersion, fmt.Errorf(ERROR_SQL_EXEC_ERROR.Error(), user.Username, err)
		}

		return isNew, isChanged, userVersion, tx.Commit()
	}

	return isNew, isChanged, userVersion, nil
}

func writeMetadata(p *Plugin, localFile string, data *core.Telegram) error {
	metaFile := fmt.Sprintf("%s.meta.json", localFile)
	data.MESSAGEMEDIA = append(data.MESSAGEMEDIA, localFile, metaFile)

	j, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return core.WriteStringToFile("", metaFile, string(j))
}

type Plugin struct {
	m sync.Mutex

	Flow *core.Flow

	LogFields log.Fields

	PluginName string
	PluginType string

	PluginDataDir string
	PluginTempDir string

	ConnectionState string

	FileListener   *client.Listener
	UpdateListener *client.Listener

	FileChannel chan int32
	DataChannel chan *core.DataItem

	ChatsCache map[int64]*core.Telegram

	ChatsDbClient *sql.DB
	UsersDbClient *sql.DB

	TdlibClient *client.Client
	TdlibParams *client.TdlibParameters

	OptionAdsEnable           bool
	OptionAdsPeriod           int64
	OptionApiHash             string
	OptionApiId               int
	OptionAppVersion          string
	OptionChatDatabase        string
	OptionChatLog             bool
	OptionDeviceModel         string
	OptionExpireAction        []string
	OptionExpireActionDelay   int64
	OptionExpireActionTimeout int
	OptionExpireInterval      int64
	OptionExpireLast          int64
	OptionFetchAll            bool
	OptionFetchAudio          bool
	OptionFetchDocument       bool
	OptionFetchMaxSize        int64
	OptionFetchMetadata       bool
	OptionFetchOrigName       bool
	OptionFetchPhoto          bool
	OptionFetchTimeout        int
	OptionFetchVideo          bool
	OptionFetchVideoNote      bool
	OptionFetchVoiceNote      bool
	OptionFilePath            string
	OptionForce               bool
	OptionForceCount          int
	OptionIgnoreFileName      bool
	OptionInput               []string
	OptionLogLevel            int
	OptionMatchSignature      []string
	OptionMatchTTL            time.Duration
	OptionMessageTypeFetch    []string
	OptionMessageTypeProcess  []string
	OptionPoolSize            int
	OptionProcessAll          bool
	OptionProcessAudio        bool
	OptionProcessDocument     bool
	OptionProcessPhoto        bool
	OptionProcessText         bool
	OptionProcessVideo        bool
	OptionProcessVideoNote    bool
	OptionProcessVoiceNote    bool
	OptionProxyEnable         bool
	OptionProxyPassword       string
	OptionProxyPort           int
	OptionProxyServer         string
	OptionProxyType           string
	OptionProxyUsername       string
	OptionSessionTTL          int
	OptionStatusEnable        bool
	OptionStatusPeriod        int64
	OptionStorageOptimize     bool
	OptionStoragePeriod       int64
	OptionTimeFormat          string
	OptionTimeZone            *time.Location
	OptionTimeout             int
	OptionUserDatabase        string
	OptionUserLog             bool
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
				case "TELEGRAM.MESSAGESENDERID":
					itemSignature += item.TELEGRAM.MESSAGESENDERID
					break
				case "TELEGRAM.MESSAGETEXT":
					itemSignature += item.TELEGRAM.MESSAGETEXT
					break
				case "TELEGRAM.MESSAGEURL":
					itemSignature += item.TELEGRAM.MESSAGEURL
					break
				}
			}

			// set default value for signature if user provided wrong values.
			if len(itemSignature) == 0 {
				itemSignature += item.TELEGRAM.CHATID
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
						"expire_action: command: %v, arguments: %v, output: %v, error: %v",
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
		PluginName:             PLUGIN_NAME,
		PluginType:             pluginConfig.PluginType,
		PluginDataDir:          filepath.Join(pluginConfig.Flow.FlowDataDir, pluginConfig.PluginType, PLUGIN_NAME),
		PluginTempDir:          filepath.Join(pluginConfig.Flow.FlowTempDir, pluginConfig.PluginType, PLUGIN_NAME),
		ConnectionState:        "unknown",
		OptionExpireLast:       0,
		OptionFetchAll:         DEFAULT_FETCH_ALL,
		OptionFetchAudio:       DEFAULT_FETCH_OTHER,
		OptionFetchDocument:    DEFAULT_FETCH_OTHER,
		OptionFetchPhoto:       DEFAULT_FETCH_OTHER,
		OptionFetchVideo:       DEFAULT_FETCH_OTHER,
		OptionFetchVideoNote:   DEFAULT_FETCH_OTHER,
		OptionFetchVoiceNote:   DEFAULT_FETCH_OTHER,
		OptionProcessAll:       DEFAULT_INCLUDE_ALL,
		OptionProcessAudio:     DEFAULT_INCLUDE_OTHER,
		OptionProcessDocument:  DEFAULT_INCLUDE_OTHER,
		OptionProcessPhoto:     DEFAULT_INCLUDE_OTHER,
		OptionProcessText:      DEFAULT_INCLUDE_OTHER,
		OptionProcessVideo:     DEFAULT_INCLUDE_OTHER,
		OptionProcessVideoNote: DEFAULT_INCLUDE_OTHER,
		OptionProcessVoiceNote: DEFAULT_INCLUDE_OTHER,
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

		"ads_enable":           -1,
		"ads_period":           1,
		"api_hash":             1,
		"api_id":               1,
		"app_version":          -1,
		"chat_database":        -1,
		"chat_log":             -1,
		"device_model":         -1,
		"fetch_max_size":       -1,
		"fetch_metadata":       -1,
		"fetch_orig_name":      -1,
		"fetch_timeout":        -1,
		"file_path":            -1,
		"input":                1,
		"log_level":            -1,
		"match_signature":      -1,
		"match_ttl":            -1,
		"message_type_fetch":   -1,
		"message_type_process": -1,
		"pool_size":            -1,
		"proxy_enable":         -1,
		"proxy_port":           -1,
		"proxy_server":         -1,
		"proxy_type":           -1,
		"session_ttl":          -1,
		"status_enable":        -1,
		"status_period":        -1,
		"storage_optimize":     -1,
		"storage_period":       -1,
		"user_database":        -1,
		"user_log":             -1,
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

	// chat_database.
	setChatDatabase := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["chat_database"] = 0
			plugin.OptionChatDatabase = v
		}
	}
	setChatDatabase(filepath.Join(plugin.PluginDataDir, DEFAULT_CHATS_DB))
	setChatDatabase(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.chat_database", template)))
	setChatDatabase((*pluginConfig.PluginParams)["chat_database"])
	core.ShowPluginParam(plugin.LogFields, "chat_database", plugin.OptionChatDatabase)

	// chat_log.
	setChatLog := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["chat_log"] = 0
			plugin.OptionChatLog = v
		}
	}
	setChatLog(DEFAULT_CHAT_LOG)
	setChatLog(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.chat_log", template)))
	setChatLog((*pluginConfig.PluginParams)["chat_log"])
	core.ShowPluginParam(plugin.LogFields, "chat_log", plugin.OptionChatLog)

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

	// fetch_max_size.
	setFetchMaxSize := func(p interface{}) {
		if v, b := core.IsSize(p); b {
			availableParams["fetch_max_size"] = 0
			plugin.OptionFetchMaxSize = v
		}
	}
	setFetchMaxSize(DEFAULT_FETCH_MAX_SIZE)
	setFetchMaxSize(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.fetch_max_size", template)))
	setFetchMaxSize((*pluginConfig.PluginParams)["fetch_max_size"])
	core.ShowPluginParam(plugin.LogFields, "fetch_max_size", plugin.OptionFetchMaxSize)

	// fetch_metadata.
	setFetchMetadata := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["fetch_metadata"] = 0
			plugin.OptionFetchMetadata = v
		}
	}
	setFetchMetadata(DEFAULT_FETCH_METADATA)
	setFetchMetadata(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.fetch_metadata", template)))
	setFetchMetadata((*pluginConfig.PluginParams)["fetch_metadata"])
	core.ShowPluginParam(plugin.LogFields, "fetch_metadata", plugin.OptionFetchMetadata)

	// fetch_orig_name.
	setFetchOrigName := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["fetch_orig_name"] = 0
			plugin.OptionFetchOrigName = v
		}
	}
	setFetchOrigName(DEFAULT_FILE_ORIG_NAME)
	setFetchOrigName(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.fetch_orig_name", template)))
	setFetchOrigName((*pluginConfig.PluginParams)["fetch_orig_name"])
	core.ShowPluginParam(plugin.LogFields, "fetch_orig_name", plugin.OptionFetchOrigName)

	if plugin.OptionFetchOrigName {
		plugin.OptionIgnoreFileName = false
	} else {
		plugin.OptionIgnoreFileName = true
	}

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

	// file_path.
	setFilePath := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["file_path"] = 0
			plugin.OptionFilePath = v
		}
	}
	setFilePath(filepath.Join(plugin.PluginDataDir, DEFAULT_FILE_DIR))
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
	core.SliceStringToUpper(&plugin.OptionMatchSignature)

	for i := 0; i < len(plugin.OptionMatchSignature); i++ {
		plugin.OptionMatchSignature[i] = strings.ToLower(plugin.OptionMatchSignature[i])
	}

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

	// message_type_fetch.
	setMessageTypeFetch := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["message_type_fetch"] = 0
			plugin.OptionMessageTypeFetch = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setMessageTypeFetch(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.message_type_fetch", template)))
	setMessageTypeFetch((*pluginConfig.PluginParams)["message_type_fetch"])
	core.ShowPluginParam(plugin.LogFields, "message_type_fetch", plugin.OptionMessageTypeFetch)
	core.SliceStringToUpper(&plugin.OptionMessageTypeFetch)

	if len(plugin.OptionMessageTypeFetch) > 0 {
		plugin.OptionFetchAll = false
		for _, v := range plugin.OptionMessageTypeFetch {
			switch v {
			case "AUDIO":
				plugin.OptionFetchAudio = true
			case "DOCUMENT":
				plugin.OptionFetchDocument = true
			case "PHOTO":
				plugin.OptionFetchPhoto = true
			case "VIDEO":
				plugin.OptionFetchVideo = true
			case "VIDEO_NOTE":
				plugin.OptionFetchVideoNote = true
			case "VOICE_NOTE":
				plugin.OptionFetchVoiceNote = true
			}
		}
	}

	// message_type_process.
	setMessageTypeProcess := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["message_type_process"] = 0
			plugin.OptionMessageTypeProcess = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setMessageTypeProcess(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.message_type_process", template)))
	setMessageTypeProcess((*pluginConfig.PluginParams)["message_type_process"])
	core.ShowPluginParam(plugin.LogFields, "message_type_process", plugin.OptionMessageTypeProcess)
	core.SliceStringToUpper(&plugin.OptionMessageTypeProcess)

	if len(plugin.OptionMessageTypeProcess) > 0 {
		plugin.OptionProcessAll = false
		for _, v := range plugin.OptionMessageTypeProcess {
			switch v {
			case "AUDIO":
				plugin.OptionProcessAudio = true
			case "DOCUMENT":
				plugin.OptionProcessDocument = true
			case "PHOTO":
				plugin.OptionProcessPhoto = true
			case "TEXT":
				plugin.OptionProcessText = true
			case "VIDEO":
				plugin.OptionProcessVideo = true
			case "VIDEO_NOTE":
				plugin.OptionProcessVideoNote = true
			case "VOICE_NOTE":
				plugin.OptionProcessVoiceNote = true
			}
		}
	}

	// pool_size.
	setPoolSize := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["pool_size"] = 0
			plugin.OptionPoolSize = v
		}
	}
	setPoolSize(DEFAULT_POOL_SIZE)
	setPoolSize(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.pool_size", template)))
	setPoolSize((*pluginConfig.PluginParams)["pool_size"])
	core.ShowPluginParam(plugin.LogFields, "pool_size", plugin.OptionPoolSize)

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

	// session_ttl.
	setSessionTTL := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["session_ttl"] = 0
			plugin.OptionSessionTTL = v
		}
	}
	setSessionTTL(DEFAULT_SESSION_TTL)
	setSessionTTL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.session_ttl", template)))
	setSessionTTL((*pluginConfig.PluginParams)["session_ttl"])
	core.ShowPluginParam(plugin.LogFields, "session_ttl", plugin.OptionSessionTTL)

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

	// user_database.
	setUserDatabase := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["user_database"] = 0
			plugin.OptionUserDatabase = v
		}
	}
	setUserDatabase(filepath.Join(plugin.PluginDataDir, DEFAULT_USERS_DB))
	setUserDatabase(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.user_database", template)))
	setUserDatabase((*pluginConfig.PluginParams)["user_database"])
	core.ShowPluginParam(plugin.LogFields, "user_database", plugin.OptionUserDatabase)

	// user_log.
	setUserLog := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["user_log"] = 0
			plugin.OptionUserLog = v
		}
	}
	setUserLog(DEFAULT_USER_LOG)
	setUserLog(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.user_log", template)))
	setUserLog((*pluginConfig.PluginParams)["user_log"])
	core.ShowPluginParam(plugin.LogFields, "user_log", plugin.OptionUserLog)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if plugin.OptionChatDatabase == "" && plugin.OptionUserDatabase == "" {
		return &Plugin{}, fmt.Errorf(ERROR_SQL_DB_OPTION.Error(),
			plugin.OptionChatDatabase, plugin.OptionUserDatabase)
	}

	if plugin.OptionProxyType != "socks" && plugin.OptionProxyType != "http" {
		return &Plugin{}, fmt.Errorf(ERROR_PROXY_TYPE_UNKNOWN.Error(),
			plugin.OptionProxyType)
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

	// Set session TTL.
	plugin.TdlibClient.SetInactiveSessionTtl(&client.SetInactiveSessionTtlRequest{InactiveSessionTtlDays: int32(plugin.OptionSessionTTL)})

	// Init chats database.
	plugin.ChatsDbClient, err = initChatsDb(&plugin)
	if err != nil {
		return &plugin, err
	}

	// Update and join to chats.
	plugin.ChatsCache = make(map[int64]*core.Telegram, len(plugin.OptionInput))

	for _, chatName := range plugin.OptionInput {
		var chatId int64
		var err error

		chatData := getChat(&plugin, chatName)

		if chatData.CHATID == "" {
			chatIdRegexp := regexp.MustCompile(`^-[0-9]+$`)

			if chatIdRegexp.Match([]byte(chatName)) {
				chatId, err = strconv.ParseInt(chatName, 10, 64)
			} else if strings.Contains(chatName, "t.me/+") {
				chatId, err = getPrivateChatId(&plugin, chatName)
			} else {
				chatId, err = getPublicChatId(&plugin, chatName)
			}

			if err != nil {
				core.LogInputPlugin(plugin.LogFields, "chat", err)
				continue
			}
		} else {
			chatId, _ = strconv.ParseInt(chatData.CHATID, 10, 64)
		}

		err = updateChat(&plugin, chatId, chatName)
		if err != nil {
			core.LogInputPlugin(plugin.LogFields, "chat", err)
			continue
		}

		err = joinToChat(&plugin, chatId, chatName)
		if err != nil {
			core.LogInputPlugin(plugin.LogFields, "chat", err)
			continue
		}

		// Get updated chat again.
		chatData = getChat(&plugin, chatName)

		plugin.ChatsCache[chatId] = &chatData
	}

	// Quit if there are no chats for join.
	if len(plugin.ChatsCache) == 0 {
		return &plugin, ERROR_NO_CHATS
	}

	// Get messages and files in background.
	plugin.FileChannel = make(chan int32, DEFAULT_CHANNEL_SIZE)
	plugin.DataChannel = make(chan *core.DataItem, DEFAULT_CHANNEL_SIZE)

	// Init message listeners.
	plugin.FileListener = plugin.TdlibClient.GetListener(DEFAULT_CHANNEL_SIZE)
	plugin.UpdateListener = plugin.TdlibClient.GetListener(int64(plugin.OptionPoolSize))

	// Run main threads.
	go receiveFiles(&plugin)

	go receiveUpdates(&plugin)

	if plugin.OptionAdsEnable {
		go receiveAds(&plugin)
	}

	if plugin.OptionStatusEnable {
		go showStatus(&plugin)
	}

	if plugin.OptionStorageOptimize {
		go storageOptimize(&plugin)
	}

	if plugin.OptionUserLog {
		plugin.UsersDbClient, err = initUsersDb(&plugin)
		if err != nil {
			return &plugin, err
		}
	}

	// -------------------------------------------------------------------------

	return &plugin, nil
}
