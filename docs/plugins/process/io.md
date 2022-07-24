### Description:

**io** process plugin is intended for IO operations with text and files.

### Generic parameters:

| Param   | Required | Type  | Template | Default | Example |
| :------ | :------: | :---: | :------: | :-----: | :-----: |
| include |    -     | bool  |    -     |  false  |  true   |
| require |    -     | array |    -     |   []    | [1, 2]  |

### Plugin parameters:

| Param         | Required |  Type  | Cred | Template | Text Template |  Default   |             Example             | Description                                                       |
| :------------ | :------: | :----: | :--: | :------: | :-----------: | :--------: | :-----------------------------: | :---------------------------------------------------------------- |
| file_in       |    -     |  bool  |  -   |    +     |       -       |   false    |              true               | Process input as files.                                           |
| file_in_mode  |    -     | string |  -   |    +     |       -       |   "text"   |             "lines"             | Read input file as text or line by line into array.               |
| file_out      |    -     |  bool  |  -   |    +     |       -       |   false    |              true               | Process output as files.                                          |
| file_out_mode |    -     | string |  -   |    +     |       -       | "truncate" |            "append"             | Append data to output files (truncate by default).                |
| file_out_wrap |    -     | string |  -   |    +     |       -       |    "\n"    |              "AAA"              | Wrap appended data to file with this characters.                          |
| text_wrap     |    -     | string |  -   |    +     |       -       |    "\n"    |              "BBB"              | Wrap processed text with this characters.                          |
| **input**     |    +     | array  |  -   |    +     |       -       |    "[]"    | ["/path/to/file1", "just text"] | Set input as text, file paths or [Datum](../../concept.md) field. |
| **output**    |    +     | array  |  -   |    +     |       -       |    "[]"    |  ["data.array0", "data.text0"]  | Set output as text, file paths or [Datum](../../concept.md) field. |

### Flow sample:

```yaml
flow:
  name: "io-process-example"

  input:
    plugin: "rss"
    params:
      input: ["http://feeds.dzone.com/home"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "io"
      alias: "copy files"
      params:
        input: ["/tmp/file1", "/tmp/file2"]
        output: ["/tmp/file1_copy", "/tmp/file2_copy"]
        file_in: true
        file_out: true

    - id: 1
      plugin: "io"
      alias: "write data to files"
      params:
        input: ["rss.categories", "rss.link"]
        output: ["/tmp/rss_categories", "/tmp/rss_link"]
        file_out: true
        file_out_mode: "append"
        text_wrap: ";"

    - id: 2
      plugin: "io"
      alias: "write data to fields"
      params:
        input: ["rss.categories", "rss.link"]
        output: ["data.array0", "data.text0"]
    
    - id: 3
      plugin: "echo"
      alias: "show copied fields"
      params:
        input: ["data.array0", "data.text0"]
    
    - id: 4
      plugin: "io"
      alias: "read data from files"
      params:
        input: ["/tmp/rss_categories", "/tmp/rss_link"]
        output: ["data.text1", "data.text2"]
        file_in: true
        file_in_mode: "text"
    
    - id: 5
      plugin: "echo"
      alias: "show read files"
      params:
        input: ["data.text1", "data.text2"]
```
