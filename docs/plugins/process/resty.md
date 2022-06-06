### Description:

**resty** process plugin is intended for [REST](https://en.wikipedia.org/wiki/Representational_state_transfer) queries.


### Generic parameters:

| Param   | Required | Type  | Template | Default | Example |
|:--------|:--------:|:-----:|:--------:|:-------:|:-------:|
| include |    -     | bool  |    -     |  true   |  false  |
| require |    -     | array |    -     |   []    | [1, 2]  |


### Plugin parameters:

| Param        | Required | Type   | Cred | Template | Text Template | Default           | Example                      | Description                                            |
|:-------------|:--------:|:------:|:----:|:--------:|:-------------:|:-----------------:|:----------------------------:|:-------------------------------------------------------|
| auth         | -        | string | -    | +        | -             | ""                | "basic"                      | Auth method (basic, bearer).                           |
| bearer_token | -        | string | +    | -        | -             | ""                | "qwerty"                     | Bearer token.                                          |
| body         | -        | string | -    | +        | +             | ""                | "{"foo": "bar"}"             | Request body.                                          |
| **input**    | +        | array  | -    | +        | -             | "[]"              | ["data.array0"]              | List of [DataItem](../../concept.md) fields with data. |
| headers      | -        | map[]  | -    | +        | +             | map[]             | see example                  | Dynamic list of request headers.                       |
| method       | -        | string | -    | +        | -             | "GET"             | "POST"                       | Request method (GET, POST).                            |
| **output**   | +        | array  | -    | +        | -             | "[]"              | ["data.array1"]              | List of target [DataItem](../../concept.md) fields.    |
| params       | -        | map[]  | -    | +        | +             | map[]             | see example                  | Dynamic list of request query parameters.              |
| password     | -        | string | +    | -        | -             | ""                | ""                           | Basic auth password.                                   |
| proxy        | -        | string | -    | +        | -             | ""                | "http://127.0.0.1:8080"      | Proxy settings.                                        |
| redirect     | -        | bool   | -    | +        | -             | true              | false                        | Follow redirects.                                      |
| ssl_verify   | -        | bool   | -    | +        | -             | true              | false                        | Verify server certificate.                             |
| **target**   | +        | string | -    | +        | -             | ""                | "http://172.17.0.2:8080/api" | REST endpoint.                                         |
| user_agent   | -        | string | -    | +        | -             | "gosquito v4.1.0" | "webchela 1.0"               | Custom User-Agent for feed access.                     |
| username     | -        | string | +    | -        | -             | ""                | ""                           | Basic auth username.                                   |


### Flow sample:

```yaml
flow:
  name: "resty-process-example"

  input:
    plugin: "rss"
    params:
      force: true
      force_count: 1
      input: ["https://www.pcweek.ru/rss/"]

  process:
    - id: 0
      plugin: "resty"
      params:
        template: "templates.resty.process"
        input:  ["rss.link"]
        output: ["data.text0"]
        target: "http://172.17.0.2:8080/api"

    - id: 1
      plugin: "echo"
      params:
        input: ["data.text0"]
```

### Config sample:

```toml
[templates.resty.process]
method = "GET"

[templates.resty.process.params]
query = '{data(url:"{{ .RSS.LINK }}"){article{text_spans{lang,text,tokens_amount}}}}'

```



