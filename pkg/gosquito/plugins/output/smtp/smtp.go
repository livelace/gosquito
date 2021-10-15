package smtpOut

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/xhit/go-simple-mail/v2"
	"path"
	"strings"
	tmpl "text/template"
	"time"
)

const (
	DEFAULT_BODY_LENGTH    = 10000
	DEFAULT_BODY_HTML      = true
	DEFAULT_SMTP_PORT      = 25
	DEFAULT_SSL_ENABLE     = false
	DEFAULT_SSL_VERIFY     = true
	DEFAULT_SUBJECT_LENGTH = 100
)

var (
	ERROR_SMTP_CONNECT_ERROR = errors.New("smtp connect error: %s")
	ERROR_SMTP_SEND_ERROR    = errors.New("smtp send error: %s")
)

type Plugin struct {
	Flow *core.Flow

	PluginName string
	PluginType string

	OptionAttachments     []string
	OptionBody            string
	OptionBodyTemplate    *tmpl.Template
	OptionBodyLength      int
	OptionBodyHTML        bool
	OptionFrom            string
	OptionHeaders         map[string]interface{}
	OptionOutput          []string
	OptionServer          string
	OptionSSL             bool
	OptionSSLVerify       bool
	OptionSubject         string
	OptionSubjectTemplate *tmpl.Template
	OptionSubjectLength   int
	OptionPassword        string
	OptionPort            int
	OptionTimeout         int
	OptionUsername        string
}

