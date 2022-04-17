package regexpmatchProcess

import (
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"reflect"
	"regexp"
)

const (
	PLUGIN_NAME = "regexpmatch"

	DEFAULT_MATCH_ALL  = false
	DEFAULT_MATCH_CASE = true
	DEFAULT_MATCH_NOT  = false
)

func matchRegexes(regexps []*regexp.Regexp, text string, isNot bool) bool {
	for _, re := range regexps {
		if re.MatchString(text) && !isNot {
			return true
		}

		if !re.MatchString(text) && isNot {
			return true
		}
	}

	return false
}

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionInclude   bool
	OptionInput     []string
	OptionMatchAll  bool
	OptionMatchCase bool
	OptionMatchNot  bool
	OptionOutput    []string
	OptionRegexp    [][]*regexp.Regexp
	OptionRequire   []int
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
  p.LogFields["run"] = p.Flow.GetRunID()

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		matched := make([]bool, len(p.OptionInput))

		// Match pattern inside different data fields (Title, Content etc.).
		for index, input := range p.OptionInput {
			var ro reflect.Value

			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)

			if len(p.OptionOutput) > 0 {
				ro, _ = core.ReflectDataField(item, p.OptionOutput[index])
			}

			// This plugin supports "string" and "[]string" data fields for matching.
			switch ri.Kind() {
			case reflect.String:
				if matchRegexes(p.OptionRegexp[index], ri.String(), p.OptionMatchNot) {
					matched[index] = true
					if len(p.OptionOutput) > 0 {
						ro.SetString(ri.String())
					}
				}
			case reflect.Slice:
				somethingWasMatched := false

				for i := 0; i < ri.Len(); i++ {
					if matchRegexes(p.OptionRegexp[index], ri.Index(i).String(), p.OptionMatchNot) {
						somethingWasMatched = true
						if len(p.OptionOutput) > 0 {
							ro.Set(reflect.Append(ro, ri.Index(i)))
						}
					}
				}

				matched[index] = somethingWasMatched
			}
		}

		// Append replaced item to results.
		matchedInSomeInputs := false
		matchedInAllInputs := true

		for _, b := range matched {
			if b {
				matchedInSomeInputs = true
			} else {
				matchedInAllInputs = false
			}
		}

		if (p.OptionMatchAll && matchedInAllInputs) || (!p.OptionMatchAll && matchedInSomeInputs) {
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

		"input":      1,
		"match_all":  -1,
		"match_case": -1,
		"match_not":  -1,
		"output":     -1,
		"regexp":     1,
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

	// match_all.
	setMatchAll := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["match_all"] = 0
			plugin.OptionMatchAll = v
		}
	}
	setMatchAll(DEFAULT_MATCH_ALL)
	setMatchAll((*pluginConfig.PluginParams)["match_all"])
	core.ShowPluginParam(plugin.LogFields, "match_all", plugin.OptionMatchAll)

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

	// match_not.
	setMatchNot := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["match_not"] = 0
			plugin.OptionMatchNot = v
		}
	}
	setMatchNot(DEFAULT_MATCH_NOT)
	setMatchNot((*pluginConfig.PluginParams)["match_not"])
	core.ShowPluginParam(plugin.LogFields, "match_not", plugin.OptionMatchNot)

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

	if availableParams["output"] == 0 {
		if len(plugin.OptionInput) != len(plugin.OptionOutput) && len(plugin.OptionOutput) != len(plugin.OptionRegexp) {
			return &Plugin{}, fmt.Errorf(
				"%s: %v, %v, %v",
				core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput, plugin.OptionRegexp)
		}

		if err := core.IsDataFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
			return &Plugin{}, err
		}

	} else {
		if len(plugin.OptionInput) != len(plugin.OptionRegexp) {
			return &Plugin{}, fmt.Errorf(
				"%s: %v, %v",
				core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionRegexp)
		}
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
