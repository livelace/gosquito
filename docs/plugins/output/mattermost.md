### Description:

**mattermost** output plugin is for sending data to Mattermost users/channels.


### Generic parameters:

| Param   | Required |  Type   | Template | Default | Description |
|:--------|:--------:|:-------:|:--------:|:-------:|:------------|
| timeout |    -     | seconds |    +     |    3    |             |


### Plugin parameters:

| Param       | Required |  Type  | Cred | Template | Text Template | Default |             Example              | Description |
|:------------|:--------:|:------:|:----:|:--------:|:-------------:|:-------:|:--------------------------------:|:------------|
| attachments |    -     |  map   |  -   |    +     |       -       |  map[]  |           see example            |             |
| files       |    -     | array  |  -   |    +     |       -       |   ""    | ["twitter.media", "data.array0"] |             |
| message     |    -     | string |  -   |    +     |       +       |   ""    |       "Hello, {{.FLOW}}!"        |             |
| output      |    +     | array  |  -   |    +     |       -       |   []    |      ["news", "@livelace"]       |             |
| password    |    +     | string |  +   |    -     |       -       |   ""    |                ""                |             |
| team        |    +     | string |  +   |    -     |       -       |   ""    |           "superteam"            |             |
| url         |    +     | string |  +   |    -     |       -       |   ""    | "https://mattermost.example.com" |             |
| username    |    +     | string |  +   |    -     |       -       |   ""    |                ""                |             |


### Attachments parameters:

| Param      | Required |  Type  | Template | Text Template |  Default  |          Example          | Description |
|:-----------|:--------:|:------:|:--------:|:-------------:|:---------:|:-------------------------:|:------------|
| color      |    -     | string |    +     |       -       | "#00C100" |         "#E40303"         |             |
| pretext    |    -     | string |    +     |       +       |    ""     | "Pretext {{.TIMEFORMAT}}" |             |
| text       |    -     | string |    +     |       +       |    ""     |    "Hello, {{.FLOW}}!"    |             |
| title      |    -     | string |    +     |       +       |    ""     |     "Title {{.UUID}}"     |             |
| title_link |    -     | string |    +     |       -       |    ""     |   "https://example.com"   |             |


### Config sample:

```toml

```

### Flow sample:

```yaml
```