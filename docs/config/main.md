### Main configuration:

```toml
[default]
# Configuration file might be placed:
# 1. /etc/gosquito/config.toml
# 2. $HOME/.gosquito/config.toml
# 3. $(pwd)/config.toml

# Interval units suffixes:
# ms - milliseconds, s - seconds, m - minutes, h - hours, d - days.
# Example: 100ms, 10s, 120m, 48h, 365d 

# Set command execution for expired input plugin sources.
# First 3 arguments always added: <flow_name> <input_source> <source_timestamp>
#expire_action           = ["/path/to/executable", "arg4", "arg5", "arg6"]

# Set delay between command execution (interval). 
#expire_action_delay     = "1d"

# Command execution timeout (seconds).
#expire_action_timeout   = 30

# Source will be considered as expired (interval).
#expire_interval         = "7d"

# Bind prometheus metrics exporter.
#exporter_listen         = ":8080"

# Should temp data be cleaned up at the end of flow execution.
#flow_cleanup            = true

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

# Main loop sleep (milliseconds).
#loop_sleep              = 1000

# Should flow send results of processing plugins with output plugin by default.
#plugin_include          = false

# Maximum plugin execution time (seconds). Some plugins ignore this value and use their own timeout.
#plugin_timeout          = 60

# GOMAXPROCS.
#proc_num                = <cpu_cores>

# Time settings for Datum.Timeformat (Datum.Time keeps original source time unchanged). 
# It needs for representing Datum datetime in user-defined format.
#time_format             = "15:04 02.01.2006"
#time_zone               = "UTC"

# Default user_agent for all compatible plugins.
#user_agent              = "gosquito v4.4.0"
```
