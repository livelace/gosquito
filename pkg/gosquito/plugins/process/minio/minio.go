package minioProcess

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	PLUGIN_NAME = "minio"

	DEFAULT_SOURCE_DELETE = false
	DEFAULT_SSL_ENABLE    = true
)

var (
	ERROR_ACTION_UNKNOWN = errors.New("action unknown: %s")
	ERROR_IN_OUT_AMOUNT  = errors.New("input and output objects amount not equal: %d != %d")
	ERROR_DELETE_FILE    = errors.New("cannot delete file: %s, %v")
)

func minioPut(p *Plugin, file string, object string, timeout int) error {
	// context.
	c := make(chan error, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// client.
	client, err := minio.New(p.OptionServer, &minio.Options{
		Creds:  credentials.NewStaticV4(p.OptionAccessKey, p.OptionSecretKey, ""),
		Secure: p.OptionSSL,
	})
	if err != nil {
		return err
	}

	// background.
	go func() {
		_, err = client.FPutObject(ctx, p.OptionBucket, object, file, minio.PutObjectOptions{ContentType: "octet/stream"})
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

	if p.OptionSourceDelete {
		err := os.Remove(file)
		if err != nil {
			core.LogProcessPlugin(p.LogFields, fmt.Errorf(ERROR_DELETE_FILE.Error(), file, err))
		}
	}

	return nil
}

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionAccessKey    string
	OptionAction       string
	OptionBucket       string
	OptionInclude      bool
	OptionInput        []string
	OptionOutput       []string
	OptionRequire      []int
	OptionSourceDelete bool
	OptionSSL          bool
	OptionSecretKey    string
	OptionServer       string
	OptionTimeout      int
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

	for _, item := range data {
		performed := false

		for index, input := range p.OptionInput {
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.OptionOutput[index])

			switch ri.Kind() {
			case reflect.String:
				if err := minioPut(p, ri.String(), ro.String(), p.OptionTimeout); err != nil {
					return temp, err
				} else {
					performed = true
					core.LogProcessPlugin(p.LogFields,
						fmt.Sprintf("put: %s/%s/%s", p.OptionServer, p.OptionBucket, ro.String()))
				}

			case reflect.Slice:
				if ro.Len() != ri.Len() {
					return temp, fmt.Errorf(ERROR_IN_OUT_AMOUNT.Error(), ri.Len(), ro.Len())
				}

				for i := 0; i < ri.Len(); i++ {
					if err := minioPut(p, ri.Index(i).String(), ro.Index(i).String(), p.OptionTimeout); err != nil {
						return temp, err
					} else {
						performed = true
						core.LogProcessPlugin(p.LogFields,
							fmt.Sprintf("put: %s/%s/%s", p.OptionServer, p.OptionBucket, ro.Index(i).String()))
					}
				}
			}
		}

		if performed {
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
		"include":  -1,
		"require":  -1,
		"template": -1,

		"access_key":    1,
		"action":        1,
		"bucket":        1,
		"cred":          -1,
		"input":         1,
		"output":        1,
		"secret_key":    1,
		"server":        1,
		"source_delete": -1,
		"ssl":           -1,
		"timeout":       -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin settings or set defaults.

	cred, _ := core.IsString((*pluginConfig.PluginParams)["cred"])
	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])
	
    vault, err := core.GetVault(pluginConfig.AppConfig.GetStringMap(fmt.Sprintf("%s.vault", cred)))
	if err != nil {
		return &plugin, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	// access_key.
	setAccessKey := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["access_key"] = 0
			plugin.OptionAccessKey = core.GetCredValue(v, vault)
		}
	}
	setAccessKey(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.access_key", cred)))
	setAccessKey((*pluginConfig.PluginParams)["access_key"])

	// secret_key.
	setSecretKey := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["secret_key"] = 0
			plugin.OptionSecretKey = core.GetCredValue(v, vault)
		}
	}
	setSecretKey(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.secret_key", cred)))
	setSecretKey((*pluginConfig.PluginParams)["secret_key"])

	// server.
	setServer := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["server"] = 0
			plugin.OptionServer = core.GetCredValue(v, vault)
		}
	}
	setServer(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.server", cred)))
	setServer((*pluginConfig.PluginParams)["server"])

	// -----------------------------------------------------------------------------------------------------------------

	// action.
	setAction := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["action"] = 0
			plugin.OptionAction = v
		}
	}
	setAction(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.action", template)))
	setAction((*pluginConfig.PluginParams)["action"])
	core.ShowPluginParam(plugin.LogFields, "action", plugin.OptionAction)

	// bucket.
	setBucket := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["bucket"] = 0
			plugin.OptionBucket = v
		}
	}
	setBucket(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.bucket", template)))
	setBucket((*pluginConfig.PluginParams)["bucket"])
	core.ShowPluginParam(plugin.LogFields, "bucket", plugin.OptionBucket)

	// include.
	setInclude := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["include"] = 0
			plugin.OptionInclude = v
		}
	}
	setInclude(pluginConfig.AppConfig.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
	setInclude(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.include", template)))
	setInclude((*pluginConfig.PluginParams)["include"])
	core.ShowPluginParam(plugin.LogFields, "include", plugin.OptionInclude)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.OptionInput = v
		}
	}
	setInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.input", template)))
	setInput((*pluginConfig.PluginParams)["input"])
	core.ShowPluginParam(plugin.LogFields, "input", plugin.OptionInput)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.OptionOutput = v
		}
	}
	setOutput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.output", template)))
	setOutput((*pluginConfig.PluginParams)["output"])
	core.ShowPluginParam(plugin.LogFields, "output", plugin.OptionOutput)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.OptionRequire = v

		}
	}
	setRequire(pluginConfig.AppConfig.GetIntSlice(fmt.Sprintf("%s.require", template)))
	setRequire((*pluginConfig.PluginParams)["require"])
	core.ShowPluginParam(plugin.LogFields, "require", plugin.OptionRequire)

	// source_delete.
	setSourceDelete := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["source_delete"] = 0
			plugin.OptionSourceDelete = v
		}
	}
	setSourceDelete(DEFAULT_SOURCE_DELETE)
	setSourceDelete(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.source_delete", template)))
	setSourceDelete((*pluginConfig.PluginParams)["source_delete"])
	core.ShowPluginParam(plugin.LogFields, "source_delete", plugin.OptionSourceDelete)

	// ssl.
	setSSL := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["ssl"] = 0
			plugin.OptionSSL = v
		}
	}
	setSSL(DEFAULT_SSL_ENABLE)
	setSSL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.ssl", template)))
	setSSL((*pluginConfig.PluginParams)["ssl"])
	core.ShowPluginParam(plugin.LogFields, "ssl", plugin.OptionSSL)

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.OptionTimeout = v
		}
	}
	setTimeout(pluginConfig.AppConfig.GetInt(core.VIPER_DEFAULT_PLUGIN_TIMEOUT))
	setTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.PluginParams)["timeout"])
	core.ShowPluginParam(plugin.LogFields, "timeout", plugin.OptionTimeout)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// input and output must have equal size.
	if len(plugin.OptionInput) != len(plugin.OptionOutput) {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput)
	}

	if err := core.IsDataFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
		return &Plugin{}, err
	}

	if plugin.OptionAction != "put" {
		return &Plugin{}, fmt.Errorf(ERROR_ACTION_UNKNOWN.Error(), plugin.OptionAction)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
