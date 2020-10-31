### Description:

**smtp** output plugin is intended for sending data as emails.


### Generic parameters:

| Param   | Required | Type | Template | Default | Description |
|:--------|:--------:|:----:|:--------:|:-------:|:------------|
| timeout |    -     | int  |    +     |   60    |             |


### Plugin parameters:

| Param          | Required |  Type  | Cred | Template | Text Template | Default |             Example              | Description |
|:---------------|:--------:|:------:|:----:|:--------:|:-------------:|:-------:|:--------------------------------:|:------------|
| attachments    |    -     | array  |  -   |    +     |       -       |   []    | ["twitter.media", "data.array0"] |             |
| **body**       |    +     | string |  -   |    +     |       +       |   ""    |        "{{.RSS.CONTENT}}"        |             |
| body_html      |    -     |  bool  |  -   |    +     |       -       |  true   |              false               |             |
| body_length    |    -     |  int   |  -   |    +     |       -       |  10000  |               1000               |             |
| **from**       |    +     | string |  -   |    +     |       -       |   ""    |      "gosquito@example.com"      |             |
| headers        |    -     | map[]  |  -   |    +     |       -       |  map[]  |           see example            |             |
| **output**     |    +     | array  |  -   |    +     |       -       |   []    |      ["user1@example.com"]       |             |
| password       |    -     | string |  +   |    -     |       -       |   ""    |                ""                |             |
| port           |    -     |  int   |  -   |    +     |       -       |   25    |               465                |             |
| **server**     |    +     | string |  -   |    +     |       -       |   ""    |        "mail.example.com"        |             |
| ssl            |    -     |  bool  |  -   |    +     |       -       |  true   |              false               |             |
| **subject**    |    +     | string |  -   |    +     |       +       |   ""    |       "{{.TWITTER.TEXT}}"        |             |
| subject_length |    -     |  int   |  -   |    +     |       -       |   100   |               300                |             |
| username       |    -     |  int   |  +   |    -     |       -       |   ""    |                ""                |             |


### Config sample:

```toml

```

### Flow sample:

```yaml
```