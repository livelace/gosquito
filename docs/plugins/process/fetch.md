### Description:

**fetch** process plugin is intended for downloading files.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |
| timeout |    -     |  int  |   60    |   300   |


### Plugin parameters:

| Param      | Required | Type  | Default |      Example      | Description                        |
|:-----------|:--------:|:-----:|:-------:|:-----------------:|:-----------------------------------|
| **input**  |    +     | array |   []    | ["twitter.media"] | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with URLs. |
| **output** |    +     | array |   []    |  ["data.array0"]  | List of target DataItem fields.    |

### Config sample:

```toml

```

### Flow sample:

```yaml
```

