package rssIn

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/mmcdole/gofeed"
)

const (
	PLUGIN_NAME = "rss"

	DEFAULT_MATCH_TTL  = "1d"
	DEFAULT_SSL_VERIFY = true
)

func fetchFeed(url string, userAgent string, sslVerify bool, timeout int) (*gofeed.Feed, error) {
	temp := &gofeed.Feed{}

	// context.
	c := make(chan error, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// http.
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !sslVerify}}
	client := http.Client{Transport: transport}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	// background.
	go func() {
		// toxiproxy is neat, btw :)
		res, err := client.Do(req)
		if err != nil {
			c <- err
			return
		}

		if res != nil {
			defer func() {
				ce := res.Body.Close()
				if ce != nil {
					c <- ce
					return
				}
			}()
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			c <- fmt.Errorf("http error: %s", res.Status)
			return
		}

		temp, err = gofeed.NewParser().Parse(res.Body)
		if err != nil {
			c <- err
			return
		}

		c <- nil
	}()

	// wait for completion.
	select {
	case <-ctx.Done():
	case err := <-c:
		if err != nil {
			return temp, fmt.Errorf("error: %s, %s", url, err)
		}
	case <-time.After(time.Duration(timeout) * time.Second):
		return temp, fmt.Errorf("timeout: %s", url)
	}

	return temp, nil
}

