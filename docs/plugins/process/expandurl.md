### Description:

**expandurl** process plugin is intended for expanding short URLs into
full URLs.


### Generic parameters:

| Param   | Required | Type  | Default | Example | Description |
|:--------|:--------:|:-----:|:-------:|:-------:|:------------|
| include |    -     | bool  |  true   |  false  |             |
| require |    -     | array |   []    | [1, 2]  |             |
| timeout |    -     |  int  |    3    |    2    |             |


### Plugin parameters:

| Param      | Required |  Type  |      Default      |     Example      | Description |
|:-----------|:--------:|:------:|:-----------------:|:----------------:|:------------|
| depth      |    -     |  int   |         3         |        10        |             |
| **input**  |    +     | array  |        []         | ["twitter.urls"] |             |
| **output** |    +     | array  |        []         | ["data.array0"]  |             |
| user_agent |    -     | string | "gosquito v1.0.0" |  "webchela 1.0"  |             |

### Config sample:

```toml

```

### Flow sample:

```yaml
```

