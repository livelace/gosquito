# gosquito


***gosquito*** ("go" + "mosquito") is a pluggable tool for data gathering from different sources (RSS, Twitter, Telegram etc.), data processing (fetch, minio, regexp, webchela etc.) and data transmitting to various destinations (SMTP, Mattermost, Kafka etc.).

### Config sample:


### Plugins:

| Plugin        | Type    | Description |
| :-------------|---------| ----------- |
| rss           | input   |
| telegram      | input   |
| twitter       | input   |
| dedup         | process |
| [regexpreplace](https://github.com/livelace/gosquito/plugins/process/regexpreplace.md) | process 
| webchela      | process |
| kafka         | output  |
| mattermost    | output  |
| smtp          | output  |