package expandurl

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
	v1, b1 := getServerRedirect(p, url)
	v2, b2 := getServerRedirect(p, swapUrlSchema(url))

	if b1 {
		return expandUrl(p, v1, url, depth-1)

	} else if b2 {
		return expandUrl(p, v2, url, depth-1)

	} else {
		return url
	}
}

func getServerRedirect(p *Plugin, url string) (string, bool) {
	f := func(req *http.Request, via []*http.Request) error {
		return errors.New("server redirect detected, not really error")
	}

	client := &http.Client{
		CheckRedirect: f,
		Timeout:       time.Duration(p.Timeout) * time.Second,
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", p.UserAgent)
	resp, _ := client.Do(req)

	if resp != nil {
		loc, err := resp.Location()

		if err == nil && loc.String() != "" {
			return loc.String(), true
		}
	}

	return url, false
}

func swapUrlSchema(s string) string {
	if httpSchema.MatchString(s) {
		return httpSchema.ReplaceAllString(s, "https://")

	} else if httpsSchema.MatchString(s) {
		return httpsSchema.ReplaceAllString(s, "http://")
	}

	return s
}

type Plugin struct {
	Hash string
	Flow string

	ID    int
	Alias string

	File string
	Name string
	Type string

	Include bool
	Require []int

	Depth     int
	Input     []string
	Output    []string
	Timeout   int
	UserAgent string
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		expanded := false

		for index, input := range p.Input {
			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.Output[index])

			switch ri.Kind() {
			case reflect.Slice:
				for i := 0; i < ri.Len(); i++ {
					expandedUrl := expandUrl(p, ri.Index(i).String(), "", p.Depth)

					if expandedUrl != ri.Index(i).String() {
						expanded = true
						ro.Set(reflect.Append(ro, reflect.ValueOf(expandedUrl)))
					}

					log.WithFields(log.Fields{
						"hash":   p.Hash,
						"flow":   p.Flow,
						"file":   p.File,
						"plugin": p.Name,
						"type":   p.Type,
						"id":     p.ID,
						"alias":  p.Alias,
						"data": fmt.Sprintf("expandurl: source url: %s, depth: %d, expanded url: %s",
							ri.Index(i).String(), p.Depth, expandedUrl),
					}).Debug(core.LOG_PLUGIN_DATA)
				}
			}
		}

		if expanded {
			temp = append(temp, item)
		}
	}

	return temp, nil
}

func (p *Plugin) GetId() int {
	return p.ID
}

func (p *Plugin) GetAlias() string {
	return p.Alias
}

func (p *Plugin) GetFile() string {
	return p.File
}

func (p *Plugin) GetName() string {
	return p.Name
}

func (p *Plugin) GetType() string {
	return p.Type
}

func (p *Plugin) GetInclude() bool {
	return p.Include
}

func (p *Plugin) GetRequire() []int {
	return p.Require
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Hash: pluginConfig.Hash,
		Flow: pluginConfig.Flow,

		ID:    pluginConfig.ID,
		Alias: pluginConfig.Alias,

		File: pluginConfig.File,
		Name: "expandurl",
		Type: "process",
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

	template, _ := core.IsString((*pluginConfig.Params)["template"])

	// depth.
	setDepth := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["depth"] = 0
			plugin.Depth = v
		}
	}
	setDepth(DEFAULT_DEPTH)
	setDepth(pluginConfig.Config.GetInt(fmt.Sprintf("%s.depth", template)))
	setDepth((*pluginConfig.Params)["depth"])
	showParam("depth", plugin.Depth)

	// include.
	setInclude := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["include"] = 0
			plugin.Include = v
		}
	}
	setInclude(pluginConfig.Config.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
	setInclude(pluginConfig.Config.GetString(fmt.Sprintf("%s.include", template)))
	setInclude((*pluginConfig.Params)["include"])
	showParam("include", plugin.Include)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			if err := core.IsDataFieldsSlice(&v); err == nil {
				availableParams["input"] = 0
				plugin.Input = v
			}
		}
	}
	setInput(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.input", template)))
	setInput((*pluginConfig.Params)["input"])
	showParam("input", plugin.Input)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			if err := core.IsDataFieldsSlice(&v); err == nil {
				availableParams["output"] = 0
				plugin.Output = v
			}
		}
	}
	setOutput(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.output", template)))
	setOutput((*pluginConfig.Params)["output"])
	showParam("output", plugin.Output)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.Require = v

		}
	}
	setRequire(pluginConfig.Config.GetIntSlice(fmt.Sprintf("%s.require", template)))
	setRequire((*pluginConfig.Params)["require"])
	showParam("require", plugin.Require)

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

	// user_agent.
	setUserAgent := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["user_agent"] = 0
			plugin.UserAgent = v
		}
	}
	setUserAgent(pluginConfig.Config.GetString(core.VIPER_DEFAULT_USER_AGENT))
	setUserAgent(pluginConfig.Config.GetString(fmt.Sprintf("%s.user_agent", template)))
	setUserAgent((*pluginConfig.Params)["user_agent"])
	showParam("user_agent", plugin.UserAgent)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// input and output must have equal size.
	if len(plugin.Input) != len(plugin.Output) {
		return &Plugin{}, fmt.Errorf("%s: %v, %v", core.ERROR_SIZE_MISMATCH.Error(), plugin.Input, plugin.Output)
	} else {
		core.SliceStringToUpper(&plugin.Input)
		core.SliceStringToUpper(&plugin.Output)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
