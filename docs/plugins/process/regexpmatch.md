### Description:

**regexpmatch** process plugin is intended for matching patterns inside
data.


### Generic parameters:

| Param   | Required | Type  | Template | Default | Example |
|:--------|:--------:|:-----:|:--------:|:-------:|:-------:|
| include |    -     | bool  |    -     |  true   |  false  |
| require |    -     | array |    -     |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Template | Default |     Example      | Description                                                                   |
|:-----------|:--------:|:-----:|:--------:|:-------:|:----------------:|:------------------------------------------------------------------------------|
| **input**  |    +     | array |    -     |   []    | ["twitter.text"] | List of [DataItem](../../concept.md) fields with data.                        |
| match_all  |    -     | bool  |    -     |  false  |       true       | Patterns must be matched in all selected [DataItem](../../concept.md) fields. |
| match_case |    -     | bool  |    -     |  true   |      false       | Case sensitive/insensitive.                                                   |
| match_not  |    -     | bool  |    -     |  false  |       true       | Logical not match pattern.                                                    |
| output     |    -     | array |    -     |   []    |  ["data.text0"]  | List of target [DataItem](../../concept.md) fields.                           |
| **regexp** |    +     | array |    +     |   []    |    ["Россия"]    | List of config templates/raw regexps for matching.                            |


### Flow sample:

```yaml
flow:
  name: "regexpmatch-example"

  input:
    plugin: "rss"
    params:
      input: ["https://www.interfax.ru/rss.asp", "https://tass.ru/rss/v2.xml"]
      force: true
      force_count: 100

  process:
    - id: 0
      plugin: "regexpmatch"
      params:
        input:  ["rss.description"]
        regexp: [".*Росси.*"]

    - id: 1
      plugin: "echo"
      alias: "echo news with Россия word"
      params:
        require: [0]
        input: ["rss.description"]
```

