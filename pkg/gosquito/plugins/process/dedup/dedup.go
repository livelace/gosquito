package dedup

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
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
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	uuidCache := make(map[uuid.UUID]bool, 0)

	// Order preserving.
	for i := 0; i < len(data); i++ {
		item := data[i]

		if _, ok := uuidCache[item.UUID]; !ok {
			uuidCache[item.UUID] = true
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
		Name: "dedup",
		Type: "process",
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"include": -1,
		"require": 1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	showParam := func(p string, v interface{}) {
		log.WithFields(log.Fields{
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
	setInclude(pluginConfig.Config.GetString(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
	setInclude((*pluginConfig.Params)["include"])
	showParam("include", plugin.Include)

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
