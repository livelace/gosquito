### Description:

**mattermost** output plugin is intended for sending data to Mattermost
users/channels.


### Generic parameters:

| Param   | Required | Type | Template | Default |
|:--------|:--------:|:----:|:--------:|:-------:|
| timeout |    -     | int  |    +     |    3    |


### Plugin parameters:

| Param        | Required |  Type  | Cred | Template | Text Template | Default |          Example           | Description                                                                                      |
|:-------------|:--------:|:------:|:----:|:--------:|:-------------:|:-------:|:--------------------------:|:-------------------------------------------------------------------------------------------------|
| attachments  |    -     |  map   |  -   |    +     |       -       |  map[]  |        see example         | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| files        |    -     | array  |  -   |    +     |       -       |   ""    |      ["data.array0"]       | List of DataItem fields which contains files paths.                                              |
| message      |    -     | string |  -   |    +     |       +       |   ""    |    "Hello, {{.FLOW}}!"     | Message text.                                                                                    |
| **output**   |    +     | array  |  -   |    +     |       -       |   []    |   ["news", "@livelace"]    | List of channels/users.                                                                          |
| **password** |    +     | string |  +   |    -     |       -       |   ""    |             ""             | Mattermost password.                                                                             |
| **team**     |    +     | string |  +   |    -     |       -       |   ""    |        "superteam"         | Mattermost team.                                                                                 |
| **url**      |    +     | string |  +   |    -     |       -       |   ""    | "https://host.example.com" | Mattermost URL.                                                                                  |
| **username** |    +     | string |  +   |    -     |       -       |   ""    |             ""             | Mattermost user.                                                                                 |


### Attachments parameters:

| Param      | Required |  Type  | Template | Text Template |  Default  |          Example          | Description                                                                                      |
|:-----------|:--------:|:------:|:--------:|:-------------:|:---------:|:-------------------------:|:-------------------------------------------------------------------------------------------------|
| color      |    -     | string |    +     |       -       | "#00C100" |         "#E40303"         | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| pretext    |    -     | string |    +     |       +       |    ""     | "Pretext {{.TIMEFORMAT}}" | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| text       |    -     | string |    +     |       +       |    ""     |    "Hello, {{.FLOW}}!"    | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| title      |    -     | string |    +     |       +       |    ""     |     "Title {{.UUID}}"     | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |
| title_link |    -     | string |    +     |       -       |    ""     |   "https://example.com"   | [Mattermost Message Attachments](https://docs.mattermost.com/developer/message-attachments.html) |


### Config sample:

```toml

```

### Flow sample:

```yaml
```

