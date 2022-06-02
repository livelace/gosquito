package gosquito

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	rssIn "github.com/livelace/gosquito/pkg/gosquito/plugins/input/rss"
	twitterIn "github.com/livelace/gosquito/pkg/gosquito/plugins/input/twitter"
	kafkaMulti "github.com/livelace/gosquito/pkg/gosquito/plugins/multi/kafka"
	restyMulti "github.com/livelace/gosquito/pkg/gosquito/plugins/multi/resty"
	telegramMulti "github.com/livelace/gosquito/pkg/gosquito/plugins/multi/telegram"
	mattermostOut "github.com/livelace/gosquito/pkg/gosquito/plugins/output/mattermost"
	slackOut "github.com/livelace/gosquito/pkg/gosquito/plugins/output/slack"
	smtpOut "github.com/livelace/gosquito/pkg/gosquito/plugins/output/smtp"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/dedup"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/dirname"
	echoProcess "github.com/livelace/gosquito/pkg/gosquito/plugins/process/echo"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/expandurl"
	"github.com/livelace/gosquito/pkg/gosquito/plugins/process/fetch"
	jqProcess "github.com/livelace/gosquito/pkg/gosquito/plugins/process/jq"
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

func getFlow(appConfig *viper.Viper) []*core.Flow {
	var flows []*core.Flow

	// -----------------------------------------------------------------------------------------------------------------
	// Early checks.

	// Enable/disable selected flows (mutual exclusive).
	flowsDisabled := appConfig.GetStringSlice(core.VIPER_DEFAULT_FLOW_DISABLE)
	flowsEnabled := appConfig.GetStringSlice(core.VIPER_DEFAULT_FLOW_ENABLE)

	if len(flowsDisabled) > 0 && len(flowsEnabled) > 0 {
		log.WithFields(log.Fields{
			"error": core.ERROR_FLOW_ENABLE_DISABLE_CONFLICT,
		}).Error(core.LOG_FLOW_READ)
		os.Exit(1)
	}

	// Checking flows names uniqueness.
	flowsNames := make(map[string]string)

	// Read "flow" files.
	files, err := readFlow(appConfig.GetString(core.VIPER_DEFAULT_FLOW_CONF))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error(core.LOG_FLOW_READ)
		os.Exit(1)
	}

	// Exit if there are no flows.
	if len(files) == 0 {
		log.WithFields(log.Fields{
			"path":  appConfig.GetString(core.VIPER_DEFAULT_FLOW_CONF),
			"error": core.ERROR_NO_VALID_FLOW,
		}).Error(core.LOG_FLOW_READ)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Need for Telegram instance amount restrictions.
	// Only one Telegram plugin instance available right now (since tdlib 1.7.0).
	//telegramPluginTotal := 0

	// Each file produces only one "flow" configuration.
	for _, file := range files {
		// ---------------------------------------------------------------------------------------------------------
		// Every flow consists of:
		// 1. Flow parameters.
		// 2. Input plugin.
		// 3. Process plugins.
		// 4. Output plugins.

		fileName := filepath.Base(file)

		// Logging.
		logFlowFileError := func(err error) {
			log.WithFields(log.Fields{
				"file":  fileName,
				"error": err,
			}).Error(core.LOG_FLOW_READ)
		}

		// ---------------------------------------------------------------------------------------------------------
		// Parse and check flow body.

		var flowUUID, _ = uuid.NewRandom()
		var flowHash = core.GenUID()
		var flowName string
		var flowRunID = int64(0)

		var flowCleanup bool
		var flowInstance int
		var flowInterval int64

		var flowParams map[string]interface{}

		var inputPlugin core.InputPlugin
		var processPlugins = make(map[int]core.ProcessPlugin, 0)
		var processPluginsNames = make([]string, 0)
		var outputPlugin core.OutputPlugin

		// Read flow body into structure.
		flowBody := core.FlowUnmarshal{}

		// Skip flow if we cannot read flow.
		data, err := ioutil.ReadFile(file)
		if err != nil {
			logFlowFileError(err)
			continue
		}

		// Skip flow if we cannot unmarshal flow yaml.
		err = yaml.Unmarshal(data, &flowBody)
		if err != nil {
			logFlowFileError(err)
			continue
		}

		// Flow name must be compatible.
		if !core.IsFlowName(flowBody.Flow.Name) {
			logFlowFileError(fmt.Errorf(core.ERROR_FLOW_NAME_COMPAT.Error(), flowBody.Flow.Name))
			continue
		}

		// Flow name must be unique.
		if v, ok := flowsNames[flowBody.Flow.Name]; ok {
			logFlowFileError(fmt.Errorf(core.ERROR_FLOW_NAME_UNIQUE.Error(), v))
			continue
		}

		// Exclude disabled flows.
		if (len(flowsEnabled) > 0 && !core.IsValueInSlice(flowBody.Flow.Name, &flowsEnabled)) ||
			(len(flowsDisabled) > 0 && core.IsValueInSlice(flowBody.Flow.Name, &flowsDisabled)) {

			log.WithFields(log.Fields{
				"flow":  flowBody.Flow.Name,
				"error": core.ERROR_FLOW_DISABLED,
			}).Warn(core.LOG_FLOW_IGNORE)

			continue
		}

		flowName = flowBody.Flow.Name
		flowsNames[flowName] = fileName

		// ---------------------------------------------------------------------------------------------------------
		// Logging.

		logFlowValid := func(p string) {
			log.WithFields(log.Fields{
				"hash": flowHash,
				"flow": flowName,
				"file": fileName,
			}).Info(core.LOG_FLOW_VALID)
		}

		logFlowInvalid := func(p string) {
			log.WithFields(log.Fields{
				"hash": flowHash,
				"flow": flowName,
				"file": fileName,
			}).Error(core.LOG_FLOW_INVALID)
		}

		logFlowParam := func(p string, v interface{}) {
			log.WithFields(log.Fields{
				"hash":  flowHash,
				"flow":  flowName,
				"file":  fileName,
				"type":  "flow",
				"value": fmt.Sprintf("%s: %v", p, v),
			}).Debug(core.LOG_SET_VALUE)
		}

		logInputOutputPluginError := func(plugin string, pluginType string, kind string, err error) {
			log.WithFields(log.Fields{
				"hash":   flowHash,
				"flow":   flowName,
				"file":   fileName,
				"plugin": plugin,
				"type":   pluginType,
				"error":  err,
			}).Error(kind)
		}

		// ---------------------------------------------------------------------------------------------------------
		// Map "flow" params.

		// Every flow has these parameters.
		flowParamsAvailable := map[string]int{
			"cleanup":  -1,
			"instance": -1,
			"interval": -1,
		}

		// Flow parameters may be not specified (use defaults).
		flowParams, _ = core.IsMapWithStringAsKey(flowBody.Flow.Params)

		// Check required and unknown parameters.
		if err := core.CheckPluginParams(&flowParamsAvailable, &flowParams); err != nil {
			log.WithFields(log.Fields{
				"flow":  flowName,
				"file":  fileName,
				"error": err,
			}).Error(core.ERROR_PARAM_ERROR)
			logFlowInvalid(flowName)
			continue
		}

		// Set flow cleanup.
		if v, b := core.IsBool(flowParams["cleanup"]); b {
			flowCleanup = v
			logFlowParam("cleanup", v)
		} else {
			flowCleanup = appConfig.GetBool(core.VIPER_DEFAULT_FLOW_CLEANUP)
			logFlowParam("cleanup", flowCleanup)
		}

		// Set flow instance limit.
		if v, b := core.IsInt(flowParams["instance"]); b {
			flowInstance = v
			logFlowParam("instance", v)
		} else {
			flowInstance = appConfig.GetInt(core.VIPER_DEFAULT_FLOW_INSTANCE)
			logFlowParam("instance", flowInstance)
		}

		// Set flow interval.
		if v, b := core.IsInterval(flowParams["interval"]); b {
			flowInterval = v
			logFlowParam("interval", v)
		} else {
			flowInterval, _ = core.IsInterval(appConfig.GetString(core.VIPER_DEFAULT_FLOW_INTERVAL))
			logFlowParam("interval", flowInterval)
		}

		// ---------------------------------------------------------------------------------------------------------
		// Create flow.

		flow := &core.Flow{
			FlowUUID:  flowUUID,
			FlowHash:  flowHash,
			FlowName:  flowName,
			FlowRunID: flowRunID,

			FlowFile:     fileName,
			FlowDataDir:  filepath.Join(appConfig.GetString(core.VIPER_DEFAULT_FLOW_DATA), flowName, core.DEFAULT_DATA_DIR),
			FlowStateDir: filepath.Join(appConfig.GetString(core.VIPER_DEFAULT_FLOW_DATA), flowName, core.DEFAULT_STATE_DIR),
			FlowTempDir:  filepath.Join(appConfig.GetString(core.VIPER_DEFAULT_FLOW_DATA), flowName, core.DEFAULT_TEMP_DIR),

			FlowCleanup:  flowCleanup,
			FlowInstance: flowInstance,
			FlowInterval: flowInterval,
		}

		// ---------------------------------------------------------------------------------------------------------
		// Map "input" plugin.

		inputParams, b := core.IsMapWithStringAsKey(flowBody.Flow.Input.Params)
		if !b {
			logInputOutputPluginError(flowBody.Flow.Input.Plugin, "input", core.ERROR_PARAM_ERROR.Error(),
				core.ERROR_PARAM_KEY_MUST_STRING)
			logFlowInvalid(flowName)
			continue
		}

		// Assemble plugin configuration.
		inputPluginConfig := core.PluginConfig{
			AppConfig:    appConfig,
			Flow:         flow,
			PluginParams: &inputParams,
			PluginType:   "input",
		}

		// Available "input" plugins.
		switch flowBody.Flow.Input.Plugin {
		case "kafka":
			inputPlugin, err = kafkaMulti.Init(&inputPluginConfig)
		case "resty":
			inputPlugin, err = restyMulti.Init(&inputPluginConfig)
		case "rss":
			inputPlugin, err = rssIn.Init(&inputPluginConfig)
		case "telegram":
			//if telegramPluginTotal < telegramMulti.MAX_INSTANCE_PER_APP {
			//	inputPlugin, err = telegramMulti.Init(&inputPluginConfig)
			//	telegramPluginTotal += 1
			//} else {
			//	logInputOutputPluginError(flowBody.Flow.Input.Plugin, "input", core.LOG_PLUGIN_INIT,
			//		fmt.Errorf(core.ERROR_PLUGIN_MAX_INSTANCE.Error(), telegramPluginTotal))
			//	continue
			//}
            inputPlugin, err = telegramMulti.Init(&inputPluginConfig)
		case "twitter":
			inputPlugin, err = twitterIn.Init(&inputPluginConfig)
		default:
			err = core.ERROR_PLUGIN_UNKNOWN
		}

		// Skip flow if we cannot initialize "input" plugin.
		if err != nil {
			logInputOutputPluginError(flowBody.Flow.Input.Plugin, "input", core.LOG_PLUGIN_INIT, err)
			logFlowInvalid(flowName)
			continue
		}

		// ---------------------------------------------------------------------------------------------------------
		// Map "process" plugins.

		for pluginIndex := 0; pluginIndex < len(flowBody.Flow.Process); pluginIndex++ {
			pluginItem := flowBody.Flow.Process[pluginIndex]

			var plugin core.ProcessPlugin
			var pluginId int
			var pluginName string

			// Validate "process" plugins items declaration.
			pluginId, a := core.IsPluginId(pluginItem["id"])
			pluginName, b := core.IsString(pluginItem["plugin"])
			pluginParams, c := core.IsMapWithStringAsKey(pluginItem["params"])
			pluginAlias, _ := core.IsString(pluginItem["alias"])

			// Logging.
			logProcessPluginError := func(err error) {
				log.WithFields(log.Fields{
					"flow":   flowName,
					"file":   fileName,
					"plugin": pluginName,
					"id":     pluginId,
					"error":  err,
				}).Error(core.LOG_PLUGIN_INIT)
			}

			// Every plugin must have: id, plugin, params.
			if !a || !b || !c {
				logProcessPluginError(core.ERROR_PLUGIN_PROCESS_PARAMS)
				break
			}

			// All "process" plugins ids must be ordered.
			if pluginId != pluginIndex {
				logProcessPluginError(fmt.Errorf("%s: %d", core.ERROR_PLUGIN_PROCESS_ORDER, pluginId))
				break
			}

			// Assemble plugin configuration.
			processPluginConfig := core.PluginConfig{
				AppConfig:    appConfig,
				Flow:         flow,
				PluginID:     pluginId,
				PluginAlias:  pluginAlias,
				PluginParams: &pluginParams,
				PluginType:   "process",
			}

			// Available "process" plugins.
			switch pluginName {
			case "dedup":
				plugin, err = dedupProcess.Init(&processPluginConfig)
			case "dirname":
				plugin, err = dirnameProcess.Init(&processPluginConfig)
			case "echo":
				plugin, err = echoProcess.Init(&processPluginConfig)
			case "expandurl":
				plugin, err = expandurlProcess.Init(&processPluginConfig)
			case "fetch":
				plugin, err = fetchProcess.Init(&processPluginConfig)
			case "jq":
				plugin, err = jqProcess.Init(&processPluginConfig)
			case "minio":
				plugin, err = minioProcess.Init(&processPluginConfig)
			case "regexpfind":
				plugin, err = regexpfindProcess.Init(&processPluginConfig)
			case "regexpmatch":
				plugin, err = regexpmatchProcess.Init(&processPluginConfig)
			case "regexpreplace":
				plugin, err = regexpreplaceProcess.Init(&processPluginConfig)
			case "resty":
				plugin, err = restyMulti.Init(&processPluginConfig)
			case "unique":
				plugin, err = uniqueProcess.Init(&processPluginConfig)
			case "webchela":
				plugin, err = webchelaProcess.Init(&processPluginConfig)
			case "xpath":
				plugin, err = xpathProcess.Init(&processPluginConfig)
			default:
				err = fmt.Errorf("%s: %s", core.ERROR_PLUGIN_UNKNOWN, pluginName)
			}

			if err != nil {
				logProcessPluginError(err)

			} else {
				processPlugins[pluginId] = plugin
				processPluginsNames = append(processPluginsNames, pluginName)
			}
		}

		// Skip flow if some "process" plugins weren't initialized.
		if len(processPlugins) != len(flowBody.Flow.Process) {
			logFlowInvalid(flowName)
			continue
		}

		// ---------------------------------------------------------------------------------------------------------
		// Map "output" plugin.

		outputParams, b := core.IsMapWithStringAsKey(flowBody.Flow.Output.Params)
		if b {
			// Assemble plugin configuration.
			outputPluginConfig := core.PluginConfig{
				AppConfig:    appConfig,
				Flow:         flow,
				PluginParams: &outputParams,
				PluginType:   "output",
			}

			// Available "output" plugins.
			switch flowBody.Flow.Output.Plugin {
			case "kafka":
				outputPlugin, err = kafkaMulti.Init(&outputPluginConfig)
			case "mattermost":
				outputPlugin, err = mattermostOut.Init(&outputPluginConfig)
			case "resty":
				outputPlugin, err = restyMulti.Init(&outputPluginConfig)
			case "slack":
				outputPlugin, err = slackOut.Init(&outputPluginConfig)
			case "smtp":
				outputPlugin, err = smtpOut.Init(&outputPluginConfig)
			case "telegram":
				//if telegramPluginTotal < telegramMulti.MAX_INSTANCE_PER_APP {
				//	outputPlugin, err = telegramMulti.Init(&outputPluginConfig)
				//	telegramPluginTotal += 1
				//} else {
				//	logInputOutputPluginError(flowBody.Flow.Output.Plugin, "output", core.LOG_PLUGIN_INIT,
				//		fmt.Errorf(core.ERROR_PLUGIN_MAX_INSTANCE.Error(), telegramPluginTotal))
				//	continue
				//}
                outputPlugin, err = telegramMulti.Init(&outputPluginConfig)
			default:
				err = fmt.Errorf("%s: %s", core.ERROR_PLUGIN_UNKNOWN, flowBody.Flow.Output.Plugin)
			}

			// Skip flow if we cannot initialize "output" plugin.
			if err != nil {
				logInputOutputPluginError(flowBody.Flow.Output.Plugin, "output", core.LOG_PLUGIN_INIT, err)
				logFlowInvalid(flowName)
				continue
			}
		}

		// ---------------------------------------------------------------------------------------------------------
		// Finish flow creation.

		flow.InputPlugin = inputPlugin
		flow.ProcessPlugins = processPlugins
		flow.ProcessPluginsNames = processPluginsNames
		flow.OutputPlugin = outputPlugin

		flows = append(flows, flow)

		logFlowValid(flowName)
	}

	return flows
}

