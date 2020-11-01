### Description:

**regexpmatch** process plugin is intended for matching patterns inside
data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Default |             Example             | Description                                        |
|:-----------|:--------:|:-----:|:-------:|:-------------------------------:|:---------------------------------------------------|
| **input**  |    +     | array |   []    |        ["twitter.text"]         | List of DataItem fields with data.                 |
| output     |    -     | array |   []    |         ["data.text0"]          | List of target DataItem fields.                    |
| **regexp** |    +     | array |   []    | ["regexps.countries", "Россия"] | List of config templates/raw regexps for matching. |

### Config sample:

```toml

```

### Flow sample:

```yaml
```

