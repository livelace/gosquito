package regexpmatch

import (
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"reflect"
	"regexp"
)

func matchRegexes(regexes []*regexp.Regexp, text string) bool {
	for _, re := range regexes {
		if re.MatchString(text) {
			return true
		}
	}

	return false
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

	Input  []string
	Output []string
	Regexp []*regexp.Regexp
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		matched := false

		// Match pattern inside different data fields (Title, Content etc.).
		for index, input := range p.Input {
			var ro reflect.Value

			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)

			if len(p.Output) > 0 {
				ro, _ = core.ReflectDataField(item, p.Output[index])
			}

			// This plugin supports "string" and "[]string" data fields for matching.
			switch ri.Kind() {
			case reflect.String:
				if matchRegexes(p.Regexp, ri.String()) {
					matched = true

					if len(p.Output) > 0 {
						ro.SetString(ri.String())
					}
				}
			case reflect.Slice:
				for i := 0; i < ri.Len(); i++ {
					if matchRegexes(p.Regexp, ri.Index(i).String()) {
						matched = true

						if len(p.Output) > 0 {
							ro.Set(reflect.Append(ro, ri.Index(i)))
						}
					}
				}
			}
		}

		// Append matched item to results.
		if matched {
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
		Name: "regexpmatch",
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

		"input":  1,
		"output": -1,
		"regexp": 1,
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

	// include.
	setInclude := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["include"] = 0
			plugin.Include = v
		}
	}
	setInclude(pluginConfig.Config.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
	setInclude((*pluginConfig.Params)["include"])
	showParam("include", plugin.Include)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.Input = v
		}
	}
	setInput((*pluginConfig.Params)["input"])
	showParam("input", plugin.Input)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.Output = v
		}
	}
	setOutput((*pluginConfig.Params)["output"])
	showParam("output", plugin.Output)

	// regexp.
	setRegexp := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["regexp"] = 0
			plugin.Regexp = core.ExtractRegexesIntoArray(pluginConfig.Config, v)
		}
	}
	setRegexp((*pluginConfig.Params)["regexp"])
	showParam("regexp", plugin.Regexp)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.Require = v

		}
	}
	setRequire((*pluginConfig.Params)["require"])
	showParam("require", plugin.Require)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// If output is set:
	// 1. input and output must have equal size.
	// 2. input and output values must have equal types.
	if availableParams["output"] == 0 {
		if len(plugin.Input) != len(plugin.Output) {
			return &Plugin{}, fmt.Errorf(core.ERROR_SIZE_MISMATCH.Error(), plugin.Input, plugin.Output)
		}

		if err := core.IsDataFieldsTypesEqual(&plugin.Input, &plugin.Output); err != nil {
			return &Plugin{}, err
		}
	} else {
		core.SliceStringToUpper(&plugin.Input)
		core.SliceStringToUpper(&plugin.Output)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
