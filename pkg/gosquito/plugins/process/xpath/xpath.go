package xpathProcess

import (
	"errors"
	"fmt"
	"github.com/antchfx/htmlquery"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"golang.org/x/net/html"
	"reflect"
	"strings"
)

const (
	PLUGIN_NAME = "xpath"

	DEFAULT_FIND_ALL        = false
	DEFAULT_XPATH_HTML      = true
	DEFAULT_XPATH_HTML_SELF = true
	DEFAULT_XPATH_SEPARATOR = "\n"
)

var (
	ERROR_NODE_ERROR  = errors.New("xpath node error: %s")
	ERROR_PARSE_ERROR = errors.New("xpath parse error: %s")
)

func findXpath(p *Plugin, xpaths []string, text string) (string, bool) {
	var doc *html.Node
	var err error

	result := ""

	// Read document from file/string.
	if core.IsFile(text, "") {
		doc, err = htmlquery.LoadDoc(text)
	} else {
		doc, err = htmlquery.Parse(strings.NewReader(text))
	}

	if err != nil {
		core.LogProcessPlugin(p.LogFields, fmt.Errorf(ERROR_PARSE_ERROR.Error(), err))
		return "", false
	}

	// Find xpaths.
	for _, xpath := range xpaths {
		nodes, err := htmlquery.QueryAll(doc, xpath)

		if err != nil {
			core.LogProcessPlugin(p.LogFields, fmt.Errorf(ERROR_NODE_ERROR.Error(), err))
			return "", false
		}

		for _, node := range nodes {
			if p.OptionXpathHtml {
				result += fmt.Sprintf("%s%s",
					htmlquery.OutputHTML(node, p.OptionXpathHtmlSelf), p.OptionXpathSeparator)

			} else {
				result += fmt.Sprintf("%s%s",
					htmlquery.InnerText(node), p.OptionXpathSeparator)
			}
		}
	}

	return result, len(result) > 0
}

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionFindAll        bool
	OptionInclude        bool
	OptionInput          []string
	OptionOutput         []string
	OptionRequire        []int
	OptionXpath          [][]string
	OptionXpathHtml      bool
	OptionXpathHtmlSelf  bool
	OptionXpathSeparator string
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
	return false
}

func (p *Plugin) GetRequire() []int {
	return []int{0}
}

func (p *Plugin) Process(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)
  p.LogFields["run"] = p.Flow.GetRunID()

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		found := make([]bool, len(p.OptionInput))

		for index, input := range p.OptionInput {
			// Reflect "input" plugin data fields.
			// Error ignored because we always check fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ := core.ReflectDataField(item, p.OptionOutput[index])

			switch ri.Kind() {
			case reflect.String:
				if result, ok := findXpath(p, p.OptionXpath[index], ri.String()); ok {
					found[index] = true
					ro.SetString(result)
				}

			case reflect.Slice:
				somethingWasFound := false

				for i := 0; i < ri.Len(); i++ {
					if result, ok := findXpath(p, p.OptionXpath[index], ri.Index(i).String()); ok {
						somethingWasFound = true
						ro.Set(reflect.Append(ro, reflect.ValueOf(result)))
					}
				}

				found[index] = somethingWasFound
			}
		}

		// Append replaced item to results.
		foundInSomeInputs := false
		foundInAllInputs := true

		for _, b := range found {
			if b {
				foundInSomeInputs = true
			} else {
				foundInAllInputs = false
			}
		}

		if (p.OptionFindAll && foundInAllInputs) || (!p.OptionFindAll && foundInSomeInputs) {
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
		"include": -1,
		"require": -1,

		"find_all":        -1,
		"input":           1,
		"output":          1,
		"xpath":           1,
		"xpath_html":      -1,
		"xpath_separator": -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	// find_all.
	setFindAll := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["find_all"] = 0
			plugin.OptionFindAll = v
		}
	}
	setFindAll(DEFAULT_FIND_ALL)
	setFindAll((*pluginConfig.PluginParams)["find_all"])
	core.ShowPluginParam(plugin.LogFields, "find_all", plugin.OptionFindAll)

	// include.
	setInclude := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["include"] = 0
			plugin.OptionInclude = v
		}
	}
	setInclude(pluginConfig.AppConfig.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
	setInclude((*pluginConfig.PluginParams)["include"])
	core.ShowPluginParam(plugin.LogFields, "include", plugin.OptionInclude)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.OptionInput = v
		}
	}
	setInput((*pluginConfig.PluginParams)["input"])
	core.ShowPluginParam(plugin.LogFields, "input", plugin.OptionInput)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.OptionOutput = v
		}
	}
	setOutput((*pluginConfig.PluginParams)["output"])
	core.ShowPluginParam(plugin.LogFields, "output", plugin.OptionOutput)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.OptionRequire = v

		}
	}
	setRequire((*pluginConfig.PluginParams)["require"])
	core.ShowPluginParam(plugin.LogFields, "require", plugin.OptionRequire)

	// xpath.
	setXpath := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["xpath"] = 0
			plugin.OptionXpath = core.ExtractXpathsIntoArrays(pluginConfig.AppConfig, v)
		}
	}
	setXpath((*pluginConfig.PluginParams)["xpath"])
	core.ShowPluginParam(plugin.LogFields, "xpath", plugin.OptionXpath)

	// xpath_html.
	setXpathHtml := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["xpath_html"] = 0
			plugin.OptionXpathHtml = v
		}
	}
	setXpathHtml(DEFAULT_XPATH_HTML)
	setXpathHtml((*pluginConfig.PluginParams)["xpath_html"])
	core.ShowPluginParam(plugin.LogFields, "xpath_html", plugin.OptionXpathHtml)

	// xpath_html_self.
	setXpathHtmlSelf := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["xpath_html_self"] = 0
			plugin.OptionXpathHtmlSelf = v
		}
	}
	setXpathHtmlSelf(DEFAULT_XPATH_HTML_SELF)
	setXpathHtmlSelf((*pluginConfig.PluginParams)["xpath_html_self"])
	core.ShowPluginParam(plugin.LogFields, "xpath_html_self", plugin.OptionXpathHtmlSelf)

	// xpath_separator.
	setXpathSeparator := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["xpath_separator"] = 0
			plugin.OptionXpathSeparator = v
		}
	}
	setXpathSeparator(DEFAULT_XPATH_SEPARATOR)
	setXpathSeparator((*pluginConfig.PluginParams)["xpath_separator"])
	core.ShowPluginParam(plugin.LogFields, "xpath_separator", plugin.OptionXpathSeparator)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if len(plugin.OptionInput) != len(plugin.OptionOutput) && len(plugin.OptionOutput) != len(plugin.OptionXpath) {
		return &Plugin{}, fmt.Errorf(
			"%s: %v, %v, %v",
			core.ERROR_SIZE_MISMATCH.Error(), plugin.OptionInput, plugin.OptionOutput, plugin.OptionXpath)
	}

	if err := core.IsDataFieldsTypesEqual(&plugin.OptionInput, &plugin.OptionOutput); err != nil {
		return &Plugin{}, err
	}

	return &plugin, nil
}
