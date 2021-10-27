package twitterIn

import (
	"context"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"strings"
	"sync"
	"time"
)

const (
	DEFAULT_MATCH_TTL = "1d"
)

func expandMedia(s *[]twitter.MediaEntity) []string {
	temp := make([]string, 0)

	for _, m := range *s {
		temp = append(temp, m.MediaURLHttps)

		for _, v := range m.VideoInfo.Variants {
			temp = append(temp, v.URL)
		}
	}

	return temp
}

func expandTag(s *[]twitter.HashtagEntity) []string {
	temp := make([]string, 0)

	for _, v := range *s {
		temp = append(temp, v.Text)
	}

	return temp
}

func expandURL(s *[]twitter.URLEntity) []string {
	temp := make([]string, 0)

	for _, v := range *s {
		temp = append(temp, v.ExpandedURL)
		temp = append(temp, v.DisplayURL)
		temp = append(temp, v.URL)
	}

	return temp
}

func fetchTweets(p *Plugin, source string) (*[]twitter.Tweet, error) {
	var temp []twitter.Tweet
	var err error

	// context.
	c := make(chan error, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// twitter client.
	// TODO: Add custom User-Agent support. Need a custom Transport with token support.
	config := oauth1.NewConfig(p.OptionConsumerKey, p.OptionConsumerSecret)
	token := oauth1.NewToken(p.OptionAccessToken, p.OptionAccessSecret)
	httpClient := config.Client(ctx, token)
	client := twitter.NewClient(httpClient)

	// background.
	go func() {
		temp, _, err = client.Timelines.UserTimeline(&twitter.UserTimelineParams{
			Count:      200,
			ScreenName: source,
			TweetMode:  "extended",
		})

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
			return &temp, fmt.Errorf("error: %s, %s", source, err)
		}
	case <-time.After(time.Duration(p.OptionTimeout) * time.Second):
		return &temp, fmt.Errorf("timeout: %s", source)
	}

	return &temp, nil
}

func logging(p *Plugin, source string, message interface{}) {
	_, ok := message.(error)

	if ok {
		log.WithFields(log.Fields{
			"hash":   p.Flow.FlowHash,
			"flow":   p.Flow.FlowName,
			"file":   p.Flow.FlowFile,
			"plugin": p.PluginName,
			"type":   p.PluginType,
			"source": source,
			"error":  fmt.Sprintf("%v", message),
		}).Error(core.LOG_PLUGIN_DATA)

	} else {
		log.WithFields(log.Fields{
			"hash":   p.Flow.FlowHash,
			"flow":   p.Flow.FlowName,
			"file":   p.Flow.FlowFile,
			"plugin": p.PluginName,
			"type":   p.PluginType,
			"source": source,
			"data":   fmt.Sprintf("%v", message),
		}).Debug(core.LOG_PLUGIN_DATA)
	}
}

type Plugin struct {
	m sync.Mutex

	Flow *core.Flow

	PluginName string
	PluginType string

	OptionAccessSecret        string
	OptionAccessToken         string
	OptionConsumerKey         string
	OptionConsumerSecret      string
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
	OptionTimeFormat          string
	OptionTimeZone            *time.Location
	OptionTimeout             int
	OptionUserAgent           string
}

func (p *Plugin) GetFile() string {
	return p.Flow.FlowFile
}

func (p *Plugin) GetInput() []string {
	return p.OptionInput
}

func (p *Plugin) GetName() string {
	return p.PluginName
}

