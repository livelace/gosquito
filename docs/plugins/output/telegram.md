### Description:

**telegram** ouput plugin is intended for data sending to [Telegram](https://telegram.org/) chats.    
  
This plugin uses [TDLib client API](https://core.telegram.org/tdlib) (not [Telegram Bot API](https://core.telegram.org/bots/api)). Public and private chats are supported. Supported message types: audio, document, text, photo, video, video and voice notes. Registration as a client happens during first start (type phone number and code).

### Generic parameters:

| Param       | Required | Type   | Template | Default               |
|:------------|:--------:|:------:|:--------:|:---------------------:|
| timeout     | -        | int    | +        | 3                     |
| time_format | -        | string | +        | "15:04:05 02.01.2006" |
| time_zone   | -        | string | +        | "UTC"                 |

### Plugin parameters:

| Param            | Required | Type   | Cred | Template | Default                   | Example                       | Description                                                                                                |
|:-----------------|:--------:|:------:|:----:|:--------:|:-------------------------:|:-----------------------------:|:-----------------------------------------------------------------------------------------------------------|
| **api_id**       | +        | string | +    | -        | ""                        | ""                            | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                            |
| **api_hash**     | +        | string | +    | -        | ""                        | ""                            | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                            |
| app_version      | -        | string | -    | -        | v4.5.0-385342             | "0.0.1"                       | Custom application version.                                                                                |
| device_model     | -        | string | -    | -        | gosquito                  | "Redmi Note 42"               | Custom device model.                                                                                       |
| chat_database    | -        | string | -    | +        | <PLUGIN_DIR>/chats.sqlite | "/path/to/chats.db"           | Path to internal chats database.                                                                           |
| chat_save        | -        | bool   | -    | +        | false                     | true                          | Try to save all seen chats in internal database.                                                           |
| file_audio       | -        | array  | -    | +        | []                        | ["data.array0", "data.text0"] | Files will be send as audio messages.                                                                      |
| file_caption     | -        | string | -    | +        | ""                        | "Hello, {{ .DATA.TEXTA }}"    | Caption for file messages.
| file_document    | -        | array  | -    | +        | []                        | ["data.array0", "data.text0"] | Files will be send as document messages.                                                                   |
| file_photo       | -        | array  | -    | +        | []                        | ["data.array0", "data.text0"] | Files will be send as photo messages.                                                                      |
| file_video       | -        | array  | -    | +        | []                        | ["data.array0", "data.text0"] | Files will be send as video messages.                                                                      |
| log_level        | -        | int    | -    | +        | 0                         | 90                            | [TDLib Log Level](https://core.telegram.org/tdlib/docs/classtd_1_1td__api_1_1set_log_verbosity_level.html) |
| message          | -        | string | -    | +        | ""                        | "Hello, {{ .DATA.TEXTA }}"    | Message text.                                                                                              |
| message_preview  | -        | bool   | -    | +        | true                      | false                         | Enable/disale web page preview.                                                                            |
| **output**       | +        | array  | -    | +        | []                        | ["gosquito"]                  | List of Telegram chats ("t.me/+" pattern is considered as a private chat).                                 |
| proxy_enable     | -        | bool   | -    | +        | false                     | true                          | Enable/disable proxy.                                                                                      |
| proxy_port       | -        | int    | -    | +        | 9050                      | true                          | Proxy port number.                                                                                         |
| proxy_server     | -        | string | -    | +        | "127.0.0.1"               | true                          | Proxy server address.                                                                                      |
| proxy_username   | -        | string | +    | -        | ""                        | "alex"                        | Proxy username.                                                                                            |
| proxy_password   | -        | string | +    | -        | ""                        | "a1eXPass"                    | Proxy password.                                                                                            |
| proxy_type       | -        | string | -    | +        | "socks"                   | "http"                        | Use original file names with random generated suffix.                                                      |
| send_album       | -        | bool   | -    | +        | true                      | false                         | Group files into an album (2-10 files).                                                                    |
| send_delay       | -        | string | -    | +        | "1s"                      | "100ms"                       | Delay between sending.                                                                                     |
| send_timeout     | -        | string | -    | +        | "1h"                      | "24h"                         | Maximum time for sending.                                                                                  |
| session_ttl      | -        | int    | -    | +        | 366                       | 90                            | Session TTL (days).                                                                                        |
| status_enable    | -        | bool   | -    | +        | true                      | false                         | Enable/disable session status.                                                                             |
| status_period    | -        | string | -    | +        | "1h"                      | "5m"                          | Interval for showing session status in plugin output.                                                      |
| storage_optimize | -        | bool   | -    | +        | true                      | false                         | Enable/disable storage optimization (clean old data).                                                      |
| storage_period   | -        | string | -    | +        | "1h"                      | "24h"                         | Storage optimization interval.                                                                             |
| user_database    | -        | string | -    | +        | <PLUGIN_DIR>/users.sqlite | "/path/to/users.db"           | Path to internal users database.                                                                           |
| user_save        | -        | bool   | -    | +        | false                     | true                          | Enable/disable passive user logging.                                                                       |


### Flow sample:

```yaml
flow:
  name: "telegram-output"

  input:
    plugin: "rss"
    params:
      input: ["https://www.opennet.ru/opennews/opennews_all.rss"]
      force: true
      force_count: 10

  output:
    plugin: "telegram"
    params:
      cred: "creds.telegram.default"
      template: "templates.rss.telegram.default"
      output: ["gosquito"]
```


### Config sample:

```toml
[creds.telegram.default]
api_id = "<API_ID>"
api_hash = "<API_HASH>"

[templates.rss.telegram.default]
message = "{{ .RSS.DESCRIPTION }}\n\n{{ .RSS.LINK }}"
```


