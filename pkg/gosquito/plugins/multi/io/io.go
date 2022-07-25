package ioMulti

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
)

const (
	PLUGIN_NAME = "io"

	DEFAULT_FILE_IN       = false
	DEFAULT_FILE_IN_MODE  = "text"
	DEFAULT_FILE_OUT      = false
	DEFAULT_FILE_OUT_MODE = "truncate"
	DEFAULT_FILE_OUT_WRAP = "\n"
	DEFAULT_TEXT_WRAP     = "\n"
	DEFAULT_MATCH_TTL     = "1d"
)

var (
	ERROR_COPY_TO_STRING = errors.New("cannot copy anything to string: %v")
	ERROR_MODE_UNKNOWN   = errors.New("mode unknown: %v")

	INFO_APPEND_FILE_TO_FILE = "append file to file: %v -> %v, %v"
	INFO_COPY_FILE_TO_FILE   = "copy file to file: %v -> %v, %v"
	INFO_READ_TEXT_FROM_FILE = "read text from file: %v -> text, %v"
	INFO_WRITE_TEXT_TO_FILE  = "write text to file: text -> %v, %v"
)

func appendFileToFile(p *Plugin, src string, dst string) error {
	if src == "" || dst == "" {
		core.LogProcessPlugin(p.LogFields, fmt.Sprintf(INFO_APPEND_FILE_TO_FILE, src, dst, "skip"))
		return nil
	}

	if _, err := core.IsFile(src); err != nil {
		core.LogProcessPlugin(p.LogFields, err)
		return err
	}

	source, err := os.ReadFile(src)
	if err != nil {
		core.LogProcessPlugin(p.LogFields,
			fmt.Errorf(INFO_APPEND_FILE_TO_FILE, src, dst, err))
		return err
	}

	destination, _ := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	defer destination.Close()

	if b, err := destination.Write(source); err == nil {
		core.LogProcessPlugin(p.LogFields,
			fmt.Sprintf(INFO_APPEND_FILE_TO_FILE, src, dst, b))
	} else {
		core.LogProcessPlugin(p.LogFields,
			fmt.Errorf(INFO_APPEND_FILE_TO_FILE, src, dst, err))
		return err
	}

	return nil
}

func copyFileToFile(p *Plugin, src string, dst string) error {
	if src == "" || dst == "" {
		core.LogProcessPlugin(p.LogFields, fmt.Sprintf(INFO_COPY_FILE_TO_FILE, src, dst, "skip"))
		return nil
	}

	if _, err := core.IsFile(src); err != nil {
		core.LogProcessPlugin(p.LogFields, err)
		return err
	}

	source, _ := os.OpenFile(src, os.O_RDONLY, os.ModePerm)
	defer source.Close()

	destination, _ := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	defer destination.Close()

	if b, err := io.Copy(destination, source); err == nil {
		core.LogProcessPlugin(p.LogFields,
			fmt.Sprintf(INFO_COPY_FILE_TO_FILE, src, dst, b))
	} else {
		core.LogProcessPlugin(p.LogFields,
			fmt.Errorf(INFO_COPY_FILE_TO_FILE, src, dst, err))
		return err
	}

	return nil
}

func readTextFromFile(p *Plugin, src string) (string, error) {
	if src == "" {
		core.LogProcessPlugin(p.LogFields,
			fmt.Sprintf(INFO_READ_TEXT_FROM_FILE, src, "skip"))
		return "", nil
	}

	if _, err := core.IsFile(src); err != nil {
		core.LogProcessPlugin(p.LogFields, err)
		return "", err
	}

	source, err := os.ReadFile(src)
	if err == nil {
		core.LogProcessPlugin(p.LogFields,
			fmt.Sprintf(INFO_READ_TEXT_FROM_FILE, src, nil))
		return string(source), nil
	} else {
		core.LogProcessPlugin(p.LogFields,
			fmt.Errorf(INFO_READ_TEXT_FROM_FILE, src, err))
		return "", err
	}
}

