### Description:

**regexpreplace** process plugin is intended for replacing patterns
inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param           | Required | Type  | Default |        Example        | Description                                         |
|:----------------|:--------:|:-----:|:-------:|:---------------------:|:----------------------------------------------------|
| **input**       |    +     | array |   []    |   ["twitter.text"]    | List of DataItem fields with data.                  |
| **output**      |    +     | array |   []    |    ["data.text0"]     | List of target DataItem fields.                     |
| **regexp**      |    +     | array |   []    | ["regexp.bad", "war"] | List of config templates/raw regexps for replacing. |
| **replacement** |    +     | array |   []    | ["vanished", "peace"] | List of replacements.                               |

### Config sample:

```toml

```

### Flow sample:

```yaml
```