type Plugin struct {
	m sync.Mutex

	Flow *core.Flow

	LogFields log.Fields

	PluginName string
	PluginType string

	OptionExpireAction        []string
	OptionExpireActionDelay   int64
	OptionExpireActionTimeout int
	OptionExpireInterval      int64
	OptionExpireLast          int64
	OptionForce               bool
	OptionForceCount          int
	OptionInput               []string
	OptionMatchSignature      []string
	OptionMatchTTL            time.Duration
	OptionSSLVerify           bool
	OptionTimeFormat          string
	OptionTimeFormatA         string
	OptionTimeFormatB         string
	OptionTimeFormatC         string
	OptionTimeZone            *time.Location
	OptionTimeZoneA           *time.Location
	OptionTimeZoneB           *time.Location
	OptionTimeZoneC           *time.Location
	OptionTimeout             int
	OptionUserAgent           string
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

func (p *Plugin) GetInput() []string {
	return p.OptionInput
}

func (p *Plugin) GetName() string {
	return p.PluginName
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

func (p *Plugin) Receive() ([]*core.Datum, error) {
	currentTime := time.Now().UTC()
	failedSources := make([]string, 0)
	temp := make([]*core.Datum, 0)
	p.LogFields["run"] = p.Flow.GetRunID()

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}
	core.LogInputPlugin(p.LogFields, "all", fmt.Sprintf("states loaded: %d", len(flowStates)))

	// Source stat.
	sourceNewStat := make(map[string]int32)

	// Fetch data from sources.
	for _, source := range p.OptionInput {
		var sourceLastTime time.Time

		// Check if we work with source first time.
		if v, ok := flowStates[source]; ok {
			sourceLastTime = v
		} else {
			sourceLastTime = time.Unix(0, 0)
		}

		// Try to fetch new articles.
		feeds, err := fetchFeed(source, p.OptionUserAgent, p.OptionSSLVerify, p.OptionTimeout)
		if err != nil {
			failedSources = append(failedSources, source)
			core.LogInputPlugin(p.LogFields, source, err)
			continue
		}

		// Process only specific amount of articles from every source if force = true.
		var start = 0
		var end = len(feeds.Items) - 1

		if p.OptionForce {
			if len(feeds.Items) > p.OptionForceCount {
				end = start + p.OptionForceCount - 1
			}
		}

		// Process fetched data.
		for i := start; i <= end; i++ {
			var itemNew = false
			var itemSignature string
			var itemSignatureHash string
			var itemTime time.Time
			var u, _ = uuid.NewRandom()

			item := feeds.Items[i]

			// Item's update time has higher priority over publishing time.
			if item.UpdatedParsed != nil {
				itemTime = *item.UpdatedParsed
			} else {
				if item.PublishedParsed != nil {
					itemTime = *item.PublishedParsed
				} else {
					itemTime = currentTime
				}
			}

			// Process only new items. Two methods:
			// 1. Match item by user provided signature.
			// 2. Compare item timestamp with source timestamp.
			if len(p.OptionMatchSignature) > 0 {
				for _, v := range p.OptionMatchSignature {
					switch v {
					case "RSS.CONTENT":
						itemSignature += item.Content
					case "RSS.DESCRIPTION":
						itemSignature += item.Description
					case "RSS.LINK":
						itemSignature += item.Link
					case "RSS.TITLE":
						itemSignature += item.Title
					}
				}

				// set default value for signature if user provided wrong values.
				if len(itemSignature) == 0 {
					itemSignature += source
				}

				itemSignatureHash = core.HashString(&itemSignature)

				if _, ok := flowStates[itemSignatureHash]; !ok || p.OptionForce {
					// save item signature hash to state.
					flowStates[itemSignatureHash] = currentTime

					// update source timestamp.
					if itemTime.Unix() > sourceLastTime.Unix() {
						sourceLastTime = itemTime
					}

					itemNew = true
				}

			} else {
				if itemTime.Unix() > sourceLastTime.Unix() || p.OptionForce {
					sourceLastTime = itemTime
					itemNew = true
				}
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

					RSS: core.Rss{
						CATEGORIES:  item.Categories,
						CONTENT:     item.Content,
						DESCRIPTION: item.Description,
						GUID:        item.GUID,
						LINK:        item.Link,
						TITLE:       item.Title,
					},

					WARNINGS: make([]string, 0),
				})

				sourceNewStat[source] += 1
			}
		}

		// always update source timestamp.
		flowStates[source] = sourceLastTime

		core.LogInputPlugin(p.LogFields, source, fmt.Sprintf("last update: %s, received data: %d, new data: %d",
			sourceLastTime, len(feeds.Items), sourceNewStat[source]))
	}

	// Save updated flow states.
	if err := p.SaveState(flowStates); err != nil {
		return temp, err
	}

	// Check every source for expiration.
	sourcesExpired := false

	// Check if any source is expired.
	for source, sourceTime := range flowStates {
		if (currentTime.Unix() - sourceTime.Unix()) > p.OptionExpireInterval / 1000 {
			sourcesExpired = true

			// Execute command if expire delay exceeded.
			// ExpireLast keeps last execution timestamp.
			if (currentTime.Unix() - p.OptionExpireLast) > p.OptionExpireActionDelay / 1000 {
				p.OptionExpireLast = currentTime.Unix()

				// Execute command with args.
				// We don't worry about command return code.
				if len(p.OptionExpireAction) > 0 {
					cmd := p.OptionExpireAction[0]
					args := []string{p.Flow.FlowName, source, fmt.Sprintf("%v", sourceTime.Unix())}
					args = append(args, p.OptionExpireAction[1:]...)

					output, err := core.ExecWithTimeout(cmd, args, p.OptionExpireActionTimeout)

					core.LogInputPlugin(p.LogFields, source, fmt.Sprintf(
						"expire_action: command: %s, arguments: %v, output: %s, error: %v",
						cmd, args, output, err))
				}
			}
		}
	}

	// Inform about expiration.
	if sourcesExpired {
		return temp, core.ERROR_FLOW_EXPIRE
	}

	// Inform about sources failures.
	if len(failedSources) > 0 {
		return temp, core.ERROR_FLOW_SOURCE_FAIL
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
		PluginName:       PLUGIN_NAME,
		PluginType:       pluginConfig.PluginType,
		OptionExpireLast: 0,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// Will be set to "0" if parameter is set somehow (defaults, template, config).

	availableParams := map[string]int{
		"expire_action":         -1,
		"expire_action_delay":   -1,
		"expire_action_timeout": -1,
		"expire_interval":       -1,
		"force":                 -1,
		"force_count":           -1,
		"ssl_verify":            -1,
		"template":              -1,
		"time_format":           -1,
		"time_format_a":         -1,
		"time_format_b":         -1,
		"time_format_c":         -1,
		"time_zone":             -1,
		"time_zone_a":           -1,
		"time_zone_b":           -1,
		"time_zone_c":           -1,
		"timeout":               -1,

		"input":           1,
		"match_signature": -1,
		"match_ttl":       -1,
		"user_agent":      -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// -----------------------------------------------------------------------------------------------------------------

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

	// force.
	setForce := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["force"] = 0
			plugin.OptionForce = v
		}
	}
	setForce(core.DEFAULT_FORCE_INPUT)
	setForce(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.force", template)))
	setForce((*pluginConfig.PluginParams)["force"])
	core.ShowPluginParam(plugin.LogFields, "force", plugin.OptionForce)

	// force_count.
	setForceCount := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["force_count"] = 0
			plugin.OptionForceCount = v
		}
	}
	setForceCount(core.DEFAULT_FORCE_COUNT)
	setForceCount(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.force_count", template)))
	setForceCount((*pluginConfig.PluginParams)["force_count"])
	core.ShowPluginParam(plugin.LogFields, "force_count", plugin.OptionForceCount)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.OptionInput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.input", template)))
	setInput((*pluginConfig.PluginParams)["input"])
	core.ShowPluginParam(plugin.LogFields, "input", plugin.OptionInput)

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

	// ssl_verify.
	setSSLVerify := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["ssl_verify"] = 0
			plugin.OptionSSLVerify = v
		}
	}
	setSSLVerify(DEFAULT_SSL_VERIFY)
	setSSLVerify(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.ssl_verify", template)))
	setSSLVerify((*pluginConfig.PluginParams)["ssl_verify"])
	core.ShowPluginParam(plugin.LogFields, "ssl_verify", plugin.OptionSSLVerify)

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


	// user_agent.
	setUserAgent := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["user_agent"] = 0
			plugin.OptionUserAgent = v
		}
	}
	setUserAgent(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_USER_AGENT))
	setUserAgent(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.user_agent", template)))
	setUserAgent((*pluginConfig.PluginParams)["user_agent"])
	core.ShowPluginParam(plugin.LogFields, "user_agent", plugin.OptionUserAgent)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
