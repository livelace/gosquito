package core

import (
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------------------------------------------------

type InputPlugin interface {
	FlowLog(message interface{})

	GetInput() []string
	GetName() string

	LoadState() (map[string]time.Time, error)
	SaveState(map[string]time.Time) error

	Receive() ([]*DataItem, error)
}

type ProcessPlugin interface {
	FlowLog(message interface{})

	GetInclude() bool
	GetRequire() []int

	Process(d []*DataItem) ([]*DataItem, error)
}

type OutputPlugin interface {
	FlowLog(message interface{})

	GetName() string
	GetOutput() []string

	Send(d []*DataItem) error
}

// ---------------------------------------------------------------------------------------------------------------------

type PluginConfig struct {
	AppConfig    *viper.Viper
	Flow         *Flow
	PluginID     int
	PluginAlias  string
	PluginParams *map[string]interface{}
	PluginType   string
}

// ---------------------------------------------------------------------------------------------------------------------

type Flow struct {
	m        sync.Mutex
	instance int

	FlowUUID  uuid.UUID
	FlowHash  string
	FlowName  string
	FlowRunID int64

	FlowFile     string
	FlowDataDir  string
	FlowStateDir string
	FlowTempDir  string

	FlowCleanup  bool
	FlowInstance int
	FlowInterval int64

	InputPlugin         InputPlugin
	ProcessPlugins      map[int]ProcessPlugin
	ProcessPluginsNames []string
	OutputPlugin        OutputPlugin

	MetricError   int32
	MetricExpire  int32
	MetricNoData  int32
	MetricReceive int32
	MetricSend    int32
}

func (f *Flow) GetInstance() int {
	f.m.Lock()
	defer f.m.Unlock()

	return f.instance
}

func (f *Flow) GetRunID() int64 {
  return f.FlowRunID
}

func (f *Flow) ResetMetric() {
	f.MetricError = 0
	f.MetricExpire = 0
	f.MetricNoData = 0
	f.MetricReceive = 0
	f.MetricSend = 0
}

func (f *Flow) Lock() bool {
	f.m.Lock()
	defer f.m.Unlock()

	if f.instance == 0 || f.instance < f.FlowInstance {
		f.FlowRunID += 1
		f.instance += 1
		return true
	}

	return false
}

func (f *Flow) Unlock() bool {
	f.m.Lock()
	defer f.m.Unlock()

	if f.instance <= 0 {
		return false
	}

	f.instance -= 1
	return true
}

type FlowCandidate struct {
	Flow    *Flow
	Counter int64
}

type FlowUnmarshal struct {
	Flow struct {
		Name string `yaml:"name"`

		Params map[interface{}]interface{} `yaml:"params"`

		Input struct {
			Plugin string                      `yaml:"plugin"`
			Params map[interface{}]interface{} `yaml:"params"`
		}

		Process []map[interface{}]interface{} `yaml:"process"`

		Output struct {
			Plugin string                      `yaml:"plugin"`
			Params map[interface{}]interface{} `yaml:"params"`
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------------

type Data struct {
	ARRAY0 []string
	ARRAY1 []string
	ARRAY2 []string
	ARRAY3 []string
	ARRAY4 []string
	ARRAY5 []string
	ARRAY6 []string
	ARRAY7 []string
	ARRAY8 []string
	ARRAY9 []string
	TEXT0  string
	TEXT1  string
	TEXT2  string
	TEXT3  string
	TEXT4  string
	TEXT5  string
	TEXT6  string
	TEXT7  string
	TEXT8  string
	TEXT9  string
}

type Iter struct {
	INDEX int
	VALUE string
}

type Resty struct {
	BODY       string
	PROTO      string
	STATUS     string
	STATUSCODE string
}

type Rss struct {
	CATEGORIES  []string
	CONTENT     string
	DESCRIPTION string
	GUID        string
	LINK        string
	TITLE       string
}

type Telegram struct {
	CHATID    string
	CHATTITLE string
	CHATTYPE  string

	MESSAGEID       string
	MESSAGEMEDIA    []string
	MESSAGESENDERID string
	MESSAGETEXT     string
	MESSAGETEXTURL  []string
	MESSAGETYPE     string
	MESSAGEURL      string

	USERID        string
	USERNAME      string
	USERTYPE      string
	USERFIRSTNAME string
	USERLASTNAME  string
	USERPHONE     string

	WARNINGS []string
}

type Twitter struct {
	LANG  string
	MEDIA []string
	TAGS  []string
	TEXT  string
	URLS  []string
}

type DataItem struct {
	FLOW       string
	PLUGIN     string
	SOURCE     string
	TIME       time.Time
	TIMEFORMAT string
	TIMEZONE   *time.Location
	UUID       uuid.UUID

	DATA Data
	ITER Iter

	RESTY    Resty
	RSS      Rss
	TELEGRAM Telegram
	TWITTER  Twitter
}

// ---------------------------------------------------------------------------------------------------------------------
