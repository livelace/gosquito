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

func (p *Plugin) GetID() int {
	return p.PluginID
}

func (p *Plugin) GetAlias() string {
	return p.PluginAlias
}

func (p *Plugin) GetFile() string {
	return p.Flow.FlowFile
}

func (p *Plugin) GetName() string {
	return p.PluginName
}

func (p *Plugin) GetType() string {
	return p.PluginType
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
		Flow:        pluginConfig.Flow,
		PluginID:    pluginConfig.PluginID,
		PluginAlias: pluginConfig.PluginAlias,
		PluginName:  "regexpreplace",
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

	// include.
	setInclude := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["include"] = 0
			plugin.OptionInclude = v
		}
	}
	setInclude(pluginConfig.AppConfig.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
	setInclude((*pluginConfig.PluginParams)["include"])
	showParam("include", plugin.OptionInclude)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.OptionInput = v
		}
	}
	setInput((*pluginConfig.PluginParams)["input"])
	showParam("input", plugin.OptionInput)

	// match_case.
	setMatchCase := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["match_case"] = 0
			plugin.OptionMatchCase = v
		}
	}
	setMatchCase(DEFAULT_MATCH_CASE)
	setMatchCase((*pluginConfig.PluginParams)["match_case"])
	showParam("match_case", plugin.OptionMatchCase)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.OptionOutput = v
		}
	}
	setOutput((*pluginConfig.PluginParams)["output"])
	showParam("output", plugin.OptionOutput)

	// regexp.
	setRegexp := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["regexp"] = 0
			plugin.OptionRegexp = core.ExtractRegexpsIntoArrays(pluginConfig.AppConfig, v, plugin.OptionMatchCase)
		}
	}
	setRegexp((*pluginConfig.PluginParams)["regexp"])
	showParam("regexp", plugin.OptionRegexp)

	// replace.
	setReplace := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["replace"] = 0
			plugin.OptionReplace = v
		}
	}
	setReplace((*pluginConfig.PluginParams)["replace"])
	showParam("replace", plugin.OptionReplace)

	// replace_all.
	setReplaceAll := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["replace_all"] = 0
			plugin.OptionReplaceAll = v
		}
	}
	setReplaceAll(DEFAULT_REPLACE_ALL)
	setReplaceAll((*pluginConfig.PluginParams)["replace_all"])
	showParam("replace_all", plugin.OptionReplaceAll)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.OptionRequire = v

		}
	}
	setRequire((*pluginConfig.PluginParams)["require"])
	showParam("require", plugin.OptionRequire)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// 1. "input, output, regexp, replace" must have equal size.
	// 2. "input, output" values must have equal types.
	minLength := 10000
	maxLength := 0
	lengths := []int{len(plugin.OptionInput), len(plugin.OptionOutput), len(plugin.OptionRegexp), len(plugin.OptionReplace)}

	for _, length := range lengths {
		if length > maxLength {
			maxLength = length
		}
		if length < minLength {
			minLength = length
		}
	}

	if minLength != maxLength {
		return &Plugin{}, fmt.Errorf(
			"%s: %v, %v, %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput, plugin.OptionRegexp, plugin.OptionReplace)

	} else if err := core.IsDataFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
		return &Plugin{}, err

	} else {
		core.SliceStringToUpper(&plugin.OptionInput)
		core.SliceStringToUpper(&plugin.OptionOutput)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
