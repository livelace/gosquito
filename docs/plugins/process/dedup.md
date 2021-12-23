### Description:

**dedup** process plugin is intended for deduplication [DataItem](../../concept.md) sets.  

Deduplication is performed over [DataItem](../../concept.md) UUID.


### Generic parameters:

| Param   | Required | Type | Default | Example |
|:--------|:--------:|:----:|:-------:|:-------:|
| include |    -     | bool |  true   |  false  |


### Plugin parameters:

| Param       | Required | Type  | Default | Example | Description                                    |
|:------------|:--------:|:-----:|:-------:|:-------:|:-----------------------------------------------|
| **require** |    +     | array |   []    | [1, 2]  | List of process plugins IDs for deduplication. |


### Flow sample:

```yaml
flow:
  name: "dedup-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["AP"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "regexpmatch"
      alias: "match tweets with a"
      params:
        input: ["twitter.text"]
        regexp: ["a"]

    - id: 1
      plugin: "regexpmatch"
      alias: "match tweets with c"
      params:
        input: ["twitter.text"]
        regexp: ["c"]

    - id: 2
      plugin: "dedup"
      alias: "dedup tweets"
      params:
        require: [0, 1]

    - id: 3
      plugin: "echo"
      alias: "show deduplicated tweets"
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



