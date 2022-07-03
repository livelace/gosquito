package sameProcess

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	edlib "github.com/hbollon/go-edlib"
	"github.com/liuzl/tokenizer"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
)

const (
	PLUGIN_NAME = "same"

	DEFAULT_SAME_ALGO       = "levenshtein"
	DEFAULT_SAME_ALL        = false
	DEFAULT_SAME_RATIO_MAX  = 100
	DEFAULT_SAME_RATIO_MIN  = 1
	DEFAULT_SAME_SHARE_MAX  = 100
	DEFAULT_SAME_SHARE_MIN  = 1
	DEFAULT_SAME_TOKENS_MAX = 30
	DEFAULT_SAME_TOKENS_MIN = 5
	DEFAULT_SAME_TTL        = "1h"

	DEFAULT_SETTINGS_DB = "settings.db"
	DEFAULT_STATE_DIR   = "state"
)

var (
	ERROR_WRONG_VALUE       = errors.New("%v: value must be between 1 and 100: %v")
	ERROR_UNKNOWN_ALGORITHM = errors.New("%v: unknown algorithm: %v")
)

func matchSimilarity(p *Plugin, states *map[string]time.Time, text string) bool {
	if len(*states) == 0 {
		return true
	}

	counter := float32(0)
	percent := float32(100) / float32(len(*states))

	for k := range *states {
		ratio, err := edlib.StringsSimilarity(k, text, p.OptionSameAlgoConst)
		ratio_human := ratio * 100

		if err == nil {
			if ratio_human >= p.OptionSameRatioMin && ratio_human <= p.OptionSameRatioMax || ratio == 0 {
				counter += 1
			} else {
				core.LogProcessPlugin(p.LogFields, fmt.Sprintf("%v <> %v, ratio: %v%%",
					text, k, ratio_human))
			}
		} else {
			return false
		}
	}
	matched_percents := counter * percent
	if matched_percents >= p.OptionSameShareMin && matched_percents <= p.OptionSameShareMax {
		core.LogProcessPlugin(p.LogFields, fmt.Sprintf(
			"ratio min: %v%%, ratio max: %v%%, share min: %v%%, share max: %v%%, share matched: %v%%, %v",
			p.OptionSameRatioMin, p.OptionSameRatioMax, p.OptionSameShareMin, p.OptionSameShareMax,
			matched_percents, text))
		return true
	}
	return false
}

func tokenizeToString(p *Plugin, text string) (bool, string) {
	result := ""
	tokens := tokenizer.Tokenize(text)

	if len(tokens) < p.OptionSameTokensMin {
		return false, ""
	}

	if len(tokens) <= p.OptionSameTokensMax {
		for _, token := range tokens {
			result += fmt.Sprintf("%s", token)
		}
		return true, strings.TrimSpace(result)
	}

	if len(tokens) > p.OptionSameTokensMax {
		for i := 0; i < p.OptionSameTokensMax; i++ {
			result += fmt.Sprintf("%s", tokens[i])
		}
		return true, strings.TrimSpace(result)
	}

	return false, ""
}