func (p *Plugin) Send(data []*core.DataItem) error {
	// Connection settings.
	server := mail.NewSMTPClient()

	server.Host = p.OptionServer
	server.Port = p.OptionPort

	if p.OptionUsername != "" && p.OptionPassword != "" {
		server.Authentication = mail.AuthPlain
		server.Username = p.OptionUsername
		server.Password = p.OptionPassword
	}

	if p.OptionSSL {
		server.Encryption = mail.EncryptionTLS
		server.TLSConfig = &tls.Config{InsecureSkipVerify: !p.OptionSSLVerify}
	}

	server.KeepAlive = true
	server.ConnectTimeout = time.Duration(p.OptionTimeout) * time.Second
	server.SendTimeout = time.Duration(p.OptionTimeout) * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		return fmt.Errorf(ERROR_SMTP_CONNECT_ERROR.Error(), err)
	}

	// Send data.
	for _, item := range data {

		for _, to := range p.OptionOutput {
			b, err := core.ExtractTemplateIntoString(item, p.OptionBodyTemplate)
			if err != nil {
				return err
			}
			body := core.ShrinkString(&b, p.OptionBodyLength)

			s, err := core.ExtractTemplateIntoString(item, p.OptionSubjectTemplate)
			if err != nil {
				return err
			}
			s = strings.ReplaceAll(s, "\n", " ")
			subject := core.ShrinkString(&s, p.OptionSubjectLength)

			// Assemble letter.
			email := mail.NewMSG()
			email.SetFrom(p.OptionFrom).AddTo(to).SetSubject(subject)

			// Set body.
			if p.OptionBodyHTML {
				email.SetBody(mail.TextHTML, body)
			} else {
				email.SetBody(mail.TextPlain, body)
			}

			// Add attachments.
			if len(p.OptionAttachments) > 0 {
				attachments := core.ExtractDataFieldIntoArray(item, p.OptionAttachments)

				for _, v := range attachments {
					email.AddAttachment(v, path.Base(v))
				}
			}

			// Add headers.
			if len(p.OptionHeaders) > 0 {
				for k, v := range p.OptionHeaders {
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

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Flow:       pluginConfig.Flow,
		PluginName: "smtp",
		PluginType: "output",
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
		"ssl_verify":     -1,
		"subject":        1,
		"subject_length": -1,
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

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// attachments.
	setAttachments := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["attachments"] = 0
			plugin.OptionAttachments = v
		}
	}
	setAttachments(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.attachments", template)))
	setAttachments((*pluginConfig.PluginParams)["attachments"])
	showParam("attachments", plugin.OptionAttachments)

	// body.
	setBody := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["body"] = 0
			plugin.OptionBody = v
		}
	}
	setBody(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.body", template)))
	setBody((*pluginConfig.PluginParams)["body"])
	showParam("body", plugin.OptionBody)

	// body template.
	if plugin.OptionBodyTemplate, err = tmpl.New("body").Funcs(core.TemplateFuncMap).Parse(plugin.OptionBody); err != nil {
		return &Plugin{}, err
	}

	// body_html.
	setBodyHTML := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["body_html"] = 0
			plugin.OptionBodyHTML = v
		}
	}
	setBodyHTML(DEFAULT_BODY_HTML)
	setBodyHTML(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.body_html", template)))
	setBodyHTML((*pluginConfig.PluginParams)["body_html"])
	showParam("body_html", plugin.OptionBodyHTML)

	// body_length.
	setBodyLength := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["body_length"] = 0
			plugin.OptionBodyLength = v
		}
	}
	setBodyLength(DEFAULT_BODY_LENGTH)
	setBodyLength(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.body_length", template)))
	setBodyLength((*pluginConfig.PluginParams)["body_length"])
	showParam("body_length", plugin.OptionBodyLength)

	// from.
	setFrom := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["from"] = 0
			plugin.OptionFrom = v
		}
	}
	setFrom(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.from", template)))
	setFrom((*pluginConfig.PluginParams)["from"])
	showParam("from", plugin.OptionFrom)

	// headers.
	templateHeaders, _ := core.IsMapWithStringAsKey(pluginConfig.AppConfig.GetStringMap(fmt.Sprintf("%s.headers", template)))
	configHeaders, _ := core.IsMapWithStringAsKey((*pluginConfig.PluginParams)["headers"])
	mergedHeaders := make(map[string]interface{}, 0)

	for k, v := range templateHeaders {
		mergedHeaders[k] = v
	}

	for k, v := range configHeaders {
		mergedHeaders[k] = v
	}

	plugin.OptionHeaders = mergedHeaders

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

	// port.
	setPort := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["port"] = 0
			plugin.OptionPort = v
		}
	}
	setPort(DEFAULT_SMTP_PORT)
	setPort(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.port", template)))
	setPort((*pluginConfig.PluginParams)["port"])
	showParam("port", plugin.OptionPort)

	// server.
	setServer := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["server"] = 0
			plugin.OptionServer = v
		}
	}
	setServer(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.server", template)))
	setServer((*pluginConfig.PluginParams)["server"])

	// ssl.
	setSSL := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["ssl"] = 0
			plugin.OptionSSL = v
		}
	}
	setSSL(DEFAULT_SSL_ENABLE)
	setSSL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.ssl", template)))
	setSSL((*pluginConfig.PluginParams)["ssl"])
	showParam("ssl", plugin.OptionSSL)

	// ssl_verify.
	setSSLVerify := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["ssl_verify"] = 0
			plugin.OptionSSLVerify = v
		}
	}
	setSSLVerify(DEFAULT_SSL_VERIFY)
	setSSLVerify(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.ssl_verify", template)))
	setSSLVerify((*pluginConfig.PluginParams)["ssl_verify"])
	showParam("ssl_verify", plugin.OptionSSLVerify)

	// subject.
	setSubject := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["subject"] = 0
			plugin.OptionSubject = v
		}
	}
	setSubject(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.subject", template)))
	setSubject((*pluginConfig.PluginParams)["subject"])
	showParam("subject", plugin.OptionSubject)

	// subject template.
	plugin.OptionSubjectTemplate, err = tmpl.New("subject").Funcs(core.TemplateFuncMap).Parse(plugin.OptionSubject)
	if err != nil {
		return &Plugin{}, err
	}

	// subject_length.
	setSubjectLength := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["subject_length"] = 0
			plugin.OptionSubjectLength = v
		}
	}
	setSubjectLength(DEFAULT_SUBJECT_LENGTH)
	setSubjectLength(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.subject_length", template)))
	setSubjectLength((*pluginConfig.PluginParams)["subject_length"])
	showParam("subject_length", plugin.OptionSubjectLength)

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

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
