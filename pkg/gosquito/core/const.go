package core

const (
	// -----------------------------------------------------------------------------------------------------------------

	APP_NAME    = "gosquito"
	APP_VERSION = "v3.0.0"

	// -----------------------------------------------------------------------------------------------------------------

	DEFAULT_CURRENT_PATH          = "."
	DEFAULT_DATA_DIR              = "data"
	DEFAULT_ETC_PATH              = "/etc/gosquito"
	DEFAULT_EXPIRE_ACTION_DELAY   = "1d"
	DEFAULT_EXPIRE_ACTION_TIMEOUT = 30
	DEFAULT_EXPIRE_INTERVAL       = "7d"
	DEFAULT_EXPORTER_LISTEN       = ":8080"
	DEFAULT_FLOW_CONF_DIR         = "conf"
	DEFAULT_FLOW_DATA_DIR         = "data"
	DEFAULT_FLOW_INSTANCE         = 1
	DEFAULT_FLOW_INTERVAL         = "5m"
	DEFAULT_FLOW_LIMIT            = 0
	DEFAULT_FORCE_INPUT           = false
	DEFAULT_FORCE_COUNT           = 100
	DEFAULT_LOG_LEVEL             = "INFO"
	DEFAULT_LOG_TIME_FORMAT       = "02.01.2006 15:04:05.000"
	DEFAULT_LOOP_SLEEP            = 300
	DEFAULT_PLUGIN_INCLUDE        = true
	DEFAULT_PLUGIN_TIMEOUT        = 60
	DEFAULT_STATE_DIR             = "state"
	DEFAULT_TEMP_DIR              = "temp"
	DEFAULT_TIME_FORMAT           = "15:04:05 02.01.2006"
	DEFAULT_TIME_ZONE             = "UTC"
	DEFAULT_USER_AGENT            = APP_NAME + " " + APP_VERSION

	// -----------------------------------------------------------------------------------------------------------------

	LOG_CONFIG_APPLY               = "config apply"
	LOG_CONFIG_ERROR               = "config error"
	LOG_FLOW_IGNORE                = "flow ignore"
	LOG_FLOW_INVALID               = "flow invalid"
	LOG_FLOW_LOCK_WARNING          = "--- flow lock warning"
	LOG_FLOW_SEND_NO_DATA          = "no data for sending"
	LOG_FLOW_SEND_NO_DATA_INCLUDED = "no data included for sending"
	LOG_FLOW_PROCESS               = "process data ..."
	LOG_FLOW_READ                  = "flow read"
	LOG_FLOW_RECEIVE               = "receive data ..."
	LOG_FLOW_SEND                  = "send data ..."
	LOG_FLOW_START                 = "--- flow start"
	LOG_FLOW_STAT                  = "flow stat"
	LOG_FLOW_STOP                  = "--- flow stop"
	LOG_FLOW_VALID                 = "flow valid"
	LOG_FLOW_WARN                  = "--- flow warn"
	LOG_PLUGIN_DATA                = "plugin data"
	LOG_PLUGIN_INIT                = "plugin init"
	LOG_SET_VALUE                  = "set value"

	// -----------------------------------------------------------------------------------------------------------------

	VIPER_DEFAULT_EXPIRE_ACTION         = "default.expire_action"
	VIPER_DEFAULT_EXPIRE_ACTION_DELAY   = "default.expire_action_delay"
	VIPER_DEFAULT_EXPIRE_ACTION_TIMEOUT = "default.expire_action_timeout"
	VIPER_DEFAULT_EXPIRE_INTERVAL       = "default.expire_interval"
	VIPER_DEFAULT_EXPORTER_LISTEN       = "default.exporter_listen"
	VIPER_DEFAULT_FLOW_CONF             = "default.flow_conf"
	VIPER_DEFAULT_FLOW_DATA             = "default.flow_data"
	VIPER_DEFAULT_FLOW_DISABLE          = "default.flow_disable"
	VIPER_DEFAULT_FLOW_ENABLE           = "default.flow_enable"
	VIPER_DEFAULT_FLOW_INSTANCE         = "default.flow_instance"
	VIPER_DEFAULT_FLOW_INTERVAL         = "default.flow_interval"
	VIPER_DEFAULT_FLOW_LIMIT            = "default.flow_limit"
	VIPER_DEFAULT_LOG_LEVEL             = "default.log_level"
	VIPER_DEFAULT_PLUGIN_INCLUDE        = "default.plugin_include"
	VIPER_DEFAULT_PLUGIN_TIMEOUT        = "default.plugin_timeout"
	VIPER_DEFAULT_PROC_NUM              = "default.proc_num"
	VIPER_DEFAULT_TIME_FORMAT           = "default.time_format"
	VIPER_DEFAULT_TIME_ZONE             = "default.time_zone"
	VIPER_DEFAULT_USER_AGENT            = "default.user_agent"

	// -----------------------------------------------------------------------------------------------------------------

	CONFIG_SAMPLE = `[default]
# Configuration file might be placed:
# 1. /etc/gosquito/config.toml
# 2. $HOME/.gosquito/config.toml
# 3. $(pwd)/config.toml

# Interval units suffixes:
# s - seconds, m - minutes, h - hours, d - days.
# Example: 10s, 120m, 48h, 365d 

# Set command execution for expired input plugin sources.
# First 3 arguments always added: <flow_name> <input_source> <source_timestamp>
#expire_action           = ["/path/to/executable", "arg4", "arg5", "arg6"]

# Set delay between command execution (interval). 
#expire_action_delay     = "1d"

# Command execution timeout (seconds).
#expire_action_timeout   = 30

# Source will be considered as expired (interval).
#expire_interval         = "7d"

# Bind prometheus exporter.
#exporter_listen         = ":8080"

# Path to flow configurations.
#flow_conf               = "/path/to/conf"

# Path to flow data.
#flow_data               = "/path/to/data"

# Disable/enable flow by names, mutually exclusive.
#flow_disable            = ["flow1", "flow2", "flow3"]
#flow_enable             = ["flow1", "flow2", "flow3"]

# How many flows may run in parallel (0 - no limits).
#flow_limit              = 0

# Default number of flow instances.
#flow_instance           = 1

# How often flow run.
#flow_interval           = "5m"

#log_level               = "DEBUG"

# Results of process plugins will be included for sending by default.
#plugin_include          = true

# Maximum plugin execution time (seconds). Some plugins ignore this value and use their own timeout.
#plugin_timeout          = 60

# GOMAXPROCS.
#proc_num                = <cpu_cores>

# Time settings for DataItem.Timeformat (DataItem.Time keeps original source time unchanged). 
# It needs for representing DataItem datetime in user-defined format.
#time_format             = "15:04 02.01.2006"
#time_zone               = "UTC"

# Default user_agent for all compatible plugins.
#user_agent              = "gosquito v3.0.0"
`
)
