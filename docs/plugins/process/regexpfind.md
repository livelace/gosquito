### Description:

**regexpfind** process plugin is intended for finding patterns inside
data.


### Generic parameters:

| Param   | Required | Type  | Template | Default | Example |
|:--------|:--------:|:-----:|:--------:|:-------:|:-------:|
| include |    -     | bool  |    -     |  true   |  false  |
| require |    -     | array |    -     |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Template | Default |         Example          | Description                                                                                                              |
|:-----------|:--------:|:-----:|:--------:|:-------:|:------------------------:|:-------------------------------------------------------------------------------------------------------------------------|
| find_all   |    -     | bool  |    -     |  false  |           true           | Patterns must be found in all selected [DataItem](../../concept.md) fields. |
| **input**  |    +     | array |    -     |   []    |     ["twitter.urls"]     | List of [DataItem](../../concept.md) fields with data.                      |
| match_case |    -     | array |    -     |  true   |          false           | Case sensitive/insensitive.                      |
| **output** |    +     | array |    -     |   []    |     ["data.array0"]      | List of target [DataItem](../../concept.md) fields.                         |
| **regexp** |    +     | array |    +     |   []    | ["http://go.tass.ru/.*"] | List of config templates/raw regexps for searching.                                                                      |

### Flow sample:

```yaml
flow:
  name: "regexpfind-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru"]
      force: true
      force_count: 100

  process:
    - id: 0
      plugin: "regexpfind"
      params:
        input:  ["twitter.text", "twitter.urls"]
        output: ["data.array0", "data.array1"]
        regexp: [".*Россия.*", "regexps.urls"]
        find_all: true

    - id: 1
      plugin: "echo"
      params:
        require: [0]
        input: ["data.array0", "data.array1"]
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

