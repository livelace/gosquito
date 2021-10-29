### Description:

**jq** process plugin is intended for parsing JSON data.


### Generic parameters:

| Param     | Required   | Type    | Template   | Default   | Example   |
| :-------- | :--------: | :-----: | :--------: | :-------: | :-------: |
| include   | -          | bool    | -          | true      | false     |
| require   | -          | array   | -          | []        | [1, 2]    |


### Plugin parameters:

| Param        | Required   | Type    | Template   | Default   | Example                    | Description                                                                                                                |
| :----------- | :--------: | :-----: | :--------: | :-------: | :------------------------: | :------------------------------------------------------------------------------------------------------------------------- |
| find_all     | -          | bool    | -          | false     | true                       | Query must be found in all selected [DataItem](../../concept.md) fields.                                                   |
| **input**    | +          | array   | -          | []        | ["data.array0"]            | List of [DataItem](../../concept.md) fields with data.                                                                     |
| **output**   | +          | array   | -          | []        | ["data.array1"]            | List of target [DataItem](../../concept.md) fields.                                                                        |
| **query**    | +          | array   | +          | []        | [".foo", ".bar"]           | List of config templates/raw queries for searching.                                                                        |

### Flow sample:

```yaml
flow:
  name: "jq-example"

  input:
    plugin: "resty"
    params:
      input: ["https://freegeoip.app/json/"]

  process:
    - id: 0
      plugin: "jq"
      params:
        input:  ["resty.body", "resty.body"]
        output: ["data.array0", "data.array1"]
        query:  ["templates.jq.example", ".ip"]

    - id:  1
      plugin: "echo"
      params:
        input: ["data.array0", "data.array1"]     
```

### Config sample:

```toml
[templates.jq.example]
query = [
    '.country_code',
    '.country_name',
]
```

