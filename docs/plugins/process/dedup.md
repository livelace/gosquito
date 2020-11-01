### Description:

**dedup** process plugin is intended for deduplication
[[DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md)](https://github.com/livelace/gosquito/blob/master/docs/data.md) sets.


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
      input: ["rianru"]
      force: true
      force_count: 10

  process:
    - id: 0
      alias: "match russia"
      plugin: "regexpmatch"
      params:
        include: false
        input: ["twitter.text"]
        regexp: ["Россия", "Russia"]

    - id: 1
      alias: "match usa"
      plugin: "regexpmatch"
      params:
        include: false
        input: ["twitter.text"]
        regexp: ["США", "US"]

    - id: 2
      alias: "dedup tweets"
      plugin: "dedup"
      params:
        require: [0, 1]
```

### Config sample:

```toml
[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"
```



