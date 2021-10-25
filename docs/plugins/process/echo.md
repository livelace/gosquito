### Description:

**echo** process plugin is intended for echoing processing data into
console.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| require |    -     | array |   []    | [1, 2]  |


### Plugin parameters:

| Param     | Required | Type  | Default |             Example             | Description                          |
|:----------|:--------:|:-----:|:-------:|:-------------------------------:|:-------------------------------------|
| **input** |    +     | array |   []    | ["twitter.urls", "data.array0"] | List of [DataItem](../../concept.md) fields for echoing. |

### Flow sample:

```yaml
flow:
  name: "echo-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "echo"
      params:
        input: ["twitter.urls"]

```

### Config sample:

```toml
[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"
```



