package dirnameProcess

import (
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"path/filepath"
	"reflect"
)

const (
	DEFAULT_DEPTH = 1
)

func getDirName(p string, d int) string {
	if p == "/" || p == "." || d == 0 {
		return p
	}

	return getDirName(filepath.Dir(p), d-1)
}

type Plugin struct {
	Flow *core.Flow

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionInclude bool
	OptionRequire []int

	OptionDepth  int
	OptionInput  []string
	OptionOutput []string
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
	return false
}

func (p *Plugin) GetRequire() []int {
	return []int{0}
}

func (p *Plugin) Process(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {

		for index, input := range p.OptionInput {
			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.OptionOutput[index])

			for i := 0; i < ri.Len(); i++ {
				dir := getDirName(ri.Index(i).String(), p.OptionDepth)
				ro.Set(reflect.Append(ro, reflect.ValueOf(dir)))
			}
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
		PluginName:  "dirname",
		PluginType:  pluginConfig.PluginType,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"depth":  -1,
		"input":  1,
		"output": 1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

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

	// depth.
	setDepth := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["depth"] = 0
			plugin.OptionDepth = v
		}
	}
	setDepth(DEFAULT_DEPTH)
	setDepth((*pluginConfig.PluginParams)["depth"])
	showParam("depth", plugin.OptionDepth)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			if err := core.IsDataFieldsSlice(&v); err == nil {
				availableParams["input"] = 0
				plugin.OptionInput = v
			}
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

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// input and output must have equal size.
	if len(plugin.OptionInput) != len(plugin.OptionOutput) {
		return &Plugin{}, fmt.Errorf("%s: %v, %v", core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput)
	} else {
		core.SliceStringToUpper(&plugin.OptionInput)
		core.SliceStringToUpper(&plugin.OptionOutput)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
