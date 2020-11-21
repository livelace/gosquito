package webchela

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
	DEFAULT_BATCH_RETRY            = 0 // no retries.
	DEFAULT_BATCH_SIZE             = 100
	DEFAULT_BROWSER_GEOMETRY       = "1024x768"
	DEFAULT_BROWSER_INSTANCE       = 1
	DEFAULT_BROWSER_INSTANCE_TAB   = 5
	DEFAULT_BROWSER_PAGE_SIZE      = "10M"
	DEFAULT_BROWSER_PAGE_TIMEOUT   = 20
	DEFAULT_BROWSER_SCRIPT_TIMEOUT = 20
	DEFAULT_BROWSER_TYPE           = "firefox"
	DEFAULT_BUFFER_LENGHT          = 1000
	DEFAULT_CHUNK_SIZE             = "3M"
	DEFAULT_CPU_LOAD               = 25
	DEFAULT_MEM_FREE               = "1G"
	DEFAULT_PAGE_BODY_FILENAME     = "page_body.html"
	DEFAULT_PAGE_TITLE_FILENAME    = "page_title.txt"
	DEFAULT_PAGE_URL_FILENAME      = "page_url.txt"
	DEFAULT_SERVER_CONNECT_TIMEOUT = 3
	DEFAULT_SERVER_REQUEST_TIMEOUT = 10
	DEFAULT_TIMEOUT                = 300
)

func getServer(p *Plugin, batchId int, serverFailStat *map[string]int) string {
	serverLoad := make(map[string]int32, 0)

	// Debug logging.
	logDebug := func(field, message string) {
		log.WithFields(log.Fields{
			"hash":   p.Hash,
			"flow":   p.Flow,
			"file":   p.File,
			"plugin": p.Name,
			"type":   p.Type,
			"id":     p.ID,
			field:    message,
		}).Debug(core.LOG_PLUGIN_DATA)
	}

	connectTimeout := time.Duration(p.ServerTimeout) * time.Second
	requestTimeout := time.Duration(p.RequestTimeout) * time.Second

	// Gather servers load scores.
	for _, server := range p.Server {
		// Try to connect to server.
		dialCtx, dialCancel := context.WithTimeout(context.Background(), connectTimeout)
		defer dialCancel()

		conn, err := grpc.DialContext(dialCtx, server, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			logDebug("error", fmt.Sprintf("batch: %d, server is not available: %s, %s",
				batchId, server, err))
			continue
		}

		// Try to get server load.
		client := pb.NewServerClient(conn)

		funcCtx, funcCancel := context.WithTimeout(context.Background(), requestTimeout)
		defer funcCancel()

		load, err := client.GetLoad(funcCtx, &pb.Empty{})
		if err != nil {
			logDebug("error", fmt.Sprintf("batch: %d, cannot get server load: %s, %s",
				batchId, server, err))
			continue
		}

		// Check server metrics.
		if load.CpuLoad == 0 || load.MemFree == 0 || load.Score == 0 {
			logDebug("error", fmt.Sprintf(
				"batch: %d, server return invalid metrics: %s, cpu_load: %d%%, mem_free: %d, score: %d",
				batchId, server, load.CpuLoad, load.MemFree, load.Score))
			continue
		}

		if load.CpuLoad <= p.CpuLoad && load.MemFree >= p.MemFree {
			serverLoad[server] = load.Score
		} else {
			logDebug("data", fmt.Sprintf(
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
		logDebug("data", fmt.Sprintf("batch: %d, best server: %s, batch failed: %d, score: %d",
			batchId, bestServer, bestFail, bestScore))
	} else {
		logDebug("error", fmt.Sprintf("batch: %d, servers not ready!", batchId))
	}

	return bestServer
}

func processBatch(p *Plugin, batchTask *BatchTask) {
	// Quick fail.
	logAndSetFail := func(msg string) {
		batchTask.Status = "fail"
		p.BatchChannel <- batchTask

		log.WithFields(log.Fields{
			"hash":   p.Hash,
			"flow":   p.Flow,
			"file":   p.File,
			"plugin": p.Name,
			"type":   p.Type,
			"id":     p.ID,
			"error":  msg,
		}).Error(core.LOG_PLUGIN_DATA)
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
		Type:          p.BrowserType,
		Argument:      p.BrowserArgument,
		Extension:     p.BrowserExtension,
		Geometry:      p.BrowserGeometry,
		Instance:      int32(p.BrowserInstance),
		InstanceTab:   int32(p.BrowserInstanceTab),
		PageSize:      p.BrowserPageSize,
		PageTimeout:   int32(p.BrowserPageTimeout),
		ScriptTimeout: int32(p.BrowserScriptTimeout),
	}

	// Set client id for identification.
	var clientId string
	if p.ClientId == "" {
		clientId = p.Flow
	} else {
		clientId = p.ClientId
	}

	webchelaTask := pb.Task{
		ClientId:  clientId,
		Urls:      batchTask.Input,
		Scripts:   p.Script,
		ChunkSize: p.ChunkSize,
		CpuLoad:   p.CpuLoad,
		MemFree:   p.MemFree,
		Timeout:   int32(p.Timeout),
		Browser:   &webchelaTaskBrowser,
	}

	client := pb.NewServerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.Timeout)*time.Second)
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
		p.BatchChannel <- batchTask
	} else {
		batchTask.Status = "fail"
		logAndSetFail(fmt.Sprintf("batch: %d, cannot save results: %s", batchTask.ID, err))
		return
	}
}

