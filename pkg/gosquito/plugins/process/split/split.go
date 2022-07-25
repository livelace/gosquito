package splitProcess

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
)

const (
	PLUGIN_NAME = "split"

	DEFAULT_MODE        = "strict"
	DEFAULT_SPARSE_STUB = "!SPARSE!"
)

var (
	ERROR_MODE_UNKNOWN   = errors.New("mode unknown: %v")
    ERROR_SIZE_NOT_EQUAL = errors.New("size not equal: %v is %v, need %v")
)

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionInclude    bool
	OptionInput      []string
	OptionMode       string
	OptionOutput     []string
	OptionRequire    []int
	OptionSparseStub string
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

		var maxSliceSize int
		for index, input := range p.OptionInput {
			ri, _ := core.ReflectDatumField(item, input)
			if index == 0 {
				maxSliceSize = ri.Len()
				continue
			}

			if p.OptionMode == "strict" && ri.Len() != maxSliceSize {
				return temp, fmt.Errorf(ERROR_SIZE_NOT_EQUAL.Error(), input, ri.Len(), maxSliceSize)
			}

			if p.OptionMode == "sparse" {
				if ri.Len() > maxSliceSize {
					maxSliceSize = ri.Len()
				}
			}
		}

		for i := 0; i < maxSliceSize; i++ {
			var u, _ = uuid.NewRandom()

			newItem := &core.Datum{
				FLOW:        item.FLOW,
				PLUGIN:      item.PLUGIN,
				SOURCE:      item.SOURCE,
				TIME:        item.TIME,
				TIMEFORMAT:  item.TIMEFORMAT,
				TIMEFORMATA: item.TIMEFORMATA,
				TIMEFORMATB: item.TIMEFORMATB,
				TIMEFORMATC: item.TIMEFORMATC,
				UUID:        u,

				WARNINGS: item.WARNINGS,
			}

			for index, input := range p.OptionInput {
				ri, _ := core.ReflectDatumField(item, input)
				ro, _ := core.ReflectDatumField(newItem, p.OptionOutput[index])

				if i >= ri.Len() {
					ro.SetString(p.OptionSparseStub)
				} else {
					ro.SetString(ri.Index(i).String())
				}
			}

			temp = append(temp, newItem)
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

		"input":  1,
		"output": 1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

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

	// mode.
	setMode := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["mode"] = 0
			plugin.OptionMode = v
		}
	}
	setMode(DEFAULT_MODE)
	setMode((*pluginConfig.PluginParams)["mode"])
	core.ShowPluginParam(plugin.LogFields, "mode", plugin.OptionMode)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.OptionOutput = v
		}
	}
	setOutput((*pluginConfig.PluginParams)["output"])
	core.ShowPluginParam(plugin.LogFields, "output", plugin.OptionOutput)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.OptionRequire = v

		}
	}
	setRequire((*pluginConfig.PluginParams)["require"])
	core.ShowPluginParam(plugin.LogFields, "require", plugin.OptionRequire)

	// sparse_stub.
	setSparseStub := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["sparse_stub"] = 0
			plugin.OptionSparseStub = v
		}
	}
	setSparseStub(DEFAULT_SPARSE_STUB)
	setSparseStub((*pluginConfig.PluginParams)["sparse_stub"])
	core.ShowPluginParam(plugin.LogFields, "sparse_stub", plugin.OptionSparseStub)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if plugin.OptionMode != "strict" && plugin.OptionMode != "sparse" {
		return &Plugin{}, fmt.Errorf(ERROR_MODE_UNKNOWN.Error(), plugin.OptionMode)
	}

	if len(plugin.OptionInput) != len(plugin.OptionOutput) {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput)
	}

	if err := core.IsDatumFieldsSlice(&plugin.OptionInput); err != nil {
		return &Plugin{}, err
	}

	if err := core.IsDatumFieldsString(&plugin.OptionOutput); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
