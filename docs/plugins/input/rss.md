### Description:

**rss** input plugin is intended for data gathering from [RSS/Atom](https://en.wikipedia.org/wiki/RSS) feeds.

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

| Param      | Required |  Type  | Template |      Default      |            Example             | Description                        |
|:-----------|:--------:|:------:|:--------:|:-----------------:|:------------------------------:|:-----------------------------------|
| **input**  |    +     | array  |    +     |       "[]"        | ["https://tass.ru/rss/v2.xml"] | List of RSS/Atom feeds.            |
| ssl_verify |    -     | bool   |    +     |       true        |             false              | Verify server certificate.         |
| user_agent |    -     | string |    +     | "gosquito v1.0.0" |         "webchela 1.0"         | Custom User-Agent for feed access. |


### Flow sample:

```yaml
flow:
  name: "rss-example"

  input:
    plugin: "rss"
    params:
      input: ["http://tass.ru/rss/v2.xml"]
      force: true
      force_count: 10

  output:
    plugin: "smtp"
    params:
      template: "templates.rss.smtp.default"
```

### Config sample:

```toml
[templates.rss.smtp.default]
server = "mail.example.com"

from = "gosquito@example.com"
output = ["user@example.com"]

subject = "{{ .RSS.TITLE }}"

body = """
    <div align="right"><b>{{ .FLOW }}&nbsp;&nbsp;&nbsp;{{ .TIMEFORMAT }}</b></div>
    {{ .RSS.TITLE }}<br>
    {{ if .RSS.DESCRIPTION }}{{ .RSS.DESCRIPTION }}<br>{{end}}
    {{ if .RSS.CONTENT }}{{ .RSS.CONTENT }}<br><br>{{else}}<br>{{end}}
    {{ if .RSS.LINK }}{{ .RSS.LINK }}{{end}}
    """
```