func runFlow(flow *core.Flow) {
	// -----------------------------------------------------------------------------------------------------------------
	var err error
	var flowLogFields log.Fields

	if flow.Lock() {
		flowLogFields = log.Fields{
			"hash": flow.FlowHash,
			"run":  flow.GetRunID(),
			"flow": flow.FlowName,
		}
		log.WithFields(flowLogFields).Info(core.LOG_FLOW_START)
		defer flow.Unlock()

	} else {
		log.WithFields(flowLogFields).Warn(core.LOG_FLOW_LOCK)
		return
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Helper functions.

	cleanFlowTemp := func() {
		if flow.FlowCleanup {
			_ = os.RemoveAll(flow.FlowTempDir)
			log.WithFields(flowLogFields).Info(core.LOG_FLOW_CLEANUP)
		}
	}

	logFlowStop := func() {
		log.WithFields(flowLogFields).Info(core.LOG_FLOW_STOP)
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Input plugin.

	log.WithFields(log.Fields{
		"hash": flow.FlowHash,
		"run":  flow.GetRunID(),
		"flow": flow.FlowName,
	}).Info(core.LOG_FLOW_RECEIVE)

	// Get data.
	inputData, err := flow.InputPlugin.Receive()
	flow.InputPlugin.FlowLog(len(inputData))

	// Process data if flow sources are expired/failed.
	// Skip flow if we have other problems.
	if err == core.ERROR_FLOW_EXPIRE {
		atomic.AddInt32(&flow.MetricExpire, 1)
		flow.InputPlugin.FlowLog(err)

	} else if err == core.ERROR_FLOW_SOURCE_FAIL {
		atomic.AddInt32(&flow.MetricError, 1)
		flow.InputPlugin.FlowLog(err)

	} else if err != nil {
		atomic.AddInt32(&flow.MetricError, 1)
		flow.InputPlugin.FlowLog(err)
		cleanFlowTemp()
		logFlowStop()
		return
	}

	// Skip flow if we don't have new data.
	if len(inputData) == 0 {
		atomic.AddInt32(&flow.MetricNoData, 1)
		flow.InputPlugin.FlowLog(core.ERROR_NO_NEW_DATA)
		logFlowStop()
		return
	} else {
		atomic.AddInt32(&flow.MetricReceive, int32(len(inputData)))
	}

	// -------------------------------------------------------------------------------------------------------------
	// Process plugins.

	processResults := make(map[int][]*core.Datum)

	if len(flow.ProcessPlugins) > 0 {
		log.WithFields(log.Fields{
			"hash":   flow.FlowHash,
			"run":    flow.GetRunID(),
			"flow":   flow.FlowName,
			"plugin": flow.ProcessPluginsNames,
		}).Info(core.LOG_FLOW_PROCESS)

		// Every "process" plugin generates its own dataset.
		// Any dataset could be excluded from sending through "output" plugin.
		for pluginID := 0; pluginID < len(flow.ProcessPlugins); pluginID++ {
			pluginResult := make([]*core.Datum, 0)

			plugin := flow.ProcessPlugins[pluginID]
			pluginRequire := flow.ProcessPlugins[pluginID].GetRequire()

			// Process data from _input plugin_ (not other process plugins) if:
			// 1. It's the first "process" plugin in the chain.
			// 2 "require" is not set for plugin.
			if pluginID == 0 || len(pluginRequire) == 0 {
				pluginResult, err = plugin.Process(inputData)

			} else {
				// Process data from _required plugins_ (not from input plugin).

				// Combine data from required plugins:
				// 1. Plugin cannot require itself (1 -> 1).
				// 2. Plugin cannot require data from higher id (1 -> 2, ordered processing).
				var combinedResult = make([]*core.Datum, 0)

				for i := 0; i < len(pluginRequire); i++ {
					requirePluginID := pluginRequire[i]

					if requirePluginID < pluginID {
						combinedResult = append(combinedResult, processResults[requirePluginID]...)
					}
				}

				pluginResult, err = plugin.Process(combinedResult)
			}

			// 1. Skip flow if we have problems with data processing.
			// 2. Save plugin results.
			if err != nil {
				plugin.FlowLog(err)
				atomic.AddInt32(&flow.MetricError, 1)
				cleanFlowTemp()
				logFlowStop()
				return

			} else {
				plugin.FlowLog(len(pluginResult))
				processResults[pluginID] = pluginResult
			}
		}
	}

	// -------------------------------------------------------------------------------------------------------------
	// Output plugin.

	if flow.OutputPlugin != nil {
		log.WithFields(log.Fields{
			"hash":   flow.FlowHash,
			"run":    flow.GetRunID(),
			"flow":   flow.FlowName,
			"plugin": flow.OutputPlugin.GetName(),
		}).Info(core.LOG_FLOW_SEND)

		// 1. Send processed data.
		// 2. Send input plugin data if there are no processing plugins.
		// 3. Show "no data" message.
		if len(flow.ProcessPlugins) > 0 && len(processResults) > 0 {
			dataIncluded := false
			dataExist := false

			for pluginID := 0; pluginID < len(processResults); pluginID++ {
				pluginData := processResults[pluginID]

				// Send only needed data (param "include" is "true").
				if flow.ProcessPlugins[pluginID].GetInclude() {
					dataIncluded = true

					// Send only not empty data (some plugins can produce zero data).
					if len(pluginData) > 0 {
						dataExist = true

						err = flow.OutputPlugin.Send(pluginData)

						// Skip flow if there are problems with sending.
						if err != nil {
							atomic.AddInt32(&flow.MetricError, 1)
							flow.OutputPlugin.FlowLog(err)
							cleanFlowTemp()
							logFlowStop()
							return

						} else {
							atomic.AddInt32(&flow.MetricSend, int32(len(pluginData)))
							flow.OutputPlugin.FlowLog(fmt.Sprintf("process plugin id: %d, send data: %d",
								pluginID, len(pluginData)))
						}
					}
				}
			}

			if !dataIncluded {
				flow.OutputPlugin.FlowLog(core.LOG_FLOW_SEND_NO_DATA_INCLUDED)
			}

			if !dataExist {
				flow.OutputPlugin.FlowLog(core.LOG_FLOW_SEND_NO_DATA)
			}

		} else if len(flow.ProcessPlugins) == 0 && len(inputData) > 0 {
			err = flow.OutputPlugin.Send(inputData)

			// Skip flow if there are problems with sending.
			if err != nil {
				atomic.AddInt32(&flow.MetricError, 1)
				flow.OutputPlugin.FlowLog(err)
				cleanFlowTemp()
				logFlowStop()
				return

			} else {
				atomic.AddInt32(&flow.MetricSend, int32(len(inputData)))
				flow.OutputPlugin.FlowLog(len(inputData))
			}

		} else {
			flow.OutputPlugin.FlowLog(core.LOG_FLOW_SEND_NO_DATA)
		}
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Cleanup at the end.

	cleanFlowTemp()
	logFlowStop()

	// -----------------------------------------------------------------------------------------------------------------
}
