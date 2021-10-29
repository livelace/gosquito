### Description:

**regexpfind** process plugin is intended for finding patterns inside
data.


### Generic parameters:

| Param     | Required   | Type    | Template   | Default   | Example   |
| :-------- | :--------: | :-----: | :--------: | :-------: | :-------: |
| include   | -          | bool    | -          | true      | false     |
| require   | -          | array   | -          | []        | [1, 2]    |


### Plugin parameters:

| Param        | Required   | Type    | Template   | Default   | Example                    | Description                                                                   |
| :----------- | :--------: | :-----: | :--------: | :-------: | :------------------------: | :---------------------------------------------------------------------------- |
| find_all     | -          | bool    | -          | false     | true                       | Patterns must be found in all selected [DataItem](../../concept.md) fields.   |
| group        | -          | array   | -          | [][]      | [[1, 2], [3, 1]]           | Specific groups inside regexps.                                               |
| group_join   | -          | array   | -          | []        | ["/", "^^^"]               | Join matched groups with string.                                              |
| **input**    | +          | array   | -          | []        | ["twitter.text"]           | List of [DataItem](../../concept.md) fields with data.                        |
| match_case   | -          | array   | -          | true      | false                      | Case sensitive/insensitive.                                                   |
| **output**   | +          | array   | -          | []        | ["data.array0"]            | List of target [DataItem](../../concept.md) fields. Must be array.            |
| **regexp**   | +          | array   | +          | []        | ["http://go.tass.ru/.*"]   | List of config templates/raw regexps for searching.                           |

### Flow sample:

```yaml
flow:
  name: "regexpfind-example"

  input:
    plugin: "rss"
    params:
      input: ["https://www.interfax.ru/rss.asp", "https://tass.ru/rss/v2.xml"]
      force: true
      force_count: 100

  process:
    - id: 0
      plugin: "regexpfind"
      params:
        input:  ["rss.description"]
        output: ["data.array0"]
        regexp: [".*Росси.*"]

    - id: 1
      plugin: "echo"
      params:
        require: [0]
        input: ["data.array0"]
```
