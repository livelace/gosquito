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

	Receive() ([]*Datum, error)
}

type ProcessPlugin interface {
	FlowLog(message interface{})

	GetInclude() bool
	GetRequire() []int

	Process(d []*Datum) ([]*Datum, error)
}

type OutputPlugin interface {
	FlowLog(message interface{})

	GetName() string
	GetOutput() []string

	Send(d []*Datum) error
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

	MetricError   int64
	MetricExpire  int64
	MetricNoData  int64
	MetricReceive int64
	MetricRun     int64
	MetricSend    int64
	MetricTime    int64
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
	f.MetricRun = 0
	f.MetricSend = 0
	f.MetricTime = 0
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
	ARRAY0  []string
	ARRAY1  []string
	ARRAY2  []string
	ARRAY3  []string
	ARRAY4  []string
	ARRAY5  []string
	ARRAY6  []string
	ARRAY7  []string
	ARRAY8  []string
	ARRAY9  []string
	ARRAY10 []string
	ARRAY11 []string
	ARRAY12 []string
	ARRAY13 []string
	ARRAY14 []string
	ARRAY15 []string
	ARRAY16 []string
	ARRAY17 []string
	ARRAY18 []string
	ARRAY19 []string
	ARRAY20 []string
	ARRAY21 []string
	ARRAY22 []string
	ARRAY23 []string
	ARRAY24 []string
	ARRAY25 []string
	ARRAY26 []string
	ARRAY27 []string
	ARRAY28 []string
	ARRAY29 []string
	ARRAY30 []string

	ARRAYA []string
	ARRAYB []string
	ARRAYC []string
	ARRAYD []string
	ARRAYE []string
	ARRAYF []string
	ARRAYG []string
	ARRAYH []string
	ARRAYI []string
	ARRAYJ []string
	ARRAYK []string
	ARRAYL []string
	ARRAYM []string
	ARRAYN []string
	ARRAYO []string
	ARRAYP []string
	ARRAYQ []string
	ARRAYR []string
	ARRAYS []string
	ARRAYT []string
	ARRAYU []string
	ARRAYV []string
	ARRAYW []string
	ARRAYX []string
	ARRAYY []string
	ARRAYZ []string

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
	TEXT10 string
	TEXT11 string
	TEXT12 string
	TEXT13 string
	TEXT14 string
	TEXT15 string
	TEXT16 string
	TEXT17 string
	TEXT18 string
	TEXT19 string
	TEXT20 string
	TEXT21 string
	TEXT22 string
	TEXT23 string
	TEXT24 string
	TEXT25 string
	TEXT26 string
	TEXT27 string
	TEXT28 string
	TEXT29 string
	TEXT30 string

	TEXTA string
	TEXTB string
	TEXTC string
	TEXTD string
	TEXTE string
	TEXTF string
	TEXTG string
	TEXTH string
	TEXTI string
	TEXTJ string
	TEXTK string
	TEXTL string
	TEXTM string
	TEXTN string
	TEXTO string
	TEXTP string
	TEXTQ string
	TEXTR string
	TEXTS string
	TEXTT string
	TEXTU string
	TEXTV string
	TEXTW string
	TEXTX string
	TEXTY string
	TEXTZ string
}

type Iter struct {
	INDEX int
	VALUE string
}

type Io struct {
	MTIME string
	SPLIT []string
	TEXT  string
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
	CHATID               string
	CHATSOURCE           string
	CHATTYPE             string
	CHATTITLE            string
	CHATCLIENTDATA       string
	CHATPROTECTEDCONTENT string
	CHATLASTINBOXID      string
	CHATLASTOUTBOXID     string
	CHATMEMBERONLINE     string
	CHATMESSAGETTL       string
	CHATUNREADCOUNT      string
	CHATFIRSTSEEN        string
	CHATLASTSEEN         string

	MESSAGEID           string
	MESSAGEMEDIA        []string
	MESSAGEMIME         string
	MESSAGESENDERID     string
	MESSAGETEXT         string
	MESSAGETEXTMARKDOWN string
	MESSAGETEXTURL      []string
	MESSAGETIMESTAMP    string
	MESSAGETYPE         string
	MESSAGEURL          string

	USERID            string
	USERVERSION       string
	USERNAME          string
	USERTYPE          string
	USERLANG          string
	USERFIRSTNAME     string
	USERLASTNAME      string
	USERPHONE         string
	USERSTATUS        string
	USERACCESSIBLE    string
	USERCONTACT       string
	USERFAKE          string
	USERMUTUALCONTACT string
	USERSCAM          string
	USERSUPPORT       string
	USERVERIFIED      string
	USERRESTRICTION   string
	USERFIRSTSEEN     string
	USERLASTSEEN      string
}

type TelegramSendingStatus struct {
	MessageId    int64
	ErrorCode    int32
	ErrorMessage string
}

type Twitter struct {
	LANG  string
	MEDIA []string
	TAGS  []string
	TEXT  string
	URLS  []string
}

type Datum struct {
	FLOW        string
	PLUGIN      string
	SOURCE      string
	TIME        time.Time
	TIMEFORMAT  string
	TIMEFORMATA string
	TIMEFORMATB string
	TIMEFORMATC string
	TIMEZONE    *time.Location
	TIMEZONEA   *time.Location
	TIMEZONEB   *time.Location
	TIMEZONEC   *time.Location
	UUID        uuid.UUID

	DATA Data
	ITER Iter

	IO       Io
	RESTY    Resty
	RSS      Rss
	TELEGRAM Telegram
	TWITTER  Twitter

	WARNINGS []string
}

// ---------------------------------------------------------------------------------------------------------------------