func writeTextToFile(p *Plugin, text string, dst string) error {
	if dst == "" {
		core.LogProcessPlugin(p.LogFields,
			fmt.Sprintf(INFO_WRITE_TEXT_TO_FILE, dst, "skip"))
		return nil
	}

	var destination *os.File
	if p.OptionFileOutMode == "append" {
		destination, _ = os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	} else {
		destination, _ = os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	}
	defer destination.Close()

	if b, err := destination.WriteString(fmt.Sprintf("%s%s", text, p.OptionFileOutWrap)); err == nil {
		core.LogProcessPlugin(p.LogFields,
			fmt.Sprintf(INFO_WRITE_TEXT_TO_FILE, dst, b))
	} else {
		core.LogProcessPlugin(p.LogFields,
			fmt.Errorf(INFO_WRITE_TEXT_TO_FILE, dst, err))
		return err
	}

	return nil
}

type Plugin struct {
	m sync.Mutex

	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionExpireAction        []string
	OptionExpireActionDelay   int64
	OptionExpireActionTimeout int
	OptionExpireInterval      int64
	OptionExpireLast          int64
	OptionFileIn              bool
	OptionFileInMode          string
	OptionFileOut             bool
	OptionFileOutMode         string
	OptionFileOutWrap         string
	OptionInclude             bool
	OptionInput               []string
	OptionMatchSignature      []string
	OptionMatchTTL            time.Duration
	OptionOutput              []string
	OptionRequire             []int
	OptionTextWrap            string
	OptionTimeFormat          string
	OptionTimeFormatA         string
	OptionTimeFormatB         string
	OptionTimeFormatC         string
	OptionTimeZone            *time.Location
	OptionTimeZoneA           *time.Location
	OptionTimeZoneB           *time.Location
	OptionTimeZoneC           *time.Location
	OptionTimeout             int
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

func (p *Plugin) GetInput() []string {
	return p.OptionInput
}

func (p *Plugin) GetName() string {
	return p.PluginName
}

func (p *Plugin) GetRequire() []int {
	return p.OptionRequire
}

func (p *Plugin) LoadState() (map[string]time.Time, error) {
	p.m.Lock()
	defer p.m.Unlock()

	data := make(map[string]time.Time, 0)

	if err := core.PluginLoadState(p.Flow.FlowStateDir, &data); err != nil {
		return data, err
	}

	return data, nil
}

func (p *Plugin) Process(data []*core.Datum) ([]*core.Datum, error) {
	temp := make([]*core.Datum, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		for index, input := range p.OptionInput {
			ri, ierr := core.ReflectDatumField(item, input)
			ro, oerr := core.ReflectDatumField(item, p.OptionOutput[index])

			// Input and output are datum fields:
			if ierr == nil && oerr == nil {

				// Input and ouput are string: +
				if ri.Kind() == reflect.String && ro.Kind() == reflect.String {
					// 1. Copy files.
					if p.OptionFileIn && p.OptionFileOut {
						copyFileToFile(p, ri.String(), ro.String())
					}

					// 2. Copy fields.
					if !p.OptionFileIn && !p.OptionFileOut {
						ro.Set(ri)
					}

					// 3. Write files.
					if !p.OptionFileIn && p.OptionFileOut {
						writeTextToFile(p, ri.String(), ro.String())
					}

					// 4. Read files.
					if p.OptionFileIn && !p.OptionFileOut {
						if s, err := readTextFromFile(p, ri.String()); err == nil {
							ro.SetString(s)
						}
					}
				}

				// Input and ouput are slice: +
				if ri.Kind() == reflect.Slice && ro.Kind() == reflect.Slice {
					// 1. Copy files.
					// Copy files only if both slices have equal size
					// We don't know where target file should be copied.
					if p.OptionFileIn && p.OptionFileOut && ri.Len() == ro.Len() {
						for i := 0; i < ri.Len(); i++ {
							copyFileToFile(p, ri.Index(i).String(), ro.Index(i).String())
						}
					}

					// 2. Copy fields.
					if !p.OptionFileIn && !p.OptionFileOut {
						ro.Set(ri)
					}

					// 3. Write files.
					// Write strings only both slices have equal size.
					// We don't know where target file should be written.
					if !p.OptionFileIn && p.OptionFileOut && ri.Len() == ro.Len() {
						for i := 0; i < ri.Len(); i++ {
							writeTextToFile(p, ri.Index(i).String(), ro.Index(i).String())
						}
					}

					// 4. Read files.
					// Output slice will overwritten even if it contains data.
					if p.OptionFileIn && !p.OptionFileOut {
						r := make([]string, ri.Len())
						for i := 0; i < ri.Len(); i++ {
							if s, err := readTextFromFile(p, ri.String()); err == nil {
								r = append(r, s)
							}
						}
						ro.Set(reflect.ValueOf(r))
					}
				}

				// Input is string, output is slice: +
				if ri.Kind() == reflect.String && ro.Kind() == reflect.Slice {

					// 1. Copy files.
					// Copy single file to multiple destinations.
					if p.OptionFileIn && p.OptionFileOut {
						for i := 0; i < ro.Len(); i++ {
							copyFileToFile(p, ri.String(), ro.Index(i).String())
						}
					}

					// 2. Copy fields.
					// If output slice is not empty: fill entire slice with single string.
					// If output slice is empty: set output slice with single value.
					if !p.OptionFileIn && !p.OptionFileOut {
						if ro.Len() > 0 {
							for i := 0; i < ro.Len(); i++ {
								ro.Index(i).SetString(ri.String())
							}
						} else {
							ro.Set(reflect.ValueOf([]string{ri.String()}))
						}
					}

					// 3. Write files.
					if !p.OptionFileIn && p.OptionFileOut {
						for i := 0; i < ro.Len(); i++ {
							writeTextToFile(p, ri.String(), ro.Index(i).String())
						}
					}

					// 4. Read files.
					// If output slice is not empty: fill entire slice with single string.
					// If output slice is empty: set output slice with single value.
					if p.OptionFileIn && !p.OptionFileOut {
						if s, err := readTextFromFile(p, ri.String()); err == nil {
							if ro.Len() > 0 {
								for i := 0; i < ro.Len(); i++ {
									ro.Index(i).SetString(s)
								}
							} else {
								ro.Set(reflect.ValueOf([]string{s}))
							}
						}
					}
				}

				// Input is slice, output is string: +
				if ri.Kind() == reflect.Slice && ro.Kind() == reflect.String {
					// 1. Copy files.
					// Append multiple files into single file.
					if p.OptionFileIn && p.OptionFileOut {
						for i := 0; i < ri.Len(); i++ {
							appendFileToFile(p, ri.Index(i).String(), ro.String())
						}
					}

					// 2. Copy fields.
					// Join slice items into text and set output string.
					if !p.OptionFileIn && !p.OptionFileOut {
						s := ""
						for i := 0; i < ri.Len(); i++ {
							s += fmt.Sprintf("%s%s", ri.Index(i).String(), p.OptionTextWrap)
						}
						ro.SetString(s)
					}

					// 3. Write files.
					// Join slice items into text and write to file.
					if !p.OptionFileIn && p.OptionFileOut {
						s := ""
						for i := 0; i < ri.Len(); i++ {
							s += fmt.Sprintf("%s%s", ri.Index(i).String(), p.OptionTextWrap)
						}
						writeTextToFile(p, s, ro.String())
					}

					// 4. Read files.
					// Read multiple files and set output string.
					if p.OptionFileIn && !p.OptionFileOut {
						r := ""
						for i := 0; i < ri.Len(); i++ {
							if s, err := readTextFromFile(p, ri.Index(i).String()); err == nil {
								r += fmt.Sprintf("%s%s", s, p.OptionTextWrap)
							}
						}
						ro.SetString(r)
					}
				}
			}

			// Input is datum field string, output is string: +
			if ierr == nil && oerr != nil && ri.Kind() == reflect.String {

				// 1. Copy files.
				if p.OptionFileIn && p.OptionFileOut {
					copyFileToFile(p, ri.String(), p.OptionOutput[index])
				}

				// 2. Copy fields.
				if !p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}

				// 3. Write files.
				if !p.OptionFileIn && p.OptionFileOut {
					writeTextToFile(p, ri.String(), p.OptionOutput[index])
				}

				// 4. Read files.
				if p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}
			}

			// Input is string, output is datum field string: +
			if ierr != nil && oerr == nil && ro.Kind() == reflect.String {

				// 1. Copy files.
				if p.OptionFileIn && p.OptionFileOut {
					copyFileToFile(p, p.OptionInput[index], ro.String())
				}

				// 2. Copy fields.
				if !p.OptionFileIn && !p.OptionFileOut {
					ro.SetString(p.OptionInput[index])
				}

				// 3. Write files.
				if !p.OptionFileIn && p.OptionFileOut {
					writeTextToFile(p, p.OptionInput[index], ro.String())
				}

				// 4. Read files.
				if p.OptionFileIn && !p.OptionFileOut {
					if s, err := readTextFromFile(p, p.OptionInput[index]); err == nil {
						ro.SetString(s)
					}
				}
			}

			// Input is datum field slice, output is string: +
			if ierr == nil && oerr != nil && ri.Kind() == reflect.Slice {

				// 1. Copy files.
				// Append multiple files into single file.
				if p.OptionFileIn && p.OptionFileOut {
					for i := 0; i < ri.Len(); i++ {
						appendFileToFile(p, ri.Index(i).String(), p.OptionOutput[index])
					}
				}

				// 2. Copy fields.
				if !p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}

				// 3. Write files.
				// Join slice items into text and write to file.
				if !p.OptionFileIn && p.OptionFileOut {
					s := ""
					for i := 0; i < ri.Len(); i++ {
						s += fmt.Sprintf("%s%s", ri.Index(i).String(), p.OptionTextWrap)
					}
					writeTextToFile(p, s, p.OptionOutput[index])
				}

				// 4. Read files.
				if p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}
			}

			// Input is string, output is datum field slice: +
			if ierr != nil && oerr == nil && ro.Kind() == reflect.Slice {

				// 1. Copy files.
				if p.OptionFileIn && p.OptionFileOut {
					for i := 0; i < ro.Len(); i++ {
						copyFileToFile(p, p.OptionInput[index], ro.Index(i).String())
					}
				}

				// 2. Copy fields.
				// If output slice is not empty: fill entire slice with single string.
				// If output slice is empty: set output slice with single value.
				if !p.OptionFileIn && !p.OptionFileOut {
					if ro.Len() > 0 {
						for i := 0; i < ro.Len(); i++ {
							ro.Index(i).SetString(p.OptionInput[index])
						}
					} else {
						ro.Set(reflect.ValueOf([]string{p.OptionInput[index]}))
					}
				}

				// 3. Write files.
				if !p.OptionFileIn && p.OptionFileOut {
					for i := 0; i < ro.Len(); i++ {
						writeTextToFile(p, p.OptionInput[index], ro.Index(i).String())
					}
				}

				// 4. Read files.
				// If output slice is not empty: fill entire slice with single string.
				// If output slice is empty: set output slice with single value.
				if p.OptionFileIn && !p.OptionFileOut {
					if s, err := readTextFromFile(p, p.OptionInput[index]); err == nil {
						if ro.Len() > 0 {
							for i := 0; i < ro.Len(); i++ {
								ro.Index(i).SetString(s)
							}
						} else {
							ro.Set(reflect.ValueOf([]string{s}))
						}
					}
				}
			}

			// Input is string, output is string:
			if ierr != nil && oerr != nil {

				// 1. Copy files.
				if p.OptionFileIn && p.OptionFileOut {
					copyFileToFile(p, p.OptionInput[index], p.OptionOutput[index])
				}

				// 2. Copy fields.
				if !p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}

				// 3. Write files.
				if !p.OptionFileIn && p.OptionFileOut {
					writeTextToFile(p, p.OptionInput[index], p.OptionOutput[index])
				}

				// 4. Read files.
				if p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}
			}
		}
	}

	return temp, nil
}

