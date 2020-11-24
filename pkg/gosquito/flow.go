package gosquito

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	rssIn "github.com/livelace/gosquito/pkg/gosquito/plugins/input/rss"
	telegramIn "github.com/livelace/gosquito/pkg/gosquito/plugins/input/telegram"
	twitterIn "github.com/livelace/gosquito/pkg/gosquito/plugins/input/twitter"
	kafkaOut "github.com/livelace/gosquito/pkg/gosquito/plugins/output/kafka"
	mattermostOut "github.com/livelace/gosquito/pkg/gosquito/plugins/output/mattermost"
	slackOut "github.com/livelace/gosquito/pkg/gosquito/plugins/output/slack"
	smtpOut "github.com/livelace/gosquito/pkg/gosquito/plugins/output/smtp"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/dedup"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/dirname"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/echo"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/expandurl"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/fetch"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/minio"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/regexpfind"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/regexpmatch"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/regexpreplace"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/unique"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/webchela"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/xpath"
	log "github.com/livelace/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sync/atomic"
)

func readFlow(dir string) ([]string, error) {
	temp := make([]string, 0)

	re1 := regexp.MustCompile("^.*\\.yml$")
	re2 := regexp.MustCompile("^.*\\.yaml$")

	err := filepath.Walk(dir, func(item string, info os.FileInfo, err error) error {

		if core.IsFile(item, "") &&
			(re1.MatchString(info.Name()) || re2.MatchString(info.Name())) {
			temp = append(temp, item)

		} else if core.IsFile(item, "") {
			log.WithFields(log.Fields{
				"file":  item,
				"error": core.ERROR_FILE_YAML,
			}).Warn(core.LOG_FLOW_IGNORE)
		}

		return nil
	})

	return temp, err
}

