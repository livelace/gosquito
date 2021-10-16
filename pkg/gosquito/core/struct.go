package core

import (
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------------------------------------------------

type InputPlugin interface {
	GetFile() string
	GetInput() []string
	GetName() string
	GetType() string

	LoadState() (map[string]time.Time, error)
	SaveState(map[string]time.Time) error

	Recv() ([]*DataItem, error)
}

type ProcessPlugin interface {
	GetId() int
	GetAlias() string

	GetFile() string
	GetName() string
	GetType() string

	GetInclude() bool
	GetRequire() []int

	Do(d []*DataItem) ([]*DataItem, error)
}

type OutputPlugin interface {
	GetFile() string
	GetName() string
	GetOutput() []string
	GetType() string

	Send(d []*DataItem) error
}

// ---------------------------------------------------------------------------------------------------------------------

type PluginConfig struct {
	AppConfig    *viper.Viper
	Flow         *Flow
	PluginID     int
	PluginAlias  string
	PluginParams *map[string]interface{}
}

// ---------------------------------------------------------------------------------------------------------------------

type Flow struct {
	m      sync.Mutex
	number int

	FlowUUID uuid.UUID
	FlowHash string
	FlowName string

	FlowFile     string
	FlowDataDir  string
	FlowStateDir string
	FlowTempDir  string

	FlowInterval int64
	FlowNumber   int

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

func (f *Flow) GetNumber() int {
	return f.number
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

	if f.number == 0 || f.number < f.FlowNumber {
		f.number += 1
		return true
	} else {
		return false
	}
}

func (f *Flow) Unlock() bool {
	f.m.Lock()
	defer f.m.Unlock()

	if f.number <= 0 {
		return false
	} else {
		f.number -= 1
		return true
	}
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

type RssData struct {
	CATEGORIES  []string
	CONTENT     string
	DESCRIPTION string
	GUID        string
	LINK        string
	TITLE       string
}

type TelegramData struct {
	USERID   string
	USERNAME string
	USERTYPE string

	FIRSTNAME string
	LASTNAME  string
	PHONE     string

	MEDIA []string
	TEXT  string
	URL   string
}

type TwitterData struct {
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

	DATA     Data
	RSS      RssData
	TELEGRAM TelegramData
	TWITTER  TwitterData
}

// ---------------------------------------------------------------------------------------------------------------------
