### Description:

**fetch** process plugin is intended for downloading files.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |
| timeout |    -     |  int  |   60    |   300   |


### Plugin parameters:

| Param      | Required | Type  | Default |      Example      | Description                        |
|:-----------|:--------:|:-----:|:-------:|:-----------------:|:-----------------------------------|
| **input**  |    +     | array |   []    | ["twitter.media"] | List of [DataItem](../../concept.md) fields with URLs. |
| **output** |    +     | array |   []    |  ["data.array0"]  | List of target [DataItem](../../concept.md) fields.    |

### Flow sample:

```yaml
flow:
  name: "fetch-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "fetch"
      params:
        input:  ["twitter.media"]
        output: ["data.array0"]

    - id: 1
      plugin: "echo"
      params:
        input:  ["data.array0"]
```

### Config sample:

```toml
[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"
```



