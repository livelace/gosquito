### Description:

**unique** process plugin is intended for remove duplicates inside data.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param      | Required | Type  | Default |             Example             | Description |
|:-----------|:--------:|:-----:|:-------:|:-------------------------------:|:------------|
| **input**  |    +     | array |   []    | ["twitter.urls", "data.array0"] | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with data.            |
| **output** |    +     | array |   []    |         ["data.array1"]         | List of target [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields.            |

### Flow sample:

```yaml
flow:
  name: "unique-sample"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru"]
      force: true
      force_count: 10

  process:
    - id: 0
      alias: "search urls in text"
      plugin: "regexpfind"
      params:
        include: false
        input:  ["twitter.text"]
        output: ["data.array0"]
        regexp: ["regexps.urls"]

    - id: 1
      alias: "search urls in urls"
      plugin: "regexpfind"
      params:
        include: false
        input:  ["twitter.urls"]
        output: ["data.array1"]
        regexp: ["regexps.urls"]
  
    - id: 2
      plugin: "unique"
      params:
        include: false
        input:  ["data.array0", "data.array1"]
        output: ["data.array2"]
        
    - id: 3
      plugin: "echo"
      params:
        require: [2]
        input: ["data.array2"]
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
