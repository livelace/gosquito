package regexpfindProcess

import (
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"reflect"
	"regexp"
	"strings"
)

const (
	PLUGIN_NAME = "regexpfind"

	DEFAULT_FIND_ALL   = false
	DEFAULT_MATCH_CASE = true
)

func findPatternsAndReturnSlice(regexps []*regexp.Regexp, groups []int, groupsJoin string, text string) []string {
	temp := make([]string, 0)

	needJoin := len(groupsJoin) > 0

	// 1. Search groups of pattern.
	// 2. Search patterns.
	if len(groups) > 0 {
		for _, re := range regexps {

			// Try to find groups.
			for _, foundGroups := range re.FindAllStringSubmatch(text, -1) {

				// Groups found.
				if len(foundGroups) > 0 {
					groupsAmount := len(foundGroups) - 1 // first value is a string: matched text + groups.
					groupsJoined := ""

					for _, group := range groups {
						if group <= groupsAmount {
							// a. Join groups if needed.
							// b. Just append groups to result.
							if needJoin {
								groupsJoined += foundGroups[group] + groupsJoin
							} else {
								temp = append(temp, foundGroups[group])
							}
						}
					}

					// Append joined groups to result.
					if needJoin {
						temp = append(temp, strings.TrimRight(groupsJoined, groupsJoin))
					}
				}
			}
		}

	} else {
		for _, re := range regexps {
			temp = append(temp, re.FindAllString(text, -1)...)
		}
	}

	return temp
}

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionFindAll   bool
	OptionGroup     [][]int
	OptionGroupJoin []string
	OptionInclude   bool
	OptionInput     []string
	OptionMatchCase bool
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

func (p *Plugin) Process(data []*core.Datum) ([]*core.Datum, error) {
	temp := make([]*core.Datum, 0)
  p.LogFields["run"] = p.Flow.GetRunID()

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		found := make([]bool, len(p.OptionInput))

		for index, input := range p.OptionInput {
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.OptionOutput[index])

			regx := p.OptionRegexp[index]

			var group []int
			if len(p.OptionGroup) > 0 {
				group = p.OptionGroup[index]
			}

			var groupJoin string
			if len(p.OptionGroupJoin) > 0 {
				groupJoin = p.OptionGroupJoin[index]
			}

			switch ri.Kind() {
			case reflect.String:
				if s := findPatternsAndReturnSlice(regx, group, groupJoin, ri.String()); len(s) > 0 {

					found[index] = true
					for _, v := range s {
						ro.Set(reflect.Append(ro, reflect.ValueOf(v)))
					}
				}
			case reflect.Slice:
				somethingWasFound := false

				for i := 0; i < ri.Len(); i++ {
					if s := findPatternsAndReturnSlice(regx, group, groupJoin, ri.Index(i).String()); len(s) > 0 {

						somethingWasFound = true
						for _, v := range s {
							ro.Set(reflect.Append(ro, reflect.ValueOf(v)))
						}
					}
				}

				found[index] = somethingWasFound
			}
		}

		// Append replaced item to results.
		foundInSomeInputs := false
		foundInAllInputs := true

		for _, b := range found {
			if b {
				foundInSomeInputs = true
			} else {
				foundInAllInputs = false
			}
		}

		if (p.OptionFindAll && foundInAllInputs) || (!p.OptionFindAll && foundInSomeInputs) {
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

		"find_all":   -1,
		"group":      -1,
		"group_join": -1,
		"input":      1,
		"match_case": -1,
		"output":     1,
		"regexp":     1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin settings or set defaults.

	// find_all.
	setFindAll := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["find_all"] = 0
			plugin.OptionFindAll = v
		}
	}
	setFindAll(DEFAULT_FIND_ALL)
	setFindAll((*pluginConfig.PluginParams)["find_all"])
	core.ShowPluginParam(plugin.LogFields, "find_all", plugin.OptionFindAll)

	// group.
	setGroup := func(p interface{}) {
		if v, b := core.IsSliceOfSliceInt(p); b {
			availableParams["group"] = 0
			plugin.OptionGroup = v
		}
	}
	setGroup((*pluginConfig.PluginParams)["group"])
	core.ShowPluginParam(plugin.LogFields, "group", plugin.OptionGroup)

	// group_join.
	setGroupJoin := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["group_join"] = 0
			plugin.OptionGroupJoin = v
		}
	}
	setGroupJoin((*pluginConfig.PluginParams)["group_join"])
	core.ShowPluginParam(plugin.LogFields, "group_join", plugin.OptionGroupJoin)

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
			if err := core.IsDataFieldsSlice(&v); err == nil {
				availableParams["output"] = 0
				plugin.OptionOutput = v
			}
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

	// "input, output, regexp" must have equal size.
	if len(plugin.OptionInput) != len(plugin.OptionOutput) && len(plugin.OptionOutput) != len(plugin.OptionRegexp) {
		return &Plugin{}, fmt.Errorf(
			"%s: %v, %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput, plugin.OptionRegexp)
	}

	// "input, output, regexp, group" must have equal size, if group is set.
	if len(plugin.OptionGroup) > 0 && len(plugin.OptionGroup) != len(plugin.OptionInput) {
		return &Plugin{}, fmt.Errorf(
			"%s: %v, %v, %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput, plugin.OptionRegexp,
			plugin.OptionGroup)
	}

	// "input, output, regexp, group, group_join" must have equal size, if group_join is set.
	if len(plugin.OptionGroupJoin) > 0 && len(plugin.OptionGroupJoin) != len(plugin.OptionGroup) {
		return &Plugin{}, fmt.Errorf(
			"%s: %v, %v, %v, %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput, plugin.OptionRegexp,
			plugin.OptionGroup, plugin.OptionGroupJoin)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
