### Description:

**rss** input plugin is intended for data gathering from RSS/Atom feeds.

### Data structure:

```go
type RssData struct {
	CATEGORIES  []string
	CONTENT     string
	DESCRIPTION string
	GUID        string
	LINK        string
	TITLE       string
}
```

### Generic parameters:

| Param                 | Required |  Type  | Template |        Default        | Description |
|:----------------------|:--------:|:------:|:--------:|:---------------------:|:------------|
| expire_action         |    -     | array  |    +     |          []           |             |
| expire_action_delay   |    -     | string |    +     |         "1d"          |             |
| expire_action_timeout |    -     |  int   |    +     |          30           |             |
| expire_interval       |    -     | string |    +     |         "7d"          |             |
| force                 |    -     |  bool  |    +     |         false         |             |
| force_count           |    -     |  int   |    +     |          100          |             |
| timeout               |    -     |  int   |    +     |          60           |             |
| time_format           |    -     | string |    +     | "15:04:05 02.01.2006" |             |
| time_zone             |    -     | string |    +     |         "UTC"         |             |


### Plugin parameters:

| Param      | Required |  Type  | Template |      Default      |            Example            | Description |
|:-----------|:--------:|:------:|:--------:|:-----------------:|:-----------------------------:|:------------|
| **input**  |    +     | array  |    +     |       "[]"        | ["http://tass.ru/rss/v2.xml"] |             |
| user_agent |    -     | string |    +     | "gosquito v1.0.0" |        "webchela 1.0"         |             |


### Config sample:

```toml

```

### Flow sample:

```yaml
```