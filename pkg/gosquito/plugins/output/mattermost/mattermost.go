package mattermostOut

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	mattermost "github.com/mattermost/mattermost-server/model"
	"io"
	"net/http"
	"os"
	"path/filepath"
	tmpl "text/template"
	"time"
)

const (
	PLUGIN_NAME = "mattermost"

	DEFAULT_ATTACHMENTS_COLOR = "#00C100"
	DEFAULT_TIMEOUT           = 3
)

var (
	ERROR_CHANNEL_NOT_FOUND    = errors.New("channel not found: %s")
	ERROR_OUTPUT_NOT_SET       = errors.New("channels and users are not set")
	ERROR_SEND_FAIL            = errors.New("sending finished with errors")
	ERROR_SEND_MESSAGE_CHANNEL = errors.New("cannot send message to channel: %v")
	ERROR_SEND_MESSAGE_USER    = errors.New("cannot send message to user: %v")
	ERROR_UPLOAD_FILE_CHANNEL  = errors.New("cannot upload file to channel: %s, %v")
	ERROR_USER_CONNECT         = errors.New("cannot establish connection to user: %v")
	ERROR_USER_NOT_FOUND       = errors.New("user not found: %s")
)

func uploadFile(p *Plugin, channel string, file string) (string, error) {
	// Form file name.
	fileExtension := ".unknown"
	mime, err := core.DetectFileType(file)
	if err == nil {
		fileExtension = mime.Extension()
	}

	fileName := fmt.Sprintf("%s%s", filepath.Base(file), fileExtension)

	// Read file.
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, f)
	if err != nil {
		return "", err
	}
	data := buf.Bytes()

	// Upload file.
	fileUploadResponse, response := p.MattermostApi.UploadFile(data, channel, fileName)
	if response.Error != nil {
		return "", response.Error
	}

	_ = f.Close()

	return fileUploadResponse.FileInfos[0].Id, nil
}

func uploadFiles(p *Plugin, channel string, files *[]string) []string {
	filesId := make([]string, 0)

	for _, file := range *files {
		if id, err := uploadFile(p, channel, file); err == nil {
			filesId = append(filesId, id)
		} else {
			core.LogOutputPlugin(p.LogFields, channel, fmt.Errorf(ERROR_UPLOAD_FILE_CHANNEL.Error(), file, err))
			continue
		}
	}

	return filesId
}

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	MattermostApi  *mattermost.Client4
	MattermostUser *mattermost.User

	PluginName string
	PluginType string

	OptionAttachments     bool
	OptionChannels        []string
	OptionColor           string
	OptionFiles           []string
	OptionMessage         string
	OptionMessageTemplate *tmpl.Template
	OptionOutput          []string
	OptionPassword        string
	OptionPretext         string
	OptionPretextTemplate *tmpl.Template
	OptionTeam            string
	OptionText            string
	OptionTextTemplate    *tmpl.Template
	OptionTimeout         int
	OptionTitle           string
	OptionTitleLink       []string
	OptionTitleTemplate   *tmpl.Template
	OptionURL             string
	OptionUsername        string
	OptionUsers           []string
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

func (p *Plugin) GetFile() string {
	return p.Flow.FlowFile
}

func (p *Plugin) GetName() string {
	return p.PluginName
}

func (p *Plugin) GetOutput() []string {
	return p.OptionOutput
}

func (p *Plugin) GetType() string {
	return p.PluginType
}

