### Input plugins:

| Plugin                                                                                      | Description                                                                   |
|:--------------------------------------------------------------------------------------------|:------------------------------------------------------------------------------|
| [rss](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/rss.md)           | [RSS/Atom](https://en.wikipedia.org/wiki/RSS) feeds (text, urls) data source. |
| [telegram](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/telegram.md) | [Telegram](https://telegram.org/) chats (text, image, video) data source.     |
| [twitter](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/twitter.md)   | [Twitter](https://twitter.com/) tweets (media, tags, text, urls) data source. |

### Process plugins:

| Plugin                                                                                                  | Description                                                                            |
|:--------------------------------------------------------------------------------------------------------|:---------------------------------------------------------------------------------------|
| [dedup](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/dedup.md)                 | Deduplicate [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md)s. |
| [echo](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/echo.md)                   | Echoing processing data.                                                               |
| [expandurl](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/expandurl.md)         | Expand short urls.                                                                     |
| [fetch](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/fetch.md)                 | Fetch remote data.                                                                     |
| [minio](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/minio.md)                 | Put data to [S3](https://en.wikipedia.org/wiki/Amazon_S3) bucket.                      |
| [regexpfind](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/regexpfind.md)       | Find patters in data.                                                                  |
| [regexpmatch](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/regexpmatch.md)     | Match data by patterns.                                                                |
| [regexpreplace](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/regexpreplace.md) | Replace patterns in data.                                                              |
| [unique](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/unique.md)               | Remove duplicates from data.                                                           |
| [webchela](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/webchela.md)           | Interact with web pages, fetch data.                                                   |
| [xpath](https://github.com/livelace/gosquito/blob/master/docs/plugins/process/xpath.md)                 | Find HTML elements with [Xpath](https://en.wikipedia.org/wiki/XPath).                  |

### Output plugins:

| Plugin                                                                                           | Description                                                        |
|:-------------------------------------------------------------------------------------------------|:-------------------------------------------------------------------|
| [kafka](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/kafka.md)           | Send data to [Kafka](https://kafka.apache.org/) topics.            |
| [mattermost](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/mattermost.md) | Send data to [Mattermost](https://mattermost.org/) channels/users. |
| [slack](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/slack.md)           | Send data to [Slack](https://slack.com) channels/users.            |
| [smtp](https://github.com/livelace/gosquito/blob/master/docs/plugins/output/smtp.md)             | Send data as emails with custom attachments/headers.               |

