package webchelaProcess

import (
	"context"
	"errors"
	"fmt"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	pb "github.com/livelace/gosquito/pkg/gosquito/plugins/process/webchela/protobuf"
	log "github.com/livelace/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"io"
	"path/filepath"
	"reflect"
	"regexp"
	"time"
)

const (
	PLUGIN_NAME = "webchela"

	DEFAULT_BATCH_RETRY              = 0 // no retries.
	DEFAULT_BATCH_SIZE               = 100
	DEFAULT_BROWSER_GEOMETRY         = "1920x1080"
	DEFAULT_BROWSER_INSTANCE         = 1
	DEFAULT_BROWSER_INSTANCE_TAB     = 5
	DEFAULT_BROWSER_PAGE_SIZE        = "10M"
	DEFAULT_BROWSER_PAGE_TIMEOUT     = 20
	DEFAULT_BROWSER_PROXY            = ""
	DEFAULT_BROWSER_SCRIPT_TIMEOUT   = 20
	DEFAULT_BROWSER_TYPE             = "chrome"
	DEFAULT_BUFFER_LENGHT            = 1000
	DEFAULT_CHUNK_SIZE               = "3M"
	DEFAULT_COOKIE_INPUT_FILE        = false
	DEFAULT_COOKIE_INPUT_FILE_MODE   = "text"
	DEFAULT_CPU_LOAD                 = 25
	DEFAULT_MEM_FREE                 = "1G"
	DEFAULT_PAGE_BODY_FILENAME       = "body.html"
	DEFAULT_PAGE_TITLE_FILENAME      = "title.txt"
	DEFAULT_PAGE_URL_FILENAME        = "url.txt"
	DEFAULT_SCREENSHOT_PREFIX_REGEXP = "^class:|^css:|^id:|^name:|^tag:|^xpath:"
	DEFAULT_SCREENSHOT_TIMEOUT       = 10
	DEFAULT_SERVER_CONNECT_TIMEOUT   = 3
	DEFAULT_SERVER_REQUEST_TIMEOUT   = 10
	DEFAULT_TIMEOUT                  = 300
)

var (
	ERROR_UNKNOWN_SCREENSHOT_PREFIX = errors.New("unknown screenshot prefix: %s. valid prefixes: class, css, id, name, tag, xpath")
)

