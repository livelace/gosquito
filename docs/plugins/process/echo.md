### Description:

**echo** process plugin is intended for echoing processing data into console.


### Generic parameters:

| Param     | Required   | Type    | Default   | Example   |
| :-------- | :--------: | :-----: | :-------: | :-------: |
| require   | -          | array   | []        | [1, 2]    |


### Plugin parameters:

| Param       | Required   | Type    | Default   | Example                           | Description                                              |
| :---------- | :--------: | :-----: | :-------: | :-------------------------------: | :-------------------------------------                   |
| **input**   | +          | array   | []        | ["rss.title"]                     | List of [DataItem](../../concept.md) fields for echoing. |

### Flow sample:

```yaml
flow:
  name: "echo-example"

  input:
    plugin: "rss"
    params:
      input: ["https://tass.ru/rss/v2.xml"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "echo"
      params:
        input: ["rss.title"]

```



