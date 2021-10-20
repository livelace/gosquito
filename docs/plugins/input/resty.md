### Description:

**resty** input plugin is intended for data gathering from [REST](https://en.wikipedia.org/wiki/Representational_state_transfer).

### Data structure:

```go
type RestyData struct {
	BODY*       string
}
```

&ast; - may be used with **match_signature** parameter.

### Generic parameters:

| Param                   | Required   | Type     | Template   | Default                 |
| :---------------------- | :--------: | :------: | :--------: | :---------------------: |
| expire_action           | -          | array    | +          | []                      |
| expire_action_delay     | -          | string   | +          | "1d"                    |
| expire_action_timeout   | -          | int      | +          | 30                      |
| expire_interval         | -          | string   | +          | "7d"                    |
| timeout                 | -          | int      | +          | 60                      |
| time_format             | -          | string   | +          | "15:04:05 02.01.2006"   |
| time_zone               | -          | string   | +          | "UTC"                   |


### Plugin parameters:

| Param           | Required   | Type     | Cred  | Template   | Text Template | Default             | Example                          | Description                                |
| :-----------    | :--------: | :------: | :---: | :--------: | :-----------: | :-----------------: | :------------------------------: | :-----------------------------------       |
| auth            | -          | string   | -     | +          | -             | ""                  | "basic"                          | Auth method (basic, bearer).               |
| body            | +          | string   | -     | +          | +             | ""                  | "{ "foo": "bar" }"               | Request body.                              |
| headers         | -          | map[]    | -     | +          | +             | map[]               | see example                      | Dynamic list of request headers.             |
| **input**       | +          | array    | -     | +          | -             | "[]"                | ["https://www.pcweek.ru/rss/"]   | List of REST endpoints.                    |
| match_signature | -          | array    | -     | +          | -             | "[]"                | ["body"]                         | Match new articles by signature.           |
| match_ttl       | -          | string   | -     | +          | -             | "1d"                | "24h"                            | TTL (Time To Live) for matched signatures. |
| password        | -          | int      | +     | -          | -             | -                   | ""                               | Basic auth password.                       |
| ssl_verify      | -          | bool     | -     | +          | -             | true                | false                            | Verify server certificate.                 |
| user_agent      | -          | string   | -     | +          | -             | "gosquito v1.0.0"   | "webchela 1.0"                   | Custom User-Agent for feed access.         |
| username        | -          | int      | +     | -          | -             | -                   | ""                               | Basic auth username.                       |


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


