### Description:

**iconv** process plugin is intended for converting text encoding.

### Generic parameters:

| Param   | Required | Type  | Template | Default | Example |
|:--------|:--------:|:-----:|:--------:|:-------:|:-------:|
| include |    -     | bool  |    -     |  false  |  true   |
| require |    -     | array |    -     |   []    | [1, 2]  |

### Plugin parameters:

| Param      | Required |  Type  | Template | Default |             Example             | Description                                         |
|:-----------|:--------:|:------:|:--------:|:-------:|:-------------------------------:|:----------------------------------------------------|
| **input**  |    +     | array  |    -     |   []    | ["twitter.text", "data.array1"] | List of [Datum](../../concept.md) fields with data. |
| **output** |    +     | array  |    -     |   []    |         ["data.text0"]          | List of target [Datum](../../concept.md) fields.    |
| **from**   |    +     | string |    +     |   ""    |             "cp866"             | Source encoding.                                    |
| to         |    +     | string |    -     | "utf-8" |            "cp1251"             | Target encoding.                                    |

### Flow sample:

```yaml
flow:
  name: "iconv-example"

  input:
    plugin: "rss"
    params:
      input: [ "https://www.interfax.ru/rss.asp" ]
      force: true
      force_count: 1

  process:
    - id: 0
      plugin: "iconv"
      params:
        input: [ "rss.title" ]
        output: [ "data.text0" ]
        from: "cp1251"
        to: "utf-8"

    - id: 1
      plugin: "echo"
      params:
        require: [ 0 ]
        input: [ "rss.title", "data.text0" ]
```

