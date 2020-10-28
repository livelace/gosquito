package core

const (
	// -----------------------------------------------------------------------------------------------------------------

	APP_NAME    = "gosquito"
	APP_VERSION = "v1.0.0"

	// -----------------------------------------------------------------------------------------------------------------

	DEFAULT_ETC_PATH              = "/etc/gosquito"
	DEFAULT_EXPIRE_ACTION_TIMEOUT = 10
	DEFAULT_EXPIRE_DELAY          = "1d"
	DEFAULT_EXPIRE_INTERVAL       = "7d"
	DEFAULT_EXPORTER_LISTEN       = ":8080"
	DEFAULT_FLOW_INTERVAL         = "60s"
	DEFAULT_FLOW_NUMBER           = 1
	DEFAULT_FORCE_INPUT           = false
	DEFAULT_LOG_LEVEL             = "INFO"
	DEFAULT_LOG_TIME_FORMAT       = "02.01.2006 15:04:05.000"
	DEFAULT_LOOP_SLEEP            = 300
	DEFAULT_PLUGIN_INCLUDE        = true
	DEFAULT_PLUGIN_TIMEOUT        = 60
	DEFAULT_TIME_FORMAT           = "15:04:05 02.01.2006"
	DEFAULT_TIME_ZONE             = "UTC"
	DEFAULT_USER_AGENT            = APP_NAME + " " + APP_VERSION

	// -----------------------------------------------------------------------------------------------------------------

	LOG_CONFIG_INIT       = "config generated"
	LOG_FLOW_IGNORE       = "flow ignore"
	LOG_FLOW_LOCK_WARNING = "--- flow lock warning"
	LOG_FLOW_PROCESS      = "process data ..."
	LOG_FLOW_RECEIVE      = "receive data ..."
	LOG_FLOW_READ         = "flow read"
	LOG_FLOW_SEND         = "send data ..."
	LOG_FLOW_SKIP         = "--- flow skip"
	LOG_FLOW_START        = "--- flow start"
	LOG_FLOW_STAT         = "flow stat"
	LOG_FLOW_STOP         = "--- flow stop"
	LOG_PLUGIN_INIT       = "plugin init"
	LOG_PLUGIN_STAT       = "plugin stat"
	LOG_SET_VALUE         = "set value"

	// -----------------------------------------------------------------------------------------------------------------

	VIPER_DEFAULT_EXPIRE_ACTION   = "default.expire_action"
	VIPER_DEFAULT_EXPIRE_DELAY    = "default.expire_delay"
	VIPER_DEFAULT_EXPIRE_INTERVAL = "default.expire_interval"
	VIPER_DEFAULT_EXPORTER_LISTEN = "default.exporter_listen"
	VIPER_DEFAULT_FLOW_DIR        = "default.flow_dir"
	VIPER_DEFAULT_FLOW_DISABLE    = "default.flow_disable"
	VIPER_DEFAULT_FLOW_ENABLE     = "default.flow_enable"
	VIPER_DEFAULT_FLOW_INTERVAL   = "default.flow_interval"
	VIPER_DEFAULT_FLOW_NUMBER     = "default.flow_number"
	VIPER_DEFAULT_LOG_LEVEL       = "default.log_level"
	VIPER_DEFAULT_PLUGIN_DIR      = "default.plugin_dir"
	VIPER_DEFAULT_PLUGIN_INCLUDE  = "default.plugin_include"
	VIPER_DEFAULT_PLUGIN_TIMEOUT  = "default.plugin_timeout"
	VIPER_DEFAULT_PROC_MAX        = "default.proc_num"
	VIPER_DEFAULT_STATE_DIR       = "default.state_dir"
	VIPER_DEFAULT_TEMP_DIR        = "default.temp_dir"
	VIPER_DEFAULT_TIME_FORMAT     = "default.time_format"
	VIPER_DEFAULT_TIME_ZONE       = "default.time_zone"
	VIPER_DEFAULT_USER_AGENT      = "default.user_agent"

	// -----------------------------------------------------------------------------------------------------------------

	CONFIG_SAMPLE = `
`
)
