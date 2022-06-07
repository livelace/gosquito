### Description:

**mattermost** output plugin is intended for sending data to [Mattermost](https://mattermost.org/) users/channels.


### Generic parameters:

| Param   | Required | Type | Template | Default |
|:--------|:--------:|:----:|:--------:|:-------:|
| timeout |    -     | int  |    +     |    3    |


### Plugin parameters:

| Param        | Required | Type   | Cred | Template | Text Template | Default | Example                    | Description                                                                                      |
|:-------------|:--------:|:------:|:----:|:--------:|:-------------:|:-------:|:--------------------------:|:-------------------------------------------------------------------------------------------------|
| attachments  | -        | map    | -    | +        | -             | map[]   | see example                | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| files        | -        | array  | -    | +        | -             | ""      | ["data.array0"]            | List of [Datum](../../concept.md) fields with files paths.                                       |
| message      | -        | string | -    | +        | +             | ""      | "{{.DATA.TEXT0}}"          | Message text.                                                                                    |
| **output**   | +        | array  | -    | +        | -             | []      | ["news", "@livelace"]      | List of channels/users.                                                                          |
| **password** | +        | string | +    | -        | -             | ""      | ""                         | Mattermost password.                                                                             |
| **team**     | +        | string | +    | -        | -             | ""      | "superteam"                | Mattermost team.                                                                                 |
| **url**      | +        | string | +    | -        | -             | ""      | "https://host.example.com" | Mattermost URL.                                                                                  |
| **username** | +        | string | +    | -        | -             | ""      | ""                         | Mattermost user.                                                                                 |


### Attachments parameters:

| Param      | Required |  Type  | Template | Text Template |  Default  |          Example          | Description                                                                                      |
|:-----------|:--------:|:------:|:--------:|:-------------:|:---------:|:-------------------------:|:-------------------------------------------------------------------------------------------------|
| color      |    -     | string |    +     |       -       | "#00C100" |         "#E40303"         | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| pretext    |    -     | string |    +     |       +       |    ""     | "Pretext {{.TIMEFORMAT}}" | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| text       |    -     | string |    +     |       +       |    ""     |     "{{.DATA.TEXT0}}"     | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| title      |    -     | string |    +     |       +       |    ""     |    "Hello, {{.FLOW}}!"    | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| title_link |    -     | string |    +     |       -       |    ""     |   "https://example.com"   | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |

### Flow sample:

```yaml
flow:
  name: "mattermost-example"

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
        include: true
        input:  ["twitter.text"]
        output: ["data.text0"]
        regexp: ["regexps.urls"]
        replace: [""]

    - id: 1
      alias: "search urls"
      plugin: "regexpfind"
      params:
        input:  ["twitter.urls"]
        output: ["data.array0"]
        regexp: ["https://ria.ru/.*"]

  output:
    plugin: "mattermost"
    params:
      cred: "creds.mattermost.default"
      template: "templates.twitter.mattermost.default"
      output: ["news", "@livelace"]
```

### Config sample:

```toml
[creds.mattermost.default]
url = "https://host.example.com"
username = "<USERNAME>"
password = "<PASSWORD>"
team = "<TEAM>"

[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"

[regexps.urls]
regexp = [
    'https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)'
]

[templates.twitter.mattermost.default]
message = """
&nbsp;&nbsp;&nbsp;
**{{ .SOURCE | ToUpper }}, {{ .TIMEFORMAT }}**
{{.DATA.TEXT0}}
&nbsp;&nbsp;&nbsp;
{{range .DATA.ARRAY0}}{{printf .}}{{end}}
"""

[templates.twitter.mattermost.default.attachments]
color      = "#E40303"
pretext    = "Pretext {{.TIMEFORMAT}}"
text       = "{{.DATA.TEXT0}}"
title      = "Hello, {{.FLOW}}!"
title_link = "https://example.com"
```


