package xpath

import (
	"fmt"
	"github.com/antchfx/htmlquery"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"golang.org/x/net/html"
	"reflect"
	"strings"
)

const (
	DEFAULT_FIND_ALL        = false
	DEFAULT_XPATH_HTML      = true
	DEFAULT_XPATH_HTML_SELF = true
	DEFAULT_XPATH_SEPARATOR = "\n"
)

type Plugin struct {
	Hash string
	Flow string

	ID    int
	Alias string

	File string
	Name string
	Type string

	Include bool
	Require []int

	FindAll        bool
	Input          []string
	Output         []string
	Xpath          [][]string
	XpathHtml      bool
	XpathHtmlSelf  bool
	XpathSeparator string
}

func findXpath(p *Plugin, xpaths []string, text string) (string, bool) {
	var doc *html.Node
	var err error

	temp := ""

	logError := func(data string, err error) {
		log.WithFields(log.Fields{
			"hash":   p.Hash,
			"flow":   p.Flow,
			"file":   p.File,
			"plugin": p.Name,
			"type":   p.Type,
			"id":     p.ID,
			"data":   data,
			"error":  err,
		}).Error(core.LOG_PLUGIN_DATA)
	}

	// Read document from file/string.
	if core.IsFile(text, "") {
		doc, err = htmlquery.LoadDoc(text)
	} else {
		doc, err = htmlquery.Parse(strings.NewReader(text))
	}

	// Find xpaths.
	if err == nil {
		for _, xpath := range xpaths {
			nodes, err := htmlquery.QueryAll(doc, xpath)

			if err == nil {
				for _, node := range nodes {
					if p.XpathHtml {
						temp += fmt.Sprintf(
							"%s%s", htmlquery.OutputHTML(node, p.XpathHtmlSelf), p.XpathSeparator)
					} else {
						temp += fmt.Sprintf("%s%s", htmlquery.InnerText(node), p.XpathSeparator)
					}
				}
			} else {
				logError(fmt.Sprintf("xpath: %s", xpath), err)
				return "", false
			}
		}

	} else {
		logError(fmt.Sprintf("xpath parse error"), err)
		return "", false
	}

	return temp, len(temp) > 0
}

func (p *Plugin) Do(data []*core.DataItem) ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)

	if len(data) == 0 {
		return temp, nil
	}

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		found := make([]bool, len(p.Input))

		for index, input := range p.Input {
			var ro reflect.Value

			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDataField(item, input)
			ro, _ = core.ReflectDataField(item, p.Output[index])

			// This plugin supports "string" and "[]string" data fields for matching.
			switch ri.Kind() {
			case reflect.String:
				if s, b := findXpath(p, p.Xpath[index], ri.String()); b {
					found[index] = true
					ro.SetString(s)
				} else {
					ro.SetString(s)
				}
			case reflect.Slice:
				somethingWasFound := false

				for i := 0; i < ri.Len(); i++ {
					if s, b := findXpath(p, p.Xpath[index], ri.Index(i).String()); b {
						somethingWasFound = true
						ro.Set(reflect.Append(ro, reflect.ValueOf(s)))
					} else {
						ro.Set(reflect.Append(ro, reflect.ValueOf(s)))
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

		if (p.FindAll && foundInAllInputs) || (!p.FindAll && foundInSomeInputs) {
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
	return false
}

func (p *Plugin) GetRequire() []int {
	return []int{0}
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Hash: pluginConfig.Hash,
		Flow: pluginConfig.Flow,

		ID:    pluginConfig.ID,
		Alias: pluginConfig.Alias,

		File: pluginConfig.File,
		Name: "xpath",
		Type: "process",
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

	// find_all.
	setFindAll := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["find_all"] = 0
			plugin.FindAll = v
		}
	}
	setFindAll(DEFAULT_FIND_ALL)
	setFindAll((*pluginConfig.Params)["find_all"])
	showParam("find_all", plugin.FindAll)

	// include.
	setInclude := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["include"] = 0
			plugin.Include = v
		}
	}
	setInclude(pluginConfig.Config.GetBool(core.VIPER_DEFAULT_PLUGIN_INCLUDE))
	setInclude((*pluginConfig.Params)["include"])
	showParam("include", plugin.Include)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.Input = v
		}
	}
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
	setOutput((*pluginConfig.Params)["output"])
	showParam("output", plugin.Output)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.Require = v

		}
	}
	setRequire((*pluginConfig.Params)["require"])
	showParam("require", plugin.Require)

	// xpath.
	setXpath := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["xpath"] = 0
			plugin.Xpath = core.ExtractXpathsIntoArrays(pluginConfig.Config, v)
		}
	}
	setXpath((*pluginConfig.Params)["xpath"])
	showParam("xpath", plugin.Xpath)

	// xpath_html.
	setXpathHtml := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["xpath_html"] = 0
			plugin.XpathHtml = v
		}
	}
	setXpathHtml(DEFAULT_XPATH_HTML)
	setXpathHtml((*pluginConfig.Params)["xpath_html"])
	showParam("xpath_html", plugin.XpathHtml)

	// xpath_html_self.
	setXpathHtmlSelf := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["xpath_html_self"] = 0
			plugin.XpathHtmlSelf = v
		}
	}
	setXpathHtmlSelf(DEFAULT_XPATH_HTML_SELF)
	setXpathHtmlSelf((*pluginConfig.Params)["xpath_html_self"])
	showParam("xpath_html_self", plugin.XpathHtmlSelf)

	// xpath_separator.
	setXpathSeparator := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["xpath_separator"] = 0
			plugin.XpathSeparator = v
		}
	}
	setXpathSeparator(DEFAULT_XPATH_SEPARATOR)
	setXpathSeparator((*pluginConfig.Params)["xpath_separator"])
	showParam("xpath_separator", plugin.XpathSeparator)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	// 1. "input, output, xpath" must have equal size.
	// 2. "input, output" values must have equal types.
	minLength := 10000
	maxLength := 0
	lengths := []int{len(plugin.Input), len(plugin.Output), len(plugin.Xpath)}

	for _, length := range lengths {
		if length > maxLength {
			maxLength = length
		}
		if length < minLength {
			minLength = length
		}
	}

	if minLength != maxLength {
		return &Plugin{}, fmt.Errorf(
			"%s: %v, %v, %v", core.ERROR_SIZE_MISMATCH.Error(), plugin.Input, plugin.Output, plugin.Xpath)

	} else if err := core.IsDataFieldsTypesEqual(&plugin.Input, &plugin.Output); err != nil {
		return &Plugin{}, err

	} else {
		core.SliceStringToUpper(&plugin.Input)
		core.SliceStringToUpper(&plugin.Output)
	}

	return &plugin, nil
}
