### Description:

**telegram** input plugin is intended for data gathering from Telegram
chats.

### Data structure:

```go
type TelegramData struct {
	MEDIA []string
	TEXT  string
	URL   string
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
| timeout               |    -     |  int   |    +     |          60           |
| time_format           |    -     | string |    +     | "15:04:05 02.01.2006" |
| time_zone             |    -     | string |    +     |         "UTC"         |


### Plugin parameters:

| Param         | Required |  Type  | Cred | Template | Default |     Example      | Description                                                                                                |
|:--------------|:--------:|:------:|:----:|:--------:|:-------:|:----------------:|:-----------------------------------------------------------------------------------------------------------|
| **api_id**    |    +     | string |  +   |    -     |   ""    |        ""        | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                            |
| **api_hash**  |    +     | string |  +   |    -     |   ""    |        ""        | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                            |
| file_max_size |    -     |  size  |  -   |    +     |  "10m"  |       "1g"       | Maximum file size for downloading.                                                                         |
| **input**     |    +     | array  |  -   |    +     |   []    | ["breakingmash"] | List of Telegram chats.                                                                                    |
| log_level     |    -     |  int   |  -   |    +     |    0    |        90        | [TDLib Log Level](https://core.telegram.org/tdlib/docs/classtd_1_1td__api_1_1set_log_verbosity_level.html) |


### Config sample:

```toml

```

### Flow sample:

```yaml
```

