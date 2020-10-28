package fetch

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
	Hash string
	Flow string

	ID    int
	Alias string

	File    string
	Name    string
	TempDir string
	Type    string

	Include bool
	Require []int

	Input   []string
	Output  []string
	Timeout int
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		fetched := false

		for index, input := range p.Input {
			outputDir := filepath.Join(p.TempDir, p.Flow, p.Type, p.Name)
			_ = core.CreateDirIfNotExist(outputDir)

			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.Output[index])

			// Every downloaded item is placed into individual directory.
			// Items with similar names don't overwrite each other.
			fetch := func(url string) {
				u, _ := uuid.NewRandom()
				fileName := path.Base(url)
				savePath := filepath.Join(outputDir, u.String(), fileName)

				if err := fetchData(url, savePath, p.Timeout); err == nil {
					fetched = true

					ro.Set(reflect.Append(ro, reflect.ValueOf(savePath)))

					log.WithFields(log.Fields{
						"hash":   p.Hash,
						"flow":   p.Flow,
						"file":   p.File,
						"plugin": p.Name,
						"type":   p.Type,
						"id":     p.ID,
						"alias":  p.Alias,
						"data":   url,
					}).Debug(core.LOG_PLUGIN_STAT)
				} else {
					log.WithFields(log.Fields{
						"hash":   p.Hash,
						"flow":   p.Flow,
						"file":   p.File,
						"plugin": p.Name,
						"type":   p.Type,
						"id":     p.ID,
						"alias":  p.Alias,
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

		File:    pluginConfig.File,
		Name:    "fetch",
		TempDir: pluginConfig.Config.GetString(core.VIPER_DEFAULT_TEMP_DIR),
		Type:    "process",
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"include": -1,
		"require": -1,

		"input":   1,
		"output":  1,
		"timeout": -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin settings or set defaults.

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

	// include.
	setInclude := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["include"] = 0
			plugin.Include = v
		}
	}
	setInclude(pluginConfig.Config.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
	setInclude((*pluginConfig.Params)["include"])
	showParam("include", plugin.Include)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.Input = v
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

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.Require = v

		}
	}
	setRequire((*pluginConfig.Params)["require"])
	showParam("require", plugin.Require)

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.Timeout = v
		}
	}
	setTimeout(pluginConfig.Config.GetInt(core.VIPER_DEFAULT_PLUGIN_TIMEOUT))
	setTimeout((*pluginConfig.Params)["timeout"])
	showParam("timeout", plugin.Timeout)

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
