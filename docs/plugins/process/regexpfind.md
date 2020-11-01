### Description:

**regexpfind** process plugin is intended for finding patterns inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Default |                  Example                  | Description                                         |
|:-----------|:--------:|:-----:|:-------:|:-----------------------------------------:|:----------------------------------------------------|
| **input**  |    +     | array |   []    |             ["twitter.urls"]              | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with data.                  |
| **output** |    +     | array |   []    |              ["data.array0"]              | List of target [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields.                     |
| **regexp** |    +     | array |   []    | ["regexps.video", "http://go.tass.ru/.*"] | List of config templates/raw regexps for searching. |

### Flow sample:

```yaml
flow:
  name: "regexpfind-sample"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "regexpfind"
      params:
        input:  ["twitter.text"]
        output: ["data.array0"]
        regexp: ["regexps.urls"]
        
    - id: 1
      plugin: "echo"
      params:
        require: [0]
        input: ["data.array0"]
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

