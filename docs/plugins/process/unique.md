### Description:

**unique** process plugin is intended for remove duplicates inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Default |             Example             | Description |
|:-----------|:--------:|:-----:|:-------:|:-------------------------------:|:------------|
| **input**  |    +     | array |   []    | ["twitter.urls", "data.array0"] | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with data.            |
| **output** |    +     | array |   []    |         ["data.array1"]         | List of target [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields.            |

### Config sample:

```toml

```

### Flow sample:

```yaml
```