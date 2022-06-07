### Description:

**unique** process plugin is intended for removing duplicates inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Default | Example                         | Description                                                                   |
|:-----------|:--------:|:-----:|:-------:|:-------------------------------:|:------------------------------------------------------------------------------|
| **input**  | +        | array | []      | ["twitter.urls", "data.array0"] | List of [Datum](../../concept.md) fields with data.                           |
| **output** | +        | array | []      | ["data.array1"]                 | List of target [Datum](../../concept.md) fields. Must be array, single value. |

### Flow sample:

```yaml
flow:
  name: "unique-example"

  input:
    plugin: "rss"
    params:
      input: ["https://www.opennet.ru/opennews/opennews_all.rss"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "regexpfind"
      alias: "find english keywords"
      params:
        input:  ["rss.description"]
        output: ["data.array0"]
        regexp: ["[a-zA-Z]+?[a-zA-Z0-9+]+([ -]+?[a-zA-Z]+?[a-zA-Z0-9]+)?"]

    - id: 1
      plugin: "unique"
      alias: "unique english words"
      params:
        input:  ["data.array0", "rss.categories"]
        output: ["data.array1"]

    - id: 2
      plugin: "echo"
      params:
        input: ["data.array1"]
```
