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
	DEFAULT_ATTACHMENTS_COLOR = "#00C100"
	DEFAULT_TIMEOUT           = 3
)

var (
	ERROR_CHANNEL_NOT_FOUND = errors.New("channel not found: %s")
	ERROR_OUTPUT_NOT_SET    = errors.New("channels and users are not set")
	ERROR_USER_NOT_FOUND    = errors.New("user not found: %s")
	ERROR_SEND_FAIL         = errors.New("sending finished with errors")
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
	fileUploadResponse, response := p.Api.UploadFile(data, channel, fileName)
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

	return filesId
}

type Plugin struct {
	Hash string
	Flow string

	File string
	Name string
	Type string

	Api  *mattermost.Client4
	User *mattermost.User

	Channels        []string
	Files           []string
	Message         string
	MessageTemplate *tmpl.Template
	Output          []string
	Password        string
	Team            string
	Timeout         int
	Username        string
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
		// files.
		files := make([]string, 0)
		for _, v := range p.Files {
			files = append(files, core.ExtractDataFieldIntoArray(item, v)...)
		}

		// attachments.
		message, err := core.ExtractTemplateIntoString(item, p.MessageTemplate)
		if err != nil {
			return err
		}

		props := make(map[string]interface{}, 0)

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
		for _, channel := range p.Channels {
			filesId := uploadFiles(p, channel, &files)

			post := mattermost.Post{
				UserId:    p.User.Id,
				ChannelId: channel,
				Message:   message,
				FileIds:   filesId,
				Props:     props,
			}

			_, res := p.Api.CreatePost(&post)
			if res.Error != nil {
				logError(fmt.Sprintf("cannot send message to channel: %s, %v", channel, res.Error))
				sendFail = true
				continue
			}
		}

		// Send to users.
		for _, user := range p.Users {
			ch, res := p.Api.CreateDirectChannel(p.User.Id, user)
			if res.Error != nil {
				logError(fmt.Sprintf("cannot establish connection to user: %s, %v", user, res.Error))
				sendFail = true
				continue
			}

			filesId := uploadFiles(p, ch.Id, &files)

			post := mattermost.Post{
				UserId:    p.User.Id,
				ChannelId: ch.Id,
				Message:   message,
				FileIds:   filesId,
				Props:     props,
			}

			_, res = p.Api.CreatePost(&post)
			if res.Error != nil {
				logError(fmt.Sprintf("cannot send message to user: %s, %v", user, res.Error))
				sendFail = true
				continue
			}
		}
	}

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
		Name: "mattermost",
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

	// username.
	setUsername := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["username"] = 0
			plugin.Username = v
		}
	}
	setUsername(pluginConfig.Config.GetString(fmt.Sprintf("%s.username", cred)))
	setUsername((*pluginConfig.Params)["username"])

	// password.
	setPassword := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["password"] = 0
			plugin.Password = v
		}
	}
	setPassword(pluginConfig.Config.GetString(fmt.Sprintf("%s.password", cred)))
	setPassword((*pluginConfig.Params)["password"])

	// team.
	setTeam := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["team"] = 0
			plugin.Team = v
		}
	}
	setTeam(pluginConfig.Config.GetString(fmt.Sprintf("%s.team", cred)))
	setTeam((*pluginConfig.Params)["team"])

	// url.
	setURL := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["url"] = 0
			plugin.URL = v
		}
	}
	setURL(pluginConfig.Config.GetString(fmt.Sprintf("%s.url", cred)))
	setURL((*pluginConfig.Params)["url"])
	showParam("url", plugin.URL)

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
	// Mattermost.

	// Login.
	plugin.Api = mattermost.NewAPIv4Client(plugin.URL)
	plugin.Api.HttpClient = &http.Client{
		Timeout: time.Duration(plugin.Timeout) * time.Second,
	}

	user, res := plugin.Api.Login(plugin.Username, plugin.Password)
	if res.Error != nil {
		return &Plugin{}, res.Error
	}
	plugin.User = user

	// Team.
	team, res := plugin.Api.GetTeamByName(plugin.Team, "")
	if res.Error != nil {
		return &Plugin{}, res.Error
	}

	// Resolve channels ids.
	channelsId := make([]string, 0)
	for _, channel := range plugin.Channels {
		ch, res := plugin.Api.GetChannelByName(channel, team.Id, "")
		if res.Error == nil {
			channelsId = append(channelsId, ch.Id)
		} else {
			return &Plugin{}, fmt.Errorf(ERROR_CHANNEL_NOT_FOUND.Error(), channel)
		}
	}

	plugin.Channels = channelsId

	// Resolve users ids.
	usersId := make([]string, 0)
	for _, user := range plugin.Users {
		u, res := plugin.Api.GetUserByUsername(user, "")
		if res.Error == nil {
			usersId = append(usersId, u.Id)
		} else {
			return &Plugin{}, fmt.Errorf(ERROR_USER_NOT_FOUND.Error(), user)
		}
	}

	plugin.Users = usersId

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// channels or users must be set.
	if len(plugin.Channels) == 0 && len(plugin.Users) == 0 {
		return &Plugin{}, ERROR_OUTPUT_NOT_SET
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