type Plugin struct {
	Flow *core.Flow

	LogFields log.Fields

	PluginID    int
	PluginAlias string
	PluginName  string
	PluginType  string

	OptionInclude       bool
	OptionInput         []string
	OptionSameAlgo      string
	OptionSameAll       bool
	OptionSameAlgoConst edlib.Algorithm
	OptionSameRatioMax  float32
	OptionSameRatioMin  float32
	OptionSameShareMax  float32
	OptionSameShareMin  float32
	OptionSameTokensMax int
	OptionSameTokensMin int
	OptionSameTTL       time.Duration
	OptionRequire       []int
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

	stateDir := filepath.Join(p.Flow.FlowDataDir, p.PluginType, p.PluginName)
	_ = core.CreateDirIfNotExist(stateDir)

	stateData := make(map[string]time.Time, 0)
	if err := core.PluginLoadState(stateDir, &stateData); err != nil {
		return temp, err
	}
	core.LogProcessPlugin(p.LogFields, fmt.Sprintf("states loaded: %v", len(stateData)))

	// Iterate over data items (articles, tweets etc.).
	for _, item := range data {
		matched := make([]bool, len(p.OptionInput))

		// Match pattern inside different data fields (Title, Content etc.).
		for index, input := range p.OptionInput {
			// Reflect "input" plugin data fields.
			// Error ignored because we always checks fields during plugin init.
			ri, _ := core.ReflectDatumField(item, input)

			// This plugin supports "string" and "[]string" data fields for matching.
			switch ri.Kind() {
			case reflect.String:
				if tv, ts := tokenizeToString(p, ri.String()); tv {
					if sv := matchSimilarity(p, &stateData, ts); sv {
						matched[index] = true
					}
					stateData[ts] = time.Now().UTC()
				}
			case reflect.Slice:
				somethingWasMatched := false

				for i := 0; i < ri.Len(); i++ {
					if tv, ts := tokenizeToString(p, ri.Index(i).String()); tv {
						if sv := matchSimilarity(p, &stateData, ts); sv {
							somethingWasMatched = true
						}
						stateData[ts] = time.Now().UTC()
					}
				}

				matched[index] = somethingWasMatched
			}
		}

		matchedInSomeInputs := false
		matchedInAllInputs := true

		for _, b := range matched {
			if b {
				matchedInSomeInputs = true
			} else {
				matchedInAllInputs = false
			}
		}

		if (p.OptionSameAll && matchedInAllInputs) || (!p.OptionSameAll && matchedInSomeInputs) {
			temp = append(temp, item)
		}
	}

	core.PluginSaveState(stateDir, &stateData, p.OptionSameTTL)

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

		"input":           1,
		"same_algo":       -1,
		"same_ratio_max":  -1,
		"same_ratio_min":  -1,
		"same_share_max":  -1,
		"same_share_min":  -1,
		"same_tokens_max": -1,
		"same_tokens_min": -1,
		"same_ttl":        -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// -----------------------------------------------------------------------------------------------------------------

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

	// same_algo.
	setSameAlgo := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["same_algo"] = 0
			plugin.OptionSameAlgo = v
		}
	}
	setSameAlgo(DEFAULT_SAME_ALGO)
	setSameAlgo(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.same_algo", template)))
	setSameAlgo((*pluginConfig.PluginParams)["same_algo"])
	core.ShowPluginParam(plugin.LogFields, "same_algo", plugin.OptionSameAlgo)

	switch strings.ToLower(plugin.OptionSameAlgo) {
	case "levenshtein":
		plugin.OptionSameAlgoConst = edlib.Levenshtein
	case "dameraulevenshtein":
		plugin.OptionSameAlgoConst = edlib.DamerauLevenshtein
	case "osadameraulevenshtein":
		plugin.OptionSameAlgoConst = edlib.OSADamerauLevenshtein
	case "lcs":
		plugin.OptionSameAlgoConst = edlib.Lcs
	case "hamming":
		plugin.OptionSameAlgoConst = edlib.Hamming
	case "jaro":
		plugin.OptionSameAlgoConst = edlib.Jaro
	case "jarowinkler":
		plugin.OptionSameAlgoConst = edlib.JaroWinkler
	case "cosine":
		plugin.OptionSameAlgoConst = edlib.Cosine
	case "jaccard":
		plugin.OptionSameAlgoConst = edlib.Jaccard
	case "sorensendice":
		plugin.OptionSameAlgoConst = edlib.SorensenDice
	case "qgram":
		plugin.OptionSameAlgoConst = edlib.Qgram
	default:
		return &Plugin{}, fmt.Errorf(ERROR_UNKNOWN_ALGORITHM.Error(), "same_algo", plugin.OptionSameAlgo)
	}

	// same_all.
	setSameAll := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["same_all"] = 0
			plugin.OptionSameAll = v
		}
	}
	setSameAll(DEFAULT_SAME_ALL)
	setSameAll(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.same_all", template)))
	setSameAll((*pluginConfig.PluginParams)["same_all"])
	core.ShowPluginParam(plugin.LogFields, "same_all", plugin.OptionSameAll)

	// same_share_max.
	setSameShareMax := func(p interface{}) {
		if v, b := core.IsFloat(p); b {
			availableParams["same_share_max"] = 0
			plugin.OptionSameShareMax = v
		}
	}
	setSameShareMax(DEFAULT_SAME_SHARE_MAX)
	setSameShareMax(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.same_share_max", template)))
	setSameShareMax((*pluginConfig.PluginParams)["same_share_max"])
	core.ShowPluginParam(plugin.LogFields, "same_share_max", plugin.OptionSameShareMax)

	// same_share_min.
	setSameShareMin := func(p interface{}) {
		if v, b := core.IsFloat(p); b {
			availableParams["same_share_min"] = 0
			plugin.OptionSameShareMin = v
		}
	}
	setSameShareMin(DEFAULT_SAME_SHARE_MIN)
	setSameShareMin(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.same_share_min", template)))
	setSameShareMin((*pluginConfig.PluginParams)["same_share_min"])
	core.ShowPluginParam(plugin.LogFields, "same_share_min", plugin.OptionSameShareMin)

	// same_ratio_max.
	setSameRatioMax := func(p interface{}) {
		if v, b := core.IsFloat(p); b {
			availableParams["same_ratio_max"] = 0
			plugin.OptionSameRatioMax = v
		}
	}
	setSameRatioMax(DEFAULT_SAME_RATIO_MAX)
	setSameRatioMax(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.same_ratio_max", template)))
	setSameRatioMax((*pluginConfig.PluginParams)["same_ratio_max"])
	core.ShowPluginParam(plugin.LogFields, "same_ratio_max", plugin.OptionSameRatioMax)

	// same_ratio_min.
	setSameRatioMin := func(p interface{}) {
		if v, b := core.IsFloat(p); b {
			availableParams["same_ratio_min"] = 0
			plugin.OptionSameRatioMin = v
		}
	}
	setSameRatioMin(DEFAULT_SAME_RATIO_MIN)
	setSameRatioMin(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.same_ratio_min", template)))
	setSameRatioMin((*pluginConfig.PluginParams)["same_ratio_min"])
	core.ShowPluginParam(plugin.LogFields, "same_ratio_min", plugin.OptionSameRatioMin)

	// same_tokens_max.
	setSameTokensMax := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["same_tokens_max"] = 0
			plugin.OptionSameTokensMax = v
		}
	}
	setSameTokensMax(DEFAULT_SAME_TOKENS_MAX)
	setSameTokensMax(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.same_tokens_max", template)))
	setSameTokensMax((*pluginConfig.PluginParams)["same_tokens_max"])
	core.ShowPluginParam(plugin.LogFields, "same_tokens_max", plugin.OptionSameTokensMax)

	// same_tokens_min.
	setSameTokensMin := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["same_tokens_min"] = 0
			plugin.OptionSameTokensMin = v
		}
	}
	setSameTokensMin(DEFAULT_SAME_TOKENS_MIN)
	setSameTokensMin(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.same_tokens_min", template)))
	setSameTokensMin((*pluginConfig.PluginParams)["same_tokens_min"])
	core.ShowPluginParam(plugin.LogFields, "same_tokens_min", plugin.OptionSameTokensMin)

	// same_ttl.
	setSameTTL := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["same_ttl"] = 0
			plugin.OptionSameTTL = time.Duration(v) * time.Millisecond
		}
	}
	setSameTTL(DEFAULT_SAME_TTL)
	setSameTTL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.same_ttl", template)))
	setSameTTL((*pluginConfig.PluginParams)["same_ttl"])
	core.ShowPluginParam(plugin.LogFields, "same_ttl", plugin.OptionSameTTL)

	// require.
	setRequire := func(p interface{}) {
		if v, b := core.IsSliceOfInt(p); b {
			availableParams["require"] = 0
			plugin.OptionRequire = v

		}
	}
	setRequire((*pluginConfig.PluginParams)["require"])
	core.ShowPluginParam(plugin.LogFields, "require", plugin.OptionRequire)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if plugin.OptionSameRatioMax < 1 || plugin.OptionSameRatioMax > 100 {
		return &Plugin{}, fmt.Errorf(ERROR_WRONG_VALUE.Error(), "same_ratio_max", plugin.OptionSameRatioMax)
	}

	if plugin.OptionSameRatioMin < 1 || plugin.OptionSameRatioMin > 100 {
		return &Plugin{}, fmt.Errorf(ERROR_WRONG_VALUE.Error(), "same_ratio_min", plugin.OptionSameRatioMin)
	}

	if plugin.OptionSameShareMax < 1 || plugin.OptionSameShareMax > 100 {
		return &Plugin{}, fmt.Errorf(ERROR_WRONG_VALUE.Error(), "same_share_max", plugin.OptionSameShareMax)
	}

	if plugin.OptionSameShareMin < 1 || plugin.OptionSameShareMin > 100 {
		return &Plugin{}, fmt.Errorf(ERROR_WRONG_VALUE.Error(), "same_share_min", plugin.OptionSameShareMin)
	}

	// -----------------------------------------------------------------------------------------------------------------
	resetState := false
	settings := make(map[string]int, 0)

	dataDir := filepath.Join(plugin.Flow.FlowDataDir, plugin.PluginType, plugin.PluginName)
	_ = core.CreateDirIfNotExist(dataDir)

	stateDir := filepath.Join(dataDir, DEFAULT_STATE_DIR)
	_ = core.CreateDirIfNotExist(stateDir)

	// Load previous settings and compare current and old values.
	if err := core.PluginLoadData(filepath.Join(dataDir, DEFAULT_SETTINGS_DB), &settings); err != nil {
		return &plugin, err
	}

	if settings["same_tokens_max"] != plugin.OptionSameTokensMax {
		settings["same_tokens_max"] = plugin.OptionSameTokensMax
		resetState = true
	}

	if settings["same_tokens_min"] != plugin.OptionSameTokensMin {
		settings["same_tokens_min"] = plugin.OptionSameTokensMin
		resetState = true
	}

	if settings["same_ttl"] != int(plugin.OptionSameTTL.Seconds()) {
		settings["same_ttl"] = int(plugin.OptionSameTTL.Seconds())
		resetState = true
	}

	// Reset states if settings have changed.
	if resetState {
		_ = os.RemoveAll(stateDir)
	}

	// Save updated settings.
	if err := core.PluginSaveData(filepath.Join(dataDir, DEFAULT_SETTINGS_DB), &settings); err != nil {
		return &plugin, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
