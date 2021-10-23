# gosquito


***gosquito*** ("go" + "mosquito") is a pluggable tool for data
gathering, data processing and data transmitting to various destinations. 
Main goal is to replace various in-house automated tasks with a single tool and move those tasks at the edge.

<img align="right" src="assets/gosquito.png">


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

<br><br><br>

### Input plugins:

| Plugin                                                                                        | Description                                                                        |
| :-------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------    |
| [resty](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/resty.md)         | [REST](https://en.wikipedia.org/wiki/Representational_state_transfer) data source. |
| [rss](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/rss.md)             | [RSS/Atom](https://en.wikipedia.org/wiki/RSS) feeds (text, urls) data source.      |
| [telegram](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/telegram.md)   | [Telegram](https://telegram.org/) chats (text, image, video) data source.          |
| [twitter](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/twitter.md)     | [Twitter](https://twitter.com/) tweets (media, tags, text, urls) data source.      |

### Process plugins:

| Plugin                                                                                                    | Description                                                                              |
| :-------------------------------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------------- |
| [dedup](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/dedup.md)                   | Deduplicate [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md)'s. |
| [echo](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/echo.md)                     | Echoing processing data.                                                                 |
| [expandurl](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/expandurl.md)           | Expand short urls.                                                                       |
| [fetch](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/fetch.md)                   | Fetch remote data.                                                                       |
| [jq](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/jq.md)                         | Extract JSON elements with [gojq](https://github.com/itchyny/gojq).                                                                       |
| [minio](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/minio.md)                   | Put data to [S3](https://en.wikipedia.org/wiki/Amazon_S3) bucket.                        |
| [regexpfind](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/regexpfind.md)         | Find patters in data.                                                                    |
| [regexpmatch](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/regexpmatch.md)       | Match data by patterns.                                                                  |
| [regexpreplace](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/regexpreplace.md)   | Replace patterns in data.                                                                |
| [resty](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/resty.md)                   | Perform REST requests with data.                                                         |
| [unique](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/unique.md)                 | Remove duplicates from data.                                                             |
| [webchela](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/webchela.md)             | Interact with web pages, fetch data.                                                     |
| [xpath](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/xpath.md)                   | Extract HTML elements with [htmlquery](https://github.com/antchfx/htmlquery).                    |

### Output plugins:

| Plugin                                                                                             | Description                                                                                   |
| :------------------------------------------------------------------------------------------------- | :-------------------------------------------------------------------                          |
| [kafka](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/kafka.md)             | Send data to [Kafka](https://kafka.apache.org/) topics.                                       |
| [mattermost](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/mattermost.md)   | Send data to [Mattermost](https://mattermost.org/) channels/users.                            |
| [resty](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/resty.md)             | Send data to [REST](https://en.wikipedia.org/wiki/Representational_state_transfer) endpoints. |
| [slack](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/slack.md)             | Send data to [Slack](https://slack.com) channels/users.                                       |
| [smtp](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/smtp.md)               | Send data as emails with custom attachments/headers.                                          |
