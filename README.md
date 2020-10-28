# gosquito


***gosquito*** ("go" + "mosquito") is a pluggable tool for data gathering from different sources (RSS, Twitter, Telegram etc.), data processing (fetch, minio, regexp, webchela etc.) and data transmitting to various destinations (SMTP, Mattermost, Kafka etc.).


### Features:

* Pluggable architecture. Data processing organized as chains of plugins.
* Flow approach. Flow consists of: input plugin, process plugins, output plugin.
* Plugins dependencies. Plugin "B" will process data only if plugin "A" derived some data. 
* Include/exclude data from all or specific plugins.
* Declarative YAML configurations with templates support.

### Config sample:


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
| [kafka](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/kafka.md)                  | output | Send data to Kafka topics. |
| [mattermost](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/mattermost.md)        | output | Send data to Mattermost channels/users. |
| [smtp](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/smtp.md)                    | output | Send data as emails with custom attachments/headers. |

### Data model: 

