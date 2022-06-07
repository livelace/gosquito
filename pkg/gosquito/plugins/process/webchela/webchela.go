package webchelaProcess

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	pb "github.com/livelace/gosquito/pkg/gosquito/plugins/process/webchela/protobuf"
	log "github.com/livelace/logrus"
	"google.golang.org/grpc"
	"io"
	"path/filepath"
	"reflect"
	"time"
)

const (
	PLUGIN_NAME = "webchela"

	DEFAULT_BATCH_RETRY            = 0 // no retries.
	DEFAULT_BATCH_SIZE             = 100
	DEFAULT_BROWSER_GEOMETRY       = "1024x768"
	DEFAULT_BROWSER_INSTANCE       = 1
	DEFAULT_BROWSER_INSTANCE_TAB   = 5
	DEFAULT_BROWSER_PAGE_SIZE      = "10M"
	DEFAULT_BROWSER_PAGE_TIMEOUT   = 20
	DEFAULT_BROWSER_PROXY          = ""
	DEFAULT_BROWSER_SCRIPT_TIMEOUT = 20
	DEFAULT_BROWSER_TYPE           = "firefox"
	DEFAULT_BUFFER_LENGHT          = 1000
	DEFAULT_CHUNK_SIZE             = "3M"
	DEFAULT_CPU_LOAD               = 25
	DEFAULT_MEM_FREE               = "1G"
	DEFAULT_OUTPUT_FILENAME        = ""
	DEFAULT_PAGE_BODY_FILENAME     = "page_body.html"
	DEFAULT_PAGE_TITLE_FILENAME    = "page_title.txt"
	DEFAULT_PAGE_URL_FILENAME      = "page_url.txt"
	DEFAULT_SERVER_CONNECT_TIMEOUT = 3
	DEFAULT_SERVER_REQUEST_TIMEOUT = 10
	DEFAULT_TIMEOUT                = 300
)

type BatchTask struct {
	ID     int
	Server string
	Status string
	Input  []string
	Output []string
}

func getServer(p *Plugin, batchId int, serverFailStat *map[string]int) string {
	serverLoad := make(map[string]int32, 0)
	connectTimeout := time.Duration(p.OptionServerTimeout) * time.Second
	requestTimeout := time.Duration(p.OptionRequestTimeout) * time.Second

	// Gather servers load scores.
	for _, server := range p.OptionServer {
		// Try to connect to server.
		dialCtx, dialCancel := context.WithTimeout(context.Background(), connectTimeout)
		defer dialCancel()

		conn, err := grpc.DialContext(dialCtx, server, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			core.LogProcessPlugin(p.LogFields,
				fmt.Errorf("batch: %d, server is not available: %s, %s", batchId, server, err))
			continue
		}

		// Try to get server load.
		client := pb.NewServerClient(conn)

		funcCtx, funcCancel := context.WithTimeout(context.Background(), requestTimeout)
		defer funcCancel()

		load, err := client.GetLoad(funcCtx, &pb.Empty{})
		if err != nil {
			core.LogProcessPlugin(p.LogFields, fmt.Errorf("batch: %d, cannot get server load: %s, %s",
				batchId, server, err))
			continue
		}

		// Check server metrics.
		if load.CpuLoad == 0 || load.MemFree == 0 || load.Score == 0 {
			core.LogProcessPlugin(p.LogFields, fmt.Errorf(
				"batch: %d, server return invalid metrics: %s, cpu_load: %d%%, mem_free: %d, score: %d",
				batchId, server, load.CpuLoad, load.MemFree, load.Score))
			continue
		}

		if load.CpuLoad <= p.OptionCpuLoad && load.MemFree >= p.OptionMemFree {
			serverLoad[server] = load.Score
		} else {
			core.LogProcessPlugin(p.LogFields, fmt.Sprintf(
				"batch: %d, server is not ready: %s, cpu_load: %d%%, mem_free: %d, score: %d",
				batchId, server, load.CpuLoad, load.MemFree, load.Score))
		}

		_ = conn.Close()
	}

	// Choose the best server.
	bestServer := ""
	bestScore := int32(0)
	bestFail := 1000

	// Firs, find lowest fail rate.
	for server := range serverLoad {
		if f := (*serverFailStat)[server]; f < bestFail {
			bestFail = f
		}
	}

	// Second, find best score across the best fail rates.
	for server, score := range serverLoad {
		if f := (*serverFailStat)[server]; f <= bestFail && score > bestScore {
			bestServer = server
			bestScore = score
			bestFail = f
		}
	}

	if bestServer != "" {
		core.LogProcessPlugin(p.LogFields, fmt.Sprintf("batch: %d, best server: %s, failed batch: %d, score: %d",
			batchId, bestServer, bestFail, bestScore))
	} else {
		core.LogProcessPlugin(p.LogFields, fmt.Errorf("batch: %d, servers not ready", batchId))
	}

	return bestServer
}

