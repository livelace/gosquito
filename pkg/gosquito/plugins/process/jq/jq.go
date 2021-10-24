package jqProcess

import (
	"encoding/json"
	"fmt"
	"github.com/itchyny/gojq"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"reflect"
)

const (
	DEFAULT_FIND_ALL = false
)

func applyQueryToText(queries []*gojq.Query, jsonText string) ([]string, error) {
	temp := make([]string, 0)

	// Try to map provided JSON text to object.
	var jsonMap map[string]interface{}
	err := json.Unmarshal([]byte(jsonText), &jsonMap)
	if err != nil {
		return temp, err
	}

	// Apply queries to object.
	for _, query := range queries {
		iter := query.Run(jsonMap)

		for {
			v, ok := iter.Next()
			if !ok {
				break
			}

			if err, ok := v.(error); ok {
				return temp, err
			}

			if v != nil {
				temp = append(temp, fmt.Sprintf("%s", v))
			}
		}
	}

	return temp, nil
}

func logQueryError(p *Plugin, err error) {
	log.WithFields(log.Fields{
		"hash":   p.Flow.FlowHash,
		"flow":   p.Flow.FlowName,
		"file":   p.Flow.FlowFile,
		"plugin": p.PluginName,
		"type":   p.PluginType,
		"id":     p.PluginID,
		"alias":  p.PluginAlias,
		"data":   fmt.Sprintf("query error: %v", err),
	}).Error(core.LOG_PLUGIN_DATA)
}

type Plugin struct {
	Flow *core.Flow

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionFindAll bool
	OptionInclude bool
	OptionInput   []string
	OptionOutput  []string
	OptionQuery   [][]*gojq.Query
	OptionRequire []int
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
		found := make([]bool, len(p.OptionInput))

		for index, input := range p.OptionInput {
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.OptionOutput[index])

			switch ri.Kind() {
			case reflect.String:
				result, err := applyQueryToText(p.OptionQuery[index], ri.String())
				if err != nil {
					logQueryError(p, err)
				}

				if len(result) > 0 {
					for _, v := range result {
						ro.Set(reflect.Append(ro, reflect.ValueOf(v)))
					}

					found[index] = true
				}

			case reflect.Slice:
				somethingWasFound := false

				for i := 0; i < ri.Len(); i++ {
					result, err := applyQueryToText(p.OptionQuery[index], ri.Index(i).String())
					if err != nil {
						logQueryError(p, err)
					}

					if len(result) > 0 {
						for _, v := range result {
							ro.Set(reflect.Append(ro, reflect.ValueOf(v)))
						}

						somethingWasFound = true
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
		Flow:        pluginConfig.Flow,
		PluginID:    pluginConfig.PluginID,
		PluginAlias: pluginConfig.PluginAlias,
		PluginName:  "jq",
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

		"find_all": -1,
		"input":    1,
		"output":   1,
		"query":    1,
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

	// find_all.
	setFindAll := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["find_all"] = 0
			plugin.OptionFindAll = v
		}
	}
	setFindAll(DEFAULT_FIND_ALL)
	setFindAll((*pluginConfig.PluginParams)["find_all"])
	showParam("find_all", plugin.OptionFindAll)

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
	showParam("output", plugin.OptionOutput)

	// query.
	setQuery := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["query"] = 0
			plugin.OptionQuery = core.ExtractJqQueriesIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setQuery((*pluginConfig.PluginParams)["query"])
	showParam("query", plugin.OptionQuery)

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

	// 1. "input, output, query" must have equal size.
	minLength := 10000
	maxLength := 0
	lengths := []int{len(plugin.OptionInput), len(plugin.OptionOutput), len(plugin.OptionQuery)}

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
			"%s: %v, %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput, len(plugin.OptionQuery))
	} else {
		core.SliceStringToUpper(&plugin.OptionInput)
		core.SliceStringToUpper(&plugin.OptionOutput)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
