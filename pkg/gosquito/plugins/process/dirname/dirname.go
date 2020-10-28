package dirname

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

	Depth  int
	Input  []string
	Output []string
}

func getDirName(p string, d int) string {
	if p == "/" || p == "." || d == 0 {
		return p
	}

	return getDirName(filepath.Dir(p), d-1)
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {

		for index, input := range p.Input {
			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.Output[index])

			for i := 0; i < ri.Len(); i++ {
				dir := getDirName(ri.Index(i).String(), p.Depth)
				ro.Set(reflect.Append(ro, reflect.ValueOf(dir)))
			}
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
	return false
}

func (p *Plugin) GetRequire() []int {
	return []int{0}
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Hash: pluginConfig.Hash,
		Flow: pluginConfig.Flow,

		ID:    pluginConfig.ID,
		Alias: pluginConfig.Alias,

		File: pluginConfig.File,
		Name: "dirname",
		Type: "process",
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
			"hash":   plugin.Hash,
			"flow":   plugin.Flow,
			"file":   plugin.File,
			"plugin": plugin.Name,
			"type":   plugin.Type,
			"value":  fmt.Sprintf("%s: %v", p, v),
		}).Debug(core.LOG_SET_VALUE)
	}

	// -----------------------------------------------------------------------------------------------------------------

	// depth.
	setDepth := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["depth"] = 0
			plugin.Depth = v
		}
	}
	setDepth(DEFAULT_DEPTH)
	setDepth((*pluginConfig.Params)["depth"])
	showParam("depth", plugin.Depth)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			if err := core.IsDataFieldsSlice(&v); err == nil {
				availableParams["input"] = 0
				plugin.Input = v
			}
		}
	}
	setInput((*pluginConfig.Params)["input"])
	showParam("input", plugin.Input)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			if err := core.IsDataFieldsSlice(&v); err == nil {
				availableParams["output"] = 0
				plugin.Output = v
			}
		}
	}
	setOutput((*pluginConfig.Params)["output"])
	showParam("output", plugin.Output)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// input and output must have equal size.
	if len(plugin.Input) != len(plugin.Output) {
		return &Plugin{}, fmt.Errorf(core.ERROR_SIZE_MISMATCH.Error(), plugin.Input, plugin.Output)
	} else {
		core.SliceStringToUpper(&plugin.Input)
		core.SliceStringToUpper(&plugin.Output)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
