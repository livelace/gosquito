package slackOut

import (
	"errors"
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/slack-go/slack"
	"path/filepath"
	tmpl "text/template"
)

const (
	DEFAULT_ATTACHMENTS_COLOR = "#00C100"
	DEFAULT_TIMEOUT           = 3
)

var (
	ERROR_CHANNEL_NOT_FOUND = errors.New("channel not found: %s")
	ERROR_OUTPUT_NOT_SET    = errors.New("channels and users are not set")
	ERROR_USER_NOT_FOUND    = errors.New("user not found: %s")
	ERROR_SEND_FAIL         = errors.New("sending finished with errors")
)

type Plugin struct {
	Hash string
	Flow string

	File string
	Name string
	Type string

	Api *slack.Client

	Channels        []string
	Files           []string
	Message         string
	MessageTemplate *tmpl.Template
	Output          []string
	Timeout         int
	Token           string
	Users           []string
	URL             string

	Attachments     bool
	Color           string
	Pretext         string
	PretextTemplate *tmpl.Template
	Title           string
	TitleTemplate   *tmpl.Template
	TitleLink       []string
	Text            string
	TextTemplate    *tmpl.Template
}

func uploadFile(p *Plugin, channel string, file string) error {
	mime, err := core.DetectFileType(file)
	if err != nil {
		return err
	}

	_, err = p.Api.UploadFile(slack.FileUploadParameters{
		Channels: []string{channel},
		File:     file,
		Filename: fmt.Sprintf("%s%s", filepath.Base(file), mime.Extension()),
	})

	return err
}

func uploadFiles(p *Plugin, channel string, files *[]string) {
	for _, file := range *files {
		if err := uploadFile(p, channel, file); err != nil {
			log.WithFields(log.Fields{
				"hash":   p.Hash,
				"flow":   p.Flow,
				"file":   p.File,
				"plugin": p.Name,
				"type":   p.Type,
				"error":  fmt.Sprintf("cannot upload file to channel: %s, %s, %v", channel, file, err),
			}).Error(core.LOG_PLUGIN_DATA)
			continue
		}
	}
}

func (p *Plugin) Send(data []*core.DataItem) error {
	sendFail := false

	// Logging.
	logError := func(msg string) {
		log.WithFields(log.Fields{
			"hash":   p.Hash,
			"flow":   p.Flow,
			"file":   p.File,
			"plugin": p.Name,
			"type":   p.Type,
			"error":  msg,
		}).Error(core.LOG_PLUGIN_DATA)
	}

	// Process and send data.
	for _, item := range data {

		// Files.
		files := make([]string, 0)
		for _, v := range p.Files {
			files = append(files, core.ExtractDataFieldIntoArray(item, v)...)
		}

		// Message text.
		message, err := core.ExtractTemplateIntoString(item, p.MessageTemplate)
		if err != nil {
			return err
		}

		// Attachments.
		attachments := slack.Attachment{}

		if p.Attachments {
			color := p.Color

			pretext, err := core.ExtractTemplateIntoString(item, p.PretextTemplate)
			if err != nil {
				return err
			}

			text, err := core.ExtractTemplateIntoString(item, p.TextTemplate)
			if err != nil {
				return err
			}

			title, err := core.ExtractTemplateIntoString(item, p.TitleTemplate)
			if err != nil {
				return err
			}

			titleLink := core.ExtractDataFieldIntoString(item, p.TitleLink)

			attachments.Color = color
			attachments.Pretext = pretext
			attachments.Text = text
			attachments.Title = title
			attachments.TitleLink = titleLink
		}

		// Send to channels.
		for _, channel := range p.Channels {
			_, _, err := p.Api.PostMessage(
				channel,
				slack.MsgOptionAsUser(true),
				slack.MsgOptionAttachments(attachments),
				slack.MsgOptionText(message, false),
			)

			if err != nil {
				logError(fmt.Sprintf("cannot send message to channel: %s, %v", channel, err))
				sendFail = true
				continue
			}

			uploadFiles(p, channel, &files)
		}

		// Send to users.
		for _, user := range p.Users {
			ch, _, _, err := p.Api.OpenConversation(&slack.OpenConversationParameters{Users: []string{user}})
			if err != nil {
				logError(fmt.Sprintf("cannot establish connection to user: %s, %v", user, err))
				sendFail = true
				continue
			}

			_, _, err = p.Api.PostMessage(
				ch.ID,
				slack.MsgOptionAsUser(true),
				slack.MsgOptionAttachments(attachments),
				slack.MsgOptionText(message, false),
			)

			if err != nil {
				logError(fmt.Sprintf("cannot send message to user: %s, %v", user, err))
				sendFail = true
				continue
			}

			uploadFiles(p, ch.ID, &files)
		}
	}

	// Inform about sending failures.
	if sendFail {
		return ERROR_SEND_FAIL
	}

	return nil
}

func (p *Plugin) GetFile() string {
	return p.File
}

func (p *Plugin) GetName() string {
	return p.Name
}

func (p *Plugin) GetOutput() []string {
	return p.Output
}