func (p *Plugin) Receive() ([]*core.Datum, error) {
	p.LogFields["run"] = p.Flow.GetRunID()
	currentTime := time.Now().UTC()
	failedSources := make([]string, 0)
	temp := make([]*core.Datum, 0)

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}
	core.LogInputPlugin(p.LogFields, "all", fmt.Sprintf("states loaded: %d", len(flowStates)))

	for _, source := range p.OptionInput {
		var itemNew = false
		var itemSignature string
		var itemSignatureHash string
		var itemTime = currentTime
		var sourceLastTime time.Time
		var u, _ = uuid.NewRandom()

		// Check if we work with source first time.
		if v, ok := flowStates[source]; ok {
			sourceLastTime = v
		} else {
			sourceLastTime = time.Unix(0, 0)
		}

		itemLines := make([]string, 0)
		itemMtime := ""
		itemText := ""

		if p.OptionFileIn {
			if mtime, err := core.IsFile(source); err == nil {
				itemMtime = fmt.Sprintf("%v", mtime.Unix())
			} else {
				failedSources = append(failedSources, source)
				core.LogProcessPlugin(p.LogFields, err)
				continue
			}

			switch p.OptionFileInMode {
			case "lines":
				if l, err := core.GetLinesFromFile(source); err == nil {
					itemLines = l
				} else {
					failedSources = append(failedSources, source)
					core.LogProcessPlugin(p.LogFields, err)
					continue
				}
			case "text":
				if s, err := core.GetStringFromFile(source); err == nil {
					itemText = s
				} else {
					failedSources = append(failedSources, source)
					core.LogProcessPlugin(p.LogFields, err)
					continue
				}
			}
		} else {
			itemText = source
		}

		// Process only new items. Two methods:
		// 1. Match item by user provided signature.
		// 2. Pass items as is.
		if len(p.OptionMatchSignature) > 0 {
			for _, v := range p.OptionMatchSignature {
				switch v {
				case "IO.MTIME":
					itemSignature += itemMtime
				case "IO.TEXT":
					itemSignature += itemText
				}
			}

			// set default value for signature if user provided wrong values.
			if len(itemSignature) == 0 {
				itemSignature += source
			}

			itemSignatureHash = core.HashString(&itemSignature)

			if _, ok := flowStates[itemSignatureHash]; !ok {
				// save item signature hash to state.
				flowStates[itemSignatureHash] = currentTime

				// update source timestamp.
				if itemTime.Unix() > sourceLastTime.Unix() {
					sourceLastTime = itemTime
				}

				itemNew = true
			}

		} else {
			sourceLastTime = itemTime
			itemNew = true
		}

		// Add item to result.
		if itemNew {
			temp = append(temp, &core.Datum{
				FLOW:        p.Flow.FlowName,
				PLUGIN:      p.PluginName,
				SOURCE:      source,
				TIME:        itemTime,
				TIMEFORMAT:  itemTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
				TIMEFORMATA: itemTime.In(p.OptionTimeZoneA).Format(p.OptionTimeFormatA),
				TIMEFORMATB: itemTime.In(p.OptionTimeZoneB).Format(p.OptionTimeFormatB),
				TIMEFORMATC: itemTime.In(p.OptionTimeZoneC).Format(p.OptionTimeFormatC),
				UUID:        u,

				IO: core.Io{
					LINES: itemLines,
					MTIME: itemMtime,
					TEXT:  itemText,
				},

				WARNINGS: make([]string, 0),
			})
		}

		flowStates[source] = sourceLastTime
		core.LogInputPlugin(p.LogFields, source,
			fmt.Sprintf("last update: %s, received data: %d, new data: %v", sourceLastTime, 1, itemNew))
	}

	// Save updated flow states.
	if err := p.SaveState(flowStates); err != nil {
		return temp, err
	}

	// Check every source for expiration.
	sourcesExpired := false

	// Check if any source is expired.
	for _, source := range p.OptionInput {
		sourceTime := flowStates[source]

		if (currentTime.Unix() - sourceTime.Unix()) > p.OptionExpireInterval/1000 {
			sourcesExpired = true

			core.LogInputPlugin(p.LogFields, source,
				fmt.Sprintf("source expired: %v", currentTime.Sub(sourceTime)))

			// Execute command if expire delay exceeded.
			// ExpireLast keeps last execution timestamp.
			if (currentTime.Unix() - p.OptionExpireLast) > p.OptionExpireActionDelay/1000 {
				p.OptionExpireLast = currentTime.Unix()

				// Execute command with args.
				// We don't worry about command return code.
				if len(p.OptionExpireAction) > 0 {
					cmd := p.OptionExpireAction[0]
					args := []string{p.Flow.FlowName, source, fmt.Sprintf("%v", sourceTime.Unix())}
					args = append(args, p.OptionExpireAction[1:]...)

					output, err := core.ExecWithTimeout(cmd, args, p.OptionExpireActionTimeout)

					core.LogInputPlugin(p.LogFields, source, fmt.Sprintf(
						"source expired action: command: %s, arguments: %v, output: %s, error: %v",
						cmd, args, output, err))
				}
			}
		}
	}

	// Inform about sources failures.
	if len(failedSources) > 0 {
		return temp, core.ERROR_FLOW_SOURCE_FAIL
	}

	// Inform about expiration.
	if sourcesExpired {
		return temp, core.ERROR_FLOW_EXPIRE
	}

	return temp, nil
}

