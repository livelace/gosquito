package restyMulti

import (
	"crypto/tls"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"reflect"
	"strings"
	"sync"
	tmpl "text/template"
	"time"
)

const (
	PLUGIN_NAME = "resty"

	DEFAULT_MATCH_TTL  = "1d"
	DEFAULT_METHOD     = "GET"
	DEFAULT_REDIRECT   = true
	DEFAULT_SEND_DELAY = "1ms"
	DEFAULT_SSL_VERIFY = true
)

func restyClient(p *Plugin) *resty.Client {
	client := resty.New()

	// auth = basic.
	if p.OptionAuth == "basic" && p.OptionUsername != "" && p.OptionPassword != "" {
		client.SetBasicAuth(p.OptionUsername, p.OptionPassword)
	}

	// auth = bearer.
	if p.OptionAuth == "bearer" && p.OptionBearerToken != "" {
		client.SetAuthToken(p.OptionBearerToken)
	}

	// Set proxy.
	if p.OptionProxy != "" {
		client.SetProxy(p.OptionProxy)
	}

	// Set redirect.
	if p.OptionRedirect {
		client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(15))
	} else {
		client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(0))
	}

	// Set ssl_verify.
	if p.OptionSSLVerify {
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: false})
	} else {
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}

	// Set timeout.
	client.SetTimeout(time.Duration(p.OptionTimeout) * time.Second)

	// Set user_agent.
	client.SetHeader("User-Agent", p.OptionUserAgent)

	return client
}