func (p *Plugin) GetType() string {
	return p.PluginType
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

func (p *Plugin) Receive() ([]*core.DataItem, error) {
	currentTime := time.Now().UTC()
	failedSources := make([]string, 0)
	temp := make([]*core.DataItem, 0)

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}

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

		// Try to fetch new tweets.
		tweets, err := fetchTweets(p, source)
		if err != nil {
			failedSources = append(failedSources, source)
			logging(p, source, err)
			continue
		}

		// Process only specific amount of tweets from every source if force = true.
		var start = len(*tweets) - 1
		var end = 0

		if p.OptionForce {
			if len(*tweets) > p.OptionForceCount {
				end = start - p.OptionForceCount + 1
			}
		}

		// Process fetched data in reverse order.
		for i := start; i >= end; i-- {
			var itemNew = false
			var itemSignature string
			var itemSignatureHash string
			var itemTime time.Time
			var u, _ = uuid.NewRandom()

			item := (*tweets)[i]

			itemTime, err = item.CreatedAtTime()
			if err != nil {
				return temp, err
			}

			// Process only new items. Two methods:
			// 1. Match item by user provided signature.
			// 2. Compare item timestamp with source timestamp.
			if len(p.OptionMatchSignature) > 0 || p.OptionForce {
				itemSignature = source

				for _, v := range p.OptionMatchSignature {
					switch v {
					case "lang":
						itemSignature += item.Lang
						break
					case "text":
						itemSignature += item.Text
						break
					case "time":
						itemSignature += itemTime.String()
						break
					}
				}

				// set default value for signature if user provided wrong values.
				if itemSignature == source {
					itemSignature += item.Text + itemTime.String()
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
				media := expandMedia(&item.Entities.Media)
				if item.ExtendedEntities != nil {
					media = append(media, expandMedia(&item.ExtendedEntities.Media)...)
				}

				tags := expandTag(&item.Entities.Hashtags)
				urls := expandURL(&item.Entities.Urls)

				temp = append(temp, &core.DataItem{
					FLOW:       p.Flow.FlowName,
					PLUGIN:     p.PluginName,
					SOURCE:     source,
					TIME:       itemTime,
					TIMEFORMAT: itemTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
					UUID:       u,

					TWITTER: core.Twitter{
						LANG:  item.Lang,
						MEDIA: core.UniqueSliceValues(&media),
						TAGS:  core.UniqueSliceValues(&tags),
						TEXT:  strings.TrimSpace(item.FullText),
						URLS:  core.UniqueSliceValues(&urls),
					},
				})

				sourceNewStat[source] += 1
			}
		}

		flowStates[source] = sourceLastTime

		logging(p, source, fmt.Sprintf("last update: %s, received data: %d, new data: %d",
			sourceLastTime, len(*tweets), sourceNewStat[source]))
	}

	// Save updated flow states.
	if err := p.SaveState(flowStates); err != nil {
		return temp, err
	}

	// Check every source for expiration.
	sourcesExpired := false

	// Check if any source is expired.
	for source, sourceTime := range flowStates {
		if (currentTime.Unix() - sourceTime.Unix()) > p.OptionExpireInterval {
			sourcesExpired = true

			// Execute command if expire delay exceeded.
			// ExpireLast keeps last execution timestamp.
			if (currentTime.Unix() - p.OptionExpireLast) > p.OptionExpireActionDelay {
				p.OptionExpireLast = currentTime.Unix()

				// Execute command with args.
				// We don't worry about command return code.
				if len(p.OptionExpireAction) > 0 {
					cmd := p.OptionExpireAction[0]
					args := []string{p.Flow.FlowName, source, fmt.Sprintf("%v", sourceTime.Unix())}
					args = append(args, p.OptionExpireAction[1:]...)

					output, err := core.ExecWithTimeout(cmd, args, p.OptionExpireActionTimeout)

					logging(p, source, fmt.Sprintf(
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
		Flow:             pluginConfig.Flow,
		PluginName:       "twitter",
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
		"expire_action_timeout": -1,
		"expire_delay":          -1,
		"expire_interval":       -1,
		"force":                 -1,
		"force_count":           -1,
		"template":              -1,
		"timeout":               -1,
		"time_format":           -1,
		"time_zone":             -1,

		"access_token":    1,
		"access_secret":   1,
		"consumer_key":    1,
		"consumer_secret": 1,
		"cred":            -1,
		"input":           1,
		"match_signature": -1,
		"match_ttl":       -1,
		"user_agent":      -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	showParam := func(p string, v interface{}) {
		log.WithFields(log.Fields{
			"hash":   plugin.Flow.FlowHash,
			"flow":   plugin.Flow.FlowName,
			"file":   plugin.Flow.FlowFile,
			"plugin": plugin.PluginName,
			"type":   plugin.PluginType,
			"value":  fmt.Sprintf("%s: %v", p, v),
		}).Debug(core.LOG_SET_VALUE)
	}

	// -----------------------------------------------------------------------------------------------------------------

	cred, _ := core.IsString((*pluginConfig.PluginParams)["cred"])

	// access_token.
	setAccessToken := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["access_token"] = 0
			plugin.OptionAccessToken = v
		}
	}
	setAccessToken(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.access_token", cred)))
	setAccessToken((*pluginConfig.PluginParams)["access_token"])

	// access_secret.
	setAccessSecret := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["access_secret"] = 0
			plugin.OptionAccessSecret = v
		}
	}
	setAccessSecret(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.access_secret", cred)))
	setAccessSecret((*pluginConfig.PluginParams)["access_secret"])

	// consumer_key.
	setConsumerKey := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["consumer_key"] = 0
			plugin.OptionConsumerKey = v
		}
	}
	setConsumerKey(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.consumer_key", cred)))
	setConsumerKey((*pluginConfig.PluginParams)["consumer_key"])

	// consumer_secret.
	setConsumerSecret := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["consumer_secret"] = 0
			plugin.OptionConsumerSecret = v
		}
	}
	setConsumerSecret(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.consumer_secret", cred)))
	setConsumerSecret((*pluginConfig.PluginParams)["consumer_secret"])

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

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
	showParam("expire_action", plugin.OptionExpireAction)

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
	showParam("expire_action_delay", plugin.OptionExpireActionDelay)

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
	showParam("expire_action_timeout", plugin.OptionExpireActionTimeout)

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
	showParam("expire_interval", plugin.OptionExpireInterval)

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
	showParam("force", plugin.OptionForce)

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
	showParam("force_count", plugin.OptionForceCount)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.OptionInput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.input", template)))
	setInput((*pluginConfig.PluginParams)["input"])
	showParam("input", plugin.OptionInput)

	// match_signature.
	setMatchSignature := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["match_signature"] = 0
			plugin.OptionMatchSignature = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setMatchSignature(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.match_signature", template)))
	setMatchSignature((*pluginConfig.PluginParams)["match_signature"])
	showParam("match_signature", plugin.OptionMatchSignature)

	// match_ttl.
	setMatchTTL := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["match_ttl"] = 0
			plugin.OptionMatchTTL = time.Duration(v) * time.Second
		}
	}
	setMatchTTL(DEFAULT_MATCH_TTL)
	setMatchTTL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.match_ttl", template)))
	setMatchTTL((*pluginConfig.PluginParams)["match_ttl"])
	showParam("match_ttl", plugin.OptionMatchTTL)

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
	showParam("timeout", plugin.OptionTimeout)

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
	showParam("time_format", plugin.OptionTimeFormat)

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
	showParam("time_zone", plugin.OptionTimeZone)

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
	showParam("user_agent", plugin.OptionUserAgent)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
