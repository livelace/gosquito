### Description:

**regexpreplace** process plugin is intended for replacing patterns
inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param           | Required | Type  | Default |        Example         | Description                                         |
|:----------------|:--------:|:-----:|:-------:|:----------------------:|:----------------------------------------------------|
| **input**       |    +     | array |   []    |    ["twitter.text"]    | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with data.                  |
| **output**      |    +     | array |   []    |     ["data.text0"]     | List of target [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields.                     |
| **regexp**      |    +     | array |   []    | ["regexps.bad", "war"] | List of config templates/raw regexps for replacing. |
| **replacement** |    +     | array |   []    | ["vanished", "peace"]  | List of replacements.                               |

### Flow sample:

```yaml
flow:
  name: "regexpreplace-sample"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["AP", "BBCNews", "BBCWorld", "business", "independent"]
      force: true
      force_count: 10

  process:
    - id: 0
      alias: "swap words"
      plugin: "regexpreplace"
      params:
        input:  ["twitter.text"]
        output: ["data.text0"]
        regexp: ["regexps.words.bad", "regexps.words.good"]
        replacement: ["GOOD", "BAD"]
        
    - id: 1
      plugin: "echo"
      params:
        require: [0]
        input: ["data.text0"]
```

### Config sample:

```toml
[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"

[regexps.words.bad]
regexp = ["war", "violence"]

[regexps.words.good]
regexp = ["peace", "humanism"]
```

