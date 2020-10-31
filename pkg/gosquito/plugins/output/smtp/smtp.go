package smtpOut

import (
	"errors"
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/xhit/go-simple-mail/v2"
	"path"
	tmpl "text/template"
	"time"
)

const (
	DEFAULT_BODY_LENGTH    = 10000
	DEFAULT_BODY_HTML      = true
	DEFAULT_SMTP_PORT      = 25
	DEFAULT_SSL_ENABLE     = true
	DEFAULT_SUBJECT_LENGTH = 100
)

var (
	ERROR_SMTP_CONNECT_ERROR = errors.New("smtp connect error: %s")
	ERROR_SMTP_SEND_ERROR    = errors.New("smtp send error: %s")
)

type Plugin struct {
	Hash string
	Flow string

	File string
	Name string
	Type string

	Attachments     []string
	Body            string
	BodyTemplate    *tmpl.Template
	BodyLength      int
	BodyHTML        bool
	From            string
	Headers         map[string]interface{}
	Output          []string
	Server          string
	SSL             bool
	Subject         string
	SubjectTemplate *tmpl.Template
	SubjectLength   int
	Password        string
	Port            int
	Timeout         int
	Username        string
}

func (p *Plugin) Send(data []*core.DataItem) error {
	// Connection settings.
	server := mail.NewSMTPClient()

	server.Host = p.Server
	server.Port = p.Port

	if p.Username != "" && p.Password != "" {
		server.Authentication = mail.AuthPlain
		server.Username = p.Username
		server.Password = p.Password
	}

	if p.SSL {
		server.Encryption = mail.EncryptionTLS
	}

	server.KeepAlive = true
	server.ConnectTimeout = time.Duration(p.Timeout) * time.Second
	server.SendTimeout = time.Duration(p.Timeout) * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		return fmt.Errorf(ERROR_SMTP_CONNECT_ERROR.Error(), err)
	}

	// Send data.
	for _, item := range data {

		for _, to := range p.Output {
			b, err := core.ExtractTemplateIntoString(item, p.BodyTemplate)
			if err != nil {
				return err
			}
			body := core.ShrinkString(&b, p.BodyLength)

			s, err := core.ExtractTemplateIntoString(item, p.SubjectTemplate)
			if err != nil {
				return err
			}
			subject := core.ShrinkString(&s, p.SubjectLength)

			// Assemble letter.
			email := mail.NewMSG()
			email.SetFrom(p.From).AddTo(to).SetSubject(subject)

			// Set body.
			if p.BodyHTML {
				email.SetBody(mail.TextHTML, body)
			} else {
				email.SetBody(mail.TextPlain, body)
			}

			// Add attachments.
			if len(p.Attachments) > 0 {
				attachments := core.ExtractDataFieldIntoArray(item, p.Attachments)

				for _, v := range attachments {
					email.AddAttachment(v, path.Base(v))
				}
			}

			// Add headers.
			if len(p.Headers) > 0 {
				for k, v := range p.Headers {
					s := core.ExtractDataFieldIntoString(item, v)
					email.AddHeader(k, s)
				}
			}

			// Send letter.
			err = email.Send(smtpClient)
			if err != nil {
				return fmt.Errorf(ERROR_SMTP_SEND_ERROR.Error(), err)
			}
		}
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
		Name: "smtp",
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

		"attachments":    -1,
		"body":           1,
		"body_html":      -1,
		"body_length":    -1,
		"from":           1,
		"headers":        -1,
		"output":         1,
		"port":           -1,
		"server":         1,
		"ssl":            -1,
		"subject":        1,
		"subject_length": -1,
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

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.Params)["template"])

	// attachments.
	setAttachments := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["attachments"] = 0
			plugin.Attachments = v
		}
	}
	setAttachments(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.attachments", template)))
	setAttachments((*pluginConfig.Params)["attachments"])
	showParam("attachments", plugin.Attachments)

	// body.
	setBody := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["body"] = 0
			plugin.Body = v
		}
	}
	setBody(pluginConfig.Config.GetString(fmt.Sprintf("%s.body", template)))
	setBody((*pluginConfig.Params)["body"])
	showParam("body", plugin.Body)

	// body template.
	if plugin.BodyTemplate, err = tmpl.New("body").Funcs(core.TemplateFuncMap).Parse(plugin.Body); err != nil {
		return &Plugin{}, err
	}

	// body_html.
	setBodyHTML := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["body_html"] = 0
			plugin.BodyHTML = v
		}
	}
	setBodyHTML(DEFAULT_BODY_HTML)
	setBodyHTML(pluginConfig.Config.GetString(fmt.Sprintf("%s.body_html", template)))
	setBodyHTML((*pluginConfig.Params)["body_html"])
	showParam("body_html", plugin.BodyHTML)

	// body_length.
	setBodyLength := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["body_length"] = 0
			plugin.BodyLength = v
		}
	}
	setBodyLength(DEFAULT_BODY_LENGTH)
	setBodyLength(pluginConfig.Config.GetInt(fmt.Sprintf("%s.body_length", template)))
	setBodyLength((*pluginConfig.Params)["body_length"])
	showParam("body_length", plugin.BodyLength)

	// from.
	setFrom := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["from"] = 0
			plugin.From = v
		}
	}
	setFrom(pluginConfig.Config.GetString(fmt.Sprintf("%s.from", template)))
	setFrom((*pluginConfig.Params)["from"])
	showParam("from", plugin.From)

	// headers.
	templateHeaders, _ := core.IsMapWithStringAsKey(pluginConfig.Config.GetStringMap(fmt.Sprintf("%s.headers", template)))
	configHeaders, _ := core.IsMapWithStringAsKey((*pluginConfig.Params)["headers"])
	mergedHeaders := make(map[string]interface{}, 0)

	for k, v := range templateHeaders {
		mergedHeaders[k] = v
	}

	for k, v := range configHeaders {
		mergedHeaders[k] = v
	}

	plugin.Headers = mergedHeaders

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

	// port.
	setPort := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["port"] = 0
			plugin.Port = v
		}
	}
	setPort(DEFAULT_SMTP_PORT)
	setPort(pluginConfig.Config.GetInt(fmt.Sprintf("%s.port", template)))
	setPort((*pluginConfig.Params)["port"])
	showParam("port", plugin.Port)

	// server.
	setServer := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["server"] = 0
			plugin.Server = v
		}
	}
	setServer(pluginConfig.Config.GetString(fmt.Sprintf("%s.server", template)))
	setServer((*pluginConfig.Params)["server"])

	// ssl.
	setSSL := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["ssl"] = 0
			plugin.SSL = v
		}
	}
	setSSL(DEFAULT_SSL_ENABLE)
	setSSL(pluginConfig.Config.GetString(fmt.Sprintf("%s.ssl", template)))
	setSSL((*pluginConfig.Params)["ssl"])
	showParam("ssl", plugin.SSL)

	// subject.
	setSubject := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["subject"] = 0
			plugin.Subject = v
		}
	}
	setSubject(pluginConfig.Config.GetString(fmt.Sprintf("%s.subject", template)))
	setSubject((*pluginConfig.Params)["subject"])
	showParam("subject", plugin.Subject)

	// subject template.
	plugin.SubjectTemplate, err = tmpl.New("subject").Funcs(core.TemplateFuncMap).Parse(plugin.Subject)
	if err != nil {
		return &Plugin{}, err
	}

	// subject_length.
	setSubjectLength := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["subject_length"] = 0
			plugin.SubjectLength = v
		}
	}
	setSubjectLength(DEFAULT_SUBJECT_LENGTH)
	setSubjectLength(pluginConfig.Config.GetInt(fmt.Sprintf("%s.subject_length", template)))
	setSubjectLength((*pluginConfig.Params)["subject_length"])
	showParam("subject_length", plugin.SubjectLength)

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

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
