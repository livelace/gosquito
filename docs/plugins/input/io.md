### Description:

**io** input plugin is intended for IO operations with text and files.

### Data structure:

```go
type Io struct {
LINES  []string
MTIME* string
TEXT*  string
}
```

\* - field may be used with **match_signature** parameter.

### Generic parameters:

| Param                 | Required |  Type  | Template |        Default        |
|:----------------------|:--------:|:------:|:--------:|:---------------------:|
| expire_action         |    -     | array  |    +     |          []           |
| expire_action_delay   |    -     | string |    +     |         "1d"          |
| expire_action_timeout |    -     |  int   |    +     |          30           |
| expire_interval       |    -     | string |    +     |         "7d"          |
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

| Param         | Required |  Type  | Cred | Template | Text Template | Default |      Example       | Description                                        |
|:--------------|:--------:|:------:|:----:|:--------:|:-------------:|:-------:|:------------------:|:---------------------------------------------------|
| file_in       |    -     |  bool  |  -   |    +     |       -       |  false  |        true        | Process input as files.                            |
| file_in_mode  |    -     | string |  -   |    +     |       -       | "text"  |      "split"       | Read input file as a whole text or split to lines. |
| file_in_pre   |    -     | string |  -   |    +     |       -       |   ""    |        "_"         | Add characters to the beginning of data.           |
| file_in_post  |    -     | string |  -   |    +     |       -       |   ""    |        "_"         | Add characters to the end of data.                 |
| file_in_split |    -     | string |  -   |    +     |       -       |  "\n"   |       "AAA"        | Separation characters in split mode.               |
| **input**     |    +     | array  |  -   |    +     |       -       |  "[]"   | ["/path/to/file1"] | Set input as text or file paths.                   |

### Flow samples:

```yaml
flow:
  name: "io-input-example"

  input:
    plugin: "io"
    params:
      input: [ "Hello", ",", "World", "!" ]

  process:
    - id: 0
      plugin: "echo"
      params:
        input: [ "io.text" ]
```

```yaml
flow:
  name: "io-input-example"

  input:
    plugin: "io"
    params:
      input: [ "/etc/passwd" ]
      file_in: true
      file_in_mode: "lines"

  process:
    - id: 0
      plugin: "echo"
      alias: "show passwd records"
      params:
        input: [ "io.lines" ]
```

```yaml
flow:
  name: "io-input-example"

  input:
    plugin: "io"
    params:
      input: [ "/tmp/externally_updated_file" ]
      file_in: true
      file_in_mode: "text"
      match_signature: [ "io.mtime" ]
      match_ttl: [ "1d" ]

  process:
    - id: 0
      plugin: "echo"
      alias: "show updated content"
      params:
        input: [ "io.text" ]
```
