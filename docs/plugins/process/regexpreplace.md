### Description:

**regexpreplace** process plugin is intended for replacing patterns
inside data.


### Generic parameters:

| Param   | Required | Type  | Template | Default | Example |
|:--------|:--------:|:-----:|:--------:|:-------:|:-------:|
| include |    -     | bool  |    -     |  true   |  false  |
| require |    -     | array |    -     |   []    | [1, 2]  |


### Plugin parameters:

| Param       | Required | Type  | Template | Default |     Example      | Description                                                                    |
|:------------|:--------:|:-----:|:--------:|:-------:|:----------------:|:-------------------------------------------------------------------------------|
| **input**   |    +     | array |    -     |   []    | ["twitter.text"] | List of [DataItem](../../concept.md) fields with data.                         |
| match_case  |    -     | array |    -     |  true   |      false       | Case sensitive/insensitive.                                                    |
| **output**  |    +     | array |    -     |   []    |  ["data.text0"]  | List of target [DataItem](../../concept.md) fields.                            |
| **regexp**  |    +     | array |    +     |   []    |     ["war"]      | List of config templates/raw regexps for replacing.                            |
| **replace** |    +     | array |    -     |   []    |    ["peace"]     | List of replacements.                                                          |
| replace_all |    -     | bool  |    -     |  false  |       true       | Patterns must be replaced in all selected [DataItem](../../concept.md) fields. |

### Flow sample:

```yaml
flow:
  name: "regexpreplace-example"

  input:
    plugin: "rss"
    params:
      input: ["https://www.interfax.ru/rss.asp"]
      force: true
      force_count: 1

  process:
    - id: 0
      plugin: "regexpreplace"
      params:
        input:   ["rss.title"]
        output:  ["data.text0"]
        regexp:  [" "]
        replace: ["_"]

    - id: 1
      plugin: "echo"
      params:
        require: [0]
        input: ["rss.title", "data.text0"]
```

