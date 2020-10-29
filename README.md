# gosquito


***gosquito*** ("go" + "mosquito") is a pluggable tool for data gathering from different sources ([RSS](https://en.wikipedia.org/wiki/RSS), [Twitter](https://twitter.com), [Telegram](https://telegram.org/) etc.), data processing (fetch, [minio](https://min.io/), regexp, [webchela](https://github.com/livelace/webchela) etc.) and data transmitting to various destinations ([SMTP](https://en.wikipedia.org/wiki/Simple_Mail_Transfer_Protocol), [Mattermost](https://mattermost.org/), [Kafka](https://kafka.apache.org/) etc.).


### Features:

* Pluggable architecture. Data processing organized as chains of plugins.
* Flow approach. Flow consists of: input plugin, process plugins, output plugin.
* Plugins dependencies. Plugin "B" will process data only if plugin "A" derived some data. 
* Include/exclude data from all or specific plugins.
* Declarative YAML configurations with templates support.
* Export flow statistics to [Prometheus](https://prometheus.io/).

### Config sample:


```toml
[default]
time_format = "15:04 02.01.2006"
time_zone = "Europe/Moscow"

[credentials.twitter.default]
access_token = "<access_token>"
access_secret = "<access_secret>"
consumer_key = "<consumer_key>"
consumer_secret = "<consumer_secret>"

[regexes.urls]
regexp = [
    'http?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)',
    'https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)'
]

[templates.smtp.twitter.default]
server = "mail.example.com"
port = 25
ssl = true

from = "gosquito@example.com"
output = ["user@example.com"]

subject = "{{ .DATA.TEXT0 }}"
subject_length = 150

body_html = true
body_length = 1000
body = """
<br>
<div align="right"><b>{{ .TIMEFORMAT }}</b></div>
{{.DATA.TEXT0}}<br><br>
{{range .TWITTER.URLS}}{{printf "%s<br>" .}}{{end}}
"""

attachments = ["data.array0"]

[templates.smtp.twitter.default.headers]
x-gosquito-flow = "flow"
x-gosquito-plugin = "plugin"
x-gosquito-source = "source"
x-gosquito-time = "time"
x-gosquito-uuid = "uuid"
```


### Flow sample:

```yaml
flow:
  name: "find-russia"
  params:
    interval: "1h"

  input:
    plugin: "twitter"
    params:
      cred: "credentials.twitter.default"
      input: [
          "izvestia_ru", "IA_REGNUM", "rianru", "tass_agency",
          "AP", "BBCNews", "BBCWorld", "bbcrussian", "business", "independent", "Telegraph"
      ]
      force: true

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
        regexp: ["regexes.urls", "\n"]
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
      template: "templates.smtp.twitter.default"
```

### Plugins:

| Plugin        | Type    | Description |
| :-------------| :-------:| ----------- |
| [rss](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/rss.md)                       |  input  | RSS/Atom feeds (text, urls) data source. |
| [telegram](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/telegram.md)             |  input  | Telegram chats (text, image, video) data source. | 
| [twitter](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/twitter.md)               |  input  | Twitter tweets (media, tags, text, urls) data source. |
| | | |
| [dedup](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/dedup.md)                 | process | Deduplicate enriched data items. |
| [echo](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/echo.md)                   | process | Echoing processing data. |
| [expandurl](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/expandurl.md)         | process | Expand short urls. |
| [fetch](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/fetch.md)                 | process | Fetch remote data. | 
| [minio](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/minio.md)                 | process | Get/put data from/to s3 bucket. |
| [regexpfind](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/regexpfind.md)       | process | Find patters in data. |
| [regexpmatch](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/regexpmatch.md)     | process | Match data by patterns. |
| [regexpreplace](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/regexpreplace.md) | process | Replace patterns in data. |
| [unique](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/unique.md)               | process | Remove duplicates from data. | 
| [webchela](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/webchela.md)           | process | Interact with web pages, fetch data. | 
| | | |
| [kafka](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/kafka.md)                  | output  | Send data to Kafka topics. |
| [mattermost](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/mattermost.md)        | output  | Send data to Mattermost channels/users. |
| [smtp](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/smtp.md)                    | output  | Send data as emails with custom attachments/headers. |

