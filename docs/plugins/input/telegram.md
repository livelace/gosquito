### Description:

**telegram** input plugin is intended for data gathering from [Telegram](https://telegram.org/) chats.    
This plugin uses [TDLib client API](https://core.telegram.org/tdlib) (not [Telegram Bot API](https://core.telegram.org/bots/api)).  
Right now plugin supports only [public chats](https://core.telegram.org/tdlib/getting-started) (find, join, receive messages and files).  
Registration as a client happens during first start (type phone number and verify code). Only one client/phone may be used per application (not flow).

### Data structure:

```go
type Telegram struct {
	FIRSTNAME* string
	LASTNAME*  string
	MEDIA      []string
	PHONE*     string
	TEXT*      string
	URL*       string
	USERNAME*  string
	USERTYPE*  string
}
```

&ast; - field may be used with **match_signature** parameter.

### Generic parameters:

| Param                   | Required   | Type     | Template   | Default                 |
| :---------------------- | :--------: | :------: | :--------: | :---------------------: |
| expire_action           | -          | array    | +          | []                      |
| expire_action_delay     | -          | string   | +          | "1d"                    |
| expire_action_timeout   | -          | int      | +          | 30                      |
| expire_interval         | -          | string   | +          | "7d"                    |
| timeout                 | -          | int      | +          | 60                      |
| time_format             | -          | string   | +          | "15:04:05 02.01.2006"   |
| time_zone               | -          | string   | +          | "UTC"                   |


### Plugin parameters:

| Param           | Required   | Type     | Cred   | Template   | Default   | Example            | Description                                                                                                  |
| :-------------- | :--------: | :------: | :----: | :--------: | :-------: | :----------------: | :----------------------------------------------------------------------------------------------------------- |
| **api_id**      | +          | string   | +      | -          | ""        | ""                 | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                              |
| **api_hash**    | +          | string   | +      | -          | ""        | ""                 | [Telegram Apps](https://core.telegram.org/api/obtaining_api_id)                                              |
| file_max_size   | -          | size     | -      | +          | "10m"     | "1g"               | Maximum file size for download.                                                                              |
| **input**       | +          | array    | -      | +          | []        | ["breakingmash"]   | List of Telegram chats.                                                                                      |
| match_signature | -          | array    | -      | +          | "[]"      | ["text", "time"]   | Match new messages by signature.                                                                               |
| match_ttl       | -          | string   | -      | +          | "1d"      | "24h"              | TTL (Time To Live) for matched signatures.                                                                   |
| log_level       | -          | int      | -      | +          | 0         | 90                 | [TDLib Log Level](https://core.telegram.org/tdlib/docs/classtd_1_1td__api_1_1set_log_verbosity_level.html)   |


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
      alias: "match all"
      plugin: "regexpmatch"
      params:
        input:  ["telegram.text"]
        regexp: [".*"]

    - id: 1
      alias: "replace newline"
      plugin: "regexpreplace"
      params:
        include: false
        input:  ["telegram.text"]
        output: ["data.text0"]
        regexp: ["\n"]
        replace: ["<br>"]

  output:
    plugin: "smtp"
    params:
      template: "templates.telegram.smtp.default"
```


### Config sample:

```toml
[creds.telegram.default]
api_id = "<API_ID>"
api_hash = "<API_HASH>"

[templates.telegram.default]
#file_max_size = "30m"
#log_level = 90

[templates.telegram.smtp.default]
server = "mail.example.com"

from = "gosquito@example.com"
output = ["user@example.com"]

subject = "{{.TELEGRAM.TEXT}}"

body = """
    <div align="right"><b>{{.FLOW}}&nbsp;&nbsp;&nbsp;{{.TIMEFORMAT}}</b></div>
    {{ .DATA.TEXT0 }}<br><br>
    {{ .TELEGRAM.URL }}
    """
attachments = ["telegram.media"]
```


