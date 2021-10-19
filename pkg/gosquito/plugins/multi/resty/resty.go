package restyMulti

import (
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"time"
	//log "github.com/livelace/logrus"
	//tmpl "text/template"
)

const (
	DEFAULT_METHOD     = "GET"
	DEFAULT_REDIRECT   = true
	DEFAULT_SSL_VERIFY = true
)

var ()

type Plugin struct {
	Flow *core.Flow

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionAuth                string
	OptionBearerToken         string
	OptionBody                string
	OptionExpireAction        []string
	OptionExpireActionDelay   int64
	OptionExpireActionTimeout int
	OptionExpireInterval      int64
	OptionExpireLast          int64
	OptionHeaders             map[string]interface{}
	OptionInclude             bool
	OptionInput               []string
	OptionMethod              string
	OptionOutput              []string
	OptionParams              map[string]interface{}
	OptionPassword            string
	OptionProxy               string
	OptionRedirect            bool
	OptionRequire             []int
	OptionSSLVerify           bool
	OptionTimeout             int
	OptionUserAgent           string
	OptionUsername            string
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	return data, nil
}

func (p *Plugin) GetAlias() string {
	return "asd"
}

func (p *Plugin) GetFile() string {
	return p.Flow.FlowFile
}

func (p *Plugin) GetId() int {
	return 42
}

func (p *Plugin) GetInclude() bool {
	return true
}

func (p *Plugin) GetInput() []string {
	return []string{}
}

func (p *Plugin) GetName() string {
	return p.PluginName
}

func (p *Plugin) GetOutput() []string {
	//return p.OptionOutput
	return []string{}
}

func (p *Plugin) GetRequire() []int {
	return []int{}
}

func (p *Plugin) GetType() string {
	return p.PluginType
}

func (p *Plugin) LoadState() (map[string]time.Time, error) {
	return make(map[string]time.Time, 0), nil
}

func (p *Plugin) Recv() ([]*core.DataItem, error) {
	return []*core.DataItem{}, nil
}

func (p *Plugin) SaveState(data map[string]time.Time) error {
	return nil
}

func (p *Plugin) Send(data []*core.DataItem) error {
	return nil
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Flow:        pluginConfig.Flow,
		PluginID:    pluginConfig.PluginID,
		PluginAlias: pluginConfig.PluginAlias,
		PluginName:  "resty",
		PluginType:  pluginConfig.PluginType,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// "0" - will be set if parameter is set somehow (defaults, template, config etc.).
	availableParams := map[string]int{
		"cred":     -1,
		"include":  -1,
		"require":  -1,
		"template": -1,
		"timeout":  -1,

		"auth":       -1,
		"body":       -1,
		"headers":    -1,
		"method":     -1,
		"params":     -1,
		"proxy":      -1,
		"redirect":   -1,
		"ssl_verify": -1,
		"user_agent": -1,
	}

	switch pluginConfig.PluginType {
	case "input":
		availableParams["expire_action"] = -1
		availableParams["expire_action_timeout"] = -1
		availableParams["expire_delay"] = -1
		availableParams["expire_interval"] = -1
		availableParams["input"] = 1
		break
	case "process":
		availableParams["input"] = 1
		availableParams["output"] = 1
		break
	case "output":
		availableParams["output"] = 1
		break
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

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

	// bearer_token.
	setBearerToken := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["bearer_token"] = 0
			plugin.OptionBearerToken = v
		}
	}
	setBearerToken(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.bearer_token", cred)))
	setBearerToken((*pluginConfig.PluginParams)["bearer_token"])

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

	switch pluginConfig.PluginType {

	case "input":
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
		showParam("expire_action", plugin.OptionExpireAction)

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
		showParam("expire_action_delay", plugin.OptionExpireActionDelay)

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
		showParam("expire_action_timeout", plugin.OptionExpireActionTimeout)

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
		showParam("expire_interval", plugin.OptionExpireInterval)

		// input.
		setInput := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["input"] = 0
				plugin.OptionInput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
			}
		}
		setInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.input", template)))
		setInput((*pluginConfig.PluginParams)["input"])
		showParam("input", plugin.OptionInput)

		break

	case "process":
		// input.
		setInput := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["input"] = 0
				plugin.OptionInput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
			}
		}
		setInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.input", template)))
		setInput((*pluginConfig.PluginParams)["input"])
		showParam("input", plugin.OptionInput)

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

		break

	case "output":
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

		break
	}

	// auth.
	setAuth := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["auth"] = 0
			plugin.OptionAuth = v
		}
	}
	setAuth(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.auth", template)))
	setAuth((*pluginConfig.PluginParams)["auth"])
	showParam("auth", plugin.OptionAuth)

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

	showParam("headers", plugin.OptionHeaders)

	// method.
	setMethod := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["method"] = 0
			plugin.OptionMethod = v
		}
	}
	setMethod(DEFAULT_METHOD)
	setMethod(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.method", template)))
	setMethod((*pluginConfig.PluginParams)["method"])
	showParam("method", plugin.OptionMethod)

	// params.
	templateParams, _ := core.IsMapWithStringAsKey(pluginConfig.AppConfig.GetStringMap(fmt.Sprintf("%s.params", template)))
	configParams, _ := core.IsMapWithStringAsKey((*pluginConfig.PluginParams)["params"])
	mergedParams := make(map[string]interface{}, 0)

	for k, v := range templateParams {
		mergedParams[k] = v
	}

	for k, v := range configParams {
		mergedParams[k] = v
	}

	plugin.OptionParams = mergedParams

	showParam("params", plugin.OptionParams)

	// proxy.
	setProxy := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["proxy"] = 0
			plugin.OptionProxy = v
		}
	}
	setProxy(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.proxy", template)))
	setProxy((*pluginConfig.PluginParams)["proxy"])
	showParam("proxy", plugin.OptionProxy)

	// redirect.
	setRedirect := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["redirect"] = 0
			plugin.OptionRedirect = v
		}
	}
	setRedirect(DEFAULT_REDIRECT)
	setRedirect(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.redirect", template)))
	setRedirect((*pluginConfig.PluginParams)["redirect"])
	showParam("redirect", plugin.OptionRedirect)

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

	// user_agent.
	setUserAgent := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["user_agent"] = 0
			plugin.OptionUserAgent = v
		}
	}
	setUserAgent(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_USER_AGENT))
	setUserAgent(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.user_agent", template)))
	setUserAgent((*pluginConfig.PluginParams)["user_agent"])
	showParam("user_agent", plugin.OptionUserAgent)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