func (p *Plugin) GetType() string {
	return p.Type
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Hash: pluginConfig.Hash,
		Flow: pluginConfig.Flow,

		File: pluginConfig.File,
		Name: "slack",
		Type: "output",
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

		"files":   -1,
		"message": -1,
		"output":  1,
		"token":   1,

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

	cred, _ := core.IsString((*pluginConfig.Params)["cred"])

	// token.
	setToken := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["token"] = 0
			plugin.Token = v
		}
	}
	setToken(pluginConfig.Config.GetString(fmt.Sprintf("%s.token", cred)))
	setToken((*pluginConfig.Params)["token"])

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.Params)["template"])

	// files.
	setFiles := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["files"] = 0
			plugin.Files = core.ExtractConfigVariableIntoArray(pluginConfig.Config, v)
		}
	}
	setFiles(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.files", template)))
	setFiles((*pluginConfig.Params)["files"])
	showParam("files", plugin.Files)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.Output = core.ExtractConfigVariableIntoArray(pluginConfig.Config, v)
		}
	}
	setOutput(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.output", template)))
	setOutput((*pluginConfig.Params)["output"])
	showParam("output", plugin.Output)

	for _, v := range plugin.Output {
		if t, b := core.IsChatUsername(v); b {
			plugin.Users = append(plugin.Users, t)
		} else {
			plugin.Channels = append(plugin.Channels, t)
		}
	}
	showParam("channels", plugin.Channels)
	showParam("users", plugin.Users)

	// message.
	setMessage := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["message"] = 0
			plugin.Message = v
		}
	}
	setMessage(pluginConfig.Config.GetString(fmt.Sprintf("%s.message", template)))
	setMessage((*pluginConfig.Params)["message"])
	showParam("message", plugin.Message)

	// message template.
	plugin.MessageTemplate, err = tmpl.New("message").Funcs(core.TemplateFuncMap).Parse(plugin.Message)
	if err != nil {
		return &Plugin{}, err
	}

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.Timeout = v
		}
	}
	setTimeout(DEFAULT_TIMEOUT)
	setTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.Params)["timeout"])
	showParam("timeout", plugin.Timeout)

	// -----------------------------------------------------------------------------------------------------------------
	// attachments.
	attachments, _ := core.IsMapWithStringAsKey((*pluginConfig.Params)["attachments"])

	// color.
	setColor := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["color"] = 0
			plugin.Color = v
		}
	}
	setColor(DEFAULT_ATTACHMENTS_COLOR)
	setColor(pluginConfig.Config.GetString(fmt.Sprintf("%s.attachments.color", template)))
	setColor(attachments["color"])
	showParam("attachments.color", plugin.Color)

	// pretext.
	setPretext := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["pretext"] = 0
			plugin.Attachments = true
			plugin.Pretext = v
		}
	}
	setPretext(pluginConfig.Config.GetString(fmt.Sprintf("%s.attachments.pretext", template)))
	setPretext(attachments["pretext"])
	showParam("attachments.pretext", plugin.Pretext)

	// pretext template.
	plugin.PretextTemplate, err = tmpl.New("pretext").Funcs(core.TemplateFuncMap).Parse(plugin.Pretext)
	if err != nil {
		return &Plugin{}, err
	}

	// text.
	setText := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["text"] = 0
			plugin.Attachments = true
			plugin.Text = v
		}
	}
	setText(pluginConfig.Config.GetString(fmt.Sprintf("%s.attachments.text", template)))
	setText(attachments["text"])
	showParam("attachments.text", plugin.Text)

	// text template.
	if plugin.TextTemplate, err = tmpl.New("text").Funcs(core.TemplateFuncMap).Parse(plugin.Text); err != nil {
		return &Plugin{}, err
	}

	// title.
	setTitle := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["title"] = 0
			plugin.Attachments = true
			plugin.Title = v
		}
	}
	setTitle(pluginConfig.Config.GetString(fmt.Sprintf("%s.attachments.title", template)))
	setText(attachments["title"])
	showParam("attachments.title", plugin.Title)

	// title template.
	if plugin.TitleTemplate, err = tmpl.New("title").Funcs(core.TemplateFuncMap).Parse(plugin.Title); err != nil {
		return &Plugin{}, err
	}

	// title_link.
	setTitleLink := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["title_link"] = 0
			plugin.Attachments = true
			plugin.TitleLink = v
		}
	}
	setTitleLink(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.attachments.title_link", template)))
	setTitleLink(attachments["title_link"])
	showParam("attachments.title_link", plugin.TitleLink)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Slack.

	plugin.Api = slack.New(plugin.Token)

	// Resolve users ids.
	workspaceUsers, err := plugin.Api.GetUsers()
	if err != nil {
		return &Plugin{}, err
	}

	usersId := make([]string, 0)
	for _, user := range plugin.Users {
		found := false

		for _, u := range workspaceUsers {
			if user == u.Name || user == u.ID {
				usersId = append(usersId, u.ID)
				found = true
				break
			}
		}

		if !found {
			return &Plugin{}, fmt.Errorf(ERROR_USER_NOT_FOUND.Error(), user)
		}
	}

	plugin.Users = usersId

	// Resolve channels ids.
	workspaceChannels, _, err := plugin.Api.GetConversations(&slack.GetConversationsParameters{Limit: 100})
	if err != nil {
		return &Plugin{}, err
	}

	channelsId := make([]string, 0)
	for _, channel := range plugin.Channels {
		found := false

		for _, c := range workspaceChannels {
			if channel == c.Name || channel == c.ID {
				channelsId = append(channelsId, c.ID)
				found = true
				break
			}
		}

		if !found {
			return &Plugin{}, fmt.Errorf(ERROR_CHANNEL_NOT_FOUND.Error(), channel)
		}
	}

	plugin.Channels = channelsId

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// channels or users must be set.
	if len(plugin.Channels) == 0 && len(plugin.Users) == 0 {
		return &Plugin{}, ERROR_OUTPUT_NOT_SET
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
