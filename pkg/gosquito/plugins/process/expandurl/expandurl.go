package expandurlProcess

import (
	"errors"
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"net/http"
	"reflect"
	"regexp"
	"time"
)

const (
	PLUGIN_NAME = "expandurl"

	DEFAULT_DEPTH   = 10
	DEFAULT_TIMEOUT = 2
)

var (
	httpSchema  = regexp.MustCompile("http://")
	httpsSchema = regexp.MustCompile("https://")
)

func expandUrl(p *Plugin, url string, previousURL string, depth int) string {
	if depth == 0 || url == previousURL {
		return url
	}

	// Try to get redirect from server.
	// We try both schemas: http, https.
	// Example:
	// 1. https://t.co/6dEOqhestf?amp=1 (301, go further)
	// 2. https://apne.ws/BvY2ib9 (<- this doesn't work, https port closed)
	// 3. we now try http://apne.ws/BvY2ib9
	// 4. that gives https://apnews.com/article/virus-outbreak-donald-trump-wisconsin-mike ...
	v1, b1 := getRedirectFromServer(p, url)
	v2, b2 := getRedirectFromServer(p, swapURLSchema(url))

	if b1 {
		return expandUrl(p, v1, url, depth-1)

	} else if b2 {
		return expandUrl(p, v2, url, depth-1)

	} else {
		return url
	}
}

func getRedirectFromServer(p *Plugin, url string) (string, bool) {
	f := func(req *http.Request, via []*http.Request) error {
		return errors.New("server redirect detected, not really error")
	}

	client := &http.Client{
		CheckRedirect: f,
		Timeout:       time.Duration(p.OptionTimeout) * time.Second,
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", p.OptionUserAgent)
	resp, _ := client.Do(req)

	if resp != nil {
		loc, err := resp.Location()

		if err == nil && loc.String() != "" {
			return loc.String(), true
		}
	}

	return url, false
}

func swapURLSchema(s string) string {
	if httpSchema.MatchString(s) {
		return httpSchema.ReplaceAllString(s, "https://")

	} else if httpsSchema.MatchString(s) {
		return httpsSchema.ReplaceAllString(s, "http://")
	}

	return s
}

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionDepth     int
	OptionInclude   bool
	OptionInput     []string
	OptionOutput    []string
	OptionRequire   []int
	OptionTimeout   int
	OptionUserAgent string
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

func (p *Plugin) GetRequire() []int {
	return p.OptionRequire
}

func (p *Plugin) Process(data []*core.Datum) ([]*core.Datum, error) {
	temp := make([]*core.Datum, 0)
  p.LogFields["run"] = p.Flow.GetRunID()

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		expanded := false

		for index, input := range p.OptionInput {
			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDatumField(item, input)
			ro, _ := core.ReflectDatumField(item, p.OptionOutput[index])

			switch ri.Kind() {
			case reflect.Slice:
				for i := 0; i < ri.Len(); i++ {
					expandedUrl := expandUrl(p, ri.Index(i).String(), "", p.OptionDepth)

					if expandedUrl != ri.Index(i).String() {
						expanded = true
						ro.Set(reflect.Append(ro, reflect.ValueOf(expandedUrl)))
					}

					core.LogProcessPlugin(p.LogFields,
						fmt.Sprintf("expandurl: source url: %s, depth: %d, expanded url: %s",
							ri.Index(i).String(), p.OptionDepth, expandedUrl))
				}
			}
		}

		if expanded {
			temp = append(temp, item)
		}
	}

	return temp, nil
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
			"id":     pluginConfig.PluginID,
			"alias":  pluginConfig.PluginAlias,
		},
		PluginID:    pluginConfig.PluginID,
		PluginAlias: pluginConfig.PluginAlias,
		PluginName:  PLUGIN_NAME,
		PluginType:  pluginConfig.PluginType,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"include": -1,
		"require": -1,
		"timeout": -1,

		"depth":      -1,
		"input":      1,
		"output":     1,
		"user_agent": -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin settings or set defaults.

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// -----------------------------------------------------------------------------------------------------------------

	// depth.
	setDepth := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["depth"] = 0
			plugin.OptionDepth = v
		}
	}
	setDepth(DEFAULT_DEPTH)
	setDepth(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.depth", template)))
	setDepth((*pluginConfig.PluginParams)["depth"])
	core.ShowPluginParam(plugin.LogFields, "depth", plugin.OptionDepth)

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
			if err := core.IsDatumFieldsSlice(&v); err == nil {
				availableParams["input"] = 0
				plugin.OptionInput = v
			}
		}
	}
	setInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.input", template)))
	setInput((*pluginConfig.PluginParams)["input"])
	core.ShowPluginParam(plugin.LogFields, "input", plugin.OptionInput)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			if err := core.IsDatumFieldsSlice(&v); err == nil {
				availableParams["output"] = 0
				plugin.OptionOutput = v
			}
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

	// 1. input and output must have equal size.
	if len(plugin.OptionInput) != len(plugin.OptionOutput) {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput)
	}

	if err := core.IsDatumFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
