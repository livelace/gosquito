# gosquito


***gosquito*** ("go" + "mosquito") is a pluggable tool for data
gathering from different sources
([RSS](https://en.wikipedia.org/wiki/RSS),
[Twitter](https://twitter.com), [Telegram](https://telegram.org/) etc.),
data processing (fetch, [minio](https://min.io/), regexp,
[webchela](https://github.com/livelace/webchela) etc.) and data
transmitting to various destinations
([SMTP](https://en.wikipedia.org/wiki/Simple_Mail_Transfer_Protocol),
[Mattermost](https://mattermost.org/),
[Kafka](https://kafka.apache.org/) etc.).

### Main goal:

To replace various in-house automated tasks for data gathering with
single tool.

### Features:

* [Pluggable](https://github.com/livelace/gosquito/blob/master/docs/plugins/plugins.md)
  architecture.
  [Data processing](https://github.com/livelace/gosquito/blob/master/docs/data.md)
  organized as chains of plugins.
* Flow approach. Flow consists of: input plugin (grab), process plugins (transform/enrich), output
  plugin (send).
* Plugins dependencies. Plugin "B" will process data only if plugin "A"
  derived some data.
* Include/exclude data from all or specific [plugins](https://github.com/livelace/gosquito/blob/master/docs/plugins/plugins.md).
* Declarative YAML configurations with templates support.
* Export flow statistics to [Prometheus](https://prometheus.io/).
* Send only new data or send all fetched data every time.
* Fetch a fully initialized web page with [Webchela](https://github.com/livelace/webchela) [plugin](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/webchela.md).


### Build dependencies:

* Kafka support: [librdkafka](https://github.com/edenhill/librdkafka)
* Telegram support: [TDLib](https://github.com/tdlib/td)

```shell script
go build -tags dynamic "github.com/livelace/gosquito/cmd/gosquito"
```

### AppImage dependencies:

* [FUSE](https://github.com/libfuse/libfuse)
* [Glibc](https://www.gnu.org/software/libc/) >= 2.29 (Ubuntu 20.04)

### Quick start:

```shell script
# Docker:
user@localhost /tmp $ docker run -ti --rm livelace/gosquito:v2.1.3 bash
gosquito@fa388e89e10e ~ $ gosquito 
INFO[03.11.2020 14:44:15.806] gosquito v2.1.3   
INFO[03.11.2020 14:44:15.807] config init        path="/home/gosquito/.gosquito"
ERRO[03.11.2020 14:44:15.807] flow read          path="/home/gosquito/.gosquito/flow/conf" error="no valid flow"
gosquito@fa388e89e10e ~ $

# AppImage:
user@localhost ~ $ curl -sL "https://github.com/livelace/gosquito/releases/download/v2.1.3/gosquito-v2.1.3.appimage" -o "/tmp/gosquito-v2.1.3.appimage" && chmod +x "/tmp/gosquito-v2.1.3.appimage"
user@localhost ~ $ /tmp/gosquito-v2.1.3.appimage 
INFO[04.11.2020 16:59:00.228] gosquito v2.1.3   
INFO[04.11.2020 16:59:00.230] config init        path="/home/user/.gosquito"
ERRO[04.11.2020 16:59:00.233] flow read          path="/home/user/.gosquito/flow/conf" error="no valid flow"
```

### Flow example ([options](https://github.com/livelace/gosquito/blob/master/docs/flow.md)):

```yaml
flow:
  name: "flow-example"
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
      force_count: 10

  process:
    - id: 0
      alias: "match russia"
      plugin: "regexpmatch"
      params:
        input: ["twitter.text"]
        regexp: ["regexps.words"]

    - id: 1
      alias: "clean text"
      plugin: "regexpreplace"
      params:
        require: [0]
        include: false
        input:  ["twitter.text"]
        output: ["data.text0"]
        regexp: ["regexps.urls"]
        replace: [""]

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

### Config example ([options](https://github.com/livelace/gosquito/blob/master/docs/config.md)):

```toml
[default]

#log_level = "DEBUG"
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

[regexps.words]
regexp = [
  "Россия", "Russia"
]

[templates.twitter.smtp.default]
server = "mail.example.com"
port = 25
ssl = true

from = "gosquito@example.com"
output = ["user@example.com"]

subject = "{{.DATA.TEXT0}}"
subject_length = 150

body = """
<br>
<div align="right"><b>{{.FLOW}}&nbsp;&nbsp;&nbsp;{{.TIMEFORMAT}}</b></div>
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

### Grafana charts:

![grafana](https://github.com/livelace/gosquito/blob/master/assets/grafana.png)
