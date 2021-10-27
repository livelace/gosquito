package jqProcess

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/itchyny/gojq"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"reflect"
)

const (
	PLUGIN_NAME = "jq"

	DEFAULT_FIND_ALL = false
)

var (
	ERROR_QUERY_ERROR = errors.New("query error: %s")
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

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

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
					core.LogProcessPlugin(p.LogFields, fmt.Errorf(ERROR_QUERY_ERROR.Error(), err))
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
						core.LogProcessPlugin(p.LogFields, fmt.Errorf(ERROR_QUERY_ERROR.Error(), err))
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

		"find_all": -1,
		"input":    1,
		"output":   1,
		"query":    1,
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

	// query.
	setQuery := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["query"] = 0
			plugin.OptionQuery = core.ExtractJqQueriesIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setQuery((*pluginConfig.PluginParams)["query"])
	core.ShowPluginParam(plugin.LogFields, "query", plugin.OptionQuery)

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

	if len(plugin.OptionInput) != len(plugin.OptionOutput) && len(plugin.OptionOutput) != len(plugin.OptionQuery) {
		return &Plugin{}, fmt.Errorf(
			"%s: %v, %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput, plugin.OptionQuery)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
