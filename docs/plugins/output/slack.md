### Description:

**slack** output plugin is intended for sending data to [Slack](https://slack.com)  users/channels.


### Generic parameters:

| Param   | Required | Type | Template | Default |
|:--------|:--------:|:----:|:--------:|:-------:|
| timeout |    -     | int  |    +     |    3    |


### Plugin parameters:

| Param       | Required | Type   | Cred | Template | Text Template | Default | Example               | Description                                                                                                        |
|:------------|:--------:|:------:|:----:|:--------:|:-------------:|:-------:|:---------------------:|:-------------------------------------------------------------------------------------------------------------------|
| attachments | -        | map    | -    | +        | -             | map[]   | see example           | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts)                                     |
| files       | -        | array  | -    | +        | -             | ""      | ["data.array0"]       | List of [Datum](../../concept.md) fields with files paths.                                                         |
| message     | -        | string | -    | +        | +             | ""      | "{{ .DATA.TEXT0 }}"   | Message text.                                                                                                      |
| **output**  | +        | array  | -    | +        | -             | []      | ["news", "@livelace"] | List of channels/users.                                                                                            |
| **token**   | +        | string | +    | -        | -             | ""      | "xoxp-1-2-3"          | [Slack Internal App Token](https://slack.com/intl/en-ru/help/articles/215770388-Create-and-regenerate-API-tokens). |


### Attachments parameters:

| Param      | Required |  Type  | Template | Text Template |  Default  |          Example          | Description                                                                    |
|:-----------|:--------:|:------:|:--------:|:-------------:|:---------:|:-------------------------:|:-------------------------------------------------------------------------------|
| color      |    -     | string |    +     |       -       | "#00C100" |         "#E40303"         | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts) |
| pretext    |    -     | string |    +     |       +       |    ""     | "Pretext {{.TIMEFORMAT}}" | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts) |
| text       |    -     | string |    +     |       +       |    ""     |    "{{ .DATA.TEXT0 }}"    | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts) |
| title      |    -     | string |    +     |       +       |    ""     |    "Hello, {{.FLOW}}!"    | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts) |
| title_link |    -     | string |    +     |       -       |    ""     |   "https://example.com"   | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts) |

### Flow sample:

```yaml
flow:
  name: "slack-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru", "tass_agency", "izvestia_ru"]
      force: true
      force_count: 1

  process:
    - id: 0
      plugin: "regexpreplace"
      alias: "clean text"
      params:
        include: true
        input:  ["twitter.text"]
        output: ["data.text0"]
        regexp: ["regexps.urls"]
        replace: [ "" ]

    - id: 1
      plugin: "regexpfind"
      alias: "search urls"
      params:
        input:  ["twitter.urls", "twitter.urls"]
        output: ["data.array0",  "data.array1"]
        regexp: ["regexps.full.urls", "regexps.short.urls"]

    - id: 2
      plugin: "expandurl"
      alias: "expand short urls"
      params:
        input:  ["data.array1"]
        output: ["data.array2"]

    - id: 3
      plugin: "regexpreplace"
      alias: "clean urls"
      params:
        input:  ["data.array0", "data.array2"]
        output: ["data.array3", "data.array4"]
        regexp: ["regexps.clean.urls", "regexps.clean.urls"]
        replace: ["", ""]

    - id: 4
      plugin: "unique"
      alias: "merge urls"
      params:
        input:  ["data.array3", "data.array4"]
        output: ["data.array5"]

  output:
    plugin: "slack"
    params:
      cred: "creds.slack.default"
      template: "templates.twitter.slack.default"
      output: ["_news", "@livelace-mobile"]
```

### Config sample:

```toml
[creds.slack.default]
token = "<INTERNAL_APP_TOKEN>"

[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"

[regexps.urls]
regexp = [
    'https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)'
]

[regexps.clean.urls]
regexp = [
    '\?[a-zA-Z_]+=.*'
]

[regexps.full.urls]
regexp = [
    "https://iz.ru/.*",
    "https://ria.ru/.*",
    "https://rsport.ria.ru/.*",
]

[regexps.short.urls]
regexp = [
    "http[s]?://go.tass.ru/.*",
    "http[s]?://youtu.be/.*",
    "http[s]?://t.co/.*",
]

[templates.twitter.slack.default]
message = """
*{{ .SOURCE | ToUpper }}, {{ .TIMEFORMAT }}*
{{.DATA.TEXT0}}

{{range .DATA.ARRAY5}}{{printf .}}{{end}}
"""
```


