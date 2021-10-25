### Description:

**regexpmatch** process plugin is intended for matching patterns inside
data.


### Generic parameters:

| Param   | Required | Type  | Template | Default | Example |
|:--------|:--------:|:-----:|:--------:|:-------:|:-------:|
| include |    -     | bool  |    -     |  true   |  false  |
| require |    -     | array |    -     |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Template | Default |     Example      | Description                                                                                                                |
|:-----------|:--------:|:-----:|:--------:|:-------:|:----------------:|:---------------------------------------------------------------------------------------------------------------------------|
| **input**  |    +     | array |    -     |   []    | ["twitter.text"] | List of [DataItem](../../concept.md) fields with data.                        |
| match_all  |    -     | bool  |    -     |  false  |       true       | Patterns must be matched in all selected [DataItem](../../concept.md) fields. |
| match_case |    -     | array |    -     |  true   |      false       | Case sensitive/insensitive.                                                                                                |
| output     |    -     | array |    -     |   []    |  ["data.text0"]  | List of target [DataItem](../../concept.md) fields.                           |
| **regexp** |    +     | array |    +     |   []    |    ["Россия"]    | List of config templates/raw regexps for matching.                                                                         |


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
      force_count: 10

  process:
    - id: 0
      plugin: "regexpmatch"
      params:
        input:  ["twitter.text", "twitter.urls"]
        output: ["data.text0", "data.array0"]
        regexp: ["Россия", "regexps.urls"]
        match_all: true

    - id: 1
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

[regexps.urls]
regexp = [
    'http?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)',
    'https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)'
]
```

