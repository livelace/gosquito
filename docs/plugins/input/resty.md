### Description:

**resty** input plugin is intended for data gathering from [REST](https://en.wikipedia.org/wiki/Representational_state_transfer).

### Data structure:

```go
type RestyData struct {
	BODY*       string
	PROTO*      string
	STATUS*     string
	STATUSCODE* string
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
| bearer_token    | -          | string   | +     | -          | -             | ""                  | "qwerty"                         | Bearer token.                              |
| body            | +          | string   | -     | +          | +             | ""                  | "{"foo": "bar"}"                 | Request body.                              |
| headers         | -          | map[]    | -     | +          | +             | map[]               | see example                      | Dynamic list of request headers.           |
| **input**       | +          | array    | -     | +          | -             | "[]"                | ["https://www.pcweek.ru/rss/"]   | List of REST endpoints.                    |
| match_signature | -          | array    | -     | +          | -             | "[]"                | ["body", "statuscode"]           | Match new articles by signature.           |
| match_ttl       | -          | string   | -     | +          | -             | "1d"                | "24h"                            | TTL (Time To Live) for matched signatures. |
| method          | -          | string   | -     | +          | -             | "GET"               | "POST"                           | Request method (GET, POST).                |
| params          | -          | map[]    | -     | +          | +             | map[]               | see example                      | Dynamic list of request query parameters.  |
| password        | -          | string   | +     | -          | -             | ""                  | ""                               | Basic auth password.                       |
| proxy           | -          | string   | -     | +          | -             | ""                  | "http://127.0.0.1:8080"          | Proxy settings.                            |
| redirect        | -          | bool     | -     | +          | -             | true                | false                            | Follow redirects.                          |
| ssl_verify      | -          | bool     | -     | +          | -             | true                | false                            | Verify server certificate.                 |
| user_agent      | -          | string   | -     | +          | -             | "gosquito v1.0.0"   | "webchela 1.0"                   | Custom User-Agent for feed access.         |
| username        | -          | string   | +     | -          | -             | ""                  | ""                               | Basic auth username.                       |


### Flow sample:

```yaml
flow:
  name: "resty-input-example"

  input:
    plugin: "resty"
    params:
      template: "templates.resty.input"
      input: ["https://freegeoip.app/json/"]

  process:
    - id: 0
      plugin: "echo"
      params:
        input: ["resty.body"]
```

### Config sample:

```toml
[templates.resty.input]
method = "GET"
proxy = "http://127.0.0.1:8081"
ssl_verify = false

[templates.resty.input.headers]
foo = "bar"

[templates.resty.input.params]
foo = "bar"

```