func processBatch(p *Plugin, batchTask *BatchTask) {
	// Quick fail.
	logAndSetFail := func(message string) {
		batchTask.Status = "fail"
		p.OptionBatchChannel <- batchTask
		core.LogProcessPlugin(p.LogFields, fmt.Errorf("%s", message))
	}

	// Connect to server.
	conn, err := grpc.Dial(batchTask.Server, grpc.WithInsecure(), grpc.WithBlock())
	if conn == nil || err != nil {
		logAndSetFail(fmt.Sprintf("batch: %d, server is not available: %s",
			batchTask.ID, batchTask.Server))
		return
	}
	defer conn.Close()

	// Form and run webchela task.
	webchelaTaskBrowser := pb.Task_Browser{
		Type:          p.OptionBrowserType,
		Argument:      p.OptionBrowserArgument,
		Extension:     p.OptionBrowserExtension,
		Geometry:      p.OptionBrowserGeometry,
		Instance:      int32(p.OptionBrowserInstance),
		InstanceTab:   int32(p.OptionBrowserInstanceTab),
		PageSize:      p.OptionBrowserPageSize,
		PageTimeout:   int32(p.OptionBrowserPageTimeout),
		Proxy:         p.OptionBrowserProxy,
		ScriptTimeout: int32(p.OptionBrowserScriptTimeout),
	}

	// Set client id for identification.
	var clientId string
	if p.OptionClientId == "" {
		clientId = p.Flow.FlowName
	} else {
		clientId = p.OptionClientId
	}

	webchelaTask := pb.Task{
		ClientId:  clientId,
		Urls:      batchTask.Input,
		Scripts:   p.OptionScript,
		ChunkSize: p.OptionChunkSize,
		CpuLoad:   p.OptionCpuLoad,
		MemFree:   p.OptionMemFree,
		Timeout:   int32(p.OptionTimeout),
		Browser:   &webchelaTaskBrowser,
	}

	client := pb.NewServerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.OptionTimeout)*time.Second)
	defer cancel()

	stream, err := client.RunTask(ctx, &webchelaTask)
	if err != nil {
		logAndSetFail(fmt.Sprintf("batch: %d, cannot run task: %s", batchTask.ID, err))
		return
	}

	// Assemble chunks into []Result.
	buffer := make([]byte, 0)
	results := make([]*pb.Result, 0)

	for {
		// Consume messages from stream.
		message, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			logAndSetFail(fmt.Sprintf("batch: %d, stream reading error: %s", batchTask.ID, err))
			return
		}

		// Join chunks bytes till "End" flag.
		// Right after create Result and append to slice.
		buffer = append(buffer, message.Chunk...)

		if message.End {
			result := pb.Result{}
			err = proto.Unmarshal(buffer, &result)
			if err != nil {
				logAndSetFail(fmt.Sprintf("batch: %d, cannot unmarshal result message: %s", batchTask.ID, err))
				return
			}

			results = append(results, &result)
			buffer = make([]byte, 0)
		}
	}

	if len(results) != len(batchTask.Input) {
		logAndSetFail(fmt.Sprintf("batch: %d, received data not equal sent data: %d != %d",
			batchTask.ID, len(results), len(batchTask.Input)))
		return
	}

	if err := saveData(p, batchTask, results); err == nil {
		batchTask.Status = "success"
		p.OptionBatchChannel <- batchTask
	} else {
		batchTask.Status = "fail"
		logAndSetFail(fmt.Sprintf("batch: %d, cannot save results: %s", batchTask.ID, err))
		return
	}
}

