package regexpreplace

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
	Hash string
	Flow string

	ID    int
	Alias string

	File string
	Name string
	Type string

	Include bool
	Require []int

	Input      []string
	MatchCase  bool
	Output     []string
	Regexp     [][]*regexp.Regexp
	Replace    []string
	ReplaceAll bool
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		replaced := make([]bool, len(p.Input))

		// Match pattern inside different data fields (Title, Content etc.).
		for index, input := range p.Input {
			var ro reflect.Value

			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ = core.ReflectDataField(item, p.Output[index])

			// This plugin supports "string" and "[]string" data fields for matching.
			switch ri.Kind() {
			case reflect.String:
				if s, b := findAndReplace(p.Regexp[index], ri.String(), p.Replace[index]); b {
					replaced[index] = true
					ro.SetString(s)
				} else {
					ro.SetString(s)
				}
			case reflect.Slice:
				somethingWasReplaced := false

				for i := 0; i < ri.Len(); i++ {
					if s, b := findAndReplace(p.Regexp[index], ri.Index(i).String(), p.Replace[index]); b {
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

		if (p.ReplaceAll && replacedInAllInputs) || (!p.ReplaceAll && replacedInSomeInputs) {
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
		Name: "regexpreplace",
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

	// match_case.
	setMatchCase := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["match_case"] = 0
			plugin.MatchCase = v
		}
	}
	setMatchCase(DEFAULT_MATCH_CASE)
	setMatchCase((*pluginConfig.Params)["match_case"])
	showParam("match_case", plugin.MatchCase)

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
			plugin.Regexp = core.ExtractRegexpsIntoArrays(pluginConfig.Config, v, plugin.MatchCase)
		}
	}
	setRegexp((*pluginConfig.Params)["regexp"])
	showParam("regexp", plugin.Regexp)

	// replace.
	setReplace := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["replace"] = 0
			plugin.Replace = v
		}
	}
	setReplace((*pluginConfig.Params)["replace"])
	showParam("replace", plugin.Replace)

	// replace_all.
	setReplaceAll := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["replace_all"] = 0
			plugin.ReplaceAll = v
		}
	}
	setReplaceAll(DEFAULT_REPLACE_ALL)
	setReplaceAll((*pluginConfig.Params)["replace_all"])
	showParam("replace_all", plugin.ReplaceAll)

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

	// 1. "input, output, regexp, replace" must have equal size.
	// 2. "input, output" values must have equal types.
	minLength := 10000
	maxLength := 0
	lengths := []int{len(plugin.Input), len(plugin.Output), len(plugin.Regexp), len(plugin.Replace)}

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
			core.ERROR_SIZE_MISMATCH.Error(), plugin.Input, plugin.Output, plugin.Regexp, plugin.Replace)

	} else if err := core.IsDataFieldsTypesEqual(&plugin.Input, &plugin.Output); err != nil {
		return &Plugin{}, err

	} else {
		core.SliceStringToUpper(&plugin.Input)
		core.SliceStringToUpper(&plugin.Output)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
