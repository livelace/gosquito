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
	DEFAULT_TWITTER_COUNT = 200
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
	config := oauth1.NewConfig(p.ConsumerKey, p.ConsumerSecret)
	token := oauth1.NewToken(p.AccessToken, p.AccessSecret)
	httpClient := config.Client(ctx, token)
	client := twitter.NewClient(httpClient)

	// background.
	go func() {
		temp, _, err = client.Timelines.UserTimeline(&twitter.UserTimelineParams{
			Count:      p.Count,
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
	case <-time.After(time.Duration(p.Timeout) * time.Second):
		return &temp, fmt.Errorf("timeout: %s", source)
	}

	return &temp, nil
}

type Plugin struct {
	m sync.Mutex

	Hash string
	Flow string

	File     string
	Name     string
	StateDir string
	Type     string

	ExpireAction        []string
	ExpireActionTimeout int
	ExpireDelay         int64
	ExpireInterval      int64
	ExpireLast          int64
	Force               bool

	AccessToken    string
	AccessSecret   string
	ConsumerKey    string
	ConsumerSecret string
	Count          int
	Input          []string
	Timeout        int
	TimeFormat     string
	TimeZone       *time.Location
	UserAgent      string
}

func (p *Plugin) Recv() ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)
	currentTime := time.Now().UTC()

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}

	// Fetch data from sources.
	for _, source := range p.Input {
		var lastTime time.Time

		// Check if we work with source first time.
		if v, ok := flowStates[source]; ok {
			lastTime = v.(time.Time)
		} else {
			lastTime = currentTime
		}
		newLastTime := lastTime

		// Try to fetch new tweets.
		tweets, err := fetchTweets(p, source)
		if err != nil {
			return temp, err
		}

		// Process fetched data in reverse order.
		for i := len(*tweets) - 1; i >= 0; i-- {
			item := (*tweets)[i]

			var itemTime time.Time
			var u, _ = uuid.NewRandom()

			itemTime, err = item.CreatedAtTime()
			if err != nil {
				return temp, err
			}

			// Process only new items.
			if itemTime.Unix() > lastTime.Unix() || p.Force {
				// Set last item time as a new checkpoint.
				// Items may arrive disordered.
				if itemTime.Unix() > newLastTime.Unix() {
					newLastTime = itemTime
				}

				// Derive various data.
				media := expandMedia(&item.Entities.Media)
				if item.ExtendedEntities != nil {
					media = append(media, expandMedia(&item.ExtendedEntities.Media)...)
				}

				tags := expandTag(&item.Entities.Hashtags)
				urls := expandURL(&item.Entities.Urls)

				temp = append(temp, &core.DataItem{
					FLOW:       p.Flow,
					LANG:       item.Lang,
					PLUGIN:     p.Name,
					SOURCE:     source,
					TIME:       itemTime,
					TIMEFORMAT: itemTime.In(p.TimeZone).Format(p.TimeFormat),
					UUID:       u,

					TWITTER: core.TwitterData{
						MEDIA: core.UniqueSliceValues(&media),
						TAGS:  core.UniqueSliceValues(&tags),
						TEXT:  strings.TrimSpace(item.FullText),
						URLS:  core.UniqueSliceValues(&urls),
					},
				})
			}
		}

		flowStates[source] = newLastTime

		log.WithFields(log.Fields{
			"hash":   p.Hash,
			"flow":   p.Flow,
			"file":   p.File,
			"plugin": p.Name,
			"type":   p.Type,
			"source": source,
			"data":   fmt.Sprintf("last update: %s, fetched data: %d", newLastTime, len(*tweets)),
		}).Debug(core.LOG_PLUGIN_STAT)
	}

	// Save updated flow states.
	if err := p.SaveState(flowStates); err != nil {
		return temp, err
	}

	// Check every source for expiration.
	sourcesExpired := false

	// Check if any source is expired.
	for source, sourceTime := range flowStates {
		if (currentTime.Unix() - sourceTime.(time.Time).Unix()) > p.ExpireInterval {
			sourcesExpired = true

			// Execute command if expire delay exceeded.
			// ExpireLast keeps last execution timestamp.
			if (currentTime.Unix() - p.ExpireLast) > p.ExpireDelay {
				p.ExpireLast = currentTime.Unix()

				// Execute command with args.
				// We don't worry about command return code.
				if len(p.ExpireAction) > 0 {
					cmd := p.ExpireAction[0]
					args := []string{p.Flow, source, fmt.Sprintf("%v", sourceTime.(time.Time).Unix())}
					args = append(args, p.ExpireAction[1:]...)

					output, err := core.ExecWithTimeout(cmd, args, p.ExpireActionTimeout)

					log.WithFields(log.Fields{
						"hash":   p.Hash,
						"flow":   p.Flow,
						"file":   p.File,
						"plugin": p.Name,
						"type":   p.Type,
						"source": source,
						"data": fmt.Sprintf(
							"expire_action: command: %s, arguments: %v, output: %s, error: %v",
							cmd, args, output, err),
					}).Debug(core.LOG_PLUGIN_STAT)
				}
			}
		}
	}

	// Inform about expiration.
	if sourcesExpired {
		return temp, core.ERROR_FLOW_EXPIRE
	}

	return temp, nil
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

