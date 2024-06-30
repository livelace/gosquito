package ioMulti

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
)

const (
	PLUGIN_NAME = "io"

	DEFAULT_DATA_APPEND     = false
	DEFAULT_FILE_IN         = false
	DEFAULT_FILE_IN_MODE    = "text"
	DEFAULT_FILE_IN_PRE     = ""
	DEFAULT_FILE_IN_POST    = ""
	DEFAULT_FILE_IN_SPLIT   = "\n"
	DEFAULT_FILE_OUT        = false
	DEFAULT_FILE_OUT_APPEND = false
	DEFAULT_FILE_OUT_MODE   = "text"
	DEFAULT_FILE_OUT_PRE    = ""
	DEFAULT_FILE_OUT_POST   = ""
	DEFAULT_FILE_OUT_SPLIT  = "\n"
	DEFAULT_MATCH_TTL       = "1d"
	DEFAULT_TEXT_MODE       = "text"
	DEFAULT_TEXT_PRE        = ""
	DEFAULT_TEXT_POST       = ""
	DEFAULT_TEXT_SPLIT      = "\n"
)

var (
	ERROR_COPY_TO_STRING = errors.New("cannot copy anything to string: %v")
	ERROR_MODE_UNKNOWN   = errors.New("mode unknown: %v")

	INFO_READ_FROM_FILE      = "read from file: %v, %v"
	INFO_WRITE_TO_FILE       = "write to file: %v, %v"
	INFO_WRITE_LINES_TO_FILE = "write lines to file: lines -> %v, %v"
)

func processText(p *Plugin, input []string) []string {
	r := make([]string, 0)

	for _, i := range input {
		if p.OptionTextMode == "split" {
			for _, l := range strings.Split(i, p.OptionTextSplit) {
				r = append(r, fmt.Sprintf("%s%s%s", p.OptionTextPre, l, p.OptionTextPost))
			}
		}

		if p.OptionTextMode == "text" {
			r = append(r, i)
		}
	}

	return r
}

func readFile(p *Plugin, input []string) ([]string, error) {
	r := make([]string, 0)

	for _, i := range input {
		if i == "" {
			core.LogProcessPlugin(p.LogFields,
				fmt.Sprintf(INFO_READ_FROM_FILE, i, "skip"))
			return r, nil
		}

		if _, err := core.IsFile(i); err != nil {
			core.LogProcessPlugin(p.LogFields, err)
			return r, err
		}

		ib, err := os.ReadFile(i)
		if err == nil {
			core.LogProcessPlugin(p.LogFields,
				fmt.Sprintf(INFO_READ_FROM_FILE, i, nil))

			text := strings.Trim(string(ib), "\n")

			if p.OptionFileInMode == "split" {
				for _, l := range strings.Split(text, p.OptionFileInSplit) {
					r = append(r, wrapFileIn(p, l))
				}
			}

			if p.OptionFileInMode == "text" {
				r = append(r, wrapFileIn(p, text))
			}

		} else {
			core.LogProcessPlugin(p.LogFields,
				fmt.Errorf(INFO_READ_FROM_FILE, i, err))
			return r, err
		}
	}

	return r, nil
}

