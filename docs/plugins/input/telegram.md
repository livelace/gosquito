### Description:

**telegram** plugin is for data gathering from Telegram chats.

### Data structure:

```go
type TelegramData struct {
	MEDIA []string
	TEXT  string
	URL   string
}
```

### Generic parameters:

| Param                 | Required |   Type   | Template |        Default        | Description |
|:----------------------|:--------:|:--------:|:--------:|:---------------------:|:------------|
| expire_action         |    -     |  array   |    +     |          []           |             |
| expire_action_delay   |    -     | interval |    +     |         "1d"          |             |
| expire_action_timeout |    -     | seconds  |    +     |          30           |             |
| expire_interval       |    -     | interval |    +     |         "7d"          |             |
| force                 |    -     |   bool   |    +     |         false         |             |
| timeout               |    -     | seconds  |    +     |          60           |             |
| time_format           |    -     |  string  |    +     | "15:04:05 02.01.2006" |             |
| time_zone             |    -     |  string  |    +     |         "UTC"         |             |


### Plugin parameters:

| Param         | Required |  Type  | Cred | Template | Default |              Example               | Description |
|:--------------|:--------:|:------:|:----:|:--------:|:-------:|:----------------------------------:|:------------|
| api_id        |    +     | string |  +   |    -     |   ""    |              "90004"               |             |
| api_hash      |    +     | string |  +   |    -     |   ""    | "a0000000000000000000000000000002" |             |
| file_max_size |    -     |  size  |  -   |    +     |  "10m"  |                "1g"                |             |
| input         |    +     | array  |  -   |    +     |   []    |          ["breakingmash"]          |             |
| log_level     |    -     |  int   |  -   |    +     |    0    |                 90                 |             |


### Config sample:

```toml

```

### Flow sample:

```yaml
```