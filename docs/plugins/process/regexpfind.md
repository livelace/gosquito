### Description:

**regexpfind** process plugin is intended for finding patterns inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Default |                 Example                  | Description                                         |
|:-----------|:--------:|:-----:|:-------:|:----------------------------------------:|:----------------------------------------------------|
| **input**  |    +     | array |   []    |             ["twitter.urls"]             | List of DataItem fields with data.                  |
| **output** |    +     | array |   []    |             ["data.array0"]              | List of target DataItem fields.                     |
| **regexp** |    +     | array |   []    | ["regexp.video", "http://go.tass.ru/.*"] | List of config templates/raw regexps for searching. |

### Config sample:

```toml

```

### Flow sample:

```yaml
```