func (p *Plugin) SaveState(data map[string]time.Time) error {
	p.m.Lock()
	defer p.m.Unlock()

	return core.PluginSaveState(p.Flow.FlowStateDir, &data, p.OptionMatchTTL)
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
		},
		PluginID:    pluginConfig.PluginID,
		PluginAlias: pluginConfig.PluginAlias,
		PluginName:  PLUGIN_NAME,
		PluginType:  pluginConfig.PluginType,
	}

	if pluginConfig.PluginType == "process" {
		plugin.LogFields["id"] = pluginConfig.PluginID
		plugin.LogFields["alias"] = pluginConfig.PluginAlias
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// "0" - will be set if parameter is set somehow (defaults, template, config etc.).
	availableParams := map[string]int{
		"cred":     -1,
		"template": -1,
		"timeout":  -1,

		"file_in":      -1,
		"file_in_mode": -1,
		"input":        1,
	}

	switch pluginConfig.PluginType {
	case "input":
		availableParams["expire_action"] = -1
		availableParams["expire_action_timeout"] = -1
		availableParams["expire_delay"] = -1
		availableParams["expire_interval"] = -1
		availableParams["match_signature"] = -1
		availableParams["match_ttl"] = -1
		availableParams["time_format"] = -1
		availableParams["time_format_a"] = -1
		availableParams["time_format_b"] = -1
		availableParams["time_format_c"] = -1
		availableParams["time_zone"] = -1
		availableParams["time_zone_a"] = -1
		availableParams["time_zone_b"] = -1
		availableParams["time_zone_c"] = -1

	case "process":
		availableParams["file_out"] = -1
		availableParams["file_out_mode"] = -1
		availableParams["file_out_wrap"] = -1
		availableParams["include"] = -1
		availableParams["output"] = 1
		availableParams["require"] = -1
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// -----------------------------------------------------------------------------------------------------------------

	switch pluginConfig.PluginType {
	case "input":
		// expire_action.
		setExpireAction := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["expire_action"] = 0
				plugin.OptionExpireAction = v
			}
		}
		setExpireAction(pluginConfig.AppConfig.GetStringSlice(core.VIPER_DEFAULT_EXPIRE_ACTION))
		setExpireAction(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.expire_action", template)))
		setExpireAction((*pluginConfig.PluginParams)["expire_action"])
		core.ShowPluginParam(plugin.LogFields, "expire_action", plugin.OptionExpireAction)

		// expire_action_delay.
		setExpireActionDelay := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["expire_action_delay"] = 0
				plugin.OptionExpireActionDelay = v
			}
		}
		setExpireActionDelay(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_EXPIRE_ACTION_DELAY))
		setExpireActionDelay(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.expire_action_delay", template)))
		setExpireActionDelay((*pluginConfig.PluginParams)["expire_action_delay"])
		core.ShowPluginParam(plugin.LogFields, "expire_action_delay", plugin.OptionExpireActionDelay)

		// expire_action_timeout.
		setExpireActionTimeout := func(p interface{}) {
			if v, b := core.IsInt(p); b {
				availableParams["expire_action_timeout"] = 0
				plugin.OptionExpireActionTimeout = v
			}
		}
		setExpireActionTimeout(pluginConfig.AppConfig.GetInt(core.VIPER_DEFAULT_EXPIRE_ACTION_TIMEOUT))
		setExpireActionTimeout(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.expire_action_timeout", template)))
		setExpireActionTimeout((*pluginConfig.PluginParams)["expire_action_timeout"])
		core.ShowPluginParam(plugin.LogFields, "expire_action_timeout", plugin.OptionExpireActionTimeout)

		// expire_interval.
		setExpireInterval := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["expire_interval"] = 0
				plugin.OptionExpireInterval = v
			}
		}
		setExpireInterval(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_EXPIRE_INTERVAL))
		setExpireInterval(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.expire_interval", template)))
		setExpireInterval((*pluginConfig.PluginParams)["expire_interval"])
		core.ShowPluginParam(plugin.LogFields, "expire_interval", plugin.OptionExpireInterval)

		// match_signature.
		setMatchSignature := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["match_signature"] = 0
				plugin.OptionMatchSignature = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
			}
		}
		setMatchSignature(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.match_signature", template)))
		setMatchSignature((*pluginConfig.PluginParams)["match_signature"])
		core.ShowPluginParam(plugin.LogFields, "match_signature", plugin.OptionMatchSignature)
		core.SliceStringToUpper(&plugin.OptionMatchSignature)

		// match_ttl.
		setMatchTTL := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["match_ttl"] = 0
				plugin.OptionMatchTTL = time.Duration(v) * time.Millisecond
			}
		}
		setMatchTTL(DEFAULT_MATCH_TTL)
		setMatchTTL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.match_ttl", template)))
		setMatchTTL((*pluginConfig.PluginParams)["match_ttl"])
		core.ShowPluginParam(plugin.LogFields, "match_ttl", plugin.OptionMatchTTL)

		// time_format.
		setTimeFormat := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["time_format"] = 0
				plugin.OptionTimeFormat = v
			}
		}
		setTimeFormat(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
		setTimeFormat(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format", template)))
		setTimeFormat((*pluginConfig.PluginParams)["time_format"])
		core.ShowPluginParam(plugin.LogFields, "time_format", plugin.OptionTimeFormat)

		// time_format_a.
		setTimeFormatA := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["time_format_a"] = 0
				plugin.OptionTimeFormatA = v
			}
		}
		setTimeFormatA(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
		setTimeFormatA(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format_a", template)))
		setTimeFormatA((*pluginConfig.PluginParams)["time_format_a"])
		core.ShowPluginParam(plugin.LogFields, "time_format_a", plugin.OptionTimeFormatA)

		// time_format_b.
		setTimeFormatB := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["time_format_b"] = 0
				plugin.OptionTimeFormatB = v
			}
		}
		setTimeFormatB(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
		setTimeFormatB(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format_b", template)))
		setTimeFormatB((*pluginConfig.PluginParams)["time_format_b"])
		core.ShowPluginParam(plugin.LogFields, "time_format_b", plugin.OptionTimeFormatB)

		// time_format_c.
		setTimeFormatC := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["time_format_c"] = 0
				plugin.OptionTimeFormatC = v
			}
		}
		setTimeFormatC(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
		setTimeFormatC(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format_c", template)))
		setTimeFormatC((*pluginConfig.PluginParams)["time_format_c"])
		core.ShowPluginParam(plugin.LogFields, "time_format_c", plugin.OptionTimeFormatC)

		// time_zone.
		setTimeZone := func(p interface{}) {
			if v, b := core.IsTimeZone(p); b {
				availableParams["time_zone"] = 0
				plugin.OptionTimeZone = v
			}
		}
		setTimeZone(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
		setTimeZone(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone", template)))
		setTimeZone((*pluginConfig.PluginParams)["time_zone"])
		core.ShowPluginParam(plugin.LogFields, "time_zone", plugin.OptionTimeZone)

		// time_zone_a.
		setTimeZoneA := func(p interface{}) {
			if v, b := core.IsTimeZone(p); b {
				availableParams["time_zone_a"] = 0
				plugin.OptionTimeZoneA = v
			}
		}
		setTimeZoneA(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
		setTimeZoneA(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone_a", template)))
		setTimeZoneA((*pluginConfig.PluginParams)["time_zone_a"])
		core.ShowPluginParam(plugin.LogFields, "time_zone_a", plugin.OptionTimeZoneA)

		// time_zone_b.
		setTimeZoneB := func(p interface{}) {
			if v, b := core.IsTimeZone(p); b {
				availableParams["time_zone_b"] = 0
				plugin.OptionTimeZoneB = v
			}
		}
		setTimeZoneB(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
		setTimeZoneB(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone_b", template)))
		setTimeZoneB((*pluginConfig.PluginParams)["time_zone_b"])
		core.ShowPluginParam(plugin.LogFields, "time_zone_b", plugin.OptionTimeZoneB)

		// time_zone_c.
		setTimeZoneC := func(p interface{}) {
			if v, b := core.IsTimeZone(p); b {
				availableParams["time_zone_c"] = 0
				plugin.OptionTimeZoneC = v
			}
		}
		setTimeZoneC(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
		setTimeZoneC(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone_c", template)))
		setTimeZoneC((*pluginConfig.PluginParams)["time_zone_c"])
		core.ShowPluginParam(plugin.LogFields, "time_zone_c", plugin.OptionTimeZoneC)

	case "process":
		// file_out.
		setFileOut := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["file_out"] = 0
				plugin.OptionFileOut = v
			}
		}
		setFileOut(DEFAULT_FILE_OUT)
		setFileOut(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_out", template)))
		setFileOut((*pluginConfig.PluginParams)["file_out"])
		core.ShowPluginParam(plugin.LogFields, "file_out", plugin.OptionFileOut)

		// file_out_mode.
		setFileOutMode := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["file_out_mode"] = 0
				plugin.OptionFileOutMode = v
			}
		}
		setFileOutMode(DEFAULT_FILE_OUT_MODE)
		setFileOutMode(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_out_mode", template)))
		setFileOutMode((*pluginConfig.PluginParams)["file_out_mode"])
		core.ShowPluginParam(plugin.LogFields, "file_out_mode", plugin.OptionFileOutMode)

		// file_out_wrap.
		setFileOutWrap := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["file_out_wrap"] = 0
				plugin.OptionFileOutWrap = v
			}
		}
		setFileOutWrap(DEFAULT_FILE_OUT_WRAP)
		setFileOutWrap(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_out_wrap", template)))
		setFileOutWrap((*pluginConfig.PluginParams)["file_out_wrap"])
		core.ShowPluginParam(plugin.LogFields, "file_out_wrap", plugin.OptionFileOutWrap)

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

		// output.
		setOutput := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["output"] = 0
				plugin.OptionOutput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
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
	}

	// file_in.
	setFileIn := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["file_in"] = 0
			plugin.OptionFileIn = v
		}
	}
	setFileIn(DEFAULT_FILE_IN)
	setFileIn(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_in", template)))
	setFileIn((*pluginConfig.PluginParams)["file_in"])
	core.ShowPluginParam(plugin.LogFields, "file_in", plugin.OptionFileIn)

	// file_in_mode.
	setFileInMode := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["file_in_mode"] = 0
			plugin.OptionFileInMode = v
		}
	}
	setFileInMode(DEFAULT_FILE_IN_MODE)
	setFileInMode(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_in_mode", template)))
	setFileInMode((*pluginConfig.PluginParams)["file_in_mode"])
	core.ShowPluginParam(plugin.LogFields, "file_in_mode", plugin.OptionFileInMode)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.OptionInput = v
		}
	}
	setInput((*pluginConfig.PluginParams)["input"])
	core.ShowPluginParam(plugin.LogFields, "input", plugin.OptionInput)

	// text_wrap.
	setTextWrap := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["text_wrap"] = 0
			plugin.OptionTextWrap = v
		}
	}
	setTextWrap(DEFAULT_TEXT_WRAP)
	setTextWrap(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.text_wrap", template)))
	setTextWrap((*pluginConfig.PluginParams)["text_wrap"])
	core.ShowPluginParam(plugin.LogFields, "text_wrap", plugin.OptionTextWrap)

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

	if plugin.OptionFileInMode != "lines" && plugin.OptionFileInMode != "text" {
		return &Plugin{}, fmt.Errorf(ERROR_MODE_UNKNOWN.Error(), plugin.OptionFileInMode)
	}

	if pluginConfig.PluginType == "process" {
		if len(plugin.OptionInput) != len(plugin.OptionOutput) {
			return &Plugin{}, fmt.Errorf(
				"%s: %v, %v",
				core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput)
		}

		if plugin.OptionFileOutMode != "append" && plugin.OptionFileOutMode != "truncate" {
			return &Plugin{}, fmt.Errorf(ERROR_MODE_UNKNOWN.Error(), plugin.OptionFileOutMode)
		}
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
