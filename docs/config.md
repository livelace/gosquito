```toml
# Configuration file may be placed:
# 1. /etc/gosquito/config.toml
# 2. $HOME/.gosquito/config.toml
# 3. $(pwd)/config.toml

[default]

expire_action           = ["/path/to/executable", "arg1", "arg2"]
expire_action_delay     = "1d"
expire_action_timeout   = 30
expire_interval         = "7d"

exporter_listen         = ":8080"

flow_disable            = ["flow1"]
flow_enable             = ["flow1"]
flow_interval           = "60s"
flow_conf               = "/path/to/config/flow"
flow_state              = "/path/to/config/state"

log_level               = "INFO"

plugin_data             = "/path/to/config/plugin"
plugin_result_include   = true
plugin_temp             = "/path/to/config/temp"
plugin_timeout          = 60

proc_num                = 10

time_format             = "15:04 02.01.2006"
time_zone               = "UTC"

user_agent              = "gosquito v1.0.0"
```