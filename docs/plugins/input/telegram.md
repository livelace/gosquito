### Description:

**telegram** input plugin is intended for data gathering from [Telegram](https://telegram.org/) chats.    
  
This plugin uses [TDLib client API](https://core.telegram.org/tdlib) (not [Telegram Bot API](https://core.telegram.org/bots/api)). Public and private chats are supported. Supported message types: audio, document, text, photo, video, voice and video notes. Registration as a client happens during first start (type phone number and code). Only single client (phone number) per gosquito instance is supported right now.

### Data structure:

```go
type Telegram struct {
	CHATID               string
	CHATNAME             string
	CHATTYPE             string
	CHATTITLE            string
	CHATCLIENTDATA       string
	CHATPROTECTEDCONTENT string
	CHATLASTINBOXID      string
	CHATLASTOUTBOXID     string
	CHATMESSAGETTL       string
	CHATUNREADCOUNT      string
	CHATTIMESTAMP        string

	MESSAGEID        string
	MESSAGEMEDIA     []string
	MESSAGESENDERID* string
	MESSAGETEXT*     string
	MESSAGETEXTURL   []string
	MESSAGETYPE      string
	MESSAGETIMESTAMP string
	MESSAGEURL*      string

	USERID            string
	USERVERSION       string
	USERNAME          string
	USERTYPE          string
	USERLANG          string
	USERFIRSTNAME     string
	USERLASTNAME      string
	USERPHONE         string
	USERSTATUS        string
	USERACCESSIBLE    string
	USERCONTACT       string
	USERFAKE          string
	USERMUTUALCONTACT string
	USERSCAM          string
	USERSUPPORT       string
	USERVERIFIED      string
	USERRESTRICTION   string
	USERTIMESTAMP     string

	WARNINGS []string
}
```

&ast; - field may be used with **match_signature** parameter.

### Generic parameters:

| Param                 | Required | Type   | Template | Default               |
|:----------------------|:--------:|:------:|:--------:|:---------------------:|
| expire_action         | -        | array  | +        | []                    |
| expire_action_delay   | -        | string | +        | "1d"                  |
| expire_action_timeout | -        | int    | +        | 30                    |
| expire_interval       | -        | string | +        | "7d"                  |
| force                 | -        | bool   | +        | false                 |
| force_count           | -        | int    | +        | 100                   |
| timeout               | -        | int    | +        | 60                    |
| time_format           | -        | string | +        | "15:04:05 02.01.2006" |
| time_zone             | -        | string | +        | "UTC"                 |


### Plugin parameters:

| Param                | Required | Type   | Cred | Template | Default                   | Example                                         | Description                                                                                                      |
|:---------------------|:--------:|:------:|:----:|:--------:|:-------------------------:|:-----------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------|
| ads_enable           | -        | bool   | -    | -        | true                      | false                                           | Enable/disable asking for sponsored messages.                                                                    |
| ads_period           | -        | string | -    | -        | "5m"                      | "1h"                                            | [Sponsored messages](https://core.telegram.org/api/sponsored-messages) receiving interval.                       |
| **api_id**           | +        | string | +    | -        | ""                        | ""                                              | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                                  |
| **api_hash**         | +        | string | +    | -        | ""                        | ""                                              | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                                  |
| app_version          | -        | string | +    | -        | v3.8.2-557f8a             | "0.0.1"                                         | Custom application version.                                                                                      |
| device_model         | -        | string | +    | -        | gosquito                  | "Redmi Note 42"                                 | Custom device model.                                                                                             |
| chat_database        | -        | string | -    | +        | <PLUGIN_DIR>/chats.sqlite | "/path/to/chats.db"                             | Path to internal chats database.                                                                                 |
| fetch_max_size       | -        | size   | -    | +        | "10m"                     | "1g"                                            | Maximum file size for download.                                                                                  |
| fetch_metadata       | -        | bool   | -    | +        | false                     | true                                            | Generate JSON metadata for downloaded file.                                                                      |
| fetch_orig_name      | -        | bool   | -    | +        | true                      | false                                           | Use original file name.                                                                                         |
| fetch_timeout        | -        | string | -    | +        | "1h"                      | "24h"                                           | Maximum time for file download.                                                                                  |
| file_path            | -        | string | -    | +        | <PLUGIN_DIR>              | "/data/files"                                   | Directory where all downloaded files will be saved.                                                              |
| **input**            | +        | array  | -    | +        | []                        | ["breakingmash"]                                | List of Telegram chats ("t.me/+" pattern is considered as a private chat).                                       |
| log_level            | -        | int    | -    | +        | 0                         | 90                                              | [TDLib Log Level](https://core.telegram.org/tdlib/docs/classtd_1_1td__api_1_1set_log_verbosity_level.html)       |
| match_signature      | -        | array  | -    | +        | "[]"                      | ["telegram.messagetext", "telegram.messageurl"] | Match new messages by signature.                                                                                 |
| match_ttl            | -        | string | -    | +        | "1d"                      | "24h"                                           | TTL (Time To Live) for matched signatures.                                                                       |
| message_type_fetch   | -        | array  | -    | +        | "[]"                      | ["audio", "document"]                           | Fetch files only for specific message types (audio, document, photo, video, video_note, voice_note). Default all |
| message_type_process | -        | array  | -    | +        | "[]"                      | ["audio", "document"]                           | Process only specific message types (audio, document, photo, text, video, video_note, voice_note). Default all   |
| pool_size            | -        | int    | -    | +        | 100000                    | 10000                                           | Put arriving messages in pool during blocking operations (file fetching).                                        |
| proxy_enable         | -        | bool   | -    | +        | false                     | true                                            | Enable/disable proxy.                                                                                            |
| proxy_port           | -        | int    | -    | +        | 9050                      | true                                            | Proxy port number.                                                                                               |
| proxy_server         | -        | string | -    | +        | "127.0.0.1"               | true                                            | Proxy server address.                                                                                            |
| proxy_username       | -        | string | +    | -        | ""                        | "alex"                                          | Proxy username.                                                                                                  |
| proxy_password       | -        | string | +    | -        | ""                        | "a1eXPass"                                      | Proxy password.                                                                                                  |
| proxy_type           | -        | string | -    | +        | "socks"                   | "http"                                          | Use original file names with random generated suffix.                                                            |
| session_ttl          | -        | int    | -    | +        | 366                       | 90                                              | Session TTL (days).                                                                                              |
| status_enable        | -        | bool   | -    | +        | true                      | false                                           | Enable/disable session status.                                                                                   |
| status_period        | -        | string | -    | +        | "1h"                      | "5m"                                            | Interval for showing session status in plugin output.                                                            |
| storage_optimize     | -        | bool   | -    | +        | true                      | false                                           | Enable/disable storage optimization (clean old data).                                                            |
| storage_period       | -        | string | -    | +        | "1h"                      | "24h"                                           | Storage optimization interval.                                                                                   |
| user_database        | -        | string | -    | +        | <PLUGIN_DIR>/users.sqlite | "/path/to/users.db"                             | Path to internal users database.                                                                                 |
| user_log             | -        | bool   | -    | +        | true                      | false                                           | Enable/disable passive user logging.                                                                             |


### Flow sample:

```yaml
# Due to Telegram _API limits_ we just wait new message events in background,
# we don't make requests for new/not received messages.
# For more information see: https://github.com/tdlib/td/issues/682
#
# Right after we received a new message event we compare event timestamp with
# last received message timestamp and if event contains new data - we process new data.
# We cannot use "force" here and have to wait new messages explicitly.
flow:
  name: "telegram-example"

  input:
    plugin: "telegram"
    params:
      cred: "creds.telegram.default"
      template: "templates.telegram.default"
      input: ["breakingmash", "interfax_ru", "izvestia"]
      fetch_timeout: "3h"
      file_path: "/tmp"
      status_period: "5m"
      storage_period: "5h"

  process:
    - id: 0
      alias: "exclude ads"
      plugin: "regexpmatch"
      params:
        include: true
        input:  ["telegram.username"]
        regexp: ["ads"]
        match_not: true

    - id: 1
      alias: "extract filename"
      plugin: "regexpfind"
      params:
        require: [0]
        input:  ["telegram.messagemedia"]
        output: ["data.array0"]
        regexp: ["^(.+)\\/([^\\/]+)$"]
        group:  [[2]]

    - id: 2
      plugin: "echo"
      alias: "show text"
      params:
        require: [0]
        input: [
          "telegram.chattitle", 
          "telegram.chattype", 
          "telegram.messageid", 
          "telegram.messagemedia", 
          "telegram.messagesenderid", 
          "telegram.messagetext", 
          "telegram.messagetexturl", 
          "telegram.messagetype", 
          "telegram.messageurl", 
          "telegram.userid", 
          "telegram.username", 
          "telegram.usertype", 
          "telegram.userfirstname", 
          "telegram.userlastname", 
          "telegram.userphone", 
          "data.array0",
          "---", 
        ]
```


### Config sample:

```toml
[creds.telegram.default]
api_id = "<API_ID>"
api_hash = "<API_HASH>"

[templates.telegram.default]
file_max_size = "3g"
#log_level = 90
```


