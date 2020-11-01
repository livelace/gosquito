### Description:

**twitter** input plugin is intended for data gathering from Twitter
channels.

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

| Param                 | Required |  Type  | Template |        Default        |
|:----------------------|:--------:|:------:|:--------:|:---------------------:|
| expire_action         |    -     | array  |    +     |          []           |
| expire_action_delay   |    -     | string |    +     |         "1d"          |
| expire_action_timeout |    -     |  int   |    +     |          30           |
| expire_interval       |    -     | string |    +     |         "7d"          |
| force                 |    -     |  bool  |    +     |         false         |
| force_count           |    -     |  int   |    +     |          100          |
| timeout               |    -     |  int   |    +     |          60           |
| time_format           |    -     | string |    +     | "15:04:05 02.01.2006" |
| time_zone             |    -     | string |    +     |         "UTC"         |


### Plugin parameters:

| Param               | Required |  Type  | Cred | Template |      Default      |     Example     | Description                                                             |
|:--------------------|:--------:|:------:|:----:|:--------:|:-----------------:|:---------------:|:------------------------------------------------------------------------|
| **access_secret**   |    +     | string |  +   |    -     |        ""         |       ""        | [Twitter Api Access](https://developer.twitter.com/en/apply-for-access) |
| **access_token**    |    +     | string |  +   |    -     |        ""         |       ""        | [Twitter Api Access](https://developer.twitter.com/en/apply-for-access) |
| **consumer_key**    |    +     | string |  +   |    -     |        ""         |       ""        | [Twitter Api Access](https://developer.twitter.com/en/apply-for-access) |
| **consumer_secret** |    +     | string |  +   |    -     |        ""         |       ""        | [Twitter Api Access](https://developer.twitter.com/en/apply-for-access) |
| **input**           |    +     | array  |  -   |    +     |        []         | ["tass_agency"] | List of Twitter channels.                                               |
| user_agent          |    -     | string |  -   |    +     | "gosquito v1.0.0" | "webchela 1.0"  | Custom User-Agent for API access.                                       |


### Config sample:

```toml

```

### Flow sample:

```yaml
```