type BatchTask struct {
	ID               int
	Server           string
	Status           string
	Input            []string
	Output           []string
	CookieInput      []string
	ScreenshotInput  []string
	ScreenshotOutput [][]string
	ScriptInput      []string
	ScriptOutput     [][]string
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
		ClientId:          clientId,
		Urls:              batchTask.Input,
		Cookies:           batchTask.CookieInput,
		Screenshots:       batchTask.ScreenshotInput,
		Scripts:           batchTask.ScriptInput,
		ChunkSize:         p.OptionChunkSize,
		CpuLoad:           p.OptionCpuLoad,
		MemFree:           p.OptionMemFree,
		Timeout:           int32(p.OptionTimeout),
		Browser:           &webchelaTaskBrowser,
		ScreenshotTimeout: int32(p.OptionScreenshotTimeout),
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

func readAndCheckJsonCookie(p *Plugin, text string, firstValue bool) (string, error) {
	var cookie string

	// cookies from files:
	if p.OptionCookieInputFile {
		switch p.OptionCookieInputFileMode {
		case "lines":
			if lines, err := core.GetLinesFromFile(text); err == nil {
				for i, l := range lines {
					// check json before sending task.
					if e := core.IsJson(l); e != nil {
						return "", e
					}

					// first cookie shouldn't be prepended with separator.
					if firstValue && i == 0 {
						cookie += l
					} else {
						cookie += fmt.Sprintf("%s%s", core.DEFAULT_UNIQUE_SEPARATOR, l)
					}
				}
			} else {
				return "", err
			}

		case "text":
			if s, err := core.GetStringFromFile(text); err == nil {
				// check json before sending task.
				if e := core.IsJson(s); e != nil {
					return "", e
				}

				cookie = s
			} else {
				return "", err
			}
		}
	}

	// cookies from text:
	if !p.OptionCookieInputFile {
		// check json before sending task.
		if e := core.IsJson(text); e != nil {
			return "", e
		}

		cookie = text
	}

	return cookie, nil
}

func saveData(p *Plugin, b *BatchTask, results []*pb.Result) error {
	for _, result := range results {
		// Create output directory in plugin's temporary directory:
		outputDir := filepath.Join(p.Flow.FlowTempDir, p.PluginType, p.PluginName, result.UUID)
		err := core.CreateDirIfNotExist(outputDir)
		if err != nil {
			return err
		}

		// Save page data:
		err = core.WriteStringToFile(outputDir, DEFAULT_PAGE_URL_FILENAME, result.PageUrl)
		if err != nil {
			return err
		}

		err = core.WriteStringToFile(outputDir, DEFAULT_PAGE_TITLE_FILENAME, result.PageTitle)
		if err != nil {
			return err
		}

		err = core.WriteStringToFile(outputDir, DEFAULT_PAGE_BODY_FILENAME, result.PageBody)
		if err != nil {
			return err
		}

		b.Output = append(b.Output, filepath.Join(outputDir, DEFAULT_PAGE_BODY_FILENAME))

		// Save screenshots data:
		screenshots := make([]string, 0, len(result.Screenshots))

		for index, output := range result.Screenshots {
			filename := fmt.Sprintf("screenshot%02d-%02d.png", index, result.ScreenshotsId[index])

			err = core.WriteBase64ToFile(outputDir, filename, &output)

			if err != nil {
				return err
			}

			screenshots = append(screenshots, filepath.Join(outputDir, filename))
		}

		b.ScreenshotOutput = append(b.ScreenshotOutput, screenshots)

		// Write scripts files:
		scripts := make([]string, 0, len(result.Scripts))

		for index, output := range result.Scripts {
			filename := fmt.Sprintf("script%02d-%02d.txt", index, result.ScriptsId[index])

			err = core.WriteStringToFile(outputDir, filename, output)

			if err != nil {
				return err
			}

			scripts = append(scripts, filepath.Join(outputDir, filename))
		}

		b.ScriptOutput = append(b.ScriptOutput, scripts)

		// Show data path in console:
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
	OptionCookieInput          []string
	OptionCookieInputSize      int
	OptionCookieInputFile      bool
	OptionCookieInputFileMode  string
	OptionCpuLoad              int32
	OptionInclude              bool
	OptionInput                []string
	OptionInputSize            int
	OptionMemFree              int64
	OptionOutput               []string
	OptionOutputSize           int
	OptionRequestTimeout       int
	OptionRequire              []int
	OptionScreenshotInput      [][]string
	OptionScreenshotInputSize  int
	OptionScreenshotOutput     []string
	OptionScreenshotOutputSize int
	OptionScreenshotTimeout    int
	OptionScriptInput          [][]string
	OptionScriptInputSize      int
	OptionScriptOutput         []string
	OptionScriptOutputSize     int
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

	// -----------------------------------------------------------------------
	// Extract URLs from data items into single flat slice.
	// This is needed for batch slicing (process URLs in sized blocks/batches).
	// Example: every datum has filled ["data.array0", "data.array1"] with URLs,
	// we extract all URLs from all datums into one flat slice: [url0, url1 ... urlN].
	// To recognize boundaries between datums in slice we save "metadata".
	// Example metadata: [0] = [20, 300].
	// that means: [datum 0] = [data.array0 = 20 urls, data.array1 = 300 urls]
	datumURLMeta := make(map[int][]int)

	// For each URL we fill/align corresponding cookies, screenshots, scripts. Exp:
	// URLs:        ['https://livelace.ru', ...]
	// Cookies:     ['{"name": "foo", "value": "bar"}<SEPARATOR>{"name": "first", "value": "second"}', ...]
	// Screenshots: ['tag:body'<SEPARATOR>'xpath://html', ...]
	// Scripts: ['return 1;<SEPARATOR>return 2;', ...]

	allURL := make([]string, 0)
	allCookies := make([]string, 0)
	allScreenshots := make([]string, 0)
	allScripts := make([]string, 0)

	for datumIndex, datumValue := range data {
		datumURLMeta[datumIndex] = make([]int, len(p.OptionInput))

		for inputIndex, inputValue := range p.OptionInput {
			// Combine cookies for URL:
			// Fail fast if there are some problems with
			// file reading and JSON format.
			var urlCookie string

			if p.OptionCookieInputSize > 0 {
				ri, ierr := core.ReflectDatumField(datumValue, p.OptionCookieInput[datumIndex])

				// input is datatum field:
				if ierr == nil {
					switch ri.Kind() {
					case reflect.String:
						c, err := readAndCheckJsonCookie(p, ri.String(), true)
						if err == nil {
							urlCookie = c
						} else {
							return temp, err
						}
					case reflect.Slice:
						var c string
						var err error

						for i := 0; i < ri.Len(); i++ {
							if i == 0 {
								c, err = readAndCheckJsonCookie(p, ri.Index(i).String(), true)
							} else {
								c, err = readAndCheckJsonCookie(p, ri.Index(i).String(), false)
							}

							if err == nil {
								urlCookie += c
							} else {
								return temp, err
							}
						}
					}
				}

				// input is just a string:
				if ierr != nil {
					c, err := readAndCheckJsonCookie(p, p.OptionCookieInput[datumIndex], true)
					if err == nil {
						urlCookie = c
					} else {
						return temp, err
					}
				}
			}

			// Combine screenshots for URL:
			var urlScreenshot string

			if p.OptionScreenshotInputSize > 0 {
				for i, v := range p.OptionScreenshotInput[inputIndex] {
					if i == 0 {
						urlScreenshot += fmt.Sprintf("%s", v)
					} else {
						urlScreenshot += fmt.Sprintf("%s%s", core.DEFAULT_UNIQUE_SEPARATOR, v)
					}
				}
			}

			// Combine scripts for URL:
			var urlScript string

			if p.OptionScriptInputSize > 0 {
				for i, v := range p.OptionScriptInput[inputIndex] {
					if i == 0 {
						urlScript += fmt.Sprintf("%s", v)
					} else {
						urlScript += fmt.Sprintf("%s%s", core.DEFAULT_UNIQUE_SEPARATOR, v)
					}
				}
			}

			// 1. Set metadata.
			// 2. Put data in flat slices.
			ri, _ := core.ReflectDatumField(datumValue, inputValue)

			switch ri.Kind() {
			case reflect.String:
				datumURLMeta[datumIndex][inputIndex] = 1

				allURL = append(allURL, ri.String())
				allCookies = append(allCookies, urlCookie)
				allScreenshots = append(allScreenshots, urlScreenshot)
				allScripts = append(allScripts, urlScript)

			case reflect.Slice:
				datumURLMeta[datumIndex][inputIndex] = ri.Len()

				for i := 0; i < ri.Len(); i++ {
					allURL = append(allURL, ri.Index(i).String())
					allCookies = append(allCookies, urlCookie)
					allScreenshots = append(allScreenshots, urlScreenshot)
					allScripts = append(allScripts, urlScript)
				}
			}
		}
	}

	// Split data into batches:
	batches := make([][]string, 0)
	batchesCookies := make([][]string, 0)
	batchesScreenshots := make([][]string, 0)
	batchesScripts := make([][]string, 0)

	for i := 0; i < len(allURL); i += p.OptionBatchSize {
		end := i + p.OptionBatchSize
		if end > len(allURL) {
			end = len(allURL)
		}

		batches = append(batches, allURL[i:end])
		batchesCookies = append(batchesCookies, allCookies[i:end])
		batchesScreenshots = append(batchesScreenshots, allScreenshots[i:end])
		batchesScripts = append(batchesScripts, allScripts[i:end])
	}

	// -----------------------------------------------------------------------
	// 1. Get suitable server (cpu/mem free enough) from pool
	// 2. Assign batches to servers concurrently
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

		for batchId, batchData := range batches {
			switch batchStatus[batchId] {
			case "":
				if server := getServer(p, batchId, &serverFailStat); server != "" {
					go processBatch(p, &BatchTask{
						ID:              batchId,
						Server:          server,
						Input:           batchData,
						CookieInput:     batchesCookies[batchId],
						ScreenshotInput: batchesScreenshots[batchId],
						ScriptInput:     batchesScripts[batchId],
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

		// Fetch completed batches and update their statuses.
		// Update statistics for servers.
		for i := 0; i < len(p.OptionBatchChannel); i++ {
			b := <-p.OptionBatchChannel

			batchStatus[b.ID] = b.Status
			batchResult[b.ID] = b

			if b.Status == "fail" {
				serverFailStat[b.Server] += 1
			}
		}

		// Wait for all batches completion.
		if completed {
			break
		} else {
			timeoutCounter += 1
			time.Sleep(1 * time.Second)
		}
	}

	// -----------------------------------------------------------------------
	// Put all batches results into flat slices:
	outputData := make([]string, 0)
	screenshotOutputData := make([][]string, 0)
	scriptOutputData := make([][]string, 0)

	for i := 0; i < len(batches); i++ {
		outputData = append(outputData, batchResult[i].Output...)
		screenshotOutputData = append(screenshotOutputData, batchResult[i].ScreenshotOutput...)
		scriptOutputData = append(scriptOutputData, batchResult[i].ScriptOutput...)
	}

	// Amount of input and output data _must_ be equal, even if some pages were processed with errors
	// (timeouts, DNS resolution problems, etc.).
	if len(allURL) != len(outputData) {
		core.LogProcessPlugin(p.LogFields, fmt.Errorf("main loop: received data not equal sent data: %d != %d",
			len(outputData), len(allURL)))
		return temp, nil
	}

	// -----------------------------------------------------------------------
	// Fill corresponding output datum with received data.
	// Use offset metadata for proper slicing.
	outputOffset := 0

	for datumIndex := 0; datumIndex < len(datumURLMeta); datumIndex++ {
		grabbed := false

		datumMeta := datumURLMeta[datumIndex]

		for dataIndex, dataValue := range datumMeta {
			if dataValue > 0 {
				grabbed = true
			}

			ro, _ := core.ReflectDatumField(data[datumIndex], p.OptionOutput[dataIndex])

			for offset := outputOffset; offset < outputOffset+dataValue; offset++ {
				switch ro.Kind() {
				case reflect.String:
					ro.SetString(outputData[offset])
				case reflect.Slice:
					for offset := outputOffset; offset < outputOffset+dataValue; offset++ {
						ro.Set(reflect.Append(ro, reflect.ValueOf(outputData[offset])))
					}
				}

				if p.OptionScreenshotOutputSize > 0 {
					screenshotRo, _ := core.ReflectDatumField(data[datumIndex], p.OptionScreenshotOutput[dataIndex])
					screenshotRo.Set(reflect.ValueOf(screenshotOutputData[offset]))
				}

				if p.OptionScriptOutputSize > 0 {
					scriptRo, _ := core.ReflectDatumField(data[datumIndex], p.OptionScriptOutput[dataIndex])
					scriptRo.Set(reflect.ValueOf(scriptOutputData[offset]))
				}
			}

			outputOffset += dataValue
		}

		if grabbed {
			temp = append(temp, data[datumIndex])
		}
	}

	// -----------------------------------------------------------------------
	// fin!

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
		"cookie_input":           -1,
		"cookie_input_file":      -1,
		"cookie_input_file_mode": -1,
		"cpu_load":               -1,
		"input":                  1,
		"mem_free":               -1,
		"output":                 -1,
		"request_timeout":        -1,
		"screenshot_input":       -1,
		"screenshot_output":      -1,
		"screenshot_timeout":     -1,
		"script_input":           -1,
		"script_output":          -1,
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

	// cookie_input.
	setCookieInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["cookie_input"] = 0
			plugin.OptionCookieInput = v
			plugin.OptionCookieInputSize = len(v)
		}
	}
	setCookieInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.cookie_input", template)))
	setCookieInput((*pluginConfig.PluginParams)["cookie_input"])
	core.ShowPluginParam(plugin.LogFields, "cookie_input", plugin.OptionCookieInput)

	// cookie_input_file.
	setCookieInputFile := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["cookie_input_file"] = 0
			plugin.OptionCookieInputFile = v
		}
	}
	setCookieInputFile(DEFAULT_COOKIE_INPUT_FILE)
	setCookieInputFile(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.cookie_input_file", template)))
	setCookieInputFile((*pluginConfig.PluginParams)["cookie_input_file"])
	core.ShowPluginParam(plugin.LogFields, "cookie_input_file", plugin.OptionCookieInputFile)

	// cookie_input_file_mode.
	setCookieInputFileInMode := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["cookie_input_file_mode"] = 0
			plugin.OptionCookieInputFileMode = v
		}
	}
	setCookieInputFileInMode(DEFAULT_COOKIE_INPUT_FILE_MODE)
	setCookieInputFileInMode(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.cookie_input_file_mode", template)))
	setCookieInputFileInMode((*pluginConfig.PluginParams)["cookie_input_file_mode"])
	core.ShowPluginParam(plugin.LogFields, "cookie_input_file_mode", plugin.OptionCookieInputFileMode)

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
			plugin.OptionInputSize = len(v)
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
			plugin.OptionOutputSize = len(v)
		}
	}
	setOutput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.output", template)))
	setOutput((*pluginConfig.PluginParams)["output"])
	core.ShowPluginParam(plugin.LogFields, "output", plugin.OptionOutput)

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

	// screenshot_input.
	setScreenshotInput := func(p interface{}) {
		if v, b := core.IsSliceOfSliceString(p); b {
			availableParams["screenshot_input"] = 0
			plugin.OptionScreenshotInput = v
			plugin.OptionScreenshotInputSize = len(v)
		}
	}
	setScreenshotInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.screenshot_input", template)))
	setScreenshotInput((*pluginConfig.PluginParams)["screenshot_input"])
	core.ShowPluginParam(plugin.LogFields, "screenshot_input", plugin.OptionScreenshotInput)

	// screenshot_output.
	setScreenshotOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			if err := core.IsDatumFieldsSlice(&v); err == nil {
				availableParams["screenshot_output"] = 0
				plugin.OptionScreenshotOutput = v
				plugin.OptionScreenshotOutputSize = len(v)
			}
		}
	}
	setScreenshotOutput((*pluginConfig.PluginParams)["screenshot_output"])
	core.ShowPluginParam(plugin.LogFields, "screenshot_output", plugin.OptionScreenshotOutput)

	// screenshot_timeout.
	setScreenshotTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["screenshot_timeout"] = 0
			plugin.OptionScreenshotTimeout = v
		}
	}
	setScreenshotTimeout(DEFAULT_SCREENSHOT_TIMEOUT)
	setScreenshotTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.screenshot_timeout", template)))
	setScreenshotTimeout((*pluginConfig.PluginParams)["screenshot_timeout"])
	core.ShowPluginParam(plugin.LogFields, "screenshot_timeout", plugin.OptionScreenshotTimeout)

	// script_input.
	setScriptInput := func(p interface{}) {
		if v, b := core.IsSliceOfSliceString(p); b {
			availableParams["script_input"] = 0
			plugin.OptionScriptInput = v
			plugin.OptionScriptInputSize = len(v)
		}
	}
	setScriptInput((*pluginConfig.PluginParams)["script_input"])
	core.ShowPluginParam(plugin.LogFields, "script_input", plugin.OptionScriptInput)

	// script_output.
	setScriptOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			if err := core.IsDatumFieldsSlice(&v); err == nil {
				availableParams["script_output"] = 0
				plugin.OptionScriptOutput = v
				plugin.OptionScriptOutputSize = len(v)
			}
		}
	}
	setScriptOutput((*pluginConfig.PluginParams)["script_output"])
	core.ShowPluginParam(plugin.LogFields, "script_output", plugin.OptionScriptOutput)

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

	// input/output:
	if plugin.OptionInputSize != plugin.OptionOutputSize {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput)
	}

	if err := core.IsDatumFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
		return &Plugin{}, err
	}

	// cookie_input:
	if plugin.OptionCookieInputSize > 0 && plugin.OptionCookieInputSize != plugin.OptionInputSize {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionCookieInput)
	}

	// screenshot_input/screenshot_output:
	if plugin.OptionScreenshotInputSize > 0 && plugin.OptionScreenshotInputSize != plugin.OptionInputSize {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionScreenshotInput)
	}

	if plugin.OptionScreenshotInputSize > 0 && plugin.OptionScreenshotOutputSize != plugin.OptionScreenshotInputSize {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionScreenshotInput, plugin.OptionScreenshotOutput)
	}

	if plugin.OptionScreenshotOutputSize > 0 && plugin.OptionScreenshotOutputSize != plugin.OptionScreenshotInputSize {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionScreenshotInput, plugin.OptionScreenshotOutput)
	}

	matchPrefix, _ := regexp.Compile(DEFAULT_SCREENSHOT_PREFIX_REGEXP)

	for _, v := range plugin.OptionScreenshotInput {
		for _, i := range v {
			if !matchPrefix.MatchString(i) {
				return &Plugin{}, fmt.Errorf(ERROR_UNKNOWN_SCREENSHOT_PREFIX.Error(), i)
			}
		}
	}

	// script_input/script_output:
	if plugin.OptionScriptInputSize > 0 && plugin.OptionScriptInputSize != plugin.OptionInputSize {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionScriptInput)
	}

	if plugin.OptionScriptInputSize > 0 && plugin.OptionScriptOutputSize != plugin.OptionScriptInputSize {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionScriptInput, plugin.OptionScriptOutput)
	}

	if plugin.OptionScriptOutputSize > 0 && plugin.OptionScriptOutputSize != plugin.OptionScriptInputSize {
		return &Plugin{}, fmt.Errorf("%s: %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionScriptInput, plugin.OptionScriptOutput)
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