func writeFile(p *Plugin, input []string, output []string) error {
	var of *os.File
	var err error

	for _, o := range output {
		if o == "" {
			core.LogProcessPlugin(p.LogFields,
				fmt.Sprintf(INFO_WRITE_LINES_TO_FILE, o, "skip"))
			continue
		}

		if p.OptionFileOutAppend {
			of, err = os.OpenFile(o, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
		} else {
			of, err = os.OpenFile(o, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		}
		defer of.Close()

		if err != nil {
			core.LogProcessPlugin(p.LogFields,
				fmt.Errorf(INFO_WRITE_TO_FILE, o, err))
			return err
		}

		for _, i := range input {
			if p.OptionFileOutMode == "split" {
				for _, line := range strings.Split(i, p.OptionFileOutSplit) {
					if b, err := of.WriteString(wrapFileOut(p, line)); err == nil {
						core.LogProcessPlugin(p.LogFields,
							fmt.Sprintf(INFO_WRITE_TO_FILE, o, b))
					} else {
						core.LogProcessPlugin(p.LogFields,
							fmt.Errorf(INFO_WRITE_TO_FILE, o, err))
						return err
					}
				}
			}

			if p.OptionFileOutMode == "text" {
				if b, err := of.WriteString(wrapFileOut(p, i)); err == nil {
					core.LogProcessPlugin(p.LogFields,
						fmt.Sprintf(INFO_WRITE_TO_FILE, o, b))
				} else {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(INFO_WRITE_TO_FILE, o, err))
					return err
				}
			}
		}
	}

	return nil
}

func wrapFileIn(p *Plugin, s string) string {
	return fmt.Sprintf("%s%s%s", p.OptionFileInPre, s, p.OptionFileInPost)
}

func wrapFileOut(p *Plugin, s string) string {
	return fmt.Sprintf("%s%s%s", p.OptionFileOutPre, s, p.OptionFileOutPost)
}

type Plugin struct {
	m sync.Mutex

	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionDataAppend          bool
	OptionExpireAction        []string
	OptionExpireActionDelay   int64
	OptionExpireActionTimeout int
	OptionExpireInterval      int64
	OptionExpireLast          int64
	OptionFileIn              bool
	OptionFileInMode          string
	OptionFileInPre           string
	OptionFileInPost          string
	OptionFileInSplit         string
	OptionFileOut             bool
	OptionFileOutAppend       bool
	OptionFileOutMode         string
	OptionFileOutPre          string
	OptionFileOutPost         string
	OptionFileOutSplit        string
	OptionInclude             bool
	OptionInput               []string
	OptionMatchSignature      []string
	OptionMatchTTL            time.Duration
	OptionOutput              []string
	OptionRequire             []int
	OptionTextMode            string
	OptionTextPre             string
	OptionTextPost            string
	OptionTextSplit           string
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
	p.LogFields["run"] = p.Flow.GetRunID()

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		for index, input := range p.OptionInput {
			ri, ierr := core.ReflectDatumField(item, input)
			ro, oerr := core.ReflectDatumField(item, p.OptionOutput[index])

			var ioError error

			// ---------------------------------------------------------------
			// 1. Input: Datum Field -> Output: Datum Field:
			if ierr == nil && oerr == nil {

				// 1.1 Input: Datum String -> Output: Datum String: ++
				if ri.Kind() == reflect.String && ro.Kind() == reflect.String {

					// 1.1.1 File -> File ++
					if p.OptionFileIn && p.OptionFileOut {
						d, err := readFile(p, []string{ri.String()})
						if err != nil {
							ioError = err
						}

						if err := writeFile(p, d, []string{ro.String()}); err != nil {
							ioError = err
						}
					}

					// 1.1.2 String -> String ++
					if !p.OptionFileIn && !p.OptionFileOut {
						if p.OptionDataAppend {
							ro.SetString(ro.String() + ri.String())
						} else {
							ro.SetString(ri.String())
						}
					}

					// 1.1.3. String -> File ++
					if !p.OptionFileIn && p.OptionFileOut {
						if err := writeFile(p, []string{ri.String()}, []string{ro.String()}); err != nil {
							ioError = err
						}
					}

					// 1.1.4 File -> String ++
					if p.OptionFileIn && !p.OptionFileOut {
						d, err := readFile(p, []string{ri.String()})
						if err != nil {
							ioError = err
						}

						if p.OptionDataAppend {
							ro.SetString(ro.String() + strings.Join(d, ""))
						} else {
							ro.SetString(strings.Join(d, ""))
						}
					}
				}

				// 1.2 Input: Datum Slice -> Output: Datum Slice: ++
				if ri.Kind() == reflect.Slice && ro.Kind() == reflect.Slice {

					// 1.2.1 [File1 .. FileN] -> [File1 .. FileN] ++
					if p.OptionFileIn && p.OptionFileOut {
						d, err := readFile(p, ri.Interface().([]string))
						if err != nil {
							ioError = err
						}

						if err := writeFile(p, d, ro.Interface().([]string)); err != nil {
							ioError = err
						}
					}

					// 1.2.2 [String1 .. StringN] -> [String1 .. StringN] ++
					if !p.OptionFileIn && !p.OptionFileOut {
						if p.OptionDataAppend {
							ro.Set(reflect.AppendSlice(ro, ri))
						} else {
							ro.Set(ri)
						}
					}

					// 1.2.3 [String1 .. StringN] -> [File1 .. FileN] ++
					if !p.OptionFileIn && p.OptionFileOut {
						if err := writeFile(p, ri.Interface().([]string), ro.Interface().([]string)); err != nil {
							ioError = err
						}
					}

					// 1.2.4 [File1 .. FileN] -> [String1 .. StringN] ++
					if p.OptionFileIn && !p.OptionFileOut {
						d, err := readFile(p, ri.Interface().([]string))
						if err != nil {
							ioError = err
						}

						if p.OptionDataAppend {
							ro.Set(reflect.AppendSlice(ro, reflect.ValueOf(d)))
						} else {
							ro.Set(reflect.ValueOf(d))
						}
					}
				}

				// 1.3 Input: Datum String -> Output: Datum Slice: ++
				if ri.Kind() == reflect.String && ro.Kind() == reflect.Slice {

					// 1.3.1 File -> [File1 .. FileN] ++
					if p.OptionFileIn && p.OptionFileOut {
						d, err := readFile(p, []string{ri.String()})
						if err != nil {
							ioError = err
						}

						if err := writeFile(p, d, ro.Interface().([]string)); err != nil {
							ioError = err
						}
					}

					// 1.3.2 String -> [String1 .. StringN] ++
					if !p.OptionFileIn && !p.OptionFileOut {
						if p.OptionDataAppend {
							ro.Set(reflect.Append(ro, reflect.ValueOf(ri.String())))
						} else {
							ro.Set(reflect.ValueOf([]string{ri.String()}))
						}
					}

					// 1.3.3 String -> [File1 .. FileN] ++
					if !p.OptionFileIn && p.OptionFileOut {
						if err := writeFile(p, []string{ri.String()}, ro.Interface().([]string)); err != nil {
							ioError = err
						}
					}

					// 1.3.4 File -> [String1 .. StringN] ++
					if p.OptionFileIn && !p.OptionFileOut {
						d, err := readFile(p, []string{ri.String()})
						if err != nil {
							ioError = err
						}

						if p.OptionDataAppend {
							ro.Set(reflect.AppendSlice(ro, reflect.ValueOf(d)))
						} else {
							ro.Set(reflect.ValueOf(d))
						}
					}
				}

				// 1.4 Input: Datum Slice -> Output: Datum String:
				if ri.Kind() == reflect.Slice && ro.Kind() == reflect.String {

					// 1.4.1 [File1 .. FileN] -> File ++
					if p.OptionFileIn && p.OptionFileOut {
						d, err := readFile(p, ri.Interface().([]string))
						if err != nil {
							ioError = err
						}

						if err := writeFile(p, d, []string{ro.String()}); err != nil {
							ioError = err
						}
					}

					// 1.4.2 [String1 .. StringN] -> String ++
					if !p.OptionFileIn && !p.OptionFileOut {
						if p.OptionDataAppend {
							ro.SetString(ro.String() + strings.Join(ri.Interface().([]string), ""))
						} else {
							ro.SetString(strings.Join(ri.Interface().([]string), ""))
						}
					}

					// 1.4.3 [String1 .. StringN] -> File ++
					if !p.OptionFileIn && p.OptionFileOut {
						if err := writeFile(p, ri.Interface().([]string), []string{ro.String()}); err != nil {
							ioError = err
						}
					}

					// 1.4.4 [File1 .. FileN] -> String ++
					if p.OptionFileIn && !p.OptionFileOut {
						d, err := readFile(p, ri.Interface().([]string))
						if err != nil {
							ioError = err
						}

						if p.OptionDataAppend {
							ro.SetString(ro.String() + strings.Join(d, ""))
						} else {
							ro.SetString(strings.Join(d, ""))
						}
					}
				}
			}

			// ---------------------------------------------------------------
			// 2. Input: Datum String -> Output: String: ++
			if ierr == nil && oerr != nil && ri.Kind() == reflect.String {

				// 2.1 File -> File ++
				if p.OptionFileIn && p.OptionFileOut {
					d, err := readFile(p, []string{ri.String()})
					if err != nil {
						ioError = err
					}

					if err := writeFile(p, d, []string{p.OptionOutput[index]}); err != nil {
						ioError = err
					}
				}

				// 2.2 String -> String ++
				if !p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}

				// 2.3 String -> File ++
				if !p.OptionFileIn && p.OptionFileOut {
					if err := writeFile(p, []string{ri.String()}, []string{p.OptionOutput[index]}); err != nil {
						ioError = err
					}
				}

				// 2.4 File -> String ++
				if p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}
			}

			// ---------------------------------------------------------------
			// 3. Input: String -> Output: Datum String: ++
			if ierr != nil && oerr == nil && ro.Kind() == reflect.String {

				// 3.1 File -> File ++
				if p.OptionFileIn && p.OptionFileOut {
					d, err := readFile(p, []string{p.OptionInput[index]})
					if err != nil {
						ioError = err
					}

					if err := writeFile(p, d, []string{ro.String()}); err != nil {
						ioError = err
					}
				}

				// 3.2 String -> String ++
				if !p.OptionFileIn && !p.OptionFileOut {
					if p.OptionDataAppend {
						ro.SetString(ro.String() + p.OptionInput[index])
					} else {
						ro.SetString(p.OptionInput[index])
					}
				}

				// 3.3 String -> File ++
				if !p.OptionFileIn && p.OptionFileOut {
					if err := writeFile(p, []string{p.OptionInput[index]}, []string{ro.String()}); err != nil {
						ioError = err
					}
				}

				// 3.4 File -> String ++
				if p.OptionFileIn && !p.OptionFileOut {
					d, err := readFile(p, []string{p.OptionInput[index]})
					if err != nil {
						ioError = err
					}

					if p.OptionDataAppend {
						ro.SetString(ro.String() + strings.Join(d, ""))
					} else {
						ro.SetString(strings.Join(d, ""))
					}
				}
			}

			// ---------------------------------------------------------------
			// 4. Input: Datum Slice -> Output: String ++
			if ierr == nil && oerr != nil && ri.Kind() == reflect.Slice {

				// 4.1 [File1 .. FileN] -> File ++
				if p.OptionFileIn && p.OptionFileOut {
					d, err := readFile(p, ri.Interface().([]string))
					if err != nil {
						ioError = err
					}

					if err := writeFile(p, d, []string{p.OptionOutput[index]}); err != nil {
						ioError = err
					}
				}

				// 4.2 [String1 .. StringN] -> String ++
				if !p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}

				// 4.3 [String1 .. StringN] -> File ++
				if !p.OptionFileIn && p.OptionFileOut {
					if err := writeFile(p, ri.Interface().([]string), []string{p.OptionOutput[index]}); err != nil {
						ioError = err
					}
				}

				// 4.4 [File1 .. FileN] -> String ++
				if p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}
			}

			// ---------------------------------------------------------------
			// 5. Input: String -> Output: Datum Slice ++
			if ierr != nil && oerr == nil && ro.Kind() == reflect.Slice {

				// 5.1 File -> [File1 .. FileN] ++
				if p.OptionFileIn && p.OptionFileOut {
					d, err := readFile(p, []string{p.OptionInput[index]})
					if err != nil {
						ioError = err
					}

					if err := writeFile(p, d, ro.Interface().([]string)); err != nil {
						ioError = err
					}
				}

				// 5.2 String -> [String1 .. StringN] ++
				if !p.OptionFileIn && !p.OptionFileOut {
					d := reflect.ValueOf(processText(p, []string{p.OptionInput[index]}))

					if p.OptionDataAppend {
						ro.Set(reflect.Append(ro, d))
					} else {
						ro.Set(d)
					}
				}

				// 5.3 String -> [File1 ... FileN] ++
				if !p.OptionFileIn && p.OptionFileOut {
					if err := writeFile(p, []string{p.OptionInput[index]}, ro.Interface().([]string)); err != nil {
						ioError = err
					}
				}

				// 5.4 File -> [String1 .. StringN] ++
				if p.OptionFileIn && !p.OptionFileOut {
					d, err := readFile(p, []string{p.OptionInput[index]})
					if err != nil {
						ioError = err
					}

					if p.OptionDataAppend {
						ro.Set(reflect.Append(ro, reflect.ValueOf(d)))
					} else {
						ro.Set(reflect.ValueOf(d))
					}
				}
			}

			// ---------------------------------------------------------------
			// 6. Input: String -> Output: String ++
			if ierr != nil && oerr != nil {

				// 6.1 File -> File ++
				if p.OptionFileIn && p.OptionFileOut {
					d, err := readFile(p, []string{p.OptionInput[index]})
					if err != nil {
						ioError = err
					}

					if err := writeFile(p, d, []string{p.OptionOutput[index]}); err != nil {
						ioError = err
					}
				}

				// 6.2 String -> String ++
				if !p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}

				// 6.3 String -> File ++
				if !p.OptionFileIn && p.OptionFileOut {
					if err := writeFile(p, []string{p.OptionInput[index]}, []string{p.OptionOutput[index]}); err != nil {
						ioError = err
					}
				}

				// 6.4 File -> String ++
				if p.OptionFileIn && !p.OptionFileOut {
					core.LogProcessPlugin(p.LogFields,
						fmt.Errorf(ERROR_COPY_TO_STRING.Error(), p.OptionOutput[index]))
				}
			}

			// ---------------------------------------------------------------

			if ioError == nil {
				temp = append(temp, item)
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

			if s, err := core.GetStringFromFile(source); err == nil {
				switch p.OptionFileInMode {
				case "split":
					for _, l := range strings.Split(s, p.OptionFileInSplit) {
						itemLines = append(itemLines, wrapFileIn(p, l))
					}
				case "text":
					itemText = wrapFileIn(p, s)
				}
			} else {
				failedSources = append(failedSources, source)
				core.LogProcessPlugin(p.LogFields, err)
				continue
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
					MTIME: itemMtime,
					SPLIT: itemLines,
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

		"file_in":       -1,
		"file_in_mode":  -1,
		"file_in_pre":   -1,
		"file_in_post":  -1,
		"file_in_split": -1,
		"input":         1,
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
		availableParams["data_append"] = -1
		availableParams["file_out"] = -1
		availableParams["file_out_append"] = -1
		availableParams["file_out_mode"] = -1
		availableParams["file_out_pre"] = -1
		availableParams["file_out_post"] = -1
		availableParams["file_out_split"] = -1
		availableParams["include"] = -1
		availableParams["output"] = 1
		availableParams["require"] = -1
		availableParams["text_mode"] = -1
		availableParams["text_pre"] = -1
		availableParams["text_post"] = -1
		availableParams["text_split"] = -1
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
		// data_append.
		setDataAppend := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["data_append"] = 0
				plugin.OptionDataAppend = v
			}
		}
		setDataAppend(DEFAULT_DATA_APPEND)
		setDataAppend(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.data_append", template)))
		setDataAppend((*pluginConfig.PluginParams)["data_append"])
		core.ShowPluginParam(plugin.LogFields, "data_append", plugin.OptionDataAppend)

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

		// file_out_append.
		setFileOutAppend := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["file_out_append"] = 0
				plugin.OptionFileOutAppend = v
			}
		}
		setFileOutAppend(DEFAULT_FILE_OUT_APPEND)
		setFileOutAppend(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_out_append", template)))
		setFileOutAppend((*pluginConfig.PluginParams)["file_out_append"])
		core.ShowPluginParam(plugin.LogFields, "file_out_append", plugin.OptionFileOutAppend)

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

		// file_out_pre.
		setFileOutPre := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["file_out_pre"] = 0
				plugin.OptionFileOutPre = v
			}
		}
		setFileOutPre(DEFAULT_FILE_OUT_PRE)
		setFileOutPre(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_out_pre", template)))
		setFileOutPre((*pluginConfig.PluginParams)["file_out_pre"])
		core.ShowPluginParam(plugin.LogFields, "file_out_pre", plugin.OptionFileOutPre)

		// file_out_post.
		setFileOutPost := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["file_out_post"] = 0
				plugin.OptionFileOutPost = v
			}
		}
		setFileOutPost(DEFAULT_FILE_OUT_POST)
		setFileOutPost(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_out_post", template)))
		setFileOutPost((*pluginConfig.PluginParams)["file_out_post"])
		core.ShowPluginParam(plugin.LogFields, "file_out_post", plugin.OptionFileOutPost)

		// file_out_split.
		setFileOutSplit := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["file_out_split"] = 0
				plugin.OptionFileOutSplit = v
			}
		}
		setFileOutSplit(DEFAULT_FILE_OUT_SPLIT)
		setFileOutSplit(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_out_split", template)))
		setFileOutSplit((*pluginConfig.PluginParams)["file_out_split"])
		core.ShowPluginParam(plugin.LogFields, "file_out_split", plugin.OptionFileOutSplit)

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

		// text_mode.
		setTextMode := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["text_mode"] = 0
				plugin.OptionTextMode = v
			}
		}
		setTextMode(DEFAULT_TEXT_MODE)
		setTextMode(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.text_mode", template)))
		setTextMode((*pluginConfig.PluginParams)["text_mode"])
		core.ShowPluginParam(plugin.LogFields, "text_mode", plugin.OptionTextMode)

		// text_pre.
		setTextPre := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["text_pre"] = 0
				plugin.OptionTextPre = v
			}
		}
		setTextPre(DEFAULT_TEXT_PRE)
		setTextPre(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.text_pre", template)))
		setTextPre((*pluginConfig.PluginParams)["text_pre"])
		core.ShowPluginParam(plugin.LogFields, "text_pre", plugin.OptionTextPre)

		// text_post.
		setTextPost := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["text_post"] = 0
				plugin.OptionTextPost = v
			}
		}
		setTextPost(DEFAULT_TEXT_POST)
		setTextPost(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.text_post", template)))
		setTextPost((*pluginConfig.PluginParams)["text_post"])
		core.ShowPluginParam(plugin.LogFields, "text_post", plugin.OptionTextPost)

		// text_split.
		setTextSplit := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["text_split"] = 0
				plugin.OptionTextSplit = v
			}
		}
		setTextSplit(DEFAULT_TEXT_SPLIT)
		setTextSplit(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.text_split", template)))
		setTextSplit((*pluginConfig.PluginParams)["text_split"])
		core.ShowPluginParam(plugin.LogFields, "text_split", plugin.OptionTextSplit)
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

	// file_in_pre.
	setFileInPre := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["file_in_pre"] = 0
			plugin.OptionFileInPre = v
		}
	}
	setFileInPre(DEFAULT_FILE_IN_PRE)
	setFileInPre(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_in_pre", template)))
	setFileInPre((*pluginConfig.PluginParams)["file_in_pre"])
	core.ShowPluginParam(plugin.LogFields, "file_in_pre", plugin.OptionFileInPre)

	// file_in_post.
	setFileInPost := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["file_in_post"] = 0
			plugin.OptionFileInPost = v
		}
	}
	setFileInPost(DEFAULT_FILE_IN_POST)
	setFileInPost(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_in_post", template)))
	setFileInPost((*pluginConfig.PluginParams)["file_in_post"])
	core.ShowPluginParam(plugin.LogFields, "file_in_post", plugin.OptionFileInPost)

	// file_in_split.
	setFileInSplit := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["file_in_split"] = 0
			plugin.OptionFileInSplit = v
		}
	}
	setFileInSplit(DEFAULT_FILE_IN_SPLIT)
	setFileInSplit(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.file_in_split", template)))
	setFileInSplit((*pluginConfig.PluginParams)["file_in_split"])
	core.ShowPluginParam(plugin.LogFields, "file_in_split", plugin.OptionFileInSplit)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.OptionInput = v
		}
	}
	setInput((*pluginConfig.PluginParams)["input"])
	core.ShowPluginParam(plugin.LogFields, "input", plugin.OptionInput)

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

	if plugin.OptionFileInMode != "split" && plugin.OptionFileInMode != "text" {
		return &Plugin{}, fmt.Errorf(ERROR_MODE_UNKNOWN.Error(), plugin.OptionFileInMode)
	}

	if pluginConfig.PluginType == "process" {
		if len(plugin.OptionInput) != len(plugin.OptionOutput) {
			return &Plugin{}, fmt.Errorf(
				"%s: %v, %v",
				core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput)
		}

		if plugin.OptionFileOutMode != "split" && plugin.OptionFileOutMode != "text" {
			return &Plugin{}, fmt.Errorf(ERROR_MODE_UNKNOWN.Error(), plugin.OptionFileOutMode)
		}
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