func getFlow(config *viper.Viper) []*core.Flow {
	var flows []*core.Flow

	// -----------------------------------------------------------------------------------------------------------------
	// Early checks.

	// Enable/disable selected flows (mutual exclusive).
	flowsDisabled := config.GetStringSlice(core.VIPER_DEFAULT_FLOW_DISABLE)
	flowsEnabled := config.GetStringSlice(core.VIPER_DEFAULT_FLOW_ENABLE)

	if len(flowsDisabled) > 0 && len(flowsEnabled) > 0 {
		log.WithFields(log.Fields{
			"error": core.ERROR_FLOW_ENABLE_DISABLE_CONFLICT,
		}).Error(core.LOG_FLOW_READ)
		os.Exit(1)
	}

	// Checking flows names uniqueness.
	flowsNames := make(map[string]string)

	// Read "flow" files.
	files, err := readFlow(config.GetString(core.VIPER_DEFAULT_FLOW_CONF))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error(core.LOG_FLOW_READ)
		os.Exit(1)
	}

	// Exit if there are no flows.
	if len(files) == 0 {
		log.WithFields(log.Fields{
			"path":  config.GetString(core.VIPER_DEFAULT_FLOW_CONF),
			"error": core.ERROR_NO_VALID_FLOW,
		}).Error(core.LOG_FLOW_READ)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------------------------------------------------

	// Each file produces only one "flow" configuration.
	for _, file := range files {
		// Every flow consists of:
		// 1. Flow parameters.
		// 2. Input plugin.
		// 3. Process plugins.
		// 4. Output plugins.

		fileName := filepath.Base(file)

		// Logging.
		logFlowError := func(msg string, err error) {
			log.WithFields(log.Fields{
				"file":  fileName,
				"error": err,
			}).Error(msg)
		}

		var flowHash = core.GenFlowHash()
		var flowUUID, _ = uuid.NewRandom()
		var flowName string
		var flowInterval int64
		var flowNumber int
		var flowParams map[string]interface{}
		var inputPlugin core.InputPlugin
		var processPlugins = make(map[int]core.ProcessPlugin, 0)
		var processPluginsNames = make([]string, 0)
		var outputPlugin core.OutputPlugin

		// Read "raw" flow data into structure.
		flowRaw := core.FlowUnmarshal{}

		// Skip flow if we cannot read flow.
		data, err := ioutil.ReadFile(file)
		if err != nil {
			logFlowError(core.LOG_FLOW_READ, err)
			continue
		}

		// Skip flow if we cannot unmarshal flow yaml.
		err = yaml.Unmarshal(data, &flowRaw)
		if err != nil {
			logFlowError(core.ERROR_FLOW_PARSE.Error(), err)
			continue
		}

		// Flow name must be compatible.
		if !core.IsFlowName(flowRaw.Flow.Name) {
			logFlowError(core.LOG_FLOW_READ,
				fmt.Errorf(core.ERROR_FLOW_NAME_COMPAT.Error(), flowRaw.Flow.Name))
			continue
		}

		// Flow name must be unique.
		if v, ok := flowsNames[flowRaw.Flow.Name]; ok {
			logFlowError(core.LOG_FLOW_READ,
				fmt.Errorf(core.ERROR_FLOW_NAME_UNIQUE.Error(), v))
			continue
		}

		// Exclude disabled flows.
		if (len(flowsEnabled) > 0 && !core.IsValueInSlice(flowRaw.Flow.Name, &flowsEnabled)) ||
			(len(flowsDisabled) > 0 && core.IsValueInSlice(flowRaw.Flow.Name, &flowsDisabled)) {
			log.WithFields(log.Fields{
				"flow":  flowRaw.Flow.Name,
				"error": core.ERROR_FLOW_DISABLED,
			}).Warn(core.LOG_FLOW_IGNORE)
			continue
		}

		flowName = flowRaw.Flow.Name
		flowsNames[flowName] = fileName

		// ---------------------------------------------------------------------------------------------------------
		// Logging.

		LogParam := func(p string, v interface{}) {
			log.WithFields(log.Fields{
				"hash":  flowHash,
				"flow":  flowName,
				"file":  fileName,
				"type":  "flow",
				"value": fmt.Sprintf("%s: %v", p, v),
			}).Debug(core.LOG_SET_VALUE)
		}

		LogPluginError := func(plugin string, pluginType string, msg string, err error) {
			log.WithFields(log.Fields{
				"hash":   flowHash,
				"flow":   flowName,
				"file":   fileName,
				"plugin": plugin,
				"type":   pluginType,
				"error":  err,
			}).Error(msg)
		}

		// ---------------------------------------------------------------------------------------------------------
		// Map "flow" params.

		// Every flow has these parameters.
		flowParamsAvailable := map[string]int{
			"interval": -1,
			"number":   -1,
		}

		// Flow parameters may be not specified (use defaults).
		flowParams, _ = core.IsMapWithStringAsKey(flowRaw.Flow.Params)

		// Check required and unknown parameters.
		if err := core.CheckPluginParams(&flowParamsAvailable, &flowParams); err != nil {
			log.WithFields(log.Fields{
				"flow":  flowName,
				"file":  fileName,
				"error": err,
			}).Error(core.ERROR_PARAM_ERROR)
			continue
		}

		// Set flow interval.
		if v, b := core.IsInterval(flowParams["interval"]); b {
			flowInterval = v
			LogParam("interval", v)
		} else {
			flowInterval, _ = core.IsInterval(config.GetString(core.VIPER_DEFAULT_FLOW_INTERVAL))
			LogParam("interval", flowInterval)
		}

		// Set flow number limit.
		if v, b := core.IsInt(flowParams["number"]); b {
			flowNumber = v
			LogParam("number", v)
		} else {
			flowNumber = config.GetInt(core.VIPER_DEFAULT_FLOW_NUMBER)
			LogParam("number", flowNumber)
		}

		// ---------------------------------------------------------------------------------------------------------
		// Map "input" plugin.

		inputParams, b := core.IsMapWithStringAsKey(flowRaw.Flow.Input.Params)
		if !b {
			LogPluginError(flowRaw.Flow.Input.Plugin, "input", core.ERROR_PARAM_ERROR.Error(),
				core.ERROR_PARAM_KEY_MUST_STRING)
			continue
		}

		// Assemble plugin configuration.
		inputPluginConfig := core.PluginConfig{
			Config: config,
			File:   fileName,
			Flow:   flowName,
			Hash:   flowHash,
			Params: &inputParams,
		}

		// Available "input" plugins.
		switch flowRaw.Flow.Input.Plugin {
		case "rss":
			inputPlugin, err = rssIn.Init(&inputPluginConfig)
		case "telegram":
			inputPlugin, err = telegramIn.Init(&inputPluginConfig)
		case "twitter":
			inputPlugin, err = twitterIn.Init(&inputPluginConfig)
		default:
			err = core.ERROR_PLUGIN_UNKNOWN
		}

		// Skip flow if we cannot initialize "input" plugin.
		if err != nil {
			LogPluginError(flowRaw.Flow.Input.Plugin, "input", core.LOG_PLUGIN_INIT, err)
			continue
		}

		// ---------------------------------------------------------------------------------------------------------
		// Map "process" plugins.

		for index := 0; index < len(flowRaw.Flow.Process); index++ {
			item := flowRaw.Flow.Process[index]

			var pluginId int
			var plugin core.ProcessPlugin
			var pluginName string

			// Validate "process" plugins items declaration.
			pluginId, a := core.IsPluginId(item["id"])
			pluginName, b := core.IsString(item["plugin"])
			pluginParams, c := core.IsMapWithStringAsKey(item["params"])

			pluginAlias, _ := core.IsString(item["alias"])

			// Every plugin must have: id, plugin, params.
			if !a || !b || !c {
				log.WithFields(log.Fields{
					"flow":   flowName,
					"file":   fileName,
					"plugin": pluginName,
					"id":     pluginId,
					"error":  core.ERROR_PLUGIN_PROCESS_PARAMS,
				}).Error(core.LOG_PLUGIN_INIT)
				break
			}

			// All "process" plugins ids must be ordered.
			if pluginId != index {
				log.WithFields(log.Fields{
					"flow":   flowName,
					"file":   fileName,
					"plugin": pluginName,
					"id":     pluginId,
					"error":  core.ERROR_PLUGIN_PROCESS_ORDER,
				}).Error(core.LOG_PLUGIN_INIT)
				break
			}

			// Assemble plugin configuration.
			processPluginConfig := core.PluginConfig{
				Alias:  pluginAlias,
				Config: config,
				File:   fileName,
				Flow:   flowName,
				Hash:   flowHash,
				ID:     pluginId,
				Params: &pluginParams,
			}

			// Available "process" plugins.
			switch pluginName {
			case "dedup":
				plugin, err = dedup.Init(&processPluginConfig)
			case "dirname":
				plugin, err = dirname.Init(&processPluginConfig)
			case "expandurl":
				plugin, err = expandurl.Init(&processPluginConfig)
			case "fetch":
				plugin, err = fetch.Init(&processPluginConfig)
			case "minio":
				plugin, err = minio.Init(&processPluginConfig)
			case "echo":
				plugin, err = echo.Init(&processPluginConfig)
			case "regexpfind":
				plugin, err = regexpfind.Init(&processPluginConfig)
			case "regexpmatch":
				plugin, err = regexpmatch.Init(&processPluginConfig)
			case "regexpreplace":
				plugin, err = regexpreplace.Init(&processPluginConfig)
			case "unique":
				plugin, err = unique.Init(&processPluginConfig)
			case "webchela":
				plugin, err = webchela.Init(&processPluginConfig)
			case "xpath":
				plugin, err = xpath.Init(&processPluginConfig)
			default:
				err = core.ERROR_PLUGIN_UNKNOWN
			}

			if err != nil {
				log.WithFields(log.Fields{
					"flow":   flowName,
					"file":   fileName,
					"plugin": pluginName,
					"id":     pluginId,
					"error":  err,
				}).Error(core.LOG_PLUGIN_INIT)

			} else {
				processPlugins[pluginId] = plugin
				processPluginsNames = append(processPluginsNames, pluginName)
			}
		}

		// Skip flow if some "process" plugins weren't initialized.
		if len(processPlugins) != len(flowRaw.Flow.Process) {
			continue
		}

		// ---------------------------------------------------------------------------------------------------------
		// Map "output" plugin.

		outputParams, b := core.IsMapWithStringAsKey(flowRaw.Flow.Output.Params)
		if b {
			// Assemble plugin configuration.
			outputPluginConfig := core.PluginConfig{
				Config: config,
				File:   fileName,
				Flow:   flowName,
				Hash:   flowHash,
				Params: &outputParams,
			}

			// Available "output" plugins.
			switch flowRaw.Flow.Output.Plugin {
			case "kafka":
				outputPlugin, err = kafkaOut.Init(&outputPluginConfig)
			case "mattermost":
				outputPlugin, err = mattermostOut.Init(&outputPluginConfig)
			case "slack":
				outputPlugin, err = slackOut.Init(&outputPluginConfig)
			case "smtp":
				outputPlugin, err = smtpOut.Init(&outputPluginConfig)
			default:
				err = core.ERROR_PLUGIN_UNKNOWN
			}

			// Skip flow if we cannot initialize "output" plugin.
			if err != nil {
				LogPluginError(flowRaw.Flow.Output.Plugin, "output", core.LOG_PLUGIN_INIT, err)
				continue
			}
		}

		flows = append(flows, &core.Flow{
			Hash: flowHash,
			UUID: flowUUID,

			Name:                flowName,
			Interval:            flowInterval,
			Number:              flowNumber,
			InputPlugin:         inputPlugin,
			ProcessPlugins:      processPlugins,
			ProcessPluginsNames: processPluginsNames,
			OutputPlugin:        outputPlugin,
		})
	}

	return flows
}

