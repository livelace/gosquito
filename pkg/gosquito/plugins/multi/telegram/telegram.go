package telegramMulti

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
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zelenin/go-tdlib/client"
	tmpl "text/template"
)

const (
	PLUGIN_NAME = "telegram"

	DEFAULT_ADS_ENABLE       = true
	DEFAULT_ADS_ID           = "sponsoredMessage"
	DEFAULT_ADS_PERIOD       = "5m"
	DEFAULT_ALBUM_SIZE       = 10
	DEFAULT_CHANNEL_SIZE     = 1000
	DEFAULT_CHAT_DB          = "chats.sqlite"
	DEFAULT_CHAT_SAVE        = false
	DEFAULT_DATABASE_DIR     = "database"
	DEFAULT_FETCH_ALL        = true
	DEFAULT_FETCH_DIR        = "files"
	DEFAULT_FETCH_MAX_SIZE   = "10m"
	DEFAULT_FETCH_METADATA   = false
	DEFAULT_FETCH_MIME_NOT   = false
	DEFAULT_FETCH_OTHER      = false
	DEFAULT_FETCH_TIMEOUT    = "1h"
	DEFAULT_FILE_ORIG_NAME   = true
	DEFAULT_INCLUDE_ALL      = true
	DEFAULT_INCLUDE_OTHER    = false
	DEFAULT_LOG_LEVEL        = 0
	DEFAULT_MATCH_TTL        = "1d"
	DEFAULT_MESSAGE_EDITED   = false
	DEFAULT_MESSAGE_PREVIEW  = true
	DEFAULT_OPEN_CHAT_ENABLE = true
	DEFAULT_OPEN_CHAT_PERIOD = "1s"
	DEFAULT_POOL_SIZE        = 100000
	DEFAULT_PROXY_ENABLE     = false
	DEFAULT_PROXY_PORT       = 9050
	DEFAULT_PROXY_SERVER     = "127.0.0.1"
	DEFAULT_PROXY_TYPE       = "socks"
	DEFAULT_SEND_ALBUM       = true
	DEFAULT_SEND_DELAY       = "10s"
	DEFAULT_SEND_TIMEOUT     = "1h"
	DEFAULT_SESSION_TTL      = 366
	DEFAULT_STATUS_ENABLE    = true
	DEFAULT_STATUS_PERIOD    = "1h"
	DEFAULT_STORAGE_OPTIMIZE = true
	DEFAULT_STORAGE_PERIOD   = "1h"
	DEFAULT_USER_DB          = "users.sqlite"
	DEFAULT_USER_SAVE        = false

	MAX_INSTANCE_PER_APP = 1

	SQL_FIND_CHAT = `
      SELECT * FROM chats WHERE source=?
    `

	SQL_COUNT_CHAT = `
      SELECT count(*) FROM chats
    `

	SQL_COUNT_USER = `
      SELECT count(DISTINCT id) FROM users
    `

	SQL_FIND_USER = `
      SELECT * FROM users WHERE id=? ORDER BY version DESC LIMIT 1
    `

	SQL_UPDATE_CHAT = `
      INSERT INTO chats (id, source, type, title, 
        client_data, has_protected_content,
        last_inbox_id, last_outbox_id, message_ttl,
        unread_count, first_seen, last_seen
      ) 
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(id) DO UPDATE SET
        type=?, title=?, 
        client_data=?, has_protected_content=?,
        last_inbox_id=?, last_outbox_id=?, message_ttl=?,
        unread_count=?, last_seen=?
    `

	SQL_UPDATE_USER = `
      INSERT INTO users (id, version, name, type, lang, 
        first_name, last_name, phone_number, status, 
        is_accessible, is_contact, is_fake, is_mutual_contact, 
        is_scam, is_support, is_verified, restriction_reason, 
        first_seen, last_seen
      ) 
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	SQL_CHATS_SCHEMA = `
      CREATE TABLE IF NOT EXISTS chats (
        id INTEGER PRIMARY KEY,
        source TEXT NOT NULL,
        type TEXT NOT NULL,
        title TEXT,
        client_data TEXT,
        has_protected_content INTEGER NOT NULL,
        last_inbox_id INTEGER NOT NULL,
        last_outbox_id INTEGER NOT NULL,
        message_ttl INTEGER NOT NULL,
        unread_count INTEGER NOT NULL,
        first_seen TEXT NOT NULL,
        last_seen TEXT NOT NULL
      )
    `

	SQL_USERS_SCHEMA = `
      CREATE TABLE IF NOT EXISTS users (
        id INTEGER NOT NULL,
        version INTEGER NOT NULL,
        name TEXT,
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
        first_seen TEXT NOT NULL,
        last_seen TEXT NOT NULL,
        UNIQUE(id, version)
      )
    `
)

var (
	ERROR_CHAT_COMMON_ERROR        = errors.New("chat error: %v, %v")
	ERROR_CHAT_GET_ERROR           = errors.New("cannot get chat: %v, %v")
	ERROR_CHAT_JOIN_ERROR          = errors.New("join chat error: %d, %v, %v")
	ERROR_CHAT_UPDATE_ERROR        = errors.New("cannnot update chat: %v, %v, %v, %v")
	ERROR_FETCH_ERROR              = errors.New("fetch error: %v")
	ERROR_FETCH_MIME               = errors.New("mime filtered: %v, %v")
	ERROR_FETCH_TIMEOUT            = errors.New("fetch timeout: %v")
	ERROR_FILE_SIZE_EXCEEDED       = errors.New("file size exceeded: %v (%v > %v)")
	ERROR_LOAD_USERS_ERROR         = errors.New("cannot load users: %v")
	ERROR_NO_CHATS                 = errors.New("no chats!")
	ERROR_PROXY_TYPE_UNKNOWN       = errors.New("proxy type unknown: %v")
	ERROR_SAVE_CHATS_ERROR         = errors.New("cannot save chats: %v")
	ERROR_SEND_ALBUM_ERROR         = errors.New("send album: %v")
	ERROR_SEND_ALBUM_MESSAGE_ERROR = errors.New("send album message: %v, %v")
	ERROR_SEND_ALBUM_TIMEOUT       = errors.New("send album timeout: %v")
	ERROR_SEND_MESSAGE_ERROR       = errors.New("send message: %v, %v")
	ERROR_SEND_MESSAGE_TIMEOUT     = errors.New("send message timeout: %v")
	ERROR_SQL_BEGIN_TRANSACTION    = errors.New("cannot start transaction: %v, %v")
	ERROR_SQL_DB_OPTION            = errors.New("chat or user database not set: %v, %v")
	ERROR_SQL_EXEC_ERROR           = errors.New("cannot execute query: %v, %v")
	ERROR_SQL_INIT_DB              = errors.New("cannot init database: %v, %v")
	ERROR_SQL_PREPARE_ERROR        = errors.New("cannot prepare query: %v, %v")
	ERROR_STATUS_ERROR             = errors.New("network error: %v, session error: %v, storage error: %v")
	ERROR_USER_UPDATE_ERROR        = errors.New("cannot save user: %v")

	INFO_SEND_ALBUM_MESSAGE_SUCCESS = "send album message: %v"
	INFO_SEND_MESSAGE_SUCCESS       = "send message: %v"
)

type clientAuthorizer struct {
	TdlibParameters chan *client.SetTdlibParametersRequest
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

func checkFileSize(p *Plugin, datum *core.Datum, fileName string, fileSize int64) bool {
	if fileSize > p.OptionFetchMaxSize {
		warning := fmt.Sprintf(ERROR_FILE_SIZE_EXCEEDED.Error(),
			fileName, core.BytesToSize(int64(fileSize)), core.BytesToSize(p.OptionFetchMaxSize))

		core.LogInputPlugin(p.LogFields, "fetch", warning)
		datum.WARNINGS = append(datum.WARNINGS, warning)

		return false
	}

	if fileSize < p.OptionFetchMaxSize {
		return true
	}

	return false
}

func checkMimeType(p *Plugin, datum *core.Datum, fileName string, mimeType string) bool {
	if (len(p.OptionFetchMimeMap) > 0 && !p.OptionFetchMimeMap[mimeType] && !p.OptionFetchMimeNot) ||
		(len(p.OptionFetchMimeMap) > 0 && p.OptionFetchMimeMap[mimeType] && p.OptionFetchMimeNot) {
		warning := fmt.Sprintf(ERROR_FETCH_MIME.Error(), fileName, mimeType)

		core.LogInputPlugin(p.LogFields, "fetch", warning)
		datum.WARNINGS = append(datum.WARNINGS, warning)

		return false
	}
	return true
}

func countChats(p *Plugin) int {
	count := 0
	stmt, _ := p.ChatDbClient.Prepare(SQL_COUNT_CHAT)
	defer stmt.Close()
	stmt.QueryRow().Scan(&count)
	return count
}

func countUsers(p *Plugin) int {
	count := 0
	stmt, _ := p.UserDbClient.Prepare(SQL_COUNT_USER)
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
		for i := 0; i < p.OptionFetchTimeout/1000; i++ {
			if len(p.InputFileChannel) > 0 {
				for id := range p.InputFileChannel {
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

func downloadFileDetectMimeWriteMeta(p *Plugin, datum *core.Datum, fileId string, fileName string) bool {
	localFile, err := downloadFile(p, fileId, fileName)

	if err == nil && p.OptionFetchMetadata {
		// Mime detection is not strictly mandatory.
		if m, e := core.GetFileMimeType(localFile); e == nil {
			datum.TELEGRAM.MESSAGEMIME = m.String()
		}

		// Return false if we cannot write metadata.
		if e := writeMetadata(p, localFile, &datum.TELEGRAM); e == nil {
			return true
		}

		return false

	} else if err == nil {
		datum.TELEGRAM.MESSAGEMEDIA = append(datum.TELEGRAM.MESSAGEMEDIA, localFile)
		return true

	} else {
		return false
	}
}

func getAudioMessage(p *Plugin, caption *client.FormattedText, file string) *client.InputMessageAudio {
	return &client.InputMessageAudio{
		Audio:   &client.InputFileLocal{Path: file},
		Caption: caption,
	}
}

func getDocumentMessage(p *Plugin, caption *client.FormattedText, file string) *client.InputMessageDocument {
	return &client.InputMessageDocument{
		Caption:  caption,
		Document: &client.InputFileLocal{Path: file},
	}
}

func getPhotoMessage(p *Plugin, caption *client.FormattedText, file string) *client.InputMessagePhoto {
	return &client.InputMessagePhoto{
		Caption: caption,
		Photo:   &client.InputFileLocal{Path: file},
	}
}

func getVideoMessage(p *Plugin, caption *client.FormattedText, file string) *client.InputMessageVideo {
	return &client.InputMessageVideo{
		Caption: caption,
		Video:   &client.InputFileLocal{Path: file},
	}
}

func getChat(p *Plugin, chatSource string) core.Telegram {
	d := core.Telegram{}

	stmt, _ := p.ChatDbClient.Prepare(SQL_FIND_CHAT)
	defer stmt.Close()
	stmt.QueryRow(chatSource).Scan(
		&d.CHATID, &d.CHATSOURCE,
		&d.CHATTYPE, &d.CHATTITLE, &d.CHATCLIENTDATA,
		&d.CHATPROTECTEDCONTENT, &d.CHATLASTINBOXID,
		&d.CHATLASTOUTBOXID, &d.CHATMESSAGETTL,
		&d.CHATUNREADCOUNT, &d.CHATFIRSTSEEN, &d.CHATLASTSEEN,
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

func getPrivateChatId(p *Plugin, chatSource string) (int64, error) {
	chatInfo, chatInfoErr := p.TdlibClient.CheckChatInviteLink(&client.CheckChatInviteLinkRequest{InviteLink: chatSource})
	chat, err := p.TdlibClient.JoinChatByInviteLink(&client.JoinChatByInviteLinkRequest{InviteLink: chatSource})

	if err != nil && err.Error() == "400 USER_ALREADY_PARTICIPANT" && chatInfoErr == nil {
		return chatInfo.ChatId, nil

	} else if err != nil {
		return 0, fmt.Errorf(ERROR_CHAT_GET_ERROR.Error(), chatSource, err)

	} else {
		return chat.Id, nil
	}
}

func getPublicChatId(p *Plugin, chatSource string) (int64, error) {
	chat, err := p.TdlibClient.SearchPublicChat(&client.SearchPublicChatRequest{Username: chatSource})
	if err != nil {
		return 0, fmt.Errorf(ERROR_CHAT_GET_ERROR.Error(), chatSource, err)
	} else {
		return chat.Id, nil
	}
}

func getUser(p *Plugin, userId int64) core.Telegram {
	d := core.Telegram{}

	stmt, _ := p.UserDbClient.Prepare(SQL_FIND_USER)
	defer stmt.Close()
	stmt.QueryRow(userId).Scan(&d.USERID, &d.USERVERSION, &d.USERNAME,
		&d.USERTYPE, &d.USERLANG, &d.USERFIRSTNAME, &d.USERLASTNAME,
		&d.USERPHONE, &d.USERSTATUS, &d.USERACCESSIBLE, &d.USERCONTACT,
		&d.USERFAKE, &d.USERMUTUALCONTACT, &d.USERSCAM, &d.USERSUPPORT,
		&d.USERVERIFIED, &d.USERRESTRICTION, &d.USERFIRSTSEEN, &d.USERLASTSEEN)

	return d
}

func handleMessageAudio(p *Plugin, datum *core.Datum, messageContent client.MessageContent) bool {
	if p.OptionProcessAll || p.OptionProcessAudio {
		audio := messageContent.(*client.MessageAudio).Audio
		datum.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessageAudio).Caption.Text

		if !checkMimeType(p, datum, audio.FileName, audio.MimeType) {
			return false
		}

		if (p.OptionFetchAll || p.OptionFetchAudio) && checkFileSize(p, datum, audio.FileName, audio.Audio.Size) {
			return downloadFileDetectMimeWriteMeta(p, datum, audio.Audio.Remote.Id, audio.FileName)
		}

		return true
	}

	return false
}

func handleMessagePhoto(p *Plugin, datum *core.Datum, messageContent client.MessageContent) bool {
	if p.OptionProcessAll || p.OptionProcessPhoto {
		photo := messageContent.(*client.MessagePhoto).Photo
		photoFile := photo.Sizes[len(photo.Sizes)-1]
		datum.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessagePhoto).Caption.Text

		if (p.OptionFetchAll || p.OptionFetchPhoto) && checkFileSize(p, datum, "photo", photoFile.Photo.Size) {
			return downloadFileDetectMimeWriteMeta(p, datum, photoFile.Photo.Remote.Id, "")
		}

		return true
	}

	return false
}

func handleMessageDocument(p *Plugin, datum *core.Datum, messageContent client.MessageContent) bool {
	if p.OptionProcessAll || p.OptionProcessDocument {
		document := messageContent.(*client.MessageDocument).Document
		datum.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessageDocument).Caption.Text

		if !checkMimeType(p, datum, document.FileName, document.MimeType) {
			return false
		}

		if (p.OptionFetchAll || p.OptionFetchDocument) && checkFileSize(p, datum, document.FileName, document.Document.Size) {
			return downloadFileDetectMimeWriteMeta(p, datum, document.Document.Remote.Id, document.FileName)
		}

		return true
	}

	return false
}

func markdownFormat(offset *int32, text *string, entity *client.TextEntity) {
	textArray := strings.Split(*text, "")

	entityBeginOffset := entity.Offset + *offset
	entityEndOffset := entity.Offset + *offset + entity.Length
	entityValue := strings.Join(textArray[entityBeginOffset:entityEndOffset], "")
	markdownItem := ""

	switch entity.Type.(type) {
	case *client.TextEntityTypeBlockQuote:
		markdownItem = fmt.Sprintf("> %s \n", entityValue)
	case *client.TextEntityTypeBold:
		markdownItem = fmt.Sprintf("**%s**", entityValue)
	case *client.TextEntityTypeItalic:
		markdownItem = fmt.Sprintf("*%s*", entityValue)
	case *client.TextEntityTypeStrikethrough:
		markdownItem = fmt.Sprintf("~~%s~~", entityValue)
	case *client.TextEntityTypeTextUrl:
		markdownItem = fmt.Sprintf("[%s](%s)", entityValue, entity.Type.(*client.TextEntityTypeTextUrl).Url)
	case *client.TextEntityTypeUnderline:
		markdownItem = fmt.Sprintf("<u>%s</u>", entityValue)
	}

	markdownBeginText := strings.Join(textArray[0:entityBeginOffset], "")
	markdownEndText := strings.Join(textArray[entityEndOffset:len(textArray)], "")

	*offset = *offset + int32(len(markdownItem)) - entity.Length
	*text = fmt.Sprintf("%s%s%s", markdownBeginText, markdownItem, markdownEndText)
}

func handleMessageText(p *Plugin, datum *core.Datum, messageContent client.MessageContent) bool {
	if p.OptionProcessAll || p.OptionProcessText {
		formattedText := messageContent.(*client.MessageText).Text
		datum.TELEGRAM.MESSAGEMIME = "text/plain"
		datum.TELEGRAM.MESSAGETEXT = formattedText.Text

		markdownOffset := int32(0)
		markdownText := formattedText.Text

		for _, entity := range formattedText.Entities {
			switch entity.Type.(type) {
			case *client.TextEntityTypeBlockQuote:
				markdownFormat(&markdownOffset, &markdownText, entity)
			case *client.TextEntityTypeBold:
				markdownFormat(&markdownOffset, &markdownText, entity)
			case *client.TextEntityTypeItalic:
				markdownFormat(&markdownOffset, &markdownText, entity)
			case *client.TextEntityTypeTextUrl:
				markdownFormat(&markdownOffset, &markdownText, entity)
				datum.TELEGRAM.MESSAGETEXTURL =
					append(datum.TELEGRAM.MESSAGETEXTURL, entity.Type.(*client.TextEntityTypeTextUrl).Url)
			case *client.TextEntityTypeUnderline:
				markdownFormat(&markdownOffset, &markdownText, entity)
			}
		}

		datum.TELEGRAM.MESSAGETEXTMARKDOWN = markdownText

		return true
	}

	return false
}

func handleMessageVideo(p *Plugin, datum *core.Datum, messageContent client.MessageContent) bool {
	if p.OptionProcessAll || p.OptionProcessVideo {
		video := messageContent.(*client.MessageVideo).Video
		datum.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessageVideo).Caption.Text

		if !checkMimeType(p, datum, video.FileName, video.MimeType) {
			return false
		}

		if (p.OptionFetchAll || p.OptionFetchVideo) && checkFileSize(p, datum, video.FileName, video.Video.Size) {
			return downloadFileDetectMimeWriteMeta(p, datum, video.Video.Remote.Id, video.FileName)
		}

		return true
	}

	return false
}

func handleMessageVideoNote(p *Plugin, datum *core.Datum, messageContent client.MessageContent) bool {
	if p.OptionProcessAll || p.OptionProcessVideoNote {
		note := messageContent.(*client.MessageVideoNote).VideoNote
		datum.TELEGRAM.MESSAGETEXT = ""

		if (p.OptionFetchAll || p.OptionFetchVideoNote) && checkFileSize(p, datum, "video_note", note.Video.Size) {
			return downloadFileDetectMimeWriteMeta(p, datum, note.Video.Remote.Id, "")
		}

		return true
	}

	return false
}

func handleMessageVoiceNote(p *Plugin, datum *core.Datum, messageContent client.MessageContent) bool {
	if p.OptionProcessAll || p.OptionProcessVoiceNote {
		note := messageContent.(*client.MessageVoiceNote).VoiceNote
		datum.TELEGRAM.MESSAGETEXT = messageContent.(*client.MessageVoiceNote).Caption.Text

		if (p.OptionFetchAll || p.OptionFetchVoiceNote) && checkFileSize(p, datum, "voice_note", note.Voice.Size) {
			return downloadFileDetectMimeWriteMeta(p, datum, note.Voice.Remote.Id, "")
		}

		return true
	}

	return false
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

func inputAds(p *Plugin) {
	for {
		for chatId, chatData := range p.ChatByIdDataCache {
			sponsoredMessages, err :=
				p.TdlibClient.GetChatSponsoredMessages(&client.GetChatSponsoredMessagesRequest{ChatId: chatId})

			if err == nil {
				for _, sponsoredMessage := range sponsoredMessages.Messages {
					switch sponsoredMessage.Content.(type) {
					case *client.MessageText:
						var u, _ = uuid.NewRandom()

						messageText := sponsoredMessage.Content.(*client.MessageText)
						messageTextURLs := make([]string, 0)
						messageTime := time.Now().UTC()

						for _, entity := range messageText.Text.Entities {
							switch entity.Type.(type) {
							case *client.TextEntityTypeTextUrl:
								messageTextURLs = append(messageTextURLs, entity.Type.(*client.TextEntityTypeTextUrl).Url)
							}
						}

						// Send data to channel.
						p.InputDatumChannel <- &core.Datum{
							FLOW:        p.Flow.FlowName,
							PLUGIN:      p.PluginName,
							SOURCE:      chatData.CHATSOURCE,
							TIME:        messageTime,
							TIMEFORMAT:  messageTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
							TIMEFORMATA: messageTime.In(p.OptionTimeZoneA).Format(p.OptionTimeFormatA),
							TIMEFORMATB: messageTime.In(p.OptionTimeZoneB).Format(p.OptionTimeFormatB),
							TIMEFORMATC: messageTime.In(p.OptionTimeZoneC).Format(p.OptionTimeFormatC),
							UUID:        u,

							TELEGRAM: core.Telegram{
								CHATID:               chatData.CHATID,
								CHATSOURCE:           chatData.CHATSOURCE,
								CHATTYPE:             chatData.CHATTYPE,
								CHATTITLE:            chatData.CHATTITLE,
								CHATCLIENTDATA:       chatData.CHATCLIENTDATA,
								CHATPROTECTEDCONTENT: chatData.CHATPROTECTEDCONTENT,
								CHATLASTINBOXID:      chatData.CHATLASTINBOXID,
								CHATLASTOUTBOXID:     chatData.CHATLASTOUTBOXID,
								CHATMESSAGETTL:       chatData.CHATMESSAGETTL,
								CHATUNREADCOUNT:      chatData.CHATUNREADCOUNT,
								CHATFIRSTSEEN:        chatData.CHATFIRSTSEEN,
								CHATLASTSEEN:         chatData.CHATLASTSEEN,

								MESSAGEID:        fmt.Sprintf("%v", sponsoredMessage.MessageId),
								MESSAGEMEDIA:     make([]string, 0),
								MESSAGESENDERID:  "",
								MESSAGETYPE:      messageText.GetType(),
								MESSAGETEXT:      messageText.Text.Text,
								MESSAGETEXTURL:   messageTextURLs,
								MESSAGETIMESTAMP: "",
								MESSAGEURL:       "",

								USERID:            "",
								USERVERSION:       "",
								USERNAME:          DEFAULT_ADS_ID,
								USERTYPE:          "",
								USERLANG:          "",
								USERFIRSTNAME:     "",
								USERLASTNAME:      "",
								USERPHONE:         "",
								USERSTATUS:        "",
								USERACCESSIBLE:    "",
								USERCONTACT:       "",
								USERFAKE:          "",
								USERMUTUALCONTACT: "",
								USERSCAM:          "",
								USERSUPPORT:       "",
								USERVERIFIED:      "",
								USERRESTRICTION:   "",
								USERFIRSTSEEN:     "",
								USERLASTSEEN:      "",
							},

							WARNINGS: make([]string, 0),
						}
					}
				}
			}
		}

		time.Sleep(p.OptionAdsPeriod)
	}
}

func inputFile(p *Plugin) {
	for {
		if len(p.InputFileListener.Updates) > 0 {
			update := <-p.InputFileListener.Updates

			switch update.(type) {
			case *client.UpdateFile:
				newFile := update.(*client.UpdateFile).File
				if newFile.Local.IsDownloadingCompleted || !newFile.Local.CanBeDownloaded {
					p.InputFileChannel <- newFile.Id
				}
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func inputDatum(p *Plugin) {
	for {
		if len(p.InputDatumListener.Updates) > 0 {
			update := <-p.InputDatumListener.Updates

			switch update.(type) {

			// Update chat's online members amount.
			case *client.UpdateChatOnlineMemberCount:
				memberData := update.(*client.UpdateChatOnlineMemberCount)
				// ToDo: Atomic ?
				p.ChatByIdDataCache[memberData.ChatId].CHATMEMBERONLINE = fmt.Sprintf("%v", memberData.OnlineMemberCount)

			// Process new and updated messages.
			case *client.UpdateNewMessage, *client.UpdateMessageContent:
				var datum = core.Datum{}
				var message *client.Message
				var messageChat *client.Chat
				var messageContent client.MessageContent
				var messageId int64
				var messageTime time.Time
				var messageTimestamp int32
				var messageType string
				var messageSenderId = int64(-1)
				var messageURL = ""
				var userData = core.Telegram{}
				var validMessage = false

				var err error

				switch v := update.(type) {
				case *client.UpdateNewMessage:
					message = v.Message
					messageChat, err = p.TdlibClient.GetChat(&client.GetChatRequest{ChatId: message.ChatId})
					if err != nil {
						continue
					}
					messageContent = message.Content
					messageId = message.Id
				case *client.UpdateMessageContent:
					if !p.OptionMessageEdited {
						continue
					}
					message, err = p.TdlibClient.GetMessage(&client.GetMessageRequest{ChatId: v.ChatId, MessageId: v.MessageId})
					if err != nil {
						continue
					}
					messageChat, err = p.TdlibClient.GetChat(&client.GetChatRequest{ChatId: v.ChatId})
					if err != nil {
						continue
					}
					messageContent = v.NewContent
					messageId = message.Id
				}

				messageTime = time.Unix(int64(message.Date), 0).UTC()
				messageTimestamp = message.Date
				messageType = messageContent.MessageContentType()

				// Get message url.
				if v, err := p.TdlibClient.GetMessageLink(&client.GetMessageLinkRequest{
					ChatId: messageChat.Id, MessageId: messageId}); err == nil {
					messageURL = v.Link
				}

				// Get sender id, saved user.
				switch messageSender := message.SenderId.(type) {
				case *client.MessageSenderChat:
					messageSenderId = int64(messageSender.ChatId)
				case *client.MessageSenderUser:
					messageSenderId = int64(messageSender.UserId)
					userData = getUser(p, messageSenderId)
				}

				// Process only target chats.
				if chatData, ok := p.ChatByIdDataCache[messageChat.Id]; ok {
					var u, _ = uuid.NewRandom()

					datum = core.Datum{
						FLOW:        p.Flow.FlowName,
						PLUGIN:      p.PluginName,
						SOURCE:      chatData.CHATSOURCE,
						TIME:        messageTime,
						TIMEFORMAT:  messageTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
						TIMEFORMATA: messageTime.In(p.OptionTimeZoneA).Format(p.OptionTimeFormatA),
						TIMEFORMATB: messageTime.In(p.OptionTimeZoneB).Format(p.OptionTimeFormatB),
						TIMEFORMATC: messageTime.In(p.OptionTimeZoneC).Format(p.OptionTimeFormatC),
						UUID:        u,

						TELEGRAM: core.Telegram{
							CHATID:               chatData.CHATID,
							CHATSOURCE:           chatData.CHATSOURCE,
							CHATTYPE:             chatData.CHATTYPE,
							CHATTITLE:            chatData.CHATTITLE,
							CHATCLIENTDATA:       chatData.CHATCLIENTDATA,
							CHATPROTECTEDCONTENT: chatData.CHATPROTECTEDCONTENT,
							CHATLASTINBOXID:      chatData.CHATLASTINBOXID,
							CHATLASTOUTBOXID:     chatData.CHATLASTOUTBOXID,
							CHATMEMBERONLINE:     chatData.CHATMEMBERONLINE,
							CHATMESSAGETTL:       chatData.CHATMESSAGETTL,
							CHATUNREADCOUNT:      chatData.CHATUNREADCOUNT,
							CHATFIRSTSEEN:        chatData.CHATFIRSTSEEN,
							CHATLASTSEEN:         chatData.CHATLASTSEEN,

							MESSAGEID:        fmt.Sprintf("%v", messageId),
							MESSAGEMEDIA:     make([]string, 0),
							MESSAGESENDERID:  fmt.Sprintf("%v", messageSenderId),
							MESSAGETYPE:      messageType,
							MESSAGETEXT:      "",
							MESSAGETEXTURL:   make([]string, 0),
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
							USERFIRSTSEEN:     userData.USERFIRSTSEEN,
							USERLASTSEEN:      userData.USERLASTSEEN,
						},

						WARNINGS: make([]string, 0),
					}

					switch messageContent.(type) {
					case *client.MessageAudio:
						validMessage = handleMessageAudio(p, &datum, messageContent)

					case *client.MessageDocument:
						validMessage = handleMessageDocument(p, &datum, messageContent)

					case *client.MessagePhoto:
						validMessage = handleMessagePhoto(p, &datum, messageContent)

					case *client.MessageText:
						validMessage = handleMessageText(p, &datum, messageContent)

					case *client.MessageVideo:
						validMessage = handleMessageVideo(p, &datum, messageContent)

					case *client.MessageVideoNote:
						validMessage = handleMessageVideoNote(p, &datum, messageContent)

					case *client.MessageVoiceNote:
						validMessage = handleMessageVoiceNote(p, &datum, messageContent)
					}

					if validMessage {
						p.InputDatumChannel <- &datum
					}

				} else {
					core.LogInputPlugin(p.LogFields, "chat",
						fmt.Sprintf("filtered: %v, %v, %v", messageChat.Id, messageChat.Type.ChatTypeType(), messageChat.Title))
				}
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func joinToChat(p *Plugin, chatId int64, chatSource string) error {
	_, err := p.TdlibClient.JoinChat(&client.JoinChatRequest{ChatId: chatId})
	if err != nil {
		return fmt.Errorf(ERROR_CHAT_JOIN_ERROR.Error(), chatId, chatSource, err)
	}
	return nil
}

func openChat(p *Plugin) {
	for {
		for chatName, chatId := range p.ChatBySourceIdCache {
			chat, err := p.TdlibClient.GetChat(&client.GetChatRequest{ChatId: chatId})

			if err == nil && chat.UnreadCount > 0 {
				_, err := p.TdlibClient.OpenChat(&client.OpenChatRequest{ChatId: chatId})

				core.LogInputPlugin(p.LogFields, "open chat",
					fmt.Sprintf("%v, %v, %v", chatName, chatId, err))

				if err == nil {
					_, err := p.TdlibClient.CloseChat(&client.CloseChatRequest{ChatId: chatId})
					core.LogInputPlugin(p.LogFields, "close chat",
						fmt.Sprintf("%v, %v, %v", chatName, chatId, err))
				}
			}
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

func outputMessage(p *Plugin) {
	for {
		if len(p.OutputMessageListener.Updates) > 0 {
			update := <-p.OutputMessageListener.Updates

			switch v := update.(type) {
			case *client.UpdateMessageSendFailed:
				p.OutputMessageChannel <- &core.TelegramSendingStatus{
					MessageId:    v.OldMessageId,
					ErrorCode:    v.Error.Code,
					ErrorMessage: v.Error.Message,
				}
			case *client.UpdateMessageSendSucceeded:
				p.OutputMessageChannel <- &core.TelegramSendingStatus{
					MessageId:    v.OldMessageId,
					ErrorCode:    0,
					ErrorMessage: "",
				}
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func saveChat(p *Plugin) {
	listener := p.TdlibClient.GetListener()

	for {
		if len(listener.Updates) > 0 {
			var chatId int64
			update := <-listener.Updates

			switch v := update.(type) {
			case *client.UpdateNewChat:
				chatId = v.Chat.Id
			case *client.UpdateNewMessage:
				chatId = v.Message.ChatId
			default:
				chatId = 0
			}

			if chatId != 0 {
				sqlUpdateChat(p, chatId, "")
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func saveUser(p *Plugin) {
	listener := p.TdlibClient.GetListener()

	for {
		if len(listener.Updates) > 0 {
			update := <-listener.Updates

			switch v := update.(type) {
			case *client.UpdateUser:
				isNew, isChanged, version, err := sqlUpdateUser(p, v.User)

				userName := "<DISABLED>"
				if v.User.Usernames != nil {
					userName = v.User.Usernames.ActiveUsernames[0]
				}

				if err == nil {
					if isNew {
						core.LogInputPlugin(p.LogFields, "user",
							fmt.Sprintf("new: %v, version: %v, username: %v",
								v.User.Id, version, userName))
					} else {
						core.LogInputPlugin(p.LogFields, "user",
							fmt.Sprintf("old: %v, version: %v, username: %v",
								v.User.Id, version, userName))
					}

					if isChanged {
						core.LogInputPlugin(p.LogFields, "user",
							fmt.Sprintf("changed: %v, version: %v, username: %v",
								v.User.Id, version, userName))
					}
				} else {
					core.LogInputPlugin(p.LogFields, "user",
						fmt.Errorf(ERROR_USER_UPDATE_ERROR.Error(), err))
				}
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func sendFiles(p *Plugin, chatId int64, fileType string, fileCaption client.FormattedText, files []string) bool {
	sendStatus := true

	if len(files) > 1 && p.OptionSendAlbum {
		// Splite files into albums.
		var albums [][]string
		for i := 0; i < len(files); i += DEFAULT_ALBUM_SIZE {
			end := i + DEFAULT_ALBUM_SIZE
			if end > len(files) {
				end = len(files)
			}
			albums = append(albums, files[i:end])
		}

		// Send albums one by one.
		for _, album := range albums {
			content := make([]client.InputMessageContent, 0)
			for _, file := range album {
				switch fileType {
				case "audio":
					content = append(content, getAudioMessage(p, &fileCaption, file))
				case "document":
					content = append(content, getDocumentMessage(p, &fileCaption, file))
				case "photo":
					content = append(content, getPhotoMessage(p, &fileCaption, file))
				case "video":
					content = append(content, getVideoMessage(p, &fileCaption, file))
				}
			}
			if !sendMessageAlbum(p, chatId, content) {
				sendStatus = false
			}
			time.Sleep(p.OptionSendDelay)
		}

	} else if len(files) > 0 {
		for _, file := range files {
			switch fileType {
			case "audio":
				if !sendMessage(p, chatId, getAudioMessage(p, &fileCaption, file)) {
					sendStatus = false
				}
			case "document":
				if !sendMessage(p, chatId, getDocumentMessage(p, &fileCaption, file)) {
					sendStatus = false
				}
			case "photo":
				if !sendMessage(p, chatId, getPhotoMessage(p, &fileCaption, file)) {
					sendStatus = false
				}
			case "video":
				if !sendMessage(p, chatId, getVideoMessage(p, &fileCaption, file)) {
					sendStatus = false
				}
			}
			time.Sleep(p.OptionSendDelay)
		}
	}
	return sendStatus
}

func sendMessage(p *Plugin, chatId int64, content client.InputMessageContent) bool {
	message, err := p.TdlibClient.SendMessage(&client.SendMessageRequest{
		ChatId:          chatId,
		MessageThreadId: 0,
		Options: &client.MessageSendOptions{
			DisableNotification: false,
			FromBackground:      true,
			SchedulingState:     nil,
		},
		InputMessageContent: content,
	})

	if err == nil {
		for i := 0; i < p.OptionSendTimeout/1000; i++ {
			if len(p.OutputMessageChannel) > 0 {
				status := <-p.OutputMessageChannel

				if status.MessageId == message.Id && status.ErrorCode == 0 {
					core.LogOutputPlugin(p.LogFields, "send",
						fmt.Sprintf(INFO_SEND_MESSAGE_SUCCESS, status.MessageId))
					return true
				}

				if status.MessageId == message.Id && status.ErrorCode != 0 {
					core.LogOutputPlugin(p.LogFields, "send",
						fmt.Errorf(ERROR_SEND_MESSAGE_ERROR.Error(),
							status.MessageId, status.ErrorMessage))
					return false
				}
			}
			time.Sleep(1 * time.Second)
		}

		core.LogOutputPlugin(p.LogFields, "send",
			fmt.Errorf(ERROR_SEND_MESSAGE_TIMEOUT.Error(), message.Id))
		return false

	} else {
		core.LogOutputPlugin(p.LogFields, "send",
			fmt.Errorf(ERROR_SEND_MESSAGE_ERROR.Error(), "", err))
		return false
	}
}

func sendMessageAlbum(p *Plugin, chatId int64, content []client.InputMessageContent) bool {
	sendStatus := true

	messages, err := p.TdlibClient.SendMessageAlbum(&client.SendMessageAlbumRequest{
		ChatId:          chatId,
		MessageThreadId: 0,
		Options: &client.MessageSendOptions{
			DisableNotification: false,
			FromBackground:      true,
			SchedulingState:     nil,
		},
		InputMessageContents: content,
	})

	if err == nil {
		messageCounter := int32(0)
		messageIdMap := make(map[int64]bool, 0)

		for _, message := range messages.Messages {
			messageIdMap[message.Id] = true
		}

		for i := 0; i < p.OptionSendTimeout/1000; i++ {
			if messageCounter == messages.TotalCount {
				return sendStatus
			}

			if len(p.OutputMessageChannel) > 0 {
				status := <-p.OutputMessageChannel

				if messageIdMap[status.MessageId] && status.ErrorCode == 0 {
					core.LogOutputPlugin(p.LogFields, "send",
						fmt.Sprintf(INFO_SEND_ALBUM_MESSAGE_SUCCESS, status.MessageId))
					messageCounter += 1

				} else if messageIdMap[status.MessageId] && status.ErrorCode != 0 {
					core.LogOutputPlugin(p.LogFields, "send",
						fmt.Errorf(ERROR_SEND_ALBUM_MESSAGE_ERROR.Error(),
							status.MessageId, status.ErrorMessage))
					messageCounter += 1
					sendStatus = false
				}
			}
			time.Sleep(1 * time.Second)
		}

		core.LogOutputPlugin(p.LogFields, "send",
			fmt.Errorf(ERROR_SEND_ALBUM_TIMEOUT.Error(), "album"))
		return false

	} else {
		core.LogOutputPlugin(p.LogFields, "send",
			fmt.Errorf(ERROR_SEND_ALBUM_ERROR.Error(), err))
		return false
	}
}

func showStatus(p *Plugin) {
	for {
		network, networkError := p.TdlibClient.GetNetworkStatistics(&client.GetNetworkStatisticsRequest{OnlyCurrent: true})
		session, sessionError := p.TdlibClient.GetActiveSessions()
		storage, storageError := p.TdlibClient.GetStorageStatisticsFast()
		user, userError := p.TdlibClient.GetMe()

		if networkError != nil || sessionError != nil || storageError != nil || userError != nil {
			core.LogInputPlugin(p.LogFields, "status",
				fmt.Errorf(ERROR_STATUS_ERROR.Error(),
					networkError, sessionError, storageError, userError))
		} else {
			networkSent := ""
			networkReceived := ""

			for _, entry := range network.Entries {
				switch v := entry.(type) {
				case *client.NetworkStatisticsEntryFile:
					networkReceived = core.BytesToSize(v.ReceivedBytes)
					networkSent = core.BytesToSize(v.SentBytes)
				}
			}

			for _, s := range session.Sessions {
				if s.IsCurrent {
					var info string

					if p.PluginType == "input" {
						m := []string{
							"database size: %v,",
							"files amount: %v,",
							"files size: %v,",
							"geo: %v,",
							"ip: %v,",
							"last active: %v,",
							"login date: %v,",
							"me id: %v,",
							"me name: %v,",
							"network received: %v,",
							"network sent: %v,",
							"pool size: %v,",
							"proxy: %v,",
							"saved chats: %v,",
							"saved users: %v",
						}

						info = fmt.Sprintf(strings.Join(m, " "),
							core.BytesToSize(storage.DatabaseSize), storage.FileCount,
							core.BytesToSize(storage.FilesSize), strings.ToLower(s.Location),
							s.IpAddress, time.Unix(int64(s.LastActiveDate), 0),
							time.Unix(int64(s.LogInDate), 0),
							user.Id, user.Usernames.ActiveUsernames[0],
							networkReceived, networkSent,
							len(p.InputDatumListener.Updates),
							p.OptionProxyEnable,
							countChats(p), countUsers(p),
						)

					} else {
						m := []string{
							"database size: %v,",
							"files amount: %v,",
							"files size: %v,",
							"geo: %v,",
							"ip: %v,",
							"last active: %v,",
							"login date: %v,",
							"me id: %v,",
							"me name: %v,",
							"network received: %v,",
							"network sent: %v,",
							"proxy: %v,",
							"saved chats: %v,",
							"saved users: %v",
						}

						info = fmt.Sprintf(strings.Join(m, " "),
							core.BytesToSize(storage.DatabaseSize), storage.FileCount,
							core.BytesToSize(storage.FilesSize), strings.ToLower(s.Location),
							s.IpAddress, time.Unix(int64(s.LastActiveDate), 0),
							time.Unix(int64(s.LogInDate), 0),
							user.Id, user.Usernames.ActiveUsernames[0],
							networkReceived, networkSent,
							p.OptionProxyEnable,
							countChats(p), countUsers(p),
						)
					}

					core.LogInputPlugin(p.LogFields, "status", info)
				}
			}
		}

		time.Sleep(p.OptionStatusPeriod)
	}
}

func storageOptimizer(p *Plugin) {
	for {
		p.m.Lock()
		_, err := p.TdlibClient.OptimizeStorage(&client.OptimizeStorageRequest{})
		p.m.Unlock()

		if err != nil {
			core.LogInputPlugin(p.LogFields, "storage", fmt.Errorf("error: %v", err))
		}

		time.Sleep(p.OptionStoragePeriod)
	}
}

func sqlUpdateChat(p *Plugin, chatId int64, chatSource string) error {
	currentTime := time.Now().UTC().Format(time.RFC3339)
	tx, err := p.ChatDbClient.Begin()
	if err != nil {
		return fmt.Errorf(ERROR_SQL_BEGIN_TRANSACTION.Error(), chatSource, err)
	}

	chat, err := p.TdlibClient.GetChat(&client.GetChatRequest{ChatId: chatId})
	if err != nil {
		return fmt.Errorf(ERROR_CHAT_GET_ERROR.Error(), chatSource, err)
	}

	stmt, err := p.ChatDbClient.Prepare(SQL_UPDATE_CHAT)
	if err != nil {
		return fmt.Errorf(ERROR_SQL_PREPARE_ERROR.Error(), chatSource, err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(chat.Id,
		chatSource, chat.Type.ChatTypeType(),
		chat.Title, chat.ClientData, chat.HasProtectedContent,
		chat.LastReadInboxMessageId, chat.LastReadOutboxMessageId,
		0, chat.UnreadCount, currentTime, currentTime,

		chat.Type.ChatTypeType(),
		chat.Title, chat.ClientData, chat.HasProtectedContent,
		chat.LastReadInboxMessageId, chat.LastReadOutboxMessageId,
		0, chat.UnreadCount, currentTime,
	)

	if err != nil {
		return fmt.Errorf(ERROR_SQL_EXEC_ERROR.Error(), chatSource, err)
	}

	return tx.Commit()
}

func sqlUpdateUser(p *Plugin, user *client.User) (bool, bool, int, error) {
	currentTime := time.Now().UTC().Format(time.RFC3339)
	userVersion := 0

	isNew := false
	isChanged := false

	firstSeen := fmt.Sprintf("%v", currentTime)
	lastSeen := fmt.Sprintf("%v", currentTime)

	oldUser := getUser(p, user.Id)

	userName := "<DISABLED>"
	if user.Usernames != nil {
		userName = user.Usernames.ActiveUsernames[0]
	}

	if oldUser.USERID == "" {
		isNew = true
	} else {
		firstSeen = oldUser.USERFIRSTSEEN

		if userName != oldUser.USERNAME || user.Type.UserTypeType() != oldUser.USERTYPE ||
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
		tx, err := p.UserDbClient.Begin()
		if err != nil {
			return isNew, isChanged, userVersion, fmt.Errorf(ERROR_SQL_BEGIN_TRANSACTION.Error(), userName, err)
		}

		stmt, err := p.UserDbClient.Prepare(SQL_UPDATE_USER)
		if err != nil {
			return isNew, isChanged, userVersion, fmt.Errorf(ERROR_SQL_PREPARE_ERROR.Error(), userName, err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(user.Id, userVersion, userName,
			user.Type.UserTypeType(), user.LanguageCode,
			user.FirstName, user.LastName, user.PhoneNumber,
			user.Status.UserStatusType(), user.HaveAccess,
			user.IsContact, user.IsFake, user.IsMutualContact,
			user.IsScam, user.IsSupport, user.IsVerified,
			user.RestrictionReason, firstSeen, lastSeen,
		)

		if err != nil {
			return isNew, isChanged, userVersion, fmt.Errorf(ERROR_SQL_EXEC_ERROR.Error(), userName, err)
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

	InputDatumChannel  chan *core.Datum
	InputDatumListener *client.Listener

	InputFileChannel  chan int32
	InputFileListener *client.Listener

	OutputMessageChannel  chan *core.TelegramSendingStatus
	OutputMessageListener *client.Listener

	ChatDbClient *sql.DB
	UserDbClient *sql.DB

	TdlibClient *client.Client
	TdlibParams *client.SetTdlibParametersRequest

	ChatByIdDataCache   map[int64]*core.Telegram
	ChatBySourceIdCache map[string]int64

	OptionAdsEnable             bool
	OptionAdsPeriod             time.Duration
	OptionApiHash               string
	OptionApiId                 int32
	OptionAppVersion            string
	OptionChatDatabase          string
	OptionChatSave              bool
	OptionDeviceModel           string
	OptionExpireAction          []string
	OptionExpireActionDelay     int64
	OptionExpireActionTimeout   int
	OptionExpireInterval        int64
	OptionExpireLast            int64
	OptionFetchAll              bool
	OptionFetchAudio            bool
	OptionFetchDir              string
	OptionFetchDocument         bool
	OptionFetchMaxSize          int64
	OptionFetchMetadata         bool
	OptionFetchMime             []string
	OptionFetchMimeMap          map[string]bool
	OptionFetchMimeNot          bool
	OptionFetchOrigName         bool
	OptionFetchPhoto            bool
	OptionFetchTimeout          int
	OptionFetchVideo            bool
	OptionFetchVideoNote        bool
	OptionFetchVoiceNote        bool
	OptionFileAudio             []string
	OptionFileCaption           string
	OptionFileCaptionTemplate   *tmpl.Template
	OptionFileDocument          []string
	OptionFilePhoto             []string
	OptionFileVideo             []string
	OptionForce                 bool
	OptionForceCount            int
	OptionIgnoreFileName        bool
	OptionInput                 []string
	OptionLogLevel              int
	OptionMatchSignature        []string
	OptionMatchTTL              time.Duration
	OptionMessage               string
	OptionMessageEdited         bool
	OptionMessagePreview        bool
	OptionMessageDisablePreview bool
	OptionMessageTemplate       *tmpl.Template
	OptionMessageTypeFetch      []string
	OptionMessageTypeProcess    []string
	OptionOpenChatEnable        bool
	OptionOpenChatPeriod        time.Duration
	OptionOutput                []string
	OptionPoolSize              int
	OptionProcessAll            bool
	OptionProcessAudio          bool
	OptionProcessDocument       bool
	OptionProcessPhoto          bool
	OptionProcessText           bool
	OptionProcessVideo          bool
	OptionProcessVideoNote      bool
	OptionProcessVoiceNote      bool
	OptionProxyEnable           bool
	OptionProxyPassword         string
	OptionProxyPort             int
	OptionProxyServer           string
	OptionProxyType             string
	OptionProxyUsername         string
	OptionSendAlbum             bool
	OptionSendDelay             time.Duration
	OptionSendTimeout           int
	OptionSessionTTL            int
	OptionSourceChat            []string
	OptionStatusEnable          bool
	OptionStatusPeriod          time.Duration
	OptionStorageOptimize       bool
	OptionStoragePeriod         time.Duration
	OptionTimeFormat            string
	OptionTimeFormatA           string
	OptionTimeFormatB           string
	OptionTimeFormatC           string
	OptionTimeZone              *time.Location
	OptionTimeZoneA             *time.Location
	OptionTimeZoneB             *time.Location
	OptionTimeZoneC             *time.Location
	OptionTimeout               int
	OptionUserDatabase          string
	OptionUserSave              bool
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

func (p *Plugin) GetOutput() []string {
	return p.OptionOutput
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

func (p *Plugin) Receive() ([]*core.Datum, error) {
	currentTime := time.Now().UTC()
	temp := make([]*core.Datum, 0)
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
	length := len(p.InputDatumChannel)

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

		item := <-p.InputDatumChannel

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
				case "TELEGRAM.MESSAGETEXT":
					itemSignature += item.TELEGRAM.MESSAGETEXT
				case "TELEGRAM.MESSAGEURL":
					itemSignature += item.TELEGRAM.MESSAGEURL
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
	for _, source := range p.OptionInput {
		sourceTime := flowStates[source]

		if (currentTime.Unix() - sourceTime.Unix()) > p.OptionExpireInterval/1000 {
			sourcesExpired = true

			core.LogInputPlugin(p.LogFields, source,
				fmt.Sprintf("source expired: %v", currentTime.Sub(sourceTime)))

			// Execute command if expire delay exceeded.
			// ExpireLast keeps last execution timestamp.
			if (currentTime.Unix() - p.OptionExpireLast) > p.OptionExpireActionDelay/1000 {
				p.OptionExpireLast = currentTime.Unix()

				// Execute command with args.
				// We don't worry about command return code.
				if len(p.OptionExpireAction) > 0 {
					cmd := p.OptionExpireAction[0]
					args := []string{p.Flow.FlowName, source, fmt.Sprintf("%v", sourceTime.Unix())}
					args = append(args, p.OptionExpireAction[1:]...)

					output, err := core.ExecWithTimeout(cmd, args, p.OptionExpireActionTimeout)

					core.LogInputPlugin(p.LogFields, source, fmt.Sprintf(
						"source expired action: command: %s, arguments: %v, output: %s, error: %v",
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

func (p *Plugin) Send(data []*core.Datum) error {
	p.LogFields["run"] = p.Flow.GetRunID()
	sendStatus := true

	for _, item := range data {
		for _, chatName := range p.OptionOutput {
			chatId := p.ChatBySourceIdCache[chatName]

			// Construct file caption.
			var fileCaption client.FormattedText
			if c, err := core.ExtractTemplateIntoString(item, p.OptionFileCaptionTemplate); err == nil && len(c) > 0 {
				fileCaption = client.FormattedText{
					Text: c,
				}
			} else if err != nil {

			}

			// Construct text message.
			if m, err := core.ExtractTemplateIntoString(item, p.OptionMessageTemplate); err == nil && len(m) > 0 {
				content := &client.InputMessageText{
					ClearDraft:         false,
					LinkPreviewOptions: &client.LinkPreviewOptions{IsDisabled: p.OptionMessagePreview},
					Text:               &client.FormattedText{Text: m},
				}
				if !sendMessage(p, chatId, content) {
					sendStatus = false
				}
				time.Sleep(p.OptionSendDelay)
			}

			// Send audio files.
			audio := core.ExtractDatumFieldIntoArray(item, p.OptionFileAudio)
			if !sendFiles(p, chatId, "audio", fileCaption, audio) {
				sendStatus = false
			}

			// Send document files.
			document := core.ExtractDatumFieldIntoArray(item, p.OptionFileDocument)
			if !sendFiles(p, chatId, "document", fileCaption, document) {
				sendStatus = false
			}

			// Send photo files.
			photo := core.ExtractDatumFieldIntoArray(item, p.OptionFilePhoto)
			if !sendFiles(p, chatId, "photo", fileCaption, photo) {
				sendStatus = false
			}

			// Send video files.
			video := core.ExtractDatumFieldIntoArray(item, p.OptionFileVideo)
			if !sendFiles(p, chatId, "video", fileCaption, video) {
				sendStatus = false
			}
		}
	}

	if !sendStatus {
		return core.ERROR_SEND_FAIL
	}

	return nil
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
		"cred":          -1,
		"template":      -1,
		"timeout":       -1,
		"time_format":   -1,
		"time_format_a": -1,
		"time_format_b": -1,
		"time_format_c": -1,
		"time_zone":     -1,
		"time_zone_a":   -1,
		"time_zone_b":   -1,
		"time_zone_c":   -1,

		"api_hash":         1,
		"api_id":           1,
		"app_version":      -1,
		"chat_database":    -1,
		"chat_save":        -1,
		"device_model":     -1,
		"log_level":        -1,
		"proxy_enable":     -1,
		"proxy_port":       -1,
		"proxy_server":     -1,
		"proxy_type":       -1,
		"session_ttl":      -1,
		"status_enable":    -1,
		"status_period":    -1,
		"storage_optimize": -1,
		"storage_period":   -1,
		"user_database":    -1,
		"user_save":        -1,
	}

	switch pluginConfig.PluginType {
	case "input":
		availableParams["ads_enable"] = -1
		availableParams["ads_period"] = -1
		availableParams["expire_action"] = -1
		availableParams["expire_action_timeout"] = -1
		availableParams["expire_delay"] = -1
		availableParams["expire_interval"] = -1
		availableParams["fetch_dir"] = -1
		availableParams["fetch_max_size"] = -1
		availableParams["fetch_metadata"] = -1
		availableParams["fetch_mime"] = -1
		availableParams["fetch_mime_not"] = -1
		availableParams["fetch_orig_name"] = -1
		availableParams["fetch_timeout"] = -1
		availableParams["force"] = -1
		availableParams["force_count"] = -1
		availableParams["input"] = 1
		availableParams["match_signature"] = -1
		availableParams["match_ttl"] = -1
		availableParams["message_edited"] = -1
		availableParams["message_type_fetch"] = -1
		availableParams["message_type_process"] = -1
		availableParams["open_chat_enable"] = -1
		availableParams["open_chat_period"] = -1
		availableParams["pool_size"] = -1
	case "output":
		availableParams["file_audio"] = -1
		availableParams["file_caption"] = -1
		availableParams["file_document"] = -1
		availableParams["file_photo"] = -1
		availableParams["file_video"] = -1
		availableParams["message"] = -1
		availableParams["message_preview"] = -1
		availableParams["output"] = 1
		availableParams["send_album"] = -1
		availableParams["send_delay"] = -1
		availableParams["send_timeout"] = -1
		break
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	cred, _ := core.IsString((*pluginConfig.PluginParams)["cred"])
	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	vault, err := core.GetVault(pluginConfig.AppConfig.GetStringMap(fmt.Sprintf("%s.vault", cred)))
	if err != nil {
		return &plugin, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	// api_id.
	setApiID := func(p interface{}) {
		if sv, sb := core.IsString(p); sb {
			if iv, ib := core.IsInt(core.GetCredValue(sv, vault)); ib {
				availableParams["api_id"] = 0
				plugin.OptionApiId = int32(iv)
			}
		}
	}
	setApiID(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.api_id", cred)))
	setApiID((*pluginConfig.PluginParams)["api_id"])

	// api_hash.
	setApiHash := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["api_hash"] = 0
			plugin.OptionApiHash = core.GetCredValue(v, vault)
		}
	}
	setApiHash(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.api_hash", cred)))
	setApiHash((*pluginConfig.PluginParams)["api_hash"])

	// proxy_username.
	setProxyUsername := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["proxy_username"] = 0
			plugin.OptionProxyUsername = core.GetCredValue(v, vault)
		}
	}
	setProxyUsername(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.proxy_userrname", cred)))
	setProxyUsername((*pluginConfig.PluginParams)["proxy_username"])

	// proxy_password.
	setProxyPassword := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["proxy_password"] = 0
			plugin.OptionProxyPassword = core.GetCredValue(v, vault)
		}
	}
	setProxyPassword(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.proxy_password", cred)))
	setProxyPassword((*pluginConfig.PluginParams)["proxy_password"])

	// -----------------------------------------------------------------------------------------------------------------

	switch pluginConfig.PluginType {

	case "input":
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
				plugin.OptionAdsPeriod = time.Duration(v) * time.Millisecond
			}
		}
		setAdsPeriod(DEFAULT_ADS_PERIOD)
		setAdsPeriod(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.ads_period", template)))
		setAdsPeriod((*pluginConfig.PluginParams)["ads_period"])
		core.ShowPluginParam(plugin.LogFields, "ads_period", plugin.OptionAdsPeriod)

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

		// fetch_dir.
		setFetchDir := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["fetch_dir"] = 0
				plugin.OptionFetchDir = v
			}
		}
		setFetchDir(filepath.Join(plugin.PluginDataDir, DEFAULT_FETCH_DIR))
		setFetchDir(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.fetch_dir", template)))
		setFetchDir((*pluginConfig.PluginParams)["fetch_dir"])
		core.ShowPluginParam(plugin.LogFields, "fetch_dir", plugin.OptionFetchDir)

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

		// fetch_mime.
		setFetchMime := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["fetch_mime"] = 0
				plugin.OptionFetchMime = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
			}
		}
		setFetchMime(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.fetch_mime", template)))
		setFetchMime((*pluginConfig.PluginParams)["fetch_mime"])
		core.ShowPluginParam(plugin.LogFields, "fetch_mime", plugin.OptionFetchMime)

		plugin.OptionFetchMimeMap = make(map[string]bool, 0)
		for _, v := range plugin.OptionFetchMime {
			plugin.OptionFetchMimeMap[v] = true
		}

		// fetch_mime_not.
		setFetchMimeNot := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["fetch_mime_not"] = 0
				plugin.OptionFetchMimeNot = v
			}
		}
		setFetchMimeNot(DEFAULT_FETCH_MIME_NOT)
		setFetchMimeNot(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.fetch_mime_not", template)))
		setFetchMimeNot((*pluginConfig.PluginParams)["fetch_mime_not"])
		core.ShowPluginParam(plugin.LogFields, "fetch_mime_not", plugin.OptionFetchMimeNot)

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

		plugin.OptionSourceChat = plugin.OptionInput

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

		// message_edited.
		setMessageEdited := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["message_edited"] = 0
				plugin.OptionMessageEdited = v
			}
		}
		setMessageEdited(DEFAULT_MESSAGE_EDITED)
		setMessageEdited(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.message_edited", template)))
		setMessageEdited((*pluginConfig.PluginParams)["message_edited"])
		core.ShowPluginParam(plugin.LogFields, "message_edited", plugin.OptionMessageEdited)

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

		// open_chat_enable.
		setOpenChatEnable := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["open_chat_enable"] = 0
				plugin.OptionOpenChatEnable = v
			}
		}
		setOpenChatEnable(DEFAULT_OPEN_CHAT_ENABLE)
		setOpenChatEnable(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.open_chat_enable", template)))
		setOpenChatEnable((*pluginConfig.PluginParams)["open_chat_enable"])
		core.ShowPluginParam(plugin.LogFields, "open_chat_enable", plugin.OptionOpenChatEnable)

		// open_chat_period.
		setOpenChatPeriod := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["status_period"] = 0
				plugin.OptionStatusPeriod = time.Duration(v) * time.Millisecond
			}
		}
		setOpenChatPeriod(DEFAULT_OPEN_CHAT_PERIOD)
		setOpenChatPeriod(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.open_chat_period", template)))
		setOpenChatPeriod((*pluginConfig.PluginParams)["open_chat_period"])
		core.ShowPluginParam(plugin.LogFields, "open_chat_period", plugin.OptionOpenChatPeriod)

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

	case "output":
		// file_audio.
		setFileAudio := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["file_audio"] = 0
				plugin.OptionFileAudio = v
			}
		}
		setFileAudio(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.file_audio", template)))
		setFileAudio((*pluginConfig.PluginParams)["file_audio"])
		core.ShowPluginParam(plugin.LogFields, "file_audio", plugin.OptionFileAudio)

		// file_caption.
		setFileCaption := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["file_caption"] = 0
				plugin.OptionFileCaption = v
			}
		}
		setFileCaption(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_caption", template)))
		setFileCaption((*pluginConfig.PluginParams)["file_caption"])
		core.ShowPluginParam(plugin.LogFields, "file_caption", plugin.OptionFileCaption)

		plugin.OptionFileCaptionTemplate, err = tmpl.New("file_caption").Funcs(core.TemplateFuncMap).Parse(plugin.OptionFileCaption)
		if err != nil {
			return &Plugin{}, err
		}

		// file_document.
		setFileDocument := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["file_document"] = 0
				plugin.OptionFileDocument = v
			}
		}
		setFileDocument(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.file_document", template)))
		setFileDocument((*pluginConfig.PluginParams)["file_document"])
		core.ShowPluginParam(plugin.LogFields, "file_document", plugin.OptionFileDocument)

		// file_photo.
		setFilePhoto := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["file_photo"] = 0
				plugin.OptionFilePhoto = v
			}
		}
		setFilePhoto(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.file_photo", template)))
		setFilePhoto((*pluginConfig.PluginParams)["file_photo"])
		core.ShowPluginParam(plugin.LogFields, "file_photo", plugin.OptionFilePhoto)

		// file_video.
		setFileVideo := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["file_video"] = 0
				plugin.OptionFileVideo = v
			}
		}
		setFileVideo(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.file_video", template)))
		setFileVideo((*pluginConfig.PluginParams)["file_video"])
		core.ShowPluginParam(plugin.LogFields, "file_video", plugin.OptionFileVideo)

		// message.
		setMessage := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["message"] = 0
				plugin.OptionMessage = v
			}
		}
		setMessage(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.message", template)))
		setMessage((*pluginConfig.PluginParams)["message"])
		core.ShowPluginParam(plugin.LogFields, "message", plugin.OptionMessage)

		plugin.OptionMessageTemplate, err = tmpl.New("message").Funcs(core.TemplateFuncMap).Parse(plugin.OptionMessage)
		if err != nil {
			return &Plugin{}, err
		}

		// message_preview.
		setMessagePreview := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["message_preview"] = 0
				plugin.OptionMessagePreview = v
			}
		}
		setMessagePreview(DEFAULT_MESSAGE_PREVIEW)
		setMessagePreview(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.message_preview", template)))
		setMessagePreview((*pluginConfig.PluginParams)["message_preview"])
		core.ShowPluginParam(plugin.LogFields, "message_preview", plugin.OptionMessagePreview)

		if plugin.OptionMessagePreview {
			plugin.OptionMessageDisablePreview = false
		} else {
			plugin.OptionMessageDisablePreview = true
		}

		// output.
		setOutput := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["output"] = 0
				plugin.OptionOutput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
			}
		}
		setOutput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.output", template)))
		setOutput((*pluginConfig.PluginParams)["output"])
		core.ShowPluginParam(plugin.LogFields, "output", plugin.OptionOutput)

		plugin.OptionSourceChat = plugin.OptionOutput

		// send_album.
		setSendAlbum := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["send_album"] = 0
				plugin.OptionSendAlbum = v
			}
		}
		setSendAlbum(DEFAULT_SEND_ALBUM)
		setSendAlbum(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.send_album", template)))
		setSendAlbum((*pluginConfig.PluginParams)["send_album"])
		core.ShowPluginParam(plugin.LogFields, "send_album", plugin.OptionSendAlbum)

		// send_delay.
		setSendDelay := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["send_delay"] = 0
				plugin.OptionSendDelay = time.Duration(v) * time.Millisecond
			}
		}
		setSendDelay(DEFAULT_SEND_DELAY)
		setSendDelay(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.send_delay", template)))
		setSendDelay((*pluginConfig.PluginParams)["send_delay"])
		core.ShowPluginParam(plugin.LogFields, "send_delay", plugin.OptionSendDelay)

		// send_timeout.
		setSendTimeout := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["send_timeout"] = 0
				plugin.OptionSendTimeout = int(v)
			}
		}
		setSendTimeout(DEFAULT_SEND_TIMEOUT)
		setSendTimeout(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.send_timeout", template)))
		setSendTimeout((*pluginConfig.PluginParams)["send_timeout"])
		core.ShowPluginParam(plugin.LogFields, "send_timeout", plugin.OptionSendTimeout)
	}

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
	setChatDatabase(filepath.Join(plugin.PluginDataDir, DEFAULT_CHAT_DB))
	setChatDatabase(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.chat_database", template)))
	setChatDatabase((*pluginConfig.PluginParams)["chat_database"])
	core.ShowPluginParam(plugin.LogFields, "chat_database", plugin.OptionChatDatabase)

	// chat_save.
	setChatSave := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["chat_save"] = 0
			plugin.OptionChatSave = v
		}
	}
	setChatSave(DEFAULT_CHAT_SAVE)
	setChatSave(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.chat_save", template)))
	setChatSave((*pluginConfig.PluginParams)["chat_save"])
	core.ShowPluginParam(plugin.LogFields, "chat_save", plugin.OptionChatSave)

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
			plugin.OptionStatusPeriod = time.Duration(v) * time.Millisecond
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
			plugin.OptionStoragePeriod = time.Duration(v) * time.Millisecond
		}
	}
	setStoragePeriod(DEFAULT_STATUS_PERIOD)
	setStoragePeriod(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.storage_period", template)))
	setStoragePeriod((*pluginConfig.PluginParams)["storage_period"])
	core.ShowPluginParam(plugin.LogFields, "storage_period", plugin.OptionStoragePeriod)

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

	// time_format_a.
	setTimeFormatA := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["time_format_a"] = 0
			plugin.OptionTimeFormatA = v
		}
	}
	setTimeFormatA(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
	setTimeFormatA(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format_a", template)))
	setTimeFormatA((*pluginConfig.PluginParams)["time_format_a"])
	core.ShowPluginParam(plugin.LogFields, "time_format_a", plugin.OptionTimeFormatA)

	// time_format_b.
	setTimeFormatB := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["time_format_b"] = 0
			plugin.OptionTimeFormatB = v
		}
	}
	setTimeFormatB(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
	setTimeFormatB(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format_b", template)))
	setTimeFormatB((*pluginConfig.PluginParams)["time_format_b"])
	core.ShowPluginParam(plugin.LogFields, "time_format_b", plugin.OptionTimeFormatB)

	// time_format_c.
	setTimeFormatC := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["time_format_c"] = 0
			plugin.OptionTimeFormatC = v
		}
	}
	setTimeFormatC(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
	setTimeFormatC(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format_c", template)))
	setTimeFormatC((*pluginConfig.PluginParams)["time_format_c"])
	core.ShowPluginParam(plugin.LogFields, "time_format_c", plugin.OptionTimeFormatC)

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

	// time_zone_a.
	setTimeZoneA := func(p interface{}) {
		if v, b := core.IsTimeZone(p); b {
			availableParams["time_zone_a"] = 0
			plugin.OptionTimeZoneA = v
		}
	}
	setTimeZoneA(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
	setTimeZoneA(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone_a", template)))
	setTimeZoneA((*pluginConfig.PluginParams)["time_zone_a"])
	core.ShowPluginParam(plugin.LogFields, "time_zone_a", plugin.OptionTimeZoneA)

	// time_zone_b.
	setTimeZoneB := func(p interface{}) {
		if v, b := core.IsTimeZone(p); b {
			availableParams["time_zone_b"] = 0
			plugin.OptionTimeZoneB = v
		}
	}
	setTimeZoneB(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
	setTimeZoneB(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone_b", template)))
	setTimeZoneB((*pluginConfig.PluginParams)["time_zone_b"])
	core.ShowPluginParam(plugin.LogFields, "time_zone_b", plugin.OptionTimeZoneB)

	// time_zone_c.
	setTimeZoneC := func(p interface{}) {
		if v, b := core.IsTimeZone(p); b {
			availableParams["time_zone_c"] = 0
			plugin.OptionTimeZoneC = v
		}
	}
	setTimeZoneC(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
	setTimeZoneC(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone_c", template)))
	setTimeZoneC((*pluginConfig.PluginParams)["time_zone_c"])
	core.ShowPluginParam(plugin.LogFields, "time_zone_c", plugin.OptionTimeZoneC)

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

	// user_database.
	setUserDatabase := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["user_database"] = 0
			plugin.OptionUserDatabase = v
		}
	}
	setUserDatabase(filepath.Join(plugin.PluginDataDir, DEFAULT_USER_DB))
	setUserDatabase(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.user_database", template)))
	setUserDatabase((*pluginConfig.PluginParams)["user_database"])
	core.ShowPluginParam(plugin.LogFields, "user_database", plugin.OptionUserDatabase)

	// user_save.
	setUserSave := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["user_save"] = 0
			plugin.OptionUserSave = v
		}
	}
	setUserSave(DEFAULT_USER_SAVE)
	setUserSave(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.user_save", template)))
	setUserSave((*pluginConfig.PluginParams)["user_save"])
	core.ShowPluginParam(plugin.LogFields, "user_save", plugin.OptionUserSave)

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

	plugin.TdlibParams = &client.SetTdlibParametersRequest{
		ApiHash:            plugin.OptionApiHash,
		ApiId:              plugin.OptionApiId,
		ApplicationVersion: plugin.OptionAppVersion,
		DatabaseDirectory:  filepath.Join(plugin.PluginDataDir, DEFAULT_DATABASE_DIR),
		DeviceModel:        plugin.OptionDeviceModel,
		//EnableStorageOptimizer: plugin.OptionStorageOptimize,
		FilesDirectory: plugin.OptionFetchDir,
		//IgnoreFileNames:        plugin.OptionIgnoreFileName,
		SystemLanguageCode:  "en",
		SystemVersion:       plugin.Flow.FlowName,
		UseChatInfoDatabase: true,
		UseFileDatabase:     true,
		UseMessageDatabase:  true,
		UseSecretChats:      true,
		UseTestDc:           false,
	}

	// Create client.
	tdlibClient, err := getClient(&plugin)
	if err != nil {
		return &Plugin{}, err
	} else {
		plugin.TdlibClient = tdlibClient
	}

	// Set session TTL.
	plugin.TdlibClient.SetInactiveSessionTtl(&client.SetInactiveSessionTtlRequest{
		InactiveSessionTtlDays: int32(plugin.OptionSessionTTL),
	})

	// -----------------------------------------------------------------------------------------------------------------
	// Init chat/user databases.

	plugin.ChatDbClient, err = initChatsDb(&plugin)
	if err != nil {
		return &plugin, err
	}

	plugin.UserDbClient, err = initUsersDb(&plugin)
	if err != nil {
		return &plugin, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Update and join to chats.
	plugin.ChatByIdDataCache = make(map[int64]*core.Telegram, 0)
	plugin.ChatBySourceIdCache = make(map[string]int64, 0)

	for _, chatSource := range plugin.OptionSourceChat {
		var chatId int64
		var err error

		chatData := getChat(&plugin, chatSource)

		// Join only to unknown chats (api limits).
		if chatData.CHATID == "" {
			chatIdRegexp := regexp.MustCompile(`^[0-9]+$`)

			if chatIdRegexp.Match([]byte(chatSource)) {
				chatId, err = strconv.ParseInt(chatSource, 10, 64)
			} else if strings.Contains(chatSource, "t.me/+") {
				chatId, err = getPrivateChatId(&plugin, chatSource)
			} else {
				chatId, err = getPublicChatId(&plugin, chatSource)
			}

			if err != nil {
				core.LogInputPlugin(plugin.LogFields, "chat", err)
				continue
			}

			err = sqlUpdateChat(&plugin, chatId, chatSource)
			if err != nil {
				core.LogInputPlugin(plugin.LogFields, "chat", err)
				continue
			}

			err = joinToChat(&plugin, chatId, chatSource)
			if err != nil {
				core.LogInputPlugin(plugin.LogFields, "chat", err)
				continue
			}

			// Get updated chat again.
			chatData = getChat(&plugin, chatSource)
		} else {
			chatId, _ = strconv.ParseInt(chatData.CHATID, 10, 64)
		}

		plugin.ChatByIdDataCache[chatId] = &chatData
		plugin.ChatBySourceIdCache[chatSource] = chatId
	}

	// Quit if there are no chats for join.
	if len(plugin.ChatByIdDataCache) == 0 {
		return &plugin, ERROR_NO_CHATS
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Input mode:

	if plugin.PluginType == "input" {
		plugin.InputFileChannel = make(chan int32, DEFAULT_CHANNEL_SIZE)
		plugin.InputFileListener = plugin.TdlibClient.GetListener()
		go inputFile(&plugin)

		plugin.InputDatumChannel = make(chan *core.Datum, DEFAULT_CHANNEL_SIZE)
		plugin.InputDatumListener = plugin.TdlibClient.GetListener()
		go inputDatum(&plugin)

		if plugin.OptionAdsEnable {
			go inputAds(&plugin)
		}

		if plugin.OptionOpenChatEnable {
			go openChat(&plugin)
		}
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Output mode:

	if plugin.PluginType == "output" {
		plugin.OutputMessageChannel = make(chan *core.TelegramSendingStatus, DEFAULT_CHANNEL_SIZE)
		plugin.OutputMessageListener = plugin.TdlibClient.GetListener()
		go outputMessage(&plugin)
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Input/output mode:

	if plugin.OptionChatSave {
		go saveChat(&plugin)
	}

	if plugin.OptionStatusEnable {
		go showStatus(&plugin)
	}

	if plugin.OptionStorageOptimize {
		go storageOptimizer(&plugin)
	}

	if plugin.OptionUserSave {
		go saveUser(&plugin)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
