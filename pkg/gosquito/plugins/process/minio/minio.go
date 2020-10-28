package minio

import (
	"context"
	"errors"
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

const (
	DEFAULT_SSL_ENABLE = true
)

var (
	ERROR_ACTION_UNKNOWN = errors.New("action unknown: %s")
)

func getLocalFiles(path string) ([]string, error) {
	temp := make([]string, 0)

	if core.IsFile(path, "") {
		temp = append(temp, path)

	} else if core.IsDir(path) {
		err := filepath.Walk(path, func(item string, info os.FileInfo, err error) error {
			if core.IsFile(item, "") {
				temp = append(temp, item)
			}
			return nil
		})

		if err != nil {
			return temp, err
		}
	}

	return temp, nil
}

func minioPut(p *Plugin, file string, object string, timeout int) error {
	// context.
	c := make(chan error, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// client.
	client, err := minio.New(p.URL, &minio.Options{
		Creds:  credentials.NewStaticV4(p.AccessKey, p.SecretKey, ""),
		Secure: p.SSL,
	})
	if err != nil {
		return err
	}

	// background.
	go func() {
		_, err = client.FPutObject(ctx, p.Bucket, object, file, minio.PutObjectOptions{ContentType: "octet/stream"})
		c <- err
	}()

	// wait for completion.
	select {
	case <-ctx.Done():
	case err := <-c:
		if err != nil {
			return fmt.Errorf("error: %s, %s", file, err)
		}
	case <-time.After(time.Duration(timeout) * time.Second):
		return fmt.Errorf("timeout: %s", file)
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
	Timeout int

	AccessKey string
	Action    string
	Bucket    string
	Input     []string
	Output    []string
	SSL       bool
	SecretKey string
	URL       string
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	for _, item := range data {
		performed := false

		for index, input := range p.Input {
			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.Output[index])

			for i := 0; i < ri.Len(); i++ {
				// Upload all found files inside provided path into one dir.
				// 1. /local/path/to/file
				// 2. /local/path/to/dir/{file1, file2 ... fileN}
				// -> <bucket>/<item_uuid>/{file1, file2 ... fileN}
				if p.Action == "put" {
					files, err := getLocalFiles(ri.Index(i).String())

					if err != nil {
						return temp, err
					}

					for _, file := range files {
						object := fmt.Sprintf("%s/%s", item.UUID, filepath.Base(file))
						if err := minioPut(p, file, object, p.Timeout); err != nil {
							return temp, err
						} else {
							log.WithFields(log.Fields{
								"hash":   p.Hash,
								"flow":   p.Flow,
								"file":   p.File,
								"plugin": p.Name,
								"type":   p.Type,
								"id":     p.ID,
								"alias":  p.Alias,
								"data":   fmt.Sprintf("put: %s/%s/%s", p.URL, p.Bucket, object),
							}).Debug(core.LOG_PLUGIN_STAT)
							ro.Set(reflect.Append(ro, reflect.ValueOf(object)))
						}
					}

					performed = true
				}
			}
		}

		if performed {
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
		Name:    "minio",
		TempDir: pluginConfig.Config.GetString(core.VIPER_DEFAULT_TEMP_DIR),
		Type:    "process",
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"include":  -1,
		"require":  -1,
		"template": -1,
		"timeout":  -1,

		"access_key": 1,
		"action":     1,
		"bucket":     1,
		"cred":       -1,
		"input":      1,
		"output":     1,
		"secret_key": 1,
		"ssl":        -1,
		"url":        1,
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

	cred, _ := core.IsString((*pluginConfig.Params)["cred"])

	// access_key.
	setAccessKey := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["access_key"] = 0
			plugin.AccessKey = v
		}
	}
	setAccessKey(pluginConfig.Config.GetString(fmt.Sprintf("%s.access_key", cred)))
	setAccessKey((*pluginConfig.Params)["access_key"])

	// secret_key.
	setSecretKey := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["secret_key"] = 0
			plugin.SecretKey = v
		}
	}
	setSecretKey(pluginConfig.Config.GetString(fmt.Sprintf("%s.secret_key", cred)))
	setSecretKey((*pluginConfig.Params)["secret_key"])

	// url.
	setURL := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["url"] = 0
			plugin.URL = v
		}
	}
	setURL(pluginConfig.Config.GetString(fmt.Sprintf("%s.url", cred)))
	setURL((*pluginConfig.Params)["url"])

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.Params)["template"])

	// action.
	setAction := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["action"] = 0
			plugin.Action = v
		}
	}
	setAction(pluginConfig.Config.GetString(fmt.Sprintf("%s.action", template)))
	setAction((*pluginConfig.Params)["action"])
	showParam("action", plugin.Action)

	// bucket.
	setBucket := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["bucket"] = 0
			plugin.Bucket = v
		}
	}
	setBucket(pluginConfig.Config.GetString(fmt.Sprintf("%s.bucket", template)))
	setBucket((*pluginConfig.Params)["bucket"])
	showParam("bucket", plugin.Action)

	// include.
	setInclude := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["include"] = 0
			plugin.Include = v
		}
	}
	setInclude(pluginConfig.Config.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
	setInclude(pluginConfig.Config.GetString(fmt.Sprintf("%s.include", template)))
	setInclude((*pluginConfig.Params)["include"])
	showParam("include", plugin.Include)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			if err := core.IsDataFieldsSlice(&v); err == nil {
				availableParams["input"] = 0
				plugin.Input = v
			}
		}
	}
	setInput(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.input", template)))
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
	setOutput(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.output", template)))
	setOutput((*pluginConfig.Params)["output"])
	showParam("output", plugin.Output)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.Require = v

		}
	}
	setRequire(pluginConfig.Config.GetIntSlice(fmt.Sprintf("%s.require", template)))
	setRequire((*pluginConfig.Params)["require"])
	showParam("require", plugin.Require)

	// ssl.
	setSSL := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["ssl"] = 0
			plugin.SSL = v
		}
	}
	setSSL(DEFAULT_SSL_ENABLE)
	setSSL(pluginConfig.Config.GetString(fmt.Sprintf("%s.ssl", template)))
	setSSL((*pluginConfig.Params)["ssl"])
	showParam("ssl", plugin.SSL)

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.Timeout = v
		}
	}
	setTimeout(pluginConfig.Config.GetInt(core.VIPER_DEFAULT_PLUGIN_TIMEOUT))
	setTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.timeout", template)))
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

	} else if plugin.Action != "put" {
		return &Plugin{}, fmt.Errorf(ERROR_ACTION_UNKNOWN.Error(), plugin.Action)

	} else {
		core.SliceStringToUpper(&plugin.Input)
		core.SliceStringToUpper(&plugin.Output)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