type Plugin struct {
	m sync.Mutex

	Flow *core.Flow

	LogFields log.Fields

	RestyClient *resty.Client

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionAuth                string
	OptionBearerToken         string
	OptionBody                string
	OptionBodyTemplate        *tmpl.Template
	OptionExpireAction        []string
	OptionExpireActionDelay   int64
	OptionExpireActionTimeout int
	OptionExpireInterval      int64
	OptionExpireLast          int64
	OptionHeaders             map[string]string
	OptionHeadersTemplate     map[string]*tmpl.Template
	OptionInclude             bool
	OptionInput               []string
	OptionMatchSignature      []string
	OptionMatchTTL            time.Duration
	OptionMethod              string
	OptionOutput              []string
	OptionParams              map[string]string
	OptionParamsTemplate      map[string]*tmpl.Template
	OptionPassword            string
	OptionProxy               string
	OptionRedirect            bool
	OptionRequire             []int
	OptionSendDelay           time.Duration
	OptionSSLVerify           bool
	OptionTarget              string
	OptionTimeFormat          string
	OptionTimeFormatA         string
	OptionTimeFormatB         string
	OptionTimeFormatC         string
	OptionTimeZone            *time.Location
	OptionTimeZoneA           *time.Location
	OptionTimeZoneB           *time.Location
	OptionTimeZoneC           *time.Location
	OptionTimeout             int
	OptionUserAgent           string
	OptionUsername            string
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

func (p *Plugin) GetInclude() bool {
	return p.OptionInclude
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

func (p *Plugin) GetRequire() []int {
	return p.OptionRequire
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

func (p *Plugin) Process(data []*core.Datum) ([]*core.Datum, error) {
	temp := make([]*core.Datum, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Perform request func.
	makeRequest := func(item *core.Datum) (*resty.Response, error) {
		var resp *resty.Response
		var err error

		// Format body.
		body, err := core.ExtractTemplateIntoString(item, p.OptionBodyTemplate)
		if err != nil {
			return resp, err
		}

		// Format headers.
		headers, err := core.ExtractTemplateMapIntoStringMap(item, p.OptionHeadersTemplate)
		if err != nil {
			return resp, err
		}
		p.RestyClient.SetHeaders(headers)

		// Format params.
		params, err := core.ExtractTemplateMapIntoStringMap(item, p.OptionParamsTemplate)
		if err != nil {
			return resp, err
		}
		p.RestyClient.SetQueryParams(params)

		switch p.OptionMethod {
		case "GET":
			resp, err = p.RestyClient.R().SetBody(body).Get(p.OptionTarget)
			break
		case "POST":
			resp, err = p.RestyClient.R().SetBody(body).Post(p.OptionTarget)
			break
		}

		if err == nil && !(resp.StatusCode() < 200 || resp.StatusCode() >= 300) {
			core.LogProcessPlugin(p.LogFields, fmt.Sprintf("%s %s %v",
				p.OptionMethod, p.OptionTarget, resp.StatusCode()))
		} else {
			core.LogProcessPlugin(p.LogFields, fmt.Errorf("%s %s %v", p.OptionMethod, p.OptionTarget, err))
		}

		return resp, err
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		for index, input := range p.OptionInput {
			ri, _ := core.ReflectDatumField(item, input)
			ro, _ := core.ReflectDatumField(item, p.OptionOutput[index])

			switch ri.Kind() {
			case reflect.String:
				item.ITER.INDEX = 0
				item.ITER.VALUE = ri.String()

				resp, err := makeRequest(item)

				if err == nil && !(resp.StatusCode() < 200 || resp.StatusCode() >= 300) {
					ro.SetString(fmt.Sprintf("%s", resp.Body()))
				}

			case reflect.Slice:
				for i := 0; i < ri.Len(); i++ {
					item.ITER.INDEX = i
					item.ITER.VALUE = ri.Index(i).String()

					resp, err := makeRequest(item)

					if err == nil && !(resp.StatusCode() < 200 || resp.StatusCode() >= 300) {
						ro.Set(reflect.Append(ro, reflect.ValueOf(fmt.Sprintf("%s", resp.Body()))))
					}
				}
			}

			if ro.Len() > 0 {
				temp = append(temp, item)
			}
		}
	}

	return temp, nil
}

func (p *Plugin) Receive() ([]*core.Datum, error) {
	p.LogFields["run"] = p.Flow.GetRunID()
	currentTime := time.Now().UTC()
	failedSources := make([]string, 0)
	temp := make([]*core.Datum, 0)

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}
	core.LogInputPlugin(p.LogFields, "all", fmt.Sprintf("states loaded: %d", len(flowStates)))

	for _, source := range p.OptionInput {
		var itemNew = false
		var itemSignature string
		var itemSignatureHash string
		var itemTime = currentTime
		var sourceLastTime time.Time
		var u, _ = uuid.NewRandom()

		var resp *resty.Response

		// Check if we work with source first time.
		if v, ok := flowStates[source]; ok {
			sourceLastTime = v
		} else {
			sourceLastTime = time.Unix(0, 0)
		}

		// Datum template for query formatting.
		itemTpl := &core.Datum{
			FLOW:        p.Flow.FlowName,
			PLUGIN:      p.PluginName,
			SOURCE:      source,
			TIME:        itemTime,
			TIMEFORMAT:  itemTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
			TIMEFORMATA: itemTime.In(p.OptionTimeZoneA).Format(p.OptionTimeFormatA),
			TIMEFORMATB: itemTime.In(p.OptionTimeZoneB).Format(p.OptionTimeFormatB),
			TIMEFORMATC: itemTime.In(p.OptionTimeZoneC).Format(p.OptionTimeFormatC),
			UUID:        u,
		}

		// Format body.
		body, err := core.ExtractTemplateIntoString(itemTpl, p.OptionBodyTemplate)
		if err != nil {
			return temp, err
		}

		// Format headers.
		headers, err := core.ExtractTemplateMapIntoStringMap(itemTpl, p.OptionHeadersTemplate)
		if err != nil {
			return temp, err
		}
		p.RestyClient.SetHeaders(headers)

		// Format params.
		params, err := core.ExtractTemplateMapIntoStringMap(itemTpl, p.OptionParamsTemplate)
		if err != nil {
			return temp, err
		}
		p.RestyClient.SetQueryParams(params)

		// Perform request.
		switch p.OptionMethod {
		case "GET":
			resp, err = p.RestyClient.R().SetBody(body).Get(source)
			break
		case "POST":
			resp, err = p.RestyClient.R().SetBody(body).Post(source)
			break
		}

		if err == nil && !(resp.StatusCode() < 200 || resp.StatusCode() >= 300) {
			itemBody := fmt.Sprintf("%s", resp.Body())

			// Process only new items. Two methods:
			// 1. Match item by user provided signature.
			// 2. Pass items as is.
			if len(p.OptionMatchSignature) > 0 {
				for _, v := range p.OptionMatchSignature {
					switch v {
					case "RESTY.BODY":
						itemSignature += itemBody
					}
				}

				// set default value for signature if user provided wrong values.
				if len(itemSignature) == 0 {
					itemSignature += source
				}

				itemSignatureHash = core.HashString(&itemSignature)

				if _, ok := flowStates[itemSignatureHash]; !ok {
					// save item signature hash to state.
					flowStates[itemSignatureHash] = currentTime

					// update source timestamp.
					if itemTime.Unix() > sourceLastTime.Unix() {
						sourceLastTime = itemTime
					}

					itemNew = true
				}

			} else {
				sourceLastTime = itemTime
				itemNew = true
			}

			// Add item to result.
			if itemNew {
				temp = append(temp, &core.Datum{
					FLOW:        p.Flow.FlowName,
					PLUGIN:      p.PluginName,
					SOURCE:      source,
					TIME:        itemTime,
					TIMEFORMAT:  itemTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
					TIMEFORMATA: itemTime.In(p.OptionTimeZoneA).Format(p.OptionTimeFormatA),
					TIMEFORMATB: itemTime.In(p.OptionTimeZoneB).Format(p.OptionTimeFormatB),
					TIMEFORMATC: itemTime.In(p.OptionTimeZoneC).Format(p.OptionTimeFormatC),
					UUID:        u,

					RESTY: core.Resty{
						BODY:       fmt.Sprintf("%s", resp.Body()),
						PROTO:      fmt.Sprintf("%s", resp.Proto()),
						STATUS:     fmt.Sprintf("%s", resp.Status()),
						STATUSCODE: fmt.Sprintf("%v", resp.StatusCode()),
					},

					WARNINGS: make([]string, 0),
				})
			}

			flowStates[source] = sourceLastTime
			core.LogInputPlugin(p.LogFields, source,
				fmt.Sprintf("last update: %s, received data: %d, new data: %v", sourceLastTime, 1, itemNew))

		} else {
			failedSources = append(failedSources, source)
			core.LogInputPlugin(p.LogFields, source, fmt.Errorf("%s %v", p.OptionMethod, err))
			continue
		}
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
		
        if (currentTime.Unix() - sourceTime.Unix()) > p.OptionExpireInterval / 1000 {
			sourcesExpired = true
            
            core.LogInputPlugin(p.LogFields, source, 
                fmt.Sprintf("source expired: %v", currentTime.Sub(sourceTime)))

			// Execute command if expire delay exceeded.
			// ExpireLast keeps last execution timestamp.
			if (currentTime.Unix() - p.OptionExpireLast) > p.OptionExpireActionDelay / 1000 {
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

	// Inform about sources failures.
	if len(failedSources) > 0 {
		return temp, core.ERROR_FLOW_SOURCE_FAIL
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

	var resp *resty.Response

	for _, output := range p.OptionOutput {
		// Iterate over data items (articles, tweets etc.).
		for _, item := range data {
			// Format body.
			body, err := core.ExtractTemplateIntoString(item, p.OptionBodyTemplate)
			if err != nil {
				return err
			}

			// Format headers.
			headers, err := core.ExtractTemplateMapIntoStringMap(item, p.OptionHeadersTemplate)
			if err != nil {
				return err
			}
			p.RestyClient.SetHeaders(headers)

			// Format params.
			params, err := core.ExtractTemplateMapIntoStringMap(item, p.OptionParamsTemplate)
			if err != nil {
				return err
			}
			p.RestyClient.SetQueryParams(params)

			// Perform request.
			switch p.OptionMethod {
			case "GET":
				resp, err = p.RestyClient.R().SetBody(body).Get(output)
			case "POST":
				resp, err = p.RestyClient.R().SetBody(body).Post(output)
			}

			if err == nil && !(resp.StatusCode() < 200 || resp.StatusCode() >= 300) {
				core.LogOutputPlugin(p.LogFields, output,
					fmt.Sprintf("%s %v", p.OptionMethod, resp.StatusCode()))
			} else {
				sendStatus = false
				core.LogOutputPlugin(p.LogFields, output,
					fmt.Errorf("%s %v", p.OptionMethod, err))
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
		PluginID:    pluginConfig.PluginID,
		PluginAlias: pluginConfig.PluginAlias,
		PluginName:  PLUGIN_NAME,
		PluginType:  pluginConfig.PluginType,
	}

	if pluginConfig.PluginType == "process" {
		plugin.LogFields["id"] = pluginConfig.PluginID
		plugin.LogFields["alias"] = pluginConfig.PluginAlias
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// "0" - will be set if parameter is set somehow (defaults, template, config etc.).
	availableParams := map[string]int{
		"cred":     -1,
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
		availableParams["match_signature"] = -1
		availableParams["match_ttl"] = -1
		availableParams["send_delay"] = -1
		availableParams["time_format"] = -1
		availableParams["time_format_a"] = -1
		availableParams["time_format_b"] = -1
		availableParams["time_format_c"] = -1
		availableParams["time_zone"] = -1
		availableParams["time_zone_a"] = -1
		availableParams["time_zone_b"] = -1
		availableParams["time_zone_c"] = -1
		break
	case "process":
		availableParams["include"] = -1
		availableParams["input"] = 1
		availableParams["output"] = 1
		availableParams["require"] = -1
		availableParams["target"] = 1
		break
	case "output":
		availableParams["output"] = 1
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

	// bearer_token.
	setBearerToken := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["bearer_token"] = 0
			plugin.OptionBearerToken = core.GetCredValue(v, vault)
		}
	}
	setBearerToken(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.bearer_token", cred)))
	setBearerToken((*pluginConfig.PluginParams)["bearer_token"])

	// username.
	setUsername := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["username"] = 0
			plugin.OptionUsername = core.GetCredValue(v, vault)
		}
	}
	setUsername(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.username", cred)))
	setUsername((*pluginConfig.PluginParams)["username"])

	// password.
	setPassword := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["password"] = 0
			plugin.OptionPassword = core.GetCredValue(v, vault)
		}
	}
	setPassword(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.password", cred)))
	setPassword((*pluginConfig.PluginParams)["password"])

	// -----------------------------------------------------------------------------------------------------------------

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
				plugin.OptionMatchTTL = time.Duration(v) * time.Millisecond
			}
		}
		setMatchTTL(DEFAULT_MATCH_TTL)
		setMatchTTL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.match_ttl", template)))
		setMatchTTL((*pluginConfig.PluginParams)["match_ttl"])
		core.ShowPluginParam(plugin.LogFields, "match_ttl", plugin.OptionMatchTTL)

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

	case "process":
		// include.
		setInclude := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["include"] = 0
				plugin.OptionInclude = v
			}
		}
		setInclude(pluginConfig.AppConfig.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
		setInclude(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.include", template)))
		setInclude((*pluginConfig.PluginParams)["include"])
		core.ShowPluginParam(plugin.LogFields, "include", plugin.OptionInclude)

		// input.
		setInput := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["input"] = 0
				plugin.OptionInput = v
			}
		}
		setInput((*pluginConfig.PluginParams)["input"])
		core.ShowPluginParam(plugin.LogFields, "input", plugin.OptionInput)

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

		// require.
		setRequire := func(p interface{}) {
			if v, b := core.IsSliceOfInt(p); b {
				availableParams["require"] = 0
				plugin.OptionRequire = v

			}
		}
		setRequire(pluginConfig.AppConfig.GetIntSlice(fmt.Sprintf("%s.require", template)))
		setRequire((*pluginConfig.PluginParams)["require"])
		core.ShowPluginParam(plugin.LogFields, "require", plugin.OptionRequire)

		// target.
		setTarget := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["target"] = 0
				plugin.OptionTarget = v
			}
		}
		setTarget(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.target", template)))
		setTarget((*pluginConfig.PluginParams)["target"])

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
		core.ShowPluginParam(plugin.LogFields, "output", plugin.OptionOutput)
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
	core.ShowPluginParam(plugin.LogFields, "auth", plugin.OptionAuth)

	// body.
	setBody := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["body"] = 0
			plugin.OptionBody = v
		}
	}
	setBody(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.body", template)))
	setBody((*pluginConfig.PluginParams)["body"])
	core.ShowPluginParam(plugin.LogFields, "body", plugin.OptionBody)

	// body template.
	if plugin.OptionBodyTemplate, err = tmpl.New("body").Funcs(core.TemplateFuncMap).Parse(plugin.OptionBody); err != nil {
		return &Plugin{}, err
	}

	// headers.
	templateHeaders, _ := core.IsMapWithStringAsKey(pluginConfig.AppConfig.GetStringMap(fmt.Sprintf("%s.headers", template)))
	configHeaders, _ := core.IsMapWithStringAsKey((*pluginConfig.PluginParams)["headers"])
	mergedHeaders := make(map[string]string, 0)
	mergedHeadersTemplate := make(map[string]*tmpl.Template, 0)

	for k, v := range templateHeaders {
		mergedHeaders[k] = fmt.Sprintf("%s", v)
	}

	for k, v := range configHeaders {
		mergedHeaders[k] = fmt.Sprintf("%s", v)
	}

	for k, v := range mergedHeaders {
		template, err := tmpl.New(k).Funcs(core.TemplateFuncMap).Parse(v)
		if err != nil {
			return &Plugin{}, err
		}
		mergedHeadersTemplate[k] = template
	}

	plugin.OptionHeaders = mergedHeaders
	plugin.OptionHeadersTemplate = mergedHeadersTemplate

	core.ShowPluginParam(plugin.LogFields, "headers", plugin.OptionHeaders)

	// method.
	setMethod := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["method"] = 0
			plugin.OptionMethod = strings.ToUpper(v)
		}
	}
	setMethod(DEFAULT_METHOD)
	setMethod(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.method", template)))
	setMethod((*pluginConfig.PluginParams)["method"])
	core.ShowPluginParam(plugin.LogFields, "method", plugin.OptionMethod)

	// params.
	templateParams, _ := core.IsMapWithStringAsKey(pluginConfig.AppConfig.GetStringMap(fmt.Sprintf("%s.params", template)))
	configParams, _ := core.IsMapWithStringAsKey((*pluginConfig.PluginParams)["params"])
	mergedParams := make(map[string]string, 0)
	mergedParamsTemplate := make(map[string]*tmpl.Template, 0)

	for k, v := range templateParams {
		mergedParams[k] = fmt.Sprintf("%s", v)
	}

	for k, v := range configParams {
		mergedParams[k] = fmt.Sprintf("%s", v)
	}

	for k, v := range mergedParams {
		template, err := tmpl.New(k).Funcs(core.TemplateFuncMap).Parse(v)
		if err != nil {
			return &Plugin{}, err
		}
		mergedParamsTemplate[k] = template
	}

	plugin.OptionParams = mergedParams
	plugin.OptionParamsTemplate = mergedParamsTemplate

	core.ShowPluginParam(plugin.LogFields, "params", plugin.OptionParams)

	// proxy.
	setProxy := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["proxy"] = 0
			plugin.OptionProxy = v
		}
	}
	setProxy(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.proxy", template)))
	setProxy((*pluginConfig.PluginParams)["proxy"])
	core.ShowPluginParam(plugin.LogFields, "proxy", plugin.OptionProxy)

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
	core.ShowPluginParam(plugin.LogFields, "redirect", plugin.OptionRedirect)

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
	core.ShowPluginParam(plugin.LogFields, "ssl_verify", plugin.OptionSSLVerify)

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
	core.ShowPluginParam(plugin.LogFields, "user_agent", plugin.OptionUserAgent)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if pluginConfig.PluginType == "process" {
		if len(plugin.OptionInput) != len(plugin.OptionOutput) {
			return &Plugin{}, fmt.Errorf(
				"%s: %v, %v",
				core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput)
		}

		if err := core.IsDatumFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
			return &Plugin{}, err
		}
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Create resty client.

	plugin.RestyClient = restyClient(&plugin)

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
