### Description:

**expandurl** process plugin is intended for expanding short URLs into full URLs.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include | -        | bool  | false   | true    |
| require | -        | array | []      | [1, 2]  |
| timeout | -        | int   | 2       | 3       |


### Plugin parameters:

| Param      | Required | Type   | Default           | Example          | Description                                         |
|:-----------|:--------:|:------:|:-----------------:|:----------------:|:----------------------------------------------------|
| depth      | -        | int    | 10                | 5                | Maximum depth of HTTP redirects.                    |
| **input**  | +        | array  | []                | ["twitter.urls"] | List of [Datum](../../concept.md) fields with URLs. |
| **output** | +        | array  | []                | ["data.array0"]  | List of target [Datum](../../concept.md) fields.    |
| user_agent | -        | string | "gosquito v4.5.0" | "webchela 1.0"   | Custom User-Agent for HTTP requests.                |

### Flow sample:

```yaml
flow:
  name: "expandurl-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["AP"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "regexpfind"
      alias: "search urls"
      params:
        input:  ["twitter.urls"]
        output: ["data.array0"]
        regexp: ["http://apne.ws/.*"]

    - id: 1
      plugin: "expandurl"
      alias: "expand urls"
      params:
        require: [0]
        include: false
        input:  ["data.array0"]
        output: ["data.array1"]

    - id: 2
      plugin: "echo"
      alias: "echo original and expanded urls"
      params:
        require: [0]
        input:  ["data.array0", "data.array1"]
```

### Config sample:

```toml
[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"
```


