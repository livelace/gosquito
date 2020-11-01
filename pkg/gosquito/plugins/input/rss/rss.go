package rssIn

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/mmcdole/gofeed"
	"net/http"
	"sync"
	"time"
)

func fetchFeed(url string, userAgent string, timeout int) (*gofeed.Feed, error) {
	temp := &gofeed.Feed{}

	// context.
	c := make(chan error, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// http.
	client := http.Client{}
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

	Hash string
	Flow string

	File     string
	Name     string
	StateDir string
	Type     string

	ExpireAction        []string
	ExpireActionDelay   int64
	ExpireActionTimeout int
	ExpireInterval      int64
	ExpireLast          int64
	Force               bool
	ForceCount          int
	Timeout             int
	TimeFormat          string
	TimeZone            *time.Location

	Input     []string
	UserAgent string
}

func (p *Plugin) Recv() ([]*core.DataItem, error) {
	temp := make([]*core.DataItem, 0)
	currentTime := time.Now().UTC()

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}

	// Delete irrelevant/obsolete sources.
	for source := range flowStates {
		if !core.IsValueInSlice(source, &p.Input) {
			delete(flowStates, source)
		}
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

		// Try to fetch new articles.
		feeds, err := fetchFeed(source, p.UserAgent, p.Timeout)
		if err != nil {
			return temp, err
		}

		// Grab only specific amount of articles from
		// every source, if p.Force = true.
		var start = 0
		var end = len(feeds.Items) - 1

		if p.Force {
			if len(feeds.Items) > p.ForceCount {
				end = start + p.ForceCount - 1
			}
		}

		// Process fetched data.
		for i := start; i <= end; i++ {
			item := feeds.Items[i]

			var itemTime time.Time
			var u, _ = uuid.NewRandom()

			// Item's update time has higher priority over publishing time.
			if item.UpdatedParsed != nil {
				itemTime = *item.UpdatedParsed

			} else {
				if item.PublishedParsed != nil {
					itemTime = *item.PublishedParsed
				} else {
					itemTime = time.Now().UTC()
				}
			}

			// Process only new items.
			if itemTime.Unix() > lastTime.Unix() || p.Force {

				// Set last item time as a new checkpoint.
				// Items may arrive disordered.
				if itemTime.Unix() > newLastTime.Unix() {
					newLastTime = itemTime
				}

				temp = append(temp, &core.DataItem{
					FLOW:       p.Flow,
					PLUGIN:     p.Name,
					SOURCE:     source,
					TIME:       itemTime,
					TIMEFORMAT: itemTime.In(p.TimeZone).Format(p.TimeFormat),
					UUID:       u,

					RSS: core.RssData{
						CATEGORIES:  item.Categories,
						CONTENT:     item.Content,
						DESCRIPTION: item.Description,
						GUID:        item.GUID,
						LINK:        item.Link,
						TITLE:       item.Title,
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
			"data":   fmt.Sprintf("last update: %s, fetched data: %d", newLastTime, len(feeds.Items)),
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
			if (currentTime.Unix() - p.ExpireLast) > p.ExpireActionDelay {
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

func (p *Plugin) GetInput() []string {
	return p.Input
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
		Name:     "rss",
		StateDir: pluginConfig.Config.GetString(core.VIPER_DEFAULT_PLUGIN_STATE),
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
		"expire_action_delay":   -1,
		"expire_action_timeout": -1,
		"expire_interval":       -1,
		"force":                 -1,
		"force_count":           -1,
		"template":              -1,
		"timeout":               -1,
		"time_format":           -1,
		"time_zone":             -1,

		"input":      1,
		"user_agent": -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	showParam := func(p string, v interface{}) {
		log.WithFields(log.Fields{
			"flow":   plugin.Flow,
			"file":   plugin.File,
			"plugin": plugin.Name,
			"type":   plugin.Type,
			"value":  fmt.Sprintf("%s: %v", p, v),
		}).Debug(core.LOG_SET_VALUE)
	}

	// -----------------------------------------------------------------------------------------------------------------
	template, _ := core.IsString((*pluginConfig.Params)["template"])

	// expire_action.
	setExpireAction := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["expire_action"] = 0
			plugin.ExpireAction = v
		}
	}
	setExpireAction(pluginConfig.Config.GetStringSlice(core.VIPER_DEFAULT_EXPIRE_ACTION))
	setExpireAction(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.expire_action", template)))
	setExpireAction((*pluginConfig.Params)["expire_action"])
	showParam("expire_action", plugin.ExpireAction)

	// expire_action_delay.
	setExpireActionDelay := func(p interface{}) {
		if v, b := core.IsInterval(p); b {
			availableParams["expire_action_delay"] = 0
			plugin.ExpireActionDelay = v
		}
	}
	setExpireActionDelay(pluginConfig.Config.GetString(core.VIPER_DEFAULT_EXPIRE_ACTION_DELAY))
	setExpireActionDelay(pluginConfig.Config.GetString(fmt.Sprintf("%s.expire_action_delay", template)))
	setExpireActionDelay((*pluginConfig.Params)["expire_action_delay"])
	showParam("expire_action_delay", plugin.ExpireActionDelay)

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

	// force_count.
	setForceCount := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["force_count"] = 0
			plugin.ForceCount = v
		}
	}
	setForceCount(core.DEFAULT_FORCE_COUNT)
	setForceCount(pluginConfig.Config.GetInt(fmt.Sprintf("%s.force_count", template)))
	setForceCount((*pluginConfig.Params)["force_count"])
	showParam("force_count", plugin.ForceCount)

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
