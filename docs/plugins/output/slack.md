### Description:

**slack** output plugin is intended for sending data to [Slack](https://slack.com)  users/channels.


### Generic parameters:

| Param     | Required   | Type   | Template   | Default   |
| :-------- | :--------: | :----: | :--------: | :-------: |
| timeout   | -          | int    | +          | 3         |


### Plugin parameters:

| Param         | Required   | Type     | Cred   | Template   | Text Template   | Default   | Example                 | Description                                                                                                          |
| :------------ | :--------: | :------: | :----: | :--------: | :-------------: | :-------: | :---------------------: | :------------------------------------------------------------------------------------------------------------------- |
| attachments   | -          | map      | -      | +          | -               | map[]     | see example             | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts)                                       |
| files         | -          | array    | -      | +          | -               | ""        | ["data.array0"]         | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with files paths.           |
| message       | -          | string   | -      | +          | +               | ""        | "{{ .DATA.TEXT0 }}"     | Message text.                                                                                                        |
| **output**    | +          | array    | -      | +          | -               | []        | ["news", "@livelace"]   | List of channels/users.                                                                                              |
| **token**     | +          | string   | +      | -          | -               | ""        | "xoxp-1-2-3"            | [Slack Internal App Token](https://slack.com/intl/en-ru/help/articles/215770388-Create-and-regenerate-API-tokens).   |


### Attachments parameters:

| Param        | Required   | Type     | Template   | Text Template   | Default     | Example                     | Description                                                                      |
| :----------- | :--------: | :------: | :--------: | :-------------: | :---------: | :-------------------------: | :------------------------------------------------------------------------------- |
| color        | -          | string   | +          | -               | "#00C100"   | "#E40303"                   | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts)   |
| pretext      | -          | string   | +          | +               | ""          | "Pretext {{.TIMEFORMAT}}"   | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts)   |
| text         | -          | string   | +          | +               | ""          | "{{ .DATA.TEXT0 }}"         | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts)   |
| title        | -          | string   | +          | +               | ""          | "Hello, {{.FLOW}}!"         | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts)   |
| title_link   | -          | string   | +          | -               | ""          | "https://example.com"       | [Slack Message Attachments](https://api.slack.com/messaging/composing/layouts)   |

### Flow sample:

```yaml
flow:
  name: "slack-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru"]
      force: true
      force_count: 10

  process:
    - id: 0
      alias: "clean text"
      plugin: "regexpreplace"
      params:
        input:  ["twitter.text"]
        output: ["data.text0"]
        regexp: ["regexps.urls"]
        replace: [""]

    - id: 1
      alias: "search urls"
      plugin: "regexpfind"
      params:
        include: false
        input:  ["twitter.urls"]
        output: ["data.array0"]
        regexp: ["https://ria.ru/.*"]

  output:
    plugin: "slack"
    params:
      cred: "creds.slack.default"
      template: "templates.twitter.slack.default"
      output: ["news", "@livelace"]
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

[templates.twitter.slack.default]
message = """
*{{ .SOURCE | ToUpper }}, {{ .TIMEFORMAT }}*
{{.DATA.TEXT0}}

{{range .DATA.ARRAY0}}{{printf .}}{{end}}
"""

[templates.twitter.slack.default.attachments]
color      = "#E40303"
pretext    = "Pretext {{.TIMEFORMAT}}"
text       = "{{.DATA.TEXT0}}"
title      = "Hello, {{.FLOW}}!"
title_link = "https://example.com"
```


