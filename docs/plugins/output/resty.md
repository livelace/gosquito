### Description:

**resty** output plugin is intended for sending data to [REST](https://en.wikipedia.org/wiki/Representational_state_transfer) endpoints.


### Generic parameters:

| Param   | Required | Type | Template | Default |
|:--------|:--------:|:----:|:--------:|:-------:|
| timeout |    -     | int  |    +     |   60    |


### Plugin parameters:

| Param        | Required |  Type  | Cred | Template | Text Template |      Default      |             Example             | Description                               |
|:-------------|:--------:|:------:|:----:|:--------:|:-------------:|:-----------------:|:-------------------------------:|:------------------------------------------|
| auth         |    -     | string |  -   |    +     |       -       |        ""         |             "basic"             | Auth method (basic, bearer).              |
| bearer_token |    -     | string |  +   |    -     |       -       |        ""         |            "qwerty"             | Bearer token.                             |
| body         |    -     | string |  -   |    +     |       +       |        ""         |        "{"foo": "bar"}"         | Request body.                             |
| headers      |    -     | map[]  |  -   |    +     |       +       |       map[]       |           see example           | Dynamic list of request headers.          |
| method       |    -     | string |  -   |    +     |       -       |       "GET"       |             "POST"              | Request method (GET, POST).               |
| **output**   |    +     | array  |  -   |    +     |       -       |       "[]"        | ["https://freegeoip.app/json/"] | List of REST endpoints.                   |
| params       |    -     | map[]  |  -   |    +     |       +       |       map[]       |           see example           | Dynamic list of request query parameters. |
| password     |    -     | string |  +   |    -     |       -       |        ""         |               ""                | Basic auth password.                      |
| proxy        |    -     | string |  -   |    +     |       -       |        ""         |     "http://127.0.0.1:8080"     | Proxy settings.                           |
| redirect     |    -     |  bool  |  -   |    +     |       -       |       true        |              false              | Follow redirects.                         |
| ssl_verify   |    -     |  bool  |  -   |    +     |       -       |       true        |              false              | Verify server certificate.                |
| user_agent   |    -     | string |  -   |    +     |       -       | "gosquito v3.1.0" |         "webchela 1.0"          | Custom User-Agent for feed access.        |
| username     |    -     | string |  +   |    -     |       -       |        ""         |               ""                | Basic auth username.                      |


### Flow sample:

```yaml
flow:
  name: "resty-output-example"

  input:
    plugin: "rss"
    params:
      force: true
      force_count: 1
      input: ["https://www.pcweek.ru/rss/"]

  output:
    plugin: "resty"
    params:
      template: "templates.resty.output"
      output: ["http://172.17.0.2:8080/api"]
```

### Config sample:

```toml
[templates.resty.output]
method = "GET"

[templates.resty.output.params]
query = '{data(url:"{{ .RSS.LINK }}"){article{text}}}'

```



