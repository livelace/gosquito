package regexpreplaceProcess

import (
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"reflect"
	"regexp"
	"strings"
)

const (
	PLUGIN_NAME = "regexpreplace"

	DEFAULT_REPLACE_ALL = false
	DEFAULT_MATCH_CASE  = true
)

func findAndReplace(regexps []*regexp.Regexp, text string, replacement string) (string, bool) {
	temp := text

	for _, re := range regexps {
		temp = re.ReplaceAllString(temp, replacement)
	}

	if temp != text {
		return strings.TrimSpace(temp), true
	} else {
		return text, false
	}
}

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionInclude    bool
	OptionInput      []string
	OptionMatchCase  bool
	OptionOutput     []string
	OptionRegexp     [][]*regexp.Regexp
	OptionReplace    []string
	OptionReplaceAll bool
	OptionRequire    []int
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

func (p *Plugin) Process(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		replaced := make([]bool, len(p.OptionInput))

		// Match pattern inside different data fields (Title, Content etc.).
		for index, input := range p.OptionInput {
			var ro reflect.Value

			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ = core.ReflectDataField(item, p.OptionOutput[index])

			// This plugin supports "string" and "[]string" data fields for matching.
			switch ri.Kind() {
			case reflect.String:
				if s, b := findAndReplace(p.OptionRegexp[index], ri.String(), p.OptionReplace[index]); b {
					replaced[index] = true
					ro.SetString(s)
				} else {
					ro.SetString(s)
				}
			case reflect.Slice:
				somethingWasReplaced := false

				for i := 0; i < ri.Len(); i++ {
					if s, b := findAndReplace(p.OptionRegexp[index], ri.Index(i).String(), p.OptionReplace[index]); b {
						somethingWasReplaced = true
						ro.Set(reflect.Append(ro, reflect.ValueOf(s)))
					} else {
						ro.Set(reflect.Append(ro, reflect.ValueOf(s)))
					}
				}

				replaced[index] = somethingWasReplaced
			}
		}

		// Append replaced item to results.
		replacedInSomeInputs := false
		replacedInAllInputs := true

		for _, b := range replaced {
			if b {
				replacedInSomeInputs = true
			} else {
				replacedInAllInputs = false
			}
		}

		if (p.OptionReplaceAll && replacedInAllInputs) || (!p.OptionReplaceAll && replacedInSomeInputs) {
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

		"input":      1,
		"match_case": -1,
		"output":     1,
		"regexp":     1,
		"replace":    1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin settings or set defaults.

	// include.
	setInclude := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["include"] = 0
			plugin.OptionInclude = v
		}
	}
	setInclude(pluginConfig.AppConfig.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
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

	// match_case.
	setMatchCase := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["match_case"] = 0
			plugin.OptionMatchCase = v
		}
	}
	setMatchCase(DEFAULT_MATCH_CASE)
	setMatchCase((*pluginConfig.PluginParams)["match_case"])
	core.ShowPluginParam(plugin.LogFields, "match_case", plugin.OptionMatchCase)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.OptionOutput = v
		}
	}
	setOutput((*pluginConfig.PluginParams)["output"])
	core.ShowPluginParam(plugin.LogFields, "output", plugin.OptionOutput)

	// regexp.
	setRegexp := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["regexp"] = 0
			plugin.OptionRegexp = core.ExtractRegexpsIntoArrays(pluginConfig.AppConfig, v, plugin.OptionMatchCase)
		}
	}
	setRegexp((*pluginConfig.PluginParams)["regexp"])
	core.ShowPluginParam(plugin.LogFields, "regexp", plugin.OptionRegexp)

	// replace.
	setReplace := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["replace"] = 0
			plugin.OptionReplace = v
		}
	}
	setReplace((*pluginConfig.PluginParams)["replace"])
	core.ShowPluginParam(plugin.LogFields, "replace", plugin.OptionReplace)

	// replace_all.
	setReplaceAll := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["replace_all"] = 0
			plugin.OptionReplaceAll = v
		}
	}
	setReplaceAll(DEFAULT_REPLACE_ALL)
	setReplaceAll((*pluginConfig.PluginParams)["replace_all"])
	core.ShowPluginParam(plugin.LogFields, "replace_all", plugin.OptionReplaceAll)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.OptionRequire = v

		}
	}
	setRequire((*pluginConfig.PluginParams)["require"])
	core.ShowPluginParam(plugin.LogFields, "require", plugin.OptionRequire)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if len(plugin.OptionInput) != len(plugin.OptionOutput) && len(plugin.OptionOutput) != len(plugin.OptionRegexp) &&
		len(plugin.OptionRegexp) != len(plugin.OptionReplace) {

		return &Plugin{}, fmt.Errorf(
			"%s: %v, %v, %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(),
			plugin.OptionInput, plugin.OptionOutput, plugin.OptionRegexp, plugin.OptionReplace)
	}

	if err := core.IsDataFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
