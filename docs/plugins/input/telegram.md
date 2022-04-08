### Description:

**telegram** input plugin is intended for data gathering from [Telegram](https://telegram.org/) chats.    
  
This plugin uses [TDLib client API](https://core.telegram.org/tdlib) (not [Telegram Bot API](https://core.telegram.org/bots/api)). Public and private chats are supported. Supported message types: audio, document, text, photo, video, voice and video notes. Registration as a client happens during first start (type phone number and code). Only single client (phone number) per gosquito instance is supported right now.

### Data structure:

```go
type Telegram struct {
    CHATID    string
    CHATTITLE string
    CHATTYPE  string
    
    MESSAGEID       string
    MESSAGEMEDIA    []string
    MESSAGESENDERID string
    MESSAGETEXT*    string
    MESSAGETEXTURL  []string
    MESSAGETYPE     string
    MESSAGEURL*     string
    
    USERID        string
    USERNAME      string
    USERTYPE      string
    USERFIRSTNAME string
    USERLASTNAME  string
    USERPHONE     string
	
    WARNINGS []string
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
| timeout               |    -     |  int   |    +     |          60           |
| time_format           |    -     | string |    +     | "15:04:05 02.01.2006" |
| time_zone             |    -     | string |    +     |         "UTC"         |


### Plugin parameters:

| Param             | Required |  Type  | Cred | Template |    Default    |      Example       | Description                                                                                                |
|:------------------|:--------:|:------:|:----:|:--------:|:-------------:|:------------------:|:-----------------------------------------------------------------------------------------------------------|
| ads_period        |    -     | string |  -   |    -     |     "5m"      |        "1h"        | [Sponsored messages](https://core.telegram.org/api/sponsored-messages) receiving interval.                 |
| **api_id**        |    +     | string |  +   |    -     |      ""       |         ""         | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                            |
| **api_hash**      |    +     | string |  +   |    -     |      ""       |         ""         | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                            |
| app_version       |    -     | string |  +   |    -     | v3.5.1-b905a4 |      "0.0.1"       | Custom application version.                                                                                |
| device_model      |    -     | string |  +   |    -     |   gosquito    |  "Redmi Note 42"   | Custom device model.                                                                                       |
| file_max_size     |    -     |  size  |  -   |    +     |     "10m"     |        "1g"        | Maximum file size for download.                                                                            |
| **input**         |    +     | array  |  -   |    +     |      []       |  ["breakingmash"]  | List of Telegram chats ("t.me/+" pattern is considered as a private chat).                                 |
| match_signature   |    -     | array  |  -   |    +     |     "[]"      | ["source", "time"] | Match new messages by signature.                                                                           |
| match_ttl         |    -     | string |  -   |    +     |     "1d"      |       "24h"        | TTL (Time To Live) for matched signatures.                                                                 |
| original_filename |    -     |  bool  |  -   |    +     |     true      |       false        | Use original file names with random generated suffix.                                                      |
| log_level         |    -     |  int   |  -   |    +     |       0       |         90         | [TDLib Log Level](https://core.telegram.org/tdlib/docs/classtd_1_1td__api_1_1set_log_verbosity_level.html) |


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

  process:
    - id: 0
      alias: "replace newline"
      plugin: "regexpreplace"
      params:
        input:  ["telegram.text"]
        output: ["data.text0"]
        regexp: ["\n"]
        replace: ["<br>"]

    - id: 1
      plugin: "echo"
      alias: "show replaced text"
      params:
        input: ["data.text0"]

```


### Config sample:

```toml
[creds.telegram.default]
api_id = "<API_ID>"
api_hash = "<API_HASH>"

[templates.telegram.default]
#file_max_size = "30m"
#log_level = 90
```


