### Available configuration options:

```toml
[default]
# Configuration file might be placed:
# 1. /etc/gosquito/config.toml
# 2. $HOME/.gosquito/config.toml
# 3. $(pwd)/config.toml

# Interval units suffixes:
# s - seconds, m - minutes, h - hours, d - days.
# Example: 10s, 120m, 48h, 365d 

# Size units suffixes:
# b - bytes, k - kilobytes, m - megabytes, g - gigabytes.
# Example: 512b, 1024k, 10m, 3g

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
#flow_conf               = "/path/to/config/flow/conf"

# Disable/enable flow by names, mutually exclusive.
#flow_disable            = ["flow1", "flow2", "flow3"]
#flow_enable             = ["flow1", "flow2", "flow3"]

# How often flows run.
#flow_interval           = "60s"

#log_level               = "INFO"

# Process plugins results will be send by default.
#plugin_include          = true

# Some plugins have their own persistent data/settings for proper work (Telegram, for instance).
#plugin_data             = "/path/to/config/plugin/data"

# Directory where plugins save their states. 
# States - it's about gosquito related features (as opposite to plugin_data).
#plugin_state            = "/path/to/config/plugin/state"

# Plugins use this dir for temporary data placing.
#plugin_temp             = "/path/to/config/plugin/temp"

# Maximum plugin execution time (seconds). Some plugins ignore this value and use their own timeout.
#plugin_timeout          = 60

# GOMAXPROCS.
#proc_num                = <cpu_cores>

# Time settings for DataItem.Timeformat (DataItem.Time keeps original source time unchanged). 
# It needs for representing DataItem datetime in user-defined format.
#time_format             = "15:04 02.01.2006"
#time_zone               = "UTC"

# Default user_agent for all compatible plugins.
#user_agent              = "gosquito v1.0.0"
```