func (p *Plugin) Send(data []*core.DataItem) error {
	sendFail := false

	// Process and send data.
	for _, item := range data {
		// files.
		files := make([]string, 0)
		for _, v := range p.OptionFiles {
			files = append(files, core.ExtractDataFieldIntoArray(item, v)...)
		}

		// attachments.
		message, err := core.ExtractTemplateIntoString(item, p.OptionMessageTemplate)
		if err != nil {
			return err
		}

		props := make(map[string]interface{}, 0)

		if p.OptionAttachments {
			color := p.OptionColor

			pretext, err := core.ExtractTemplateIntoString(item, p.OptionPretextTemplate)
			if err != nil {
				return err
			}

			text, err := core.ExtractTemplateIntoString(item, p.OptionTextTemplate)
			if err != nil {
				return err
			}

			title, err := core.ExtractTemplateIntoString(item, p.OptionTitleTemplate)
			if err != nil {
				return err
			}

			titleLink := core.ExtractDataFieldIntoString(item, p.OptionTitleLink)

			props["attachments"] = []interface{}{
				map[string]string{
					"color":      color,
					"pretext":    pretext,
					"text":       text,
					"title":      title,
					"title_link": titleLink,
				},
			}
		}

		// Send to channels.
		for _, channel := range p.OptionChannels {
			filesId := uploadFiles(p, channel, &files)

			post := mattermost.Post{
				UserId:    p.MattermostUser.Id,
				ChannelId: channel,
				Message:   message,
				FileIds:   filesId,
				Props:     props,
			}

			_, res := p.MattermostApi.CreatePost(&post)
			if res.Error != nil {
				sendFail = true
				core.LogOutputPlugin(p.LogFields, channel, fmt.Errorf(ERROR_SEND_MESSAGE_CHANNEL.Error(), res.Error))
				continue
			}
		}

		// Send to users.
		for _, user := range p.OptionUsers {
			ch, res := p.MattermostApi.CreateDirectChannel(p.MattermostUser.Id, user)
			if res.Error != nil {
				sendFail = true
				core.LogOutputPlugin(p.LogFields, user, fmt.Errorf(ERROR_USER_CONNECT.Error(), res.Error))
				continue
			}

			filesId := uploadFiles(p, ch.Id, &files)

			post := mattermost.Post{
				UserId:    p.MattermostUser.Id,
				ChannelId: ch.Id,
				Message:   message,
				FileIds:   filesId,
				Props:     props,
			}

			_, res = p.MattermostApi.CreatePost(&post)
			if res.Error != nil {
				sendFail = true
				core.LogOutputPlugin(p.LogFields, user, fmt.Errorf(ERROR_SEND_MESSAGE_USER.Error(), res.Error))
				continue
			}
		}
	}

	if sendFail {
		return ERROR_SEND_FAIL
	}

	return nil
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Flow: pluginConfig.Flow,
		LogFields: log.Fields{
			"hash":   pluginConfig.Flow.FlowHash,
			"flow":   pluginConfig.Flow.FlowName,
			"file":   pluginConfig.Flow.FlowFile,
			"plugin": PLUGIN_NAME,
			"type":   pluginConfig.PluginType,
		},
		PluginName: PLUGIN_NAME,
		PluginType: pluginConfig.PluginType,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"cred":     -1,
		"template": -1,
		"timeout":  -1,

		"files":    -1,
		"message":  -1,
		"output":   1,
		"password": 1,
		"team":     1,
		"url":      1,
		"username": 1,

		"attachments": -1,
		"color":       -1,
		"pretext":     -1,
		"text":        -1,
		"title":       -1,
		"title_link":  -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	var err error

	cred, _ := core.IsString((*pluginConfig.PluginParams)["cred"])
	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// -----------------------------------------------------------------------------------------------------------------

	// username.
	setUsername := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["username"] = 0
			plugin.OptionUsername = v
		}
	}
	setUsername(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.username", cred)))
	setUsername((*pluginConfig.PluginParams)["username"])

	// password.
	setPassword := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["password"] = 0
			plugin.OptionPassword = v
		}
	}
	setPassword(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.password", cred)))
	setPassword((*pluginConfig.PluginParams)["password"])

	// team.
	setTeam := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["team"] = 0
			plugin.OptionTeam = v
		}
	}
	setTeam(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.team", cred)))
	setTeam((*pluginConfig.PluginParams)["team"])

	// url.
	setURL := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["url"] = 0
			plugin.OptionURL = v
		}
	}
	setURL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.url", cred)))
	setURL((*pluginConfig.PluginParams)["url"])
	core.ShowPluginParam(plugin.LogFields, "url", plugin.OptionURL)

	// -----------------------------------------------------------------------------------------------------------------

	// files.
	setFiles := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["files"] = 0
			plugin.OptionFiles = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setFiles(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.files", template)))
	setFiles((*pluginConfig.PluginParams)["files"])
	core.ShowPluginParam(plugin.LogFields, "files", plugin.OptionFiles)

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

	for _, v := range plugin.OptionOutput {
		if t, b := core.IsChatUsername(v); b {
			plugin.OptionUsers = append(plugin.OptionUsers, t)
		} else {
			plugin.OptionChannels = append(plugin.OptionChannels, t)
		}
	}
	core.ShowPluginParam(plugin.LogFields, "channels", plugin.OptionChannels)
	core.ShowPluginParam(plugin.LogFields, "users", plugin.OptionUsers)

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

	// message template.
	plugin.OptionMessageTemplate, err = tmpl.New("message").Funcs(core.TemplateFuncMap).Parse(plugin.OptionMessage)
	if err != nil {
		return &Plugin{}, err
	}

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.OptionTimeout = v
		}
	}
	setTimeout(DEFAULT_TIMEOUT)
	setTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.PluginParams)["timeout"])
	core.ShowPluginParam(plugin.LogFields, "timeout", plugin.OptionTimeout)

	// -----------------------------------------------------------------------------------------------------------------
	// attachments.
	attachments, _ := core.IsMapWithStringAsKey((*pluginConfig.PluginParams)["attachments"])

	// color.
	setColor := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["color"] = 0
			plugin.OptionColor = v
		}
	}
	setColor(DEFAULT_ATTACHMENTS_COLOR)
	setColor(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.attachments.color", template)))
	setColor(attachments["color"])
	core.ShowPluginParam(plugin.LogFields, "attachments.color", plugin.OptionColor)

	// pretext.
	setPretext := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["pretext"] = 0
			plugin.OptionAttachments = true
			plugin.OptionPretext = v
		}
	}
	setPretext(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.attachments.pretext", template)))
	setPretext(attachments["pretext"])
	core.ShowPluginParam(plugin.LogFields, "attachments.pretext", plugin.OptionPretext)

	// pretext template.
	plugin.OptionPretextTemplate, err = tmpl.New("pretext").Funcs(core.TemplateFuncMap).Parse(plugin.OptionPretext)
	if err != nil {
		return &Plugin{}, err
	}

	// text.
	setText := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["text"] = 0
			plugin.OptionAttachments = true
			plugin.OptionText = v
		}
	}
	setText(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.attachments.text", template)))
	setText(attachments["text"])
	core.ShowPluginParam(plugin.LogFields, "attachments.text", plugin.OptionText)

	// text template.
	if plugin.OptionTextTemplate, err = tmpl.New("text").Funcs(core.TemplateFuncMap).Parse(plugin.OptionText); err != nil {
		return &Plugin{}, err
	}

	// title.
	setTitle := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["title"] = 0
			plugin.OptionAttachments = true
			plugin.OptionTitle = v
		}
	}
	setTitle(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.attachments.title", template)))
	setText(attachments["title"])
	core.ShowPluginParam(plugin.LogFields, "attachments.title", plugin.OptionTitle)

	// title template.
	if plugin.OptionTitleTemplate, err = tmpl.New("title").Funcs(core.TemplateFuncMap).Parse(plugin.OptionTitle); err != nil {
		return &Plugin{}, err
	}

	// title_link.
	setTitleLink := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["title_link"] = 0
			plugin.OptionAttachments = true
			plugin.OptionTitleLink = v
		}
	}
	setTitleLink(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.attachments.title_link", template)))
	setTitleLink(attachments["title_link"])
	core.ShowPluginParam(plugin.LogFields, "attachments.title_link", plugin.OptionTitleLink)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Mattermost.

	// Login.
	plugin.MattermostApi = mattermost.NewAPIv4Client(plugin.OptionURL)
	plugin.MattermostApi.HttpClient = &http.Client{
		Timeout: time.Duration(plugin.OptionTimeout) * time.Second,
	}

	user, res := plugin.MattermostApi.Login(plugin.OptionUsername, plugin.OptionPassword)
	if res.Error != nil {
		return &Plugin{}, res.Error
	}
	plugin.MattermostUser = user

	// Team.
	team, res := plugin.MattermostApi.GetTeamByName(plugin.OptionTeam, "")
	if res.Error != nil {
		return &Plugin{}, res.Error
	}

	// Resolve channels ids.
	channelsId := make([]string, 0)
	for _, channel := range plugin.OptionChannels {
		ch, res := plugin.MattermostApi.GetChannelByName(channel, team.Id, "")
		if res.Error == nil {
			channelsId = append(channelsId, ch.Id)
		} else {
			return &Plugin{}, fmt.Errorf(ERROR_CHANNEL_NOT_FOUND.Error(), channel)
		}
	}

	plugin.OptionChannels = channelsId

	// Resolve users ids.
	usersId := make([]string, 0)
	for _, user := range plugin.OptionUsers {
		u, res := plugin.MattermostApi.GetUserByUsername(user, "")
		if res.Error == nil {
			usersId = append(usersId, u.Id)
		} else {
			return &Plugin{}, fmt.Errorf(ERROR_USER_NOT_FOUND.Error(), user)
		}
	}

	plugin.OptionUsers = usersId

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// channels or users must be set.
	if len(plugin.OptionChannels) == 0 && len(plugin.OptionUsers) == 0 {
		return &Plugin{}, ERROR_OUTPUT_NOT_SET
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
