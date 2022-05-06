### Description:

**rss** input plugin is intended for data gathering from [RSS/Atom](https://en.wikipedia.org/wiki/RSS) feeds.

### Data structure:

```go
type Rss struct {
	CATEGORIES   []string
	CONTENT*     string
	DESCRIPTION* string
	GUID         string
	LINK*        string
	TITLE*       string
}
```

&ast; - field may be used with **match_signature** parameter.

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

| Param           | Required | Type   | Template | Default           | Example                        | Description                                |
|:----------------|:--------:|:------:|:--------:|:-----------------:|:------------------------------:|:-------------------------------------------|
| **input**       | +        | array  | +        | "[]"              | ["https://tass.ru/rss/v2.xml"] | List of RSS/Atom feeds.                    |
| match_signature | -        | array  | +        | "[]"              | ["rss.link", "rss.title"]      | Match new articles by signature.           |
| match_ttl       | -        | string | +        | "1d"              | "24h"                          | TTL (Time To Live) for matched signatures. |
| ssl_verify      | -        | bool   | +        | true              | false                          | Verify server certificate.                 |
| user_agent      | -        | string | +        | "gosquito v3.8.0" | "webchela 1.0"                 | Custom User-Agent for feed access.         |


### Flow sample:

```yaml
flow:
  name: "rss-example"

  input:
    plugin: "rss"
    params:
      input: ["https://tass.ru/rss/v2.xml"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "echo"
      alias: "show title and url"
      params:
        input: ["rss.title", "rss.link"]

```


