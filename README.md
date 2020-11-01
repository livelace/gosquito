# gosquito


***gosquito*** ("go" + "mosquito") is a pluggable tool for data gathering from different sources ([RSS](https://en.wikipedia.org/wiki/RSS), [Twitter](https://twitter.com), [Telegram](https://telegram.org/) etc.), data processing (fetch, [minio](https://min.io/), regexp, [webchela](https://github.com/livelace/webchela) etc.) and data transmitting to various destinations ([SMTP](https://en.wikipedia.org/wiki/Simple_Mail_Transfer_Protocol), [Mattermost](https://mattermost.org/), [Kafka](https://kafka.apache.org/) etc.).

### Main goal:

To replace various in-house automated tasks for data gathering with single tool.

### Features:

* [Pluggable](https://github.com/livelace/gosquito/blob/master/docs/plugins/plugins.md) architecture. [Data processing](https://github.com/livelace/gosquito/blob/master/docs/data.md) organized as chains of plugins.
* Flow approach. Flow consists of: input plugin, process plugins, output plugin.
* Plugins dependencies. Plugin "B" will process data only if plugin "A" derived some data. 
* Include/exclude data from all or specific plugins.
* Declarative YAML configurations with templates support.
* Export flow statistics to [Prometheus](https://prometheus.io/).
* Send only new data or send fetched data every time. 

### Flow sample ([options](https://github.com/livelace/gosquito/blob/master/docs/flow.md)):

```yaml
flow:
  name: "russia"
  params:
    interval: "10m"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: [
          "izvestia_ru", "IA_REGNUM", "rianru", "tass_agency",
          "AP", "BBCNews", "BBCWorld", "business", "independent"
      ]
      force: true
      force_count: 3

  process:
    - id: 0
      alias: "match russia"
      plugin: "regexpmatch"
      params:
        input: ["twitter.text"]
        regexp: ["Россия", "Russia"]

    - id: 1
      alias: "clean text"
      plugin: "regexpreplace"
      params:
        require: [0]
        include: false
        input:  ["twitter.text"]
        output: ["data.text0"]
        regexp: ["regexps.urls", "\n"]
        replacement: [""]

    - id: 2
      alias: "fetch media"
      plugin: "fetch"
      params:
        require: [1]
        include: false
        input:  ["twitter.media"]
        output: ["data.array0"]

  output:
    plugin: "smtp"
    params:
      template: "templates.twitter.smtp.default"
```

### Config sample ([options](https://github.com/livelace/gosquito/blob/master/docs/config.md)):

```toml
[default]
time_format = "15:04 02.01.2006"
time_zone = "Europe/Moscow"

[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"

[regexps.urls]
regexp = [
    'http?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)',
    'https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)'
]

[templates.twitter.smtp.default]
server = "mail.example.com"
port = 25
ssl = true

from = "gosquito@example.com"
output = ["user@example.com"]

subject = "{{ .DATA.TEXT0 }}"
subject_length = 150

body = """
<br>
<div align="right"><b>{{ .FLOW }}&nbsp;&nbsp;&nbsp;{{ .TIMEFORMAT }}</b></div>
{{.DATA.TEXT0}}<br><br>
{{range .TWITTER.URLS}}{{printf "%s<br>" .}}{{end}}
"""

body_html = true
body_length = 1000

attachments = ["data.array0"]

[templates.twitter.smtp.default.headers]
x-gosquito-flow   = "flow"
x-gosquito-plugin = "plugin"
x-gosquito-source = "source"
x-gosquito-time   = "time"
x-gosquito-uuid   = "uuid"
```