func saveData(p *Plugin, b *BatchTask, results []*pb.Result) error {
	for _, result := range results {
		// Create output directory in plugin temporary directory.
		outputDir := filepath.Join(p.TempDir, p.Flow, p.Type, p.Name, result.UUID)
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

		log.WithFields(log.Fields{
			"hash":   p.Hash,
			"flow":   p.Flow,
			"file":   p.File,
			"plugin": p.Name,
			"type":   p.Type,
			"id":     p.ID,
			"data":   fmt.Sprintf("batch: %d, save received data into: %s", b.ID, outputDir),
		}).Debug(core.LOG_PLUGIN_DATA)

		b.Output = append(b.Output, outputDir)
	}

	return nil
}

type BatchTask struct {
	ID     int
	Server string
	Status string
	Input  []string
	Output []string
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

	BatchChannel         chan *BatchTask
	BatchRetry           int
	BatchSize            int
	BrowserType          string
	BrowserArgument      []string
	BrowserExtension     []string
	BrowserGeometry      string
	BrowserInstance      int
	BrowserInstanceTab   int
	BrowserPageSize      int64
	BrowserPageTimeout   int
	BrowserScriptTimeout int
	ChunkSize            int64
	ClientId             string
	CpuLoad              int32
	Input                []string
	MemFree              int64
	Output               []string
	RequestTimeout       int
	Script               []string
	Server               []string
	ServerTimeout        int
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	logError := func(msg string) {
		log.WithFields(log.Fields{
			"hash":   p.Hash,
			"flow":   p.Flow,
			"file":   p.File,
			"plugin": p.Name,
			"type":   p.Type,
			"id":     p.ID,
			"error":  msg,
		}).Error(core.LOG_PLUGIN_DATA)
	}

	// Gather input data from all data items into one flat slice.
	// This needed for batch slicing (process URLs in sized blocks/batches).
	// Example: every "DataItem" has filled ["data.array0", "data.array1"] with URLs,
	// we extract all URLs from all "DataItems" into one flat slice: [url0, url1 ... urlN].
	// To recognize boundaries between DataItems in slice we save "metadata".
	// Example metadata: [0] = [20, 300].
	// that means: [DataItem 0] = [data.array0 = 20 urls, data.array1 = 300 urls].
	inputData := make([]string, 0)
	inputMeta := make(map[int][]int, 0)

	for itemIndex, itemData := range data {
		inputMeta[itemIndex] = make([]int, len(p.Input))

		for inputIndex, inputField := range p.Input {
			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(itemData, inputField)

			// fill metadata.
			inputMeta[itemIndex][inputIndex] = ri.Len()

			// append urls to flat slice.
			for i := 0; i < ri.Len(); i++ {
				inputData = append(inputData, ri.Index(i).String())
			}
		}
	}

	// Split input data into batches.
	batches := make([][]string, 0)

	for i := 0; i < len(inputData); i += p.BatchSize {
		end := i + p.BatchSize
		if end > len(inputData) {
			end = len(inputData)
		}
		batches = append(batches, inputData[i:end])
	}

	// Send batches to webchela servers concurrently.
	batchStatus := make(map[int]string, len(batches))
	batchResult := make(map[int]*BatchTask, len(batches))
	batchRetryStat := make(map[int]int, len(batches))
	serverFailStat := make(map[string]int, len(p.Server))

	timeoutCounter := 0

	for {
		if timeoutCounter > p.Timeout {
			logError(fmt.Sprintf("main loop: timeout reached: total batches: %d, timeout: %d",
				len(batches), p.Timeout))
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
				if batchRetryStat[batchId] < p.BatchRetry {
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
		for i := 0; i < len(p.BatchChannel); i++ {
			b := <-p.BatchChannel

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

	// amount of input and output data must be the same,
	// even if some pages were opened with errors (timeouts, not known DNS names etc.).
	if len(inputData) != len(outputData) {
		logError(fmt.Sprintf("main loop: received data not equal sent data: %d != %d",
			len(outputData), len(inputData)))
		return temp, nil
	}

	// fill corresponding DataItem with output data.
	outputOffset := 0

	for itemIndex := 0; itemIndex < len(inputMeta); itemIndex++ {
		grabbed := false

		itemMeta := inputMeta[itemIndex]

		for index, value := range itemMeta {
			if value > 0 {
				grabbed = true
			}

			ro, _ := core.ReflectDataField(data[itemIndex], p.Output[index])

			for offset := outputOffset; offset < outputOffset+value; offset++ {
				ro.Set(reflect.Append(ro, reflect.ValueOf(outputData[offset])))
			}

			outputOffset += value
		}

		if grabbed {
			temp = append(temp, data[itemIndex])
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
		Name:    "webchela",
		TempDir: pluginConfig.Config.GetString(core.VIPER_DEFAULT_PLUGIN_TEMP),
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
		"browser_script_timeout": -1,
		"chunk_size":             -1,
		"client_id":              -1,
		"cpu_load":               -1,
		"input":                  1,
		"mem_free":               -1,
		"output":                 -1,
		"request_timeout":        -1,
		"script":                 -1,
		"server":                 1,
		"server_timeout":         -1,
		"timeout":                -1,
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

	template, _ := core.IsString((*pluginConfig.Params)["template"])

	// batch_retry.
	setBatchRetry := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["batch_retry"] = 0
			plugin.BatchRetry = v
		}
	}
	setBatchRetry(DEFAULT_BATCH_RETRY)
	setBatchRetry(pluginConfig.Config.GetInt(fmt.Sprintf("%s.batch_retry", template)))
	setBatchRetry((*pluginConfig.Params)["batch_retry"])
	showParam("batch_retry", plugin.BatchRetry)

	// batch_size.
	setBatchSize := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["batch_size"] = 0
			plugin.BatchSize = v
		}
	}
	setBatchSize(DEFAULT_BATCH_SIZE)
	setBatchSize(pluginConfig.Config.GetInt(fmt.Sprintf("%s.batch_size", template)))
	setBatchSize((*pluginConfig.Params)["batch_size"])
	showParam("batch_size", plugin.BatchSize)

	// browser_type.
	setBrowserType := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["browser_type"] = 0
			plugin.BrowserType = v
		}
	}
	setBrowserType(DEFAULT_BROWSER_TYPE)
	setBrowserType(pluginConfig.Config.GetString(fmt.Sprintf("%s.browser_type", template)))
	setBrowserType((*pluginConfig.Params)["browser_type"])
	showParam("browser_type", plugin.BrowserType)

	// browser_argument.
	setBrowserArgument := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["browser_argument"] = 0
			plugin.BrowserArgument = v
		}
	}
	setBrowserArgument(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.browser_argument", template)))
	setBrowserArgument((*pluginConfig.Params)["browser_argument"])
	showParam("browser_argument", plugin.BrowserArgument)

	// browser_extension.
	setBrowserExtensions := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["browser_extension"] = 0
			plugin.BrowserExtension = v
		}
	}
	setBrowserExtensions(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.browser_extension", template)))
	setBrowserExtensions((*pluginConfig.Params)["browser_extension"])
	showParam("browser_extension", plugin.BrowserExtension)

	// browser_geometry.
	setBrowserGeometry := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["browser_geometry"] = 0
			plugin.BrowserGeometry = v
		}
	}
	setBrowserGeometry(DEFAULT_BROWSER_GEOMETRY)
	setBrowserGeometry(pluginConfig.Config.GetString(fmt.Sprintf("%s.browser_geometry", template)))
	setBrowserGeometry((*pluginConfig.Params)["browser_geometry"])
	showParam("browser_geometry", plugin.BrowserGeometry)

	// browser_instance.
	setBrowserInstance := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["browser_instance"] = 0
			plugin.BrowserInstance = v
		}
	}
	setBrowserInstance(DEFAULT_BROWSER_INSTANCE)
	setBrowserInstance(pluginConfig.Config.GetInt(fmt.Sprintf("%s.browser_instance", template)))
	setBrowserInstance((*pluginConfig.Params)["browser_instance"])
	showParam("browser_instance", plugin.BrowserInstance)

	// browser_instance_tab.
	setBrowserInstanceTab := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["browser_instance_tab"] = 0
			plugin.BrowserInstanceTab = v
		}
	}
	setBrowserInstanceTab(DEFAULT_BROWSER_INSTANCE_TAB)
	setBrowserInstanceTab(pluginConfig.Config.GetInt(fmt.Sprintf("%s.browser_instance_tab", template)))
	setBrowserInstanceTab((*pluginConfig.Params)["browser_instance_tab"])
	showParam("browser_instance_tab", plugin.BrowserInstanceTab)

	// browser_page_size.
	setBrowserPageSize := func(p interface{}) {
		if v, b := core.IsSize(p); b {
			availableParams["browser_page_size"] = 0
			plugin.BrowserPageSize = v
		}
	}
	setBrowserPageSize(DEFAULT_BROWSER_PAGE_SIZE)
	setBrowserPageSize(pluginConfig.Config.GetString(fmt.Sprintf("%s.browser_page_size", template)))
	setBrowserPageSize((*pluginConfig.Params)["browser_page_size"])
	showParam("browser_page_size", plugin.BrowserPageSize)

	// browser_page_timeout.
	setBrowserPageTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["browser_page_timeout"] = 0
			plugin.BrowserPageTimeout = v
		}
	}
	setBrowserPageTimeout(DEFAULT_BROWSER_PAGE_TIMEOUT)
	setBrowserPageTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.browser_page_timeout", template)))
	setBrowserPageTimeout((*pluginConfig.Params)["browser_page_timeout"])
	showParam("browser_page_timeout", plugin.BrowserPageTimeout)

	// browser_script_timeout.
	setBrowserScriptTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["browser_script_timeout"] = 0
			plugin.BrowserScriptTimeout = v
		}
	}
	setBrowserScriptTimeout(DEFAULT_BROWSER_SCRIPT_TIMEOUT)
	setBrowserScriptTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.browser_script_timeout", template)))
	setBrowserScriptTimeout((*pluginConfig.Params)["browser_script_timeout"])
	showParam("browser_script_timeout", plugin.BrowserScriptTimeout)

	// chunk_size.
	setChunkSize := func(p interface{}) {
		if v, b := core.IsSize(p); b {
			availableParams["chunk_size"] = 0
			plugin.ChunkSize = v
		}
	}
	setChunkSize(DEFAULT_CHUNK_SIZE)
	setChunkSize(pluginConfig.Config.GetString(fmt.Sprintf("%s.chunk_size", template)))
	setChunkSize((*pluginConfig.Params)["chunk_size"])
	showParam("chunk_size", plugin.ChunkSize)

	// client_id.
	setClientId := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["client_id"] = 0
			plugin.ClientId = v
		}
	}
	setClientId(pluginConfig.Config.GetString(fmt.Sprintf("%s.client_id", template)))
	setClientId((*pluginConfig.Params)["client_id"])
	showParam("client_id", plugin.ClientId)

	// cpu_load.
	setBrowserCpuLoad := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["cpu_load"] = 0
			plugin.CpuLoad = int32(v)
		}
	}
	setBrowserCpuLoad(DEFAULT_CPU_LOAD)
	setBrowserCpuLoad(pluginConfig.Config.GetInt(fmt.Sprintf("%s.cpu_load", template)))
	setBrowserCpuLoad((*pluginConfig.Params)["cpu_load"])
	showParam("cpu_load", plugin.CpuLoad)

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

	// mem_free.
	setBrowserMemFree := func(p interface{}) {
		if v, b := core.IsSize(p); b {
			availableParams["mem_free"] = 0
			plugin.MemFree = v
		}
	}
	setBrowserMemFree(DEFAULT_MEM_FREE)
	setBrowserMemFree(pluginConfig.Config.GetString(fmt.Sprintf("%s.mem_free", template)))
	setBrowserMemFree((*pluginConfig.Params)["mem_free"])
	showParam("mem_free", plugin.MemFree)

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

	// request_timeout.
	setRequestTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["request_timeout"] = 0
			plugin.RequestTimeout = v
		}
	}
	setRequestTimeout(DEFAULT_SERVER_REQUEST_TIMEOUT)
	setRequestTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.request_timeout", template)))
	setRequestTimeout((*pluginConfig.Params)["request_timeout"])
	showParam("request_timeout", plugin.RequestTimeout)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.Require = v

		}
	}
	setRequire((*pluginConfig.Params)["require"])
	showParam("require", plugin.Require)

	// script.
	setScript := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["script"] = 0
			plugin.Script = core.ExtractScripts(pluginConfig.Config, v)
		}
	}
	setScript(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.script", template)))
	setScript((*pluginConfig.Params)["script"])
	showParam("script", plugin.Script)

	// server.
	setServer := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["server"] = 0
			plugin.Server = v
		}
	}
	setServer(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.server", template)))
	setServer((*pluginConfig.Params)["server"])
	showParam("server", plugin.Server)

	// server_timeout.
	setServerTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["server_timeout"] = 0
			plugin.ServerTimeout = v
		}
	}
	setServerTimeout(DEFAULT_SERVER_CONNECT_TIMEOUT)
	setServerTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.server_timeout", template)))
	setServerTimeout((*pluginConfig.Params)["server_timeout"])
	showParam("server_timeout", plugin.ServerTimeout)

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.Timeout = v
		}
	}
	setTimeout(DEFAULT_TIMEOUT)
	setTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.Params)["timeout"])
	showParam("timeout", plugin.Timeout)

	// -----------------------------------------------------------------------------------------------------------------

	plugin.BatchChannel = make(chan *BatchTask, DEFAULT_BUFFER_LENGHT)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if len(plugin.Input) != len(plugin.Output) {
		return &Plugin{}, fmt.Errorf(core.ERROR_SIZE_MISMATCH.Error(), plugin.Input, plugin.Output)

	} else {
		core.SliceStringToUpper(&plugin.Input)
		core.SliceStringToUpper(&plugin.Output)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
