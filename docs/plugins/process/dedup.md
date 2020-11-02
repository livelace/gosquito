### Description:

**dedup** process plugin is intended for deduplication
[DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) sets.


### Generic parameters:

| Param   | Required | Type | Default | Example |
|:--------|:--------:|:----:|:-------:|:-------:|
| include |    -     | bool |  true   |  false  |


### Plugin parameters:

| Param       | Required | Type  | Default | Example | Description                                    |
|:------------|:--------:|:-----:|:-------:|:-------:|:-----------------------------------------------|
| **require** |    +     | array |   []    | [1, 2]  | List of process plugins ids for deduplication. |


### Flow sample:

```yaml
flow:
  name: "dedup-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru", "AP"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "regexpmatch"
      params:
        input: ["twitter.text"]
        regexp: ["а", "a"]

    - id: 1
      plugin: "regexpmatch"
      params:
        input: ["twitter.text"]
        regexp: ["с", "s"]

    - id: 2
      alias: "dedup tweets"
      plugin: "dedup"
      params:
        require: [0, 1]

    # Duplicates shouldn't exist.
    - id: 3
      plugin: "echo"
      params:
        require: [2]
        input: ["twitter.text"]
```

### Config sample:

```toml
[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"
```



