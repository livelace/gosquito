### Description:

**regexpmatch** process plugin is intended for matching patterns inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example | Description |
|:--------|:--------:|:-----:|:-------:|:-------:|:------------|
| include |    -     | bool  |  true   |  false  |             |
| require |    -     | array |   []    | [1, 2]  |             |


### Plugin parameters:

| Param      | Required | Type  | Default |     Example      | Description |
|:-----------|:--------:|:-----:|:-------:|:----------------:|:------------|
| **input**  |    +     | array |   []    | ["twitter.text"] |             |
| output     |    -     | array |   []    |  ["data.text0"]  |             |
| **regexp** |    +     | array |   []    |    ["Россия"]    |             |

### Config sample:

```toml

```

### Flow sample:

```yaml
```