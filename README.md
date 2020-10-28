# gosquito


***gosquito*** ("go" + "mosquito") is a pluggable tool for data gathering from different sources (RSS, Twitter, Telegram etc.), data processing (fetch, minio, regexp, webchela etc.) and data transmitting to various destinations (SMTP, Mattermost, Kafka etc.).

### Config sample:


### Plugins:

| Plugin        | Type    | Description |
| :-------------| :-------:| ----------- |
| [rss](https://github.com/livelace/gosquito/docs/plugins/input/rss.md)                       |  input  |
| [telegram](https://github.com/livelace/gosquito/docs/plugins/input/telegram.md)             |  input  |
| [twitter](https://github.com/livelace/gosquito/docs/plugins/input/twitter.md)               |  input  |
| | | |
| [dedup](https://github.com/livelace/gosquito/docs/plugins/process/dedup.md)                 | process |
| [echo](https://github.com/livelace/gosquito/docs/plugins/process/echo.md)                   | process |
| [expandurl](https://github.com/livelace/gosquito/docs/plugins/process/expandurl.md)         | process |
| [fetch](https://github.com/livelace/gosquito/docs/plugins/process/fetch.md)                 | process |
| [minio](https://github.com/livelace/gosquito/docs/plugins/process/minio.md)                 | process |
| [regexpfind](https://github.com/livelace/gosquito/docs/plugins/process/regexpfind.md)       | process |
| [regexpmatch](https://github.com/livelace/gosquito/docs/plugins/process/regexpmatch.md)     | process |
| [regexpreplace](https://github.com/livelace/gosquito/docs/plugins/process/regexpreplace.md) | process |
| [unique](https://github.com/livelace/gosquito/docs/plugins/process/unique.md)               | process |
| [webchela](https://github.com/livelace/gosquito/docs/plugins/process/webchela.md)           | process |
| | | |
| [kafka](https://github.com/livelace/gosquito/docs/plugins/output/kafka.md)                  | output |
| [mattermost](https://github.com/livelace/gosquito/docs/plugins/output/mattermost.md)        | output |
| [smtp](https://github.com/livelace/gosquito/docs/plugins/output/smtp.md)                    | output |