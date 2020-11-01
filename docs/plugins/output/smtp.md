### Description:

**smtp** output plugin is intended for sending data as emails.


### Generic parameters:

| Param   | Required | Type | Template | Default | Description |
|:--------|:--------:|:----:|:--------:|:-------:|:------------|
| timeout |    -     | int  |    +     |   60    |             |


### Plugin parameters:

| Param          | Required |  Type  | Cred | Template | Text Template | Default |        Example         | Description                               |
|:---------------|:--------:|:------:|:----:|:--------:|:-------------:|:-------:|:----------------------:|:------------------------------------------|
| attachments    |    -     | array  |  -   |    +     |       -       |   []    |    ["data.array0"]     | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with files paths. |
| **body**       |    +     | string |  -   |    +     |       +       |   ""    |   "{{.RSS.CONTENT}}"   | Email body.                               |
| body_html      |    -     |  bool  |  -   |    +     |       -       |  true   |         false          | Send body as HTML.                        |
| body_length    |    -     |  int   |  -   |    +     |       -       |  10000  |          1000          | Maximum body length in letters.           |
| **from**       |    +     | string |  -   |    +     |       -       |   ""    | "gosquito@example.com" | Email from.                               |
| headers        |    -     | map[]  |  -   |    +     |       -       |  map[]  |      see example       | Dynamic list of email headers.            |
| **output**     |    +     | array  |  -   |    +     |       -       |   []    | ["user1@example.com"]  | List of recipients.                       |
| password       |    -     | string |  +   |    -     |       -       |   ""    |           ""           | SMTP password.                            |
| port           |    -     |  int   |  -   |    +     |       -       |   25    |          465           | SMTP port.                                |
| **server**     |    +     | string |  -   |    +     |       -       |   ""    |   "mail.example.com"   | SMTP server.                              |
| ssl            |    -     |  bool  |  -   |    +     |       -       |  true   |         false          | Use SSL for connection.                   |
| **subject**    |    +     | string |  -   |    +     |       +       |   ""    |  "{{.TWITTER.TEXT}}"   | Email subject.                            |
| subject_length |    -     |  int   |  -   |    +     |       -       |   100   |          300           | Maximum subject length in letters.        |
| username       |    -     |  int   |  +   |    -     |       -       |   ""    |           ""           | SMTP user.                                |


### Flow sample:

```yaml
flow:
  name: "smtp-example"

  input:
    plugin: "rss"
    params:
      input: ["https://iz.ru/xml/rss/all.xml", "http://tass.ru/rss/v2.xml"]

  output:
    plugin: "smtp"
    params:
      cred: "creds.smtp.default"
      template: "templates.rss.smtp.default"
      headers:
        foo: "bar"
```

### Config sample:

```toml
[creds.smtp.default]
username = "<USERNAME>"
password = "<PASSWORD>"

[templates.rss.smtp.default]
server = "mail.example.com"
port = 25
ssl = true

from = "gosquito@example.com"
output = ["user@example.com"]

subject = "{{ .RSS.TITLE }}"
subject_length = 150

body = """
    <div align="right"><b>{{ .FLOW }}&nbsp;&nbsp;&nbsp;{{ .TIMEFORMAT }}</b></div>
    {{ .RSS.TITLE }}<br>
    {{ if .RSS.DESCRIPTION }}{{ .RSS.DESCRIPTION }}<br>{{end}}
    {{ if .RSS.CONTENT }}{{ .RSS.CONTENT }}<br><br>{{else}}<br>{{end}}
    {{ if .RSS.LINK }}{{ .RSS.LINK }}{{end}}
    """
    
body_html = true
body_length = 10000

[templates.rss.smtp.default.headers]
x-gosquito-flow   = "flow"
x-gosquito-plugin = "plugin"
x-gosquito-source = "source"
x-gosquito-time   = "time"
x-gosquito-uuid   = "uuid"
```
