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

func uploadFile(p *Plugin, channel string, file string) error {
	mime, err := core.DetectFileType(file)
	if err != nil {
		return err
	}

	_, err = p.SlackClient.UploadFile(slack.FileUploadParameters{
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
				"hash":   p.Flow.FlowHash,
				"flow":   p.Flow.FlowName,
				"file":   p.Flow.FlowFile,
				"plugin": p.PluginName,
				"type":   p.PluginType,
				"error":  fmt.Sprintf("cannot upload file to channel: %s, %s, %v", channel, file, err),
			}).Error(core.LOG_PLUGIN_DATA)
			continue
		}
	}
}

type Plugin struct {
	Flow *core.Flow

	PluginName string
	PluginType string

	SlackClient *slack.Client

	OptionChannels        []string
	OptionFiles           []string
	OptionMessage         string
	OptionMessageTemplate *tmpl.Template
	OptionOutput          []string
	OptionTimeout         int
	OptionToken           string
	OptionUsers           []string
	OptionURL             string

	OptionAttachments     bool
	OptionColor           string
	OptionPretext         string
	OptionPretextTemplate *tmpl.Template
	OptionTitle           string
	OptionTitleTemplate   *tmpl.Template
	OptionTitleLink       []string
	OptionText            string
	OptionTextTemplate    *tmpl.Template
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

	// Logging.
	logError := func(msg string) {
		log.WithFields(log.Fields{
			"hash":   p.Flow.FlowHash,
			"flow":   p.Flow.FlowName,
			"file":   p.Flow.FlowFile,
			"plugin": p.PluginName,
			"type":   p.PluginType,
			"error":  msg,
		}).Error(core.LOG_PLUGIN_DATA)
	}

	// Process and send data.
	for _, item := range data {

		// Files.
		files := make([]string, 0)
		for _, v := range p.OptionFiles {
			files = append(files, core.ExtractDataFieldIntoArray(item, v)...)
		}

		// Message text.
		message, err := core.ExtractTemplateIntoString(item, p.OptionMessageTemplate)
		if err != nil {
			return err
		}

		// Attachments.
		attachments := slack.Attachment{}

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

			attachments.Color = color
			attachments.Pretext = pretext
			attachments.Text = text
			attachments.Title = title
			attachments.TitleLink = titleLink
		}

		// Send to channels.
		for _, channel := range p.OptionChannels {
			_, _, err := p.SlackClient.PostMessage(
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
		for _, user := range p.OptionUsers {
			ch, _, _, err := p.SlackClient.OpenConversation(&slack.OpenConversationParameters{Users: []string{user}})
			if err != nil {
				logError(fmt.Sprintf("cannot establish connection to user: %s, %v", user, err))
				sendFail = true
				continue
			}

			_, _, err = p.SlackClient.PostMessage(
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

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Flow:       pluginConfig.Flow,
		PluginName: "slack",
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
			"hash":   plugin.Flow.FlowHash,
			"flow":   plugin.Flow.FlowName,
			"file":   plugin.Flow.FlowFile,
			"plugin": plugin.PluginName,
			"type":   plugin.PluginType,
			"value":  fmt.Sprintf("%s: %v", p, v),
		}).Debug(core.LOG_SET_VALUE)
	}

	// -----------------------------------------------------------------------------------------------------------------

	cred, _ := core.IsString((*pluginConfig.PluginParams)["cred"])

	// token.
	setToken := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["token"] = 0
			plugin.OptionToken = v
		}
	}
	setToken(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.token", cred)))
	setToken((*pluginConfig.PluginParams)["token"])

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// files.
	setFiles := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["files"] = 0
			plugin.OptionFiles = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setFiles(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.files", template)))
	setFiles((*pluginConfig.PluginParams)["files"])
	showParam("files", plugin.OptionFiles)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.OptionOutput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setOutput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.output", template)))
	setOutput((*pluginConfig.PluginParams)["output"])
	showParam("output", plugin.OptionOutput)

	for _, v := range plugin.OptionOutput {
		if t, b := core.IsChatUsername(v); b {
			plugin.OptionUsers = append(plugin.OptionUsers, t)
		} else {
			plugin.OptionChannels = append(plugin.OptionChannels, t)
		}
	}
	showParam("channels", plugin.OptionChannels)
	showParam("users", plugin.OptionUsers)

	// message.
	setMessage := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["message"] = 0
			plugin.OptionMessage = v
		}
	}
	setMessage(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.message", template)))
	setMessage((*pluginConfig.PluginParams)["message"])
	showParam("message", plugin.OptionMessage)

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
	showParam("timeout", plugin.OptionTimeout)

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
	showParam("attachments.color", plugin.OptionColor)

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
	showParam("attachments.pretext", plugin.OptionPretext)

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
	showParam("attachments.text", plugin.OptionText)

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
	showParam("attachments.title", plugin.OptionTitle)

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
	showParam("attachments.title_link", plugin.OptionTitleLink)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Slack.

	plugin.SlackClient = slack.New(plugin.OptionToken)

	// Resolve users ids.
	workspaceUsers, err := plugin.SlackClient.GetUsers()
	if err != nil {
		return &Plugin{}, err
	}

	usersId := make([]string, 0)
	for _, user := range plugin.OptionUsers {
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

	plugin.OptionUsers = usersId

	// Resolve channels ids.
	workspaceChannels, _, err := plugin.SlackClient.GetConversations(&slack.GetConversationsParameters{Limit: 100})
	if err != nil {
		return &Plugin{}, err
	}

	channelsId := make([]string, 0)
	for _, channel := range plugin.OptionChannels {
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

	plugin.OptionChannels = channelsId

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// channels or users must be set.
	if len(plugin.OptionChannels) == 0 && len(plugin.OptionUsers) == 0 {
		return &Plugin{}, ERROR_OUTPUT_NOT_SET
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
