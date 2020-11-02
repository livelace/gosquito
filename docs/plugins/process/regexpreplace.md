### Description:

**regexpreplace** process plugin is intended for replacing patterns
inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param       | Required | Type  | Default |        Example         | Description                                                                                                                 |
|:------------|:--------:|:-----:|:-------:|:----------------------:|:----------------------------------------------------------------------------------------------------------------------------|
| **input**   |    +     | array |   []    |    ["twitter.text"]    | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with data.                         |
| **output**  |    +     | array |   []    |     ["data.text0"]     | List of target [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields.                            |
| **regexp**  |    +     | array |   []    | ["regexps.bad", "war"] | List of config templates/raw regexps for replacing.                                                                         |
| **replace** |    +     | array |   []    | ["vanished", "peace"]  | List of replacements.                                                                                                       |
| replace_all |    -     | bool  |  false  | ["vanished", "peace"]  | Patterns must be replaced in all selected [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields. |

### Flow sample:

```yaml
flow:
  name: "regexpreplace-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["AP"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "regexpreplace"
      params:
        input:  ["twitter.text", "twitter.urls"]
        output: ["data.text0", "data.array0"]
        regexp:  ["for", "regexps.urls"]
        replace: ["FOR", "<URL>"]
        replace_all: true

    - id: 1
      alias: "echo text"
      plugin: "echo"
      params:
        require: [0]
        input: ["data.text0", "data.array0"]
```

### Config sample:

```toml
[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"

[regexps.words]
regexp = [" to "]
```

