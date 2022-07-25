### Description:

**split** process plugin is intended for splitting single datum into multiple datums.

### Generic parameters:

| Param   | Required | Type  | Default | Example |
| :------ | :------: | :---: | :-----: | :-----: |
| include |    -     | bool  |  false  |  true   |
| require |    -     | array |   []    | [1, 2]  |
| timeout |    -     |  int  |   60    |   300   |

### Plugin parameters:

| Param       | Required |  Type  |  Default   |            Example             | Description                                             |
| :---------- | :------: | :----: | :--------: | :----------------------------: | :------------------------------------------------------ |
| **input**   |    +     | array  |     []     | ["data.array0", "data.array1"] | Slice of strings [Datum](../../concept.md) fields.      |
| **output**  |    +     | array  |     []     |  ["data.text0", "data.text1"]  | Strings [Datum](../../concept.md) fields.               |
| mode        |    -     | string |  "strict"  |            "sparse"            | Input arrays may have different sizes if mode "sparse". |
| sparse_stub |    -     | string | "!SPARSE!" |             "AAA"              | Stub for absent values.                                 |

### Flow sample:

```yaml
flow:
  name: "split-example"

  input:
    plugin: "resty"
    params:
      input: ["https://www.drupal.org/api-d7/user.json"]

  process:
    - id: 0
      plugin: "jq"
      alias: "extract users"
      params:
        input: ["resty.body", "resty.body"]
        output: ["data.array0", "data.array1"]
        query: [".list[].uid", ".list[].name"]

    - id: 1
      plugin: "split"
      alias: "split users into datums"
      params:
        require: [0]
        input: ["data.array0", "data.array1"]
        output: ["data.text0", "data.text1"]

    - id: 2
      plugin: "echo"
      alias: "echo users"
      params:
        require: [1]
        input: ["data.text0", "data.text1", "---"]
```
