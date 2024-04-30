package iconvProcess

import (
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/qiniu/iconv"
	"reflect"
	"strings"
)

const (
	PLUGIN_NAME = "iconv"

	DEFAULT_TO = "utf-8"
)

func convert(p *Plugin, text string) (string, bool) {
	temp := text

	cd, err := iconv.Open(p.OptionTo, p.OptionFrom)
	if err != nil {
		p.FlowLog(err)
		return text, false
	}
	defer cd.Close()

	temp = cd.ConvString(text)

	if temp != text {
		return strings.TrimSpace(temp), true
	} else {
		return text, false
	}
}

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionFrom    string
	OptionInclude bool
	OptionInput   []string
	OptionOutput  []string
	OptionRequire []int
	OptionTo      string
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
		// Match pattern inside different data fields (Title, Content etc.).
		for index, input := range p.OptionInput {
			var ro reflect.Value

			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDatumField(item, input)
			ro, _ = core.ReflectDatumField(item, p.OptionOutput[index])

			// This plugin supports "string" and "[]string" data fields for matching.
			switch ri.Kind() {
			case reflect.String:
				if s, b := convert(p, ri.String()); b {
					ro.SetString(s)
				} else {
					ro.SetString(ri.String())
				}
			case reflect.Slice:
				for i := 0; i < ri.Len(); i++ {
					if s, b := convert(p, ri.Index(i).String()); b {
						ro.Set(reflect.Append(ro, reflect.ValueOf(s)))
					} else {
						ro.Set(reflect.Append(ro, ri.Index(i)))
					}
				}
			}
		}

		temp = append(temp, item)
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

		"from":   1,
		"input":  1,
		"output": 1,
		"to":     -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin settings or set defaults.

	// from.
	setFrom := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["from"] = 0
			plugin.OptionFrom = v
		}
	}
	setFrom((*pluginConfig.PluginParams)["from"])
	core.ShowPluginParam(plugin.LogFields, "from", plugin.OptionFrom)

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

	// to.
	setTo := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["to"] = 0
			plugin.OptionTo = v
		}
	}
	setTo(DEFAULT_TO)
	setTo((*pluginConfig.PluginParams)["to"])
	core.ShowPluginParam(plugin.LogFields, "to", plugin.OptionTo)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if len(plugin.OptionInput) != len(plugin.OptionOutput) {
		return &Plugin{}, fmt.Errorf(
			"%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(),
			plugin.OptionInput, plugin.OptionOutput)
	}

	if err := core.IsDatumFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
