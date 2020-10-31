### Description:

**twitter** input plugin is intended for data gathering from Twitter channels.

### Data structure:

```go
type TwitterData struct {
	LANG  string
	MEDIA []string
	TAGS  []string
	TEXT  string
	URLS  []string
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
| force_count           |    -     |   int    |    +     |          100          |             |
| timeout               |    -     | seconds  |    +     |          60           |             |
| time_format           |    -     |  string  |    +     | "15:04:05 02.01.2006" |             |
| time_zone             |    -     |  string  |    +     |         "UTC"         |             |


### Plugin parameters:

| Param               | Required |  Type  | Cred | Template |      Default      |     Example     | Description |
|:--------------------|:--------:|:------:|:----:|:--------:|:-----------------:|:---------------:|:------------|
| **access_token**    |    +     | string |  +   |    -     |        ""         |       ""        |             |
| **access_secret**   |    +     | string |  +   |    -     |        ""         |       ""        |             |
| **consumer_key**    |    +     | string |  +   |    -     |        ""         |       ""        |             |
| **consumer_secret** |    +     | string |  +   |    -     |        ""         |       ""        |             |
| **input**           |    +     | array  |  -   |    +     |        []         | ["tass_agency"] |             |
| user_agent          |    -     | string |  -   |    +     | "gosquito v1.0.0" | "webchela 1.0"  |             |


### Config sample:

```toml

```

### Flow sample:

```yaml
```