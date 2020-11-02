### Description:

**regexpmatch** process plugin is intended for matching patterns inside
data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Default |             Example             | Description                                                                                                                |
|:-----------|:--------:|:-----:|:-------:|:-------------------------------:|:---------------------------------------------------------------------------------------------------------------------------|
| **input**  |    +     | array |   []    |        ["twitter.text"]         | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with data.                        |
| match_all  |    -     | bool  |  false  |              true               | Patterns must be matched in all selected [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields. |
| output     |    -     | array |   []    |         ["data.text0"]          | List of target [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields.                           |
| **regexp** |    +     | array |   []    | ["regexps.countries", "Россия"] | List of config templates/raw regexps for matching.                                                                         |


### Flow sample:

```yaml
flow:
  name: "regexpmatch-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru"]
      force: true
      force_count: 100

  process:
    - id: 0
      plugin: "regexpmatch"
      params:
        input:  ["twitter.text"]
        output: ["data.text0"]
        regexp: ["regexps.words", "Россия"]
        
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

[regexps.words]
regexp = ["матрёшка", "балалайка"]
```