func (p *Plugin) LoadState() (map[string]interface{}, error) {
	p.m.Lock()
	defer p.m.Unlock()

	return core.PluginLoadData(p.StateDir, p.Flow)
}

func (p *Plugin) SaveState(data map[string]interface{}) error {
	p.m.Lock()
	defer p.m.Unlock()

	return core.PluginSaveData(p.StateDir, p.Flow, data)
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Hash: pluginConfig.Hash,
		Flow: pluginConfig.Flow,

		File:     pluginConfig.File,
		Name:     "twitter",
		StateDir: pluginConfig.Config.GetString(core.VIPER_DEFAULT_STATE_DIR),
		Type:     "input",

		ExpireLast: 0,
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
		"template":              -1,
		"timeout":               -1,
		"time_format":           -1,
		"time_zone":             -1,

		"access_token":    1,
		"access_secret":   1,
		"consumer_key":    1,
		"consumer_secret": 1,
		"count":           -1,
		"cred":            -1,
		"input":           1,
		"user_agent":      -1,
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

	cred, _ := core.IsString((*pluginConfig.Params)["cred"])

	// access_token.
	setAccessToken := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["access_token"] = 0
			plugin.AccessToken = v
		}
	}
	setAccessToken(pluginConfig.Config.GetString(fmt.Sprintf("%s.access_token", cred)))
	setAccessToken((*pluginConfig.Params)["access_token"])

	// access_secret.
	setAccessSecret := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["access_secret"] = 0
			plugin.AccessSecret = v
		}
	}
	setAccessSecret(pluginConfig.Config.GetString(fmt.Sprintf("%s.access_secret", cred)))
	setAccessSecret((*pluginConfig.Params)["access_secret"])

	// consumer_key.
	setConsumerKey := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["consumer_key"] = 0
			plugin.ConsumerKey = v
		}
	}
	setConsumerKey(pluginConfig.Config.GetString(fmt.Sprintf("%s.consumer_key", cred)))
	setConsumerKey((*pluginConfig.Params)["consumer_key"])

	// consumer_secret.
	setConsumerSecret := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["consumer_secret"] = 0
			plugin.ConsumerSecret = v
		}
	}
	setConsumerSecret(pluginConfig.Config.GetString(fmt.Sprintf("%s.consumer_secret", cred)))
	setConsumerSecret((*pluginConfig.Params)["consumer_secret"])

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.Params)["template"])

	// expire_interval.
	setExpireInterval := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["expire_interval"] = 0
			plugin.ExpireInterval = v
		}
	}
	setExpireInterval(pluginConfig.Config.GetString(core.VIPER_DEFAULT_EXPIRE_INTERVAL))
	setExpireInterval(pluginConfig.Config.GetString(fmt.Sprintf("%s.expire_interval", template)))
	setExpireInterval((*pluginConfig.Params)["expire_interval"])
	showParam("expire_interval", plugin.ExpireInterval)

	// expire_action.
	setExpireAction := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["expire_action"] = 0
			plugin.ExpireAction = v
		}
	}
	setExpireAction(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.expire_action", template)))
	setExpireAction((*pluginConfig.Params)["expire_action"])
	showParam("expire_action", plugin.ExpireAction)

	// expire_action_timeout.
	setExpireActionTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["expire_action_timeout"] = 0
			plugin.ExpireActionTimeout = v
		}
	}
	setExpireActionTimeout(pluginConfig.Config.GetInt(core.VIPER_DEFAULT_EXPIRE_ACTION_TIMEOUT))
	setExpireActionTimeout(pluginConfig.Config.GetString(fmt.Sprintf("%s.expire_action_timeout", template)))
	setExpireActionTimeout((*pluginConfig.Params)["expire_action_timeout"])
	showParam("expire_action_timeout", plugin.ExpireActionTimeout)

	// expire_delay.
	setExpireDelay := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["expire_delay"] = 0
			plugin.ExpireDelay = v
		}
	}
	setExpireDelay(pluginConfig.Config.GetString(core.VIPER_DEFAULT_EXPIRE_DELAY))
	setExpireDelay(pluginConfig.Config.GetString(fmt.Sprintf("%s.expire_delay", template)))
	setExpireDelay((*pluginConfig.Params)["expire_delay"])
	showParam("expire_delay", plugin.ExpireDelay)

	// count.
	setCount := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["count"] = 0
			plugin.Count = v
		}
	}
	setCount(DEFAULT_TWITTER_COUNT)
	setCount(pluginConfig.Config.GetInt(fmt.Sprintf("%s.count", template)))
	setCount((*pluginConfig.Params)["count"])
	showParam("count", plugin.Count)

	// force.
	setForce := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["force"] = 0
			plugin.Force = v
		}
	}
	setForce(core.DEFAULT_FORCE_INPUT)
	setForce(pluginConfig.Config.GetString(fmt.Sprintf("%s.force", template)))
	setForce((*pluginConfig.Params)["force"])
	showParam("force", plugin.Force)

	// input.
	setInput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["input"] = 0
			plugin.Input = core.ExtractConfigVariableIntoArray(pluginConfig.Config, v)
		}
	}
	setInput(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.input", template)))
	setInput((*pluginConfig.Params)["input"])
	showParam("input", plugin.Input)

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.Timeout = v
		}
	}
	setTimeout(pluginConfig.Config.GetInt(core.VIPER_DEFAULT_PLUGIN_TIMEOUT))
	setTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.Params)["timeout"])
	showParam("timeout", plugin.Timeout)

	// time_format.
	setTimeFormat := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["time_format"] = 0
			plugin.TimeFormat = v
		}
	}
	setTimeFormat(pluginConfig.Config.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
	setTimeFormat(pluginConfig.Config.GetString(fmt.Sprintf("%s.time_format", template)))
	setTimeFormat((*pluginConfig.Params)["time_format"])
	showParam("time_format", plugin.TimeFormat)

	// time_zone.
	setTimeZone := func(p interface{}) {
		if v, b := core.IsTimeZone(p); b {
			availableParams["time_zone"] = 0
			plugin.TimeZone = v
		}
	}
	setTimeZone(pluginConfig.Config.GetString(core.VIPER_DEFAULT_TIME_ZONE))
	setTimeZone(pluginConfig.Config.GetString(fmt.Sprintf("%s.time_zone", template)))
	setTimeZone((*pluginConfig.Params)["time_zone"])
	showParam("time_zone", plugin.TimeZone)

	// user_agent.
	setUserAgent := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["user_agent"] = 0
			plugin.UserAgent = v
		}
	}
	setUserAgent(pluginConfig.Config.GetString(core.VIPER_DEFAULT_USER_AGENT))
	setUserAgent(pluginConfig.Config.GetString(fmt.Sprintf("%s.user_agent", template)))
	setUserAgent((*pluginConfig.Params)["user_agent"])
	showParam("user_agent", plugin.UserAgent)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