func runFlow(config *viper.Viper, flow *core.Flow) {
	// -----------------------------------------------------------------------------------------------------------------
	// Let's get started :)

	if flow.Lock() {
		log.WithFields(log.Fields{
			"hash": flow.Hash,
			"flow": flow.Name,
		}).Info(core.LOG_FLOW_START)

		defer flow.Unlock()

	} else {
		log.WithFields(log.Fields{
			"hash": flow.Hash,
			"flow": flow.Name,
		}).Warn(core.LOG_FLOW_LOCK_WARNING)

		return
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Cleanup flow temp dir.

	flowTempDir := filepath.Join(config.GetString(core.VIPER_DEFAULT_PLUGIN_TEMP), flow.Name)
	cleanFlowTemp := func() {
		_ = os.RemoveAll(flowTempDir)
	}

	// -----------------------------------------------------------------------------------------------------------------
	var err error
	results := make(map[int][]*core.DataItem)

	// -----------------------------------------------------------------------------------------------------------------
	// Get data from "input" plugin.

	logFlowWarn := func(err error) {
		log.WithFields(log.Fields{
			"hash":   flow.Hash,
			"flow":   flow.Name,
			"plugin": flow.InputPlugin.GetName(),
			"type":   flow.InputPlugin.GetType(),
			"error":  err,
		}).Warn(core.LOG_FLOW_WARN)
	}

	logFlowStat := func(msg interface{}) {
		log.WithFields(log.Fields{
			"hash":   flow.Hash,
			"flow":   flow.Name,
			"file":   flow.InputPlugin.GetFile(),
			"plugin": flow.InputPlugin.GetName(),
			"type":   flow.InputPlugin.GetType(),
			"data":   fmt.Sprintf("%v", msg),
		}).Debug(core.LOG_FLOW_STAT)
	}

	logFlowStop := func() {
		log.WithFields(log.Fields{
			"hash": flow.Hash,
			"flow": flow.Name,
		}).Info(core.LOG_FLOW_STOP)
	}

	log.WithFields(log.Fields{
		"hash":   flow.Hash,
		"flow":   flow.Name,
		"plugin": flow.InputPlugin.GetName(),
		"type":   flow.InputPlugin.GetType(),
	}).Info(core.LOG_FLOW_RECEIVE)

	// Get data.
	inputData, err := flow.InputPlugin.Recv()
	logFlowStat(len(inputData))

	// Process data if some of flow sources are expired/failed.
	// Skip flow if we have other problems.
	if err == core.ERROR_FLOW_EXPIRE {
		atomic.AddInt32(&flow.MetricExpire, 1)
		logFlowWarn(err)

	} else if err == core.ERROR_FLOW_SOURCE_FAIL {
		atomic.AddInt32(&flow.MetricError, 1)
		logFlowWarn(err)

	} else if err != nil {
		atomic.AddInt32(&flow.MetricError, 1)
		logFlowWarn(err)
		cleanFlowTemp()
		logFlowStop()
		return
	}

	// Skip flow if we don't have new data.
	if len(inputData) == 0 {
		atomic.AddInt32(&flow.MetricNoData, 1)
		logFlowWarn(core.ERROR_NO_NEW_DATA)
		cleanFlowTemp()
		return
	} else {
		atomic.AddInt32(&flow.MetricReceive, int32(len(inputData)))
	}

	// -------------------------------------------------------------------------------------------------------------
	// Process received data with "process" plugins.

	if len(flow.ProcessPlugins) > 0 {
		log.WithFields(log.Fields{
			"hash":   flow.Hash,
			"flow":   flow.Name,
			"plugin": flow.ProcessPluginsNames,
			"type":   "process",
		}).Info(core.LOG_FLOW_PROCESS)

		// Every "process" plugin generates its own dataset.
		// Any dataset could be excluded from sending through "output" plugin.
		for index := 0; index < len(flow.ProcessPlugins); index++ {
			result := make([]*core.DataItem, 0)

			plugin := flow.ProcessPlugins[index]
			require := flow.ProcessPlugins[index].GetRequire()

			// Work with data from inputPlugin if:
			// 1. It's the first "process" plugin on the list.
			// 2 "require" is not set for plugin.
			if index == 0 || len(require) == 0 {
				result, err = plugin.Do(inputData)
			} else {
				// Combine datasets from required process plugins.
				var combined = make([]*core.DataItem, 0)

				// 1. Plugin cannot require itself.
				// 2. Plugin cannot require higher id (ordered processing).
				for i := 0; i < len(require); i++ {
					id := require[i]

					if id < index {
						combined = append(combined, results[id]...)
					}
				}

				result, err = plugin.Do(combined)
			}

			// Skip flow if we have problems with data processing.
			if err != nil {
				log.WithFields(log.Fields{
					"hash":   flow.Hash,
					"flow":   flow.Name,
					"file":   plugin.GetFile(),
					"plugin": plugin.GetName(),
					"type":   plugin.GetType(),
					"id":     plugin.GetId(),
					"alias":  plugin.GetAlias(),
					"error":  err,
				}).Warn(core.LOG_FLOW_WARN)

				atomic.AddInt32(&flow.MetricError, 1)
				cleanFlowTemp()
				return

			} else {
				log.WithFields(log.Fields{
					"hash":   flow.Hash,
					"flow":   flow.Name,
					"file":   plugin.GetFile(),
					"plugin": plugin.GetName(),
					"type":   plugin.GetType(),
					"id":     plugin.GetId(),
					"alias":  plugin.GetAlias(),
					"data":   len(result),
				}).Debug(core.LOG_FLOW_STAT)

				results[index] = result
			}
		}
	}

	// -------------------------------------------------------------------------------------------------------------
	// Send data through "output" plugin.

	if flow.OutputPlugin != nil {

		logFlowWarn = func(err error) {
			log.WithFields(log.Fields{
				"hash":   flow.Hash,
				"flow":   flow.Name,
				"plugin": flow.OutputPlugin.GetName(),
				"type":   flow.OutputPlugin.GetType(),
				"error":  err,
			}).Warn(core.LOG_FLOW_WARN)
		}

		logFlowStat = func(msg interface{}) {
			log.WithFields(log.Fields{
				"hash":   flow.Hash,
				"flow":   flow.Name,
				"file":   flow.OutputPlugin.GetFile(),
				"plugin": flow.OutputPlugin.GetName(),
				"type":   flow.OutputPlugin.GetType(),
				"data":   fmt.Sprintf("%v", msg),
			}).Debug(core.LOG_FLOW_STAT)
		}

		log.WithFields(log.Fields{
			"hash":   flow.Hash,
			"flow":   flow.Name,
			"plugin": flow.OutputPlugin.GetName(),
			"type":   flow.OutputPlugin.GetType(),
		}).Info(core.LOG_FLOW_SEND)

		// 1. Pass data from "process" plugins to "output" plugin.
		// 2. Pass data from "input" plugin to "output" plugin directly.
		if len(results) > 0 {
			somethingIncluded := false
			somethingHasData := false

			for index := 0; index < len(results); index++ {
				data := results[index]

				// Send only needed data (param "include" is "true").
				if flow.ProcessPlugins[index].GetInclude() {
					somethingIncluded = true

					// Send only not empty data (some plugins can produce zero data).
					if len(data) > 0 {
						somethingHasData = true
						err = flow.OutputPlugin.Send(data)

						if err != nil {
							atomic.AddInt32(&flow.MetricError, 1)
							logFlowWarn(err)
							cleanFlowTemp()
							return

						} else {
							atomic.AddInt32(&flow.MetricSend, int32(len(data)))
							logFlowStat(fmt.Sprintf("process plugin id: %d, send data: %d", index, len(data)))
						}
					}
				}
			}

			// More informative messages.
			if !somethingIncluded {
				logFlowStat(core.LOG_FLOW_SEND_NO_DATA_INCLUDED)
			}

			if !somethingHasData {
				logFlowStat(core.LOG_FLOW_SEND_NO_DATA)
			}

		} else if len(inputData) > 0 {
			err = flow.OutputPlugin.Send(inputData)

			if err != nil {
				atomic.AddInt32(&flow.MetricError, 1)
				logFlowWarn(err)
				cleanFlowTemp()
				return

			} else {
				atomic.AddInt32(&flow.MetricSend, int32(len(inputData)))
				logFlowStat(fmt.Sprintf("input plugin: %s, send data: %d",
					flow.InputPlugin.GetName(), len(inputData)))
			}
		} else {
			logFlowStat(fmt.Sprintf("no data for sending"))
		}
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Cleanup at the end.

	cleanFlowTemp()

	// -----------------------------------------------------------------------------------------------------------------
	// Fin.

	logFlowStop()
}
