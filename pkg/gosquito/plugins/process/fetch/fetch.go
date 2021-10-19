package fetchProcess

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/go-getter"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"path"
	"path/filepath"
	"reflect"
	"time"
)

const (
	LOG_FETCH_ERROR = "fetch error"
)

func fetchData(url string, dst string, timeout int) error {
	// context.
	c := make(chan error, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// getter.
	client := getter.Client{Ctx: ctx, Src: url, Dst: dst}

	// background.
	go func() {
		err := client.Get()
		c <- err
	}()

	// wait for completion.
	select {
	case <-ctx.Done():
	case err := <-c:
		if err != nil {
			return fmt.Errorf("error: %s, %s", url, err)
		}
	case <-time.After(time.Duration(timeout) * time.Second):
		return fmt.Errorf("timeout: %s", url)
	}

	return nil
}

type Plugin struct {
	Flow *core.Flow

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionInclude bool
	OptionRequire []int

	OptionInput   []string
	OptionOutput  []string
	OptionTimeout int
}

func (p *Plugin) Process(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		fetched := false

		for index, input := range p.OptionInput {
			outputDir := filepath.Join(p.Flow.FlowTempDir, p.PluginType, p.PluginName)
			_ = core.CreateDirIfNotExist(outputDir)

			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.OptionOutput[index])

			// Every downloaded item is placed into individual directory.
			// Items with similar names don't overwrite each other.
			fetch := func(url string) {
				u, _ := uuid.NewRandom()
				fileName := path.Base(url)
				savePath := filepath.Join(outputDir, u.String(), fileName)

				if err := fetchData(url, savePath, p.OptionTimeout); err == nil {
					fetched = true

					ro.Set(reflect.Append(ro, reflect.ValueOf(savePath)))

					log.WithFields(log.Fields{
						"hash":   p.Flow.FlowHash,
						"flow":   p.Flow.FlowName,
						"file":   p.Flow.FlowFile,
						"plugin": p.PluginName,
						"type":   p.PluginType,
						"id":     p.PluginID,
						"alias":  p.PluginAlias,
						"data":   fmt.Sprintf("%s -> %s", url, savePath),
					}).Debug(core.LOG_PLUGIN_DATA)

				} else {
					log.WithFields(log.Fields{
						"hash":   p.Flow.FlowHash,
						"flow":   p.Flow.FlowName,
						"file":   p.Flow.FlowFile,
						"plugin": p.PluginName,
						"type":   p.PluginType,
						"id":     p.PluginID,
						"alias":  p.PluginAlias,
						"error":  err,
					}).Debug(LOG_FETCH_ERROR)
				}
			}

			switch ri.Kind() {
			case reflect.String:
				fetch(ri.String())
			case reflect.Slice:
				for i := 0; i < ri.Len(); i++ {
					fetch(ri.Index(i).String())
				}
			}
		}

		if fetched {
			temp = append(temp, item)
		}
	}

	return temp, nil
}

func (p *Plugin) GetId() int {
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

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Flow:        pluginConfig.Flow,
		PluginID:    pluginConfig.PluginID,
		PluginAlias: pluginConfig.PluginAlias,
		PluginName:  "fetch",
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

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.OptionRequire = v

		}
	}
	setRequire((*pluginConfig.PluginParams)["require"])
	showParam("require", plugin.OptionRequire)

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.OptionTimeout = v
		}
	}
	setTimeout(pluginConfig.AppConfig.GetInt(core.VIPER_DEFAULT_PLUGIN_TIMEOUT))
	setTimeout((*pluginConfig.PluginParams)["timeout"])
	showParam("timeout", plugin.OptionTimeout)

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