func saveData(p *Plugin, b *BatchTask, results []*pb.Result) error {
	for _, result := range results {
		// Create output directory in plugin temporary directory.
		outputDir := filepath.Join(p.Flow.FlowTempDir, p.PluginType, p.PluginName, result.UUID)
		err := core.CreateDirIfNotExist(outputDir)
		if err != nil {
			return err
		}

		// Write page url.
		err = core.WriteStringToFile(outputDir, DEFAULT_PAGE_URL_FILENAME, result.PageUrl)
		if err != nil {
			return err
		}

		// Write page title.
		err = core.WriteStringToFile(outputDir, DEFAULT_PAGE_TITLE_FILENAME, result.PageTitle)
		if err != nil {
			return err
		}

		// Write page body.
		err = core.WriteStringToFile(outputDir, DEFAULT_PAGE_BODY_FILENAME, result.PageBody)
		if err != nil {
			return err
		}

		// Write scripts output.
		for index, output := range result.ScriptOutput {
			err = core.WriteStringToFile(outputDir, fmt.Sprintf("webchela_script%d_output.txt", index), output)
			if err != nil {
				return err
			}
		}

		b.Output = append(b.Output, filepath.Join(outputDir, p.OptionOutputFilename))

		core.LogProcessPlugin(p.LogFields,
			fmt.Sprintf("batch: %d, save received data into: %s", b.ID, outputDir))
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

	OptionBatchChannel         chan *BatchTask
	OptionBatchRetry           int
	OptionBatchSize            int
	OptionBrowserArgument      []string
	OptionBrowserExtension     []string
	OptionBrowserGeometry      string
	OptionBrowserInstance      int
	OptionBrowserInstanceTab   int
	OptionBrowserPageSize      int64
	OptionBrowserPageTimeout   int
	OptionBrowserProxy         string
	OptionBrowserScriptTimeout int
	OptionBrowserType          string
	OptionChunkSize            int64
	OptionClientId             string
	OptionCpuLoad              int32
	OptionInclude              bool
	OptionInput                []string
	OptionMemFree              int64
	OptionOutput               []string
	OptionOutputFilename       string
	OptionRequestTimeout       int
	OptionRequire              []int
	OptionScript               []string
	OptionServer               []string
	OptionServerTimeout        int
	OptionTimeout              int
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

	// Extract URLs from data items into single flat slice.
	// This is needed for batch slicing (process URLs in sized blocks/batches).
	// Example: every "Datum" has filled ["data.array0", "data.array1"] with URLs,
	// we extract all URLs from all "Datums" into one flat slice: [url0, url1 ... urlN].
	// To recognize boundaries between Datums in slice we save "metadata".
	// Example metadata: [0] = [20, 300].
	// that means: [Datum 0] = [data.array0 = 20 urls, data.array1 = 300 urls].
	allURL := make([]string, 0)
	itemURLMeta := make(map[int][]int, 0)

	for itemIndex, itemData := range data {
		itemURLMeta[itemIndex] = make([]int, len(p.OptionInput))

		for inputIndex, inputField := range p.OptionInput {
			ri, _ := core.ReflectDatumField(itemData, inputField)

			// 1. Set amount of URLs for specific data item and input.
			// 2. Append URLs to flat slice.
			switch ri.Kind() {
			case reflect.String:
				itemURLMeta[itemIndex][inputIndex] = 1
				allURL = append(allURL, ri.String())

			case reflect.Slice:
				itemURLMeta[itemIndex][inputIndex] = ri.Len()
				for i := 0; i < ri.Len(); i++ {
					allURL = append(allURL, ri.Index(i).String())
				}
			}
		}
	}

	// Split input data into batches.
	batches := make([][]string, 0)

	for i := 0; i < len(allURL); i += p.OptionBatchSize {
		end := i + p.OptionBatchSize
		if end > len(allURL) {
			end = len(allURL)
		}
		batches = append(batches, allURL[i:end])
	}

	// Send batches to webchela servers concurrently.
	batchStatus := make(map[int]string, len(batches))
	batchResult := make(map[int]*BatchTask, len(batches))
	batchRetryStat := make(map[int]int, len(batches))
	serverFailStat := make(map[string]int, len(p.OptionServer))

	timeoutCounter := 0

	for {
		if timeoutCounter > p.OptionTimeout {
			core.LogProcessPlugin(p.LogFields,
				fmt.Errorf("main loop: timeout reached: total batches: %d, timeout: %d",
					len(batches), p.OptionTimeout))
			return temp, nil
		}

		completed := true

		// Run batches one-by-one on suitable servers (cpu/mem load is fine).
		for batchId, batchData := range batches {
			switch batchStatus[batchId] {
			case "":
				if server := getServer(p, batchId, &serverFailStat); server != "" {
					go processBatch(p, &BatchTask{
						ID:     batchId,
						Server: server,
						Input:  batchData,
					})
					batchStatus[batchId] = "progress"
				}
				completed = false
			case "fail":
				if batchRetryStat[batchId] < p.OptionBatchRetry {
					batchRetryStat[batchId] += 1
					batchStatus[batchId] = ""
					completed = false
				}
			case "progress":
				completed = false
			case "success":
				continue
			}
		}

		// Get completed batches and update statuses.
		// Update stat for servers where fails appeared somehow.
		for i := 0; i < len(p.OptionBatchChannel); i++ {
			b := <-p.OptionBatchChannel

			batchStatus[b.ID] = b.Status
			batchResult[b.ID] = b

			if b.Status == "fail" {
				serverFailStat[b.Server] += 1
			}
		}

		// Iterate until all batches will have "success" or "fail" status.
		if completed {
			break
		} else {
			timeoutCounter += 1
			time.Sleep(1 * time.Second)
		}
	}

	// Reassemble batches into output.
	outputData := make([]string, 0)

	// put batches results into flat output slice.
	for i := 0; i < len(batches); i++ {
		outputData = append(outputData, batchResult[i].Output...)
	}

	// Amount of input and output data must be equal, even if some pages were processed
	// with errors (timeouts, not known DNS names etc.).
	if len(allURL) != len(outputData) {
		core.LogProcessPlugin(p.LogFields, fmt.Errorf("main loop: received data not equal sent data: %d != %d",
			len(outputData), len(allURL)))
		return temp, nil
	}

	// Fill corresponding Datum with output data.
	outputOffset := 0

	for itemIndex := 0; itemIndex < len(itemURLMeta); itemIndex++ {
		grabbed := false

		itemMeta := itemURLMeta[itemIndex]

		for index, value := range itemMeta {
			if value > 0 {
				grabbed = true
			}

			ro, _ := core.ReflectDatumField(data[itemIndex], p.OptionOutput[index])

			switch ro.Kind() {
			case reflect.String:
				for offset := outputOffset; offset < outputOffset+value; offset++ {
					ro.SetString(outputData[offset])
				}
			case reflect.Slice:
				for offset := outputOffset; offset < outputOffset+value; offset++ {
					ro.Set(reflect.Append(ro, reflect.ValueOf(outputData[offset])))
				}
			}

			outputOffset += value
		}

		if grabbed {
			temp = append(temp, data[itemIndex])
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

		"batch_retry":            -1,
		"batch_size":             -1,
		"browser_type":           -1,
		"browser_argument":       -1,
		"browser_extension":      -1,
		"browser_geometry":       -1,
		"browser_instance":       -1,
		"browser_instance_tab":   -1,
		"browser_page_size":      -1,
		"browser_page_timeout":   -1,
		"browser_proxy":          -1,
		"browser_script_timeout": -1,
		"chunk_size":             -1,
		"client_id":              -1,
		"cpu_load":               -1,
		"input":                  1,
		"mem_free":               -1,
		"output":                 -1,
		"output_filename":        -1,
		"request_timeout":        -1,
		"script":                 -1,
		"server":                 1,
		"server_timeout":         -1,
		"timeout":                -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin settings or set defaults.

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// -----------------------------------------------------------------------------------------------------------------

	// batch_retry.
	setBatchRetry := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["batch_retry"] = 0
			plugin.OptionBatchRetry = v
		}
	}
	setBatchRetry(DEFAULT_BATCH_RETRY)
	setBatchRetry(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.batch_retry", template)))
	setBatchRetry((*pluginConfig.PluginParams)["batch_retry"])
	core.ShowPluginParam(plugin.LogFields, "batch_retry", plugin.OptionBatchRetry)

	// batch_size.
	setBatchSize := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["batch_size"] = 0
			plugin.OptionBatchSize = v
		}
	}
	setBatchSize(DEFAULT_BATCH_SIZE)
	setBatchSize(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.batch_size", template)))
	setBatchSize((*pluginConfig.PluginParams)["batch_size"])
	core.ShowPluginParam(plugin.LogFields, "batch_size", plugin.OptionBatchSize)

	// browser_type.
	setBrowserType := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["browser_type"] = 0
			plugin.OptionBrowserType = v
		}
	}
	setBrowserType(DEFAULT_BROWSER_TYPE)
	setBrowserType(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.browser_type", template)))
	setBrowserType((*pluginConfig.PluginParams)["browser_type"])
	core.ShowPluginParam(plugin.LogFields, "browser_type", plugin.OptionBrowserType)

	// browser_argument.
	setBrowserArgument := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["browser_argument"] = 0
			plugin.OptionBrowserArgument = v
		}
	}
	setBrowserArgument(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.browser_argument", template)))
	setBrowserArgument((*pluginConfig.PluginParams)["browser_argument"])
	core.ShowPluginParam(plugin.LogFields, "browser_argument", plugin.OptionBrowserArgument)

	// browser_extension.
	setBrowserExtensions := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["browser_extension"] = 0
			plugin.OptionBrowserExtension = v
		}
	}
	setBrowserExtensions(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.browser_extension", template)))
	setBrowserExtensions((*pluginConfig.PluginParams)["browser_extension"])
	core.ShowPluginParam(plugin.LogFields, "browser_extension", plugin.OptionBrowserExtension)

	// browser_geometry.
	setBrowserGeometry := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["browser_geometry"] = 0
			plugin.OptionBrowserGeometry = v
		}
	}
	setBrowserGeometry(DEFAULT_BROWSER_GEOMETRY)
	setBrowserGeometry(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.browser_geometry", template)))
	setBrowserGeometry((*pluginConfig.PluginParams)["browser_geometry"])
	core.ShowPluginParam(plugin.LogFields, "browser_geometry", plugin.OptionBrowserGeometry)

	// browser_instance.
	setBrowserInstance := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["browser_instance"] = 0
			plugin.OptionBrowserInstance = v
		}
	}
	setBrowserInstance(DEFAULT_BROWSER_INSTANCE)
	setBrowserInstance(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.browser_instance", template)))
	setBrowserInstance((*pluginConfig.PluginParams)["browser_instance"])
	core.ShowPluginParam(plugin.LogFields, "browser_instance", plugin.OptionBrowserInstance)

	// browser_instance_tab.
	setBrowserInstanceTab := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["browser_instance_tab"] = 0
			plugin.OptionBrowserInstanceTab = v
		}
	}
	setBrowserInstanceTab(DEFAULT_BROWSER_INSTANCE_TAB)
	setBrowserInstanceTab(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.browser_instance_tab", template)))
	setBrowserInstanceTab((*pluginConfig.PluginParams)["browser_instance_tab"])
	core.ShowPluginParam(plugin.LogFields, "browser_instance_tab", plugin.OptionBrowserInstanceTab)

	// browser_page_size.
	setBrowserPageSize := func(p interface{}) {
		if v, b := core.IsSize(p); b {
			availableParams["browser_page_size"] = 0
			plugin.OptionBrowserPageSize = v
		}
	}
	setBrowserPageSize(DEFAULT_BROWSER_PAGE_SIZE)
	setBrowserPageSize(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.browser_page_size", template)))
	setBrowserPageSize((*pluginConfig.PluginParams)["browser_page_size"])
	core.ShowPluginParam(plugin.LogFields, "browser_page_size", plugin.OptionBrowserPageSize)

	// browser_page_timeout.
	setBrowserPageTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["browser_page_timeout"] = 0
			plugin.OptionBrowserPageTimeout = v
		}
	}
	setBrowserPageTimeout(DEFAULT_BROWSER_PAGE_TIMEOUT)
	setBrowserPageTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.browser_page_timeout", template)))
	setBrowserPageTimeout((*pluginConfig.PluginParams)["browser_page_timeout"])
	core.ShowPluginParam(plugin.LogFields, "browser_page_timeout", plugin.OptionBrowserPageTimeout)

	// browser_proxy.
	setBrowserProxy := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["browser_proxy"] = 0
			plugin.OptionBrowserProxy = v
		}
	}
	setBrowserProxy(DEFAULT_BROWSER_PROXY)
	setBrowserProxy(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.browser_proxy", template)))
	setBrowserProxy((*pluginConfig.PluginParams)["browser_proxy"])
	core.ShowPluginParam(plugin.LogFields, "browser_proxy", plugin.OptionBrowserProxy)

	// browser_script_timeout.
	setBrowserScriptTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["browser_script_timeout"] = 0
			plugin.OptionBrowserScriptTimeout = v
		}
	}
	setBrowserScriptTimeout(DEFAULT_BROWSER_SCRIPT_TIMEOUT)
	setBrowserScriptTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.browser_script_timeout", template)))
	setBrowserScriptTimeout((*pluginConfig.PluginParams)["browser_script_timeout"])
	core.ShowPluginParam(plugin.LogFields, "browser_script_timeout", plugin.OptionBrowserScriptTimeout)

	// chunk_size.
	setChunkSize := func(p interface{}) {
		if v, b := core.IsSize(p); b {
			availableParams["chunk_size"] = 0
			plugin.OptionChunkSize = v
		}
	}
	setChunkSize(DEFAULT_CHUNK_SIZE)
	setChunkSize(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.chunk_size", template)))
	setChunkSize((*pluginConfig.PluginParams)["chunk_size"])
	core.ShowPluginParam(plugin.LogFields, "chunk_size", plugin.OptionChunkSize)

	// client_id.
	setClientId := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["client_id"] = 0
			plugin.OptionClientId = v
		}
	}
	setClientId(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.client_id", template)))
	setClientId((*pluginConfig.PluginParams)["client_id"])
	core.ShowPluginParam(plugin.LogFields, "client_id", plugin.OptionClientId)

	// cpu_load.
	setBrowserCpuLoad := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["cpu_load"] = 0
			plugin.OptionCpuLoad = int32(v)
		}
	}
	setBrowserCpuLoad(DEFAULT_CPU_LOAD)
	setBrowserCpuLoad(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.cpu_load", template)))
	setBrowserCpuLoad((*pluginConfig.PluginParams)["cpu_load"])
	core.ShowPluginParam(plugin.LogFields, "cpu_load", plugin.OptionCpuLoad)

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

	// mem_free.
	setBrowserMemFree := func(p interface{}) {
		if v, b := core.IsSize(p); b {
			availableParams["mem_free"] = 0
			plugin.OptionMemFree = v
		}
	}
	setBrowserMemFree(DEFAULT_MEM_FREE)
	setBrowserMemFree(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.mem_free", template)))
	setBrowserMemFree((*pluginConfig.PluginParams)["mem_free"])
	core.ShowPluginParam(plugin.LogFields, "mem_free", plugin.OptionMemFree)

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

	// output_filename.
	setOutputFilename := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["output_filename"] = 0
			plugin.OptionOutputFilename = v
		}
	}
	setOutputFilename(DEFAULT_OUTPUT_FILENAME)
	setOutputFilename(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.output_filename", template)))
	setOutputFilename((*pluginConfig.PluginParams)["output_filename"])
	core.ShowPluginParam(plugin.LogFields, "output_filename", plugin.OptionOutputFilename)

	// request_timeout.
	setRequestTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["request_timeout"] = 0
			plugin.OptionRequestTimeout = v
		}
	}
	setRequestTimeout(DEFAULT_SERVER_REQUEST_TIMEOUT)
	setRequestTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.request_timeout", template)))
	setRequestTimeout((*pluginConfig.PluginParams)["request_timeout"])
	core.ShowPluginParam(plugin.LogFields, "request_timeout", plugin.OptionRequestTimeout)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.OptionRequire = v

		}
	}
	setRequire((*pluginConfig.PluginParams)["require"])
	core.ShowPluginParam(plugin.LogFields, "require", plugin.OptionRequire)

	// script.
	setScript := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["script"] = 0
			plugin.OptionScript = core.ExtractScripts(pluginConfig.AppConfig, v)
		}
	}
	setScript(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.script", template)))
	setScript((*pluginConfig.PluginParams)["script"])
	core.ShowPluginParam(plugin.LogFields, "script", plugin.OptionScript)

	// server.
	setServer := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["server"] = 0
			plugin.OptionServer = v
		}
	}
	setServer(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.server", template)))
	setServer((*pluginConfig.PluginParams)["server"])
	core.ShowPluginParam(plugin.LogFields, "server", plugin.OptionServer)

	// server_timeout.
	setServerTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["server_timeout"] = 0
			plugin.OptionServerTimeout = v
		}
	}
	setServerTimeout(DEFAULT_SERVER_CONNECT_TIMEOUT)
	setServerTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.server_timeout", template)))
	setServerTimeout((*pluginConfig.PluginParams)["server_timeout"])
	core.ShowPluginParam(plugin.LogFields, "server_timeout", plugin.OptionServerTimeout)

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.OptionTimeout = v
		}
	}
	setTimeout(DEFAULT_TIMEOUT)
	setTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.PluginParams)["timeout"])
	core.ShowPluginParam(plugin.LogFields, "timeout", plugin.OptionTimeout)

	// -----------------------------------------------------------------------------------------------------------------

	plugin.OptionBatchChannel = make(chan *BatchTask, DEFAULT_BUFFER_LENGHT)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if len(plugin.OptionInput) != len(plugin.OptionOutput) {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput)
	}

	if err := core.IsDatumFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
