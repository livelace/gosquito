### Description:

**telegram** input plugin is intended for data gathering from [Telegram](https://telegram.org/) chats.

This plugin uses [TDLib client API](https://core.telegram.org/tdlib) (
not [Telegram Bot API](https://core.telegram.org/bots/api)). Public and private chats are supported. Supported message
types: audio, document, text, photo, video, video and voice notes. Registration as a client happens during first start (
type phone number and code).

### Data structure:

```go
type Telegram struct {
CHATID               string
CHATSOURCE           string
CHATTYPE             string
CHATTITLE            string
CHATCLIENTDATA       string
CHATLASTINBOXID      string
CHATLASTOUTBOXID     string
CHATMEMBERONLINE     string
CHATMESSAGETTL       string
CHATPROTECTEDCONTENT string
CHATUNREADCOUNT      string
CHATTIMESTAMP        string

MESSAGEID            string
MESSAGEMEDIA         []string
MESSAGESENDERID*     string
MESSAGETEXT*         string
MESSAGETEXTMARKDOWN  string
MESSAGETEXTURL       []string
MESSAGETYPE          string
MESSAGETIMESTAMP     string
MESSAGEURL*          string

USERID               string
USERVERSION          string
USERNAME             string
USERTYPE             string
USERLANG             string
USERFIRSTNAME        string
USERLASTNAME         string
USERPHONE            string
USERSTATUS           string
USERACCESSIBLE       string
USERCONTACT          string
USERFAKE             string
USERMUTUALCONTACT    string
USERSCAM             string
USERSUPPORT          string
USERVERIFIED         string
USERRESTRICTION      string
USERTIMESTAMP        string
}
```

&ast; - field may be used with **match_signature** parameter.

### Generic parameters:

| Param                 | Required |  Type  | Template |        Default        |
|:----------------------|:--------:|:------:|:--------:|:---------------------:|
| expire_action         |    -     | array  |    +     |          []           |
| expire_action_delay   |    -     | string |    +     |         "1d"          |
| expire_action_timeout |    -     |  int   |    +     |          30           |
| expire_interval       |    -     | string |    +     |         "7d"          |
| force                 |    -     |  bool  |    +     |         false         |
| force_count           |    -     |  int   |    +     |          100          |
| time_format           |    -     | string |    +     | "15:04:05 02.01.2006" |
| time_format_a         |    -     | string |    +     | "15:04:05 02.01.2006" |
| time_format_b         |    -     | string |    +     | "15:04:05 02.01.2006" |
| time_format_c         |    -     | string |    +     | "15:04:05 02.01.2006" |
| time_zone             |    -     | string |    +     |         "UTC"         |
| time_zone_a           |    -     | string |    +     |         "UTC"         |
| time_zone_b           |    -     | string |    +     |         "UTC"         |
| time_zone_c           |    -     | string |    +     |         "UTC"         |
| timeout               |    -     |  int   |    +     |          60           |

### Plugin parameters:

| Param                | Required |  Type  | Cred | Template |          Default          |                     Example                     | Description                                                                                                |
|:---------------------|:--------:|:------:|:----:|:--------:|:-------------------------:|:-----------------------------------------------:|:-----------------------------------------------------------------------------------------------------------|
| ads_enable           |    -     |  bool  |  -   |    -     |           true            |                      false                      | Enable/disable asking for sponsored messages (USERNAME will be "sponsoredMessage").                        |
| ads_period           |    -     | string |  -   |    -     |           "5m"            |                      "1h"                       | [Sponsored messages](https://core.telegram.org/api/sponsored-messages) receiving interval.                 |
| **api_id**           |    +     | string |  +   |    -     |            ""             |                       ""                        | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                            |
| **api_hash**         |    +     | string |  +   |    -     |            ""             |                       ""                        | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                            |
| app_version          |    -     | string |  -   |    -     |       v4.5.0-385342       |                     "0.0.1"                     | Custom application version.                                                                                |
| device_model         |    -     | string |  -   |    -     |         gosquito          |                 "Redmi Note 42"                 | Custom device model.                                                                                       |
| chat_database        |    -     | string |  -   |    +     | <PLUGIN_DIR>/chats.sqlite |               "/path/to/chats.db"               | Path to internal chats database.                                                                           |
| chat_save            |    -     |  bool  |  -   |    +     |           false           |                      true                       | Try to save all seen chats in internal database.                                                           |
| fetch_dir            |    -     | string |  -   |    +     |       <PLUGIN_DIR>        |                  "/data/files"                  | Directory where fetched files will be saved.                                                               |
| fetch_max_size       |    -     |  size  |  -   |    +     |           "10m"           |                      "1g"                       | Maximum file size for fetching.                                                                            |
| fetch_metadata       |    -     |  bool  |  -   |    +     |           false           |                      true                       | Generate JSON metadata for fetched file.                                                                   |
| fetch_mime           |    -     | array  |  -   |    +     |           "[]"            |   ["application/zip", "application/vnd.rar"]    | Fetch only specific mime types (audio, document, video).                                                   |
| fetch_mime_not       |    -     |  bool  |  -   |    +     |           false           |                      true                       | Fetch all mime types except specified in fetch_mime (if true).                                             |
| fetch_orig_name      |    -     |  bool  |  -   |    +     |           true            |                      false                      | Use original file name.                                                                                    |
| fetch_timeout        |    -     | string |  -   |    +     |           "1h"            |                      "24h"                      | Maximum time for fetching.                                                                                 |
| **input**            |    +     | array  |  -   |    +     |            []             |       ["breakingmash", "-1001117628569"]        | List of Telegram chats ("t.me/+" pattern is considered as a private chat).                                 |
| log_level            |    -     |  int   |  -   |    +     |             0             |                       90                        | [TDLib Log Level](https://core.telegram.org/tdlib/docs/classtd_1_1td__api_1_1set_log_verbosity_level.html) |
| match_signature      |    -     | array  |  -   |    +     |           "[]"            | ["telegram.messagetext", "telegram.messageurl"] | Match new messages by signature.                                                                           |
| match_ttl            |    -     | string |  -   |    +     |           "1d"            |                      "24h"                      | TTL (Time To Live) for matched signatures.                                                                 |
| message_edited       |    -     |  bool  |  -   |    +     |           false           |                      true                       | Include edited messages.                                                                                   |
| message_markdown     |    -     | string |  -   |    +     |        "internal"         |                   "telegram"                    | Algorithm for converting Telegram formatting to Markdown.                                                  |
| message_type_fetch   |    -     | array  |  -   |    +     |           "[]"            |              ["audio", "document"]              | Fetch files only for specific message types (audio, document, photo, video, video_note, voice_note).       |
| message_type_process |    -     | array  |  -   |    +     |           "[]"            |              ["audio", "document"]              | Process only specific message types (audio, document, photo, text, video, video_note, voice_note).         |
| message_translate    |    -     | string |  -   |    +     |            ""             |                      "en"                       | Language code to translate message text.                                                                   |
| message_view         |    -     |  bool  |  -   |    +     |           true            |                      false                      | Mark received messages as read.                                                                            |
| open_chat_enable     |    -     |  bool  |  -   |    +     |           true            |                      false                      | Enable/disable open/close chats for generating move updates events.                                        |
| open_chat_period     |    -     | string |  -   |    +     |           "10s"           |                      "1h"                       | Interval for opening/closing chats.                                                                        |
| pool_size            |    -     |  int   |  -   |    +     |          100000           |                      10000                      | Spool size for receiving updates.                                                                          |
| proxy_enable         |    -     |  bool  |  -   |    +     |           false           |                      true                       | Enable/disable proxy.                                                                                      |
| proxy_port           |    -     |  int   |  -   |    +     |           9050            |                      true                       | Proxy port number.                                                                                         |
| proxy_server         |    -     | string |  -   |    +     |        "127.0.0.1"        |                      true                       | Proxy server address.                                                                                      |
| proxy_username       |    -     | string |  +   |    -     |            ""             |                     "alex"                      | Proxy username.                                                                                            |
| proxy_password       |    -     | string |  +   |    -     |            ""             |                   "a1eXPass"                    | Proxy password.                                                                                            |
| proxy_type           |    -     | string |  -   |    +     |          "socks"          |                     "http"                      | Use original file names with random generated suffix.                                                      |
| session_ttl          |    -     |  int   |  -   |    +     |            366            |                       90                        | Session TTL (days).                                                                                        |
| status_enable        |    -     |  bool  |  -   |    +     |           true            |                      false                      | Enable/disable session status.                                                                             |
| status_period        |    -     | string |  -   |    +     |           "1h"            |                      "5m"                       | Interval for showing session status in plugin output.                                                      |
| storage_optimize     |    -     |  bool  |  -   |    +     |           true            |                      false                      | Enable/disable storage optimization (clean old data).                                                      |
| user_database        |    -     | string |  -   |    +     | <PLUGIN_DIR>/users.sqlite |               "/path/to/users.db"               | Path to internal users database.                                                                           |
| user_save            |    -     |  bool  |  -   |    +     |           false           |                      true                       | Enable/disable passive user logging.                                                                       |

### Flow sample:

```yaml
flow:
  name: "telegram-input"

  input:
    plugin: "telegram"
    params:
      cred: "creds.telegram.default"
      template: "templates.telegram.default"
      input: [ "breakingmash", "interfax_ru", "izvestia" ]
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
        input: [ "telegram.username" ]
        regexp: [ "sponsoredMessage" ]
        match_not: true

    - id: 1
      alias: "extract filename"
      plugin: "regexpfind"
      params:
        require: [ 0 ]
        input: [ "telegram.messagemedia" ]
        output: [ "data.array0" ]
        regexp: [ "^(.+)\\/([^\\/]+)$" ]
        group: [ [ 2 ] ]

    - id: 2
      plugin: "echo"
      alias: "show text"
      params:
        require: [ 0 ]
        input: [
          "telegram.chattitle",
          "telegram.chattype",
          "telegram.messageid",
          "telegram.messagemedia",
          "telegram.messagesenderid",
          "telegram.messagetext",
          "telegram.messagetextmarkdown",
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


