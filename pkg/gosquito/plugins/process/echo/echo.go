package echo

import (
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"reflect"
)

const (
	LOG_PLUGIN_ECHO_DATA = "echo data"
)

type Plugin struct {
	Hash string
	Flow string

	ID    int
	Alias string

	File string
	Name string
	Type string

	Input   []string
	Require []int
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Echoing.
	f := func(msg string) {
		log.WithFields(log.Fields{
			"hash":   p.Hash,
			"flow":   p.Flow,
			"file":   p.File,
			"plugin": p.Name,
			"type":   p.Type,
			"id":     p.ID,
			"alias":  p.Alias,
			"data":   msg,
		}).Debug(LOG_PLUGIN_ECHO_DATA)
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {

		for _, input := range p.Input {
			// Reflect "input" plugin data fields.
			ri, err := core.ReflectDataField(item, input)

			if err == nil {
				switch ri.Kind() {
				case reflect.Slice:
					for i := 0; i < ri.Len(); i++ {
						f(ri.Index(i).String())
					}
				default:
					f(ri.String())
				}

			} else {
				f(input)
			}
		}

		temp = append(temp, item)
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
		Name: "echo",
		Type: "process",
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"input":   1,
		"require": -1,
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

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.Input = v
		}
	}
	setInput((*pluginConfig.Params)["input"])
	showParam("input", plugin.Input)

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

	return &plugin, nil
}
