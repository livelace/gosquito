### Description:

**twitter** input plugin is intended for data gathering from [Twitter](https://twitter.com/) channels.

### Data structure:

```go
type Twitter struct {
	LANG  string
	MEDIA []string
	TAGS  []string
	TEXT* string
	URLS  []string
}
```

&ast; - field may be used with **match_signature** parameter.

### Generic parameters:

| Param                 | Required | Type   | Template | Default               |
|:----------------------|:--------:|:------:|:--------:|:---------------------:|
| expire_action         | -        | array  | +        | []                    |
| expire_action_delay   | -        | string | +        | "1d"                  |
| expire_action_timeout | -        | int    | +        | 30                    |
| expire_interval       | -        | string | +        | "7d"                  |
| force                 | -        | bool   | +        | false                 |
| force_count           | -        | int    | +        | 100                   |
| time_format           | -        | string | +        | "15:04:05 02.01.2006" |
| time_format_a         | -        | string | +        | "15:04:05 02.01.2006" |
| time_format_b         | -        | string | +        | "15:04:05 02.01.2006" |
| time_format_c         | -        | string | +        | "15:04:05 02.01.2006" |
| time_zone             | -        | string | +        | "UTC"                 |
| time_zone_a           | -        | string | +        | "UTC"                 |
| time_zone_b           | -        | string | +        | "UTC"                 |
| time_zone_c           | -        | string | +        | "UTC"                 |
| timeout               | -        | int    | +        | 60                    |


### Plugin parameters:

| Param               | Required | Type   | Cred | Template | Default           | Example                          | Description                                                             |
|:--------------------|:--------:|:------:|:----:|:--------:|:-----------------:|:--------------------------------:|:------------------------------------------------------------------------|
| **access_secret**   | +        | string | +    | -        | ""                | ""                               | [Twitter API Access](https://developer.twitter.com/en/apply-for-access) |
| **access_token**    | +        | string | +    | -        | ""                | ""                               | [Twitter API Access](https://developer.twitter.com/en/apply-for-access) |
| **consumer_key**    | +        | string | +    | -        | ""                | ""                               | [Twitter API Access](https://developer.twitter.com/en/apply-for-access) |
| **consumer_secret** | +        | string | +    | -        | ""                | ""                               | [Twitter API Access](https://developer.twitter.com/en/apply-for-access) |
| **input**           | +        | array  | -    | +        | []                | ["tass_agency"]                  | List of Twitter channels.                                               |
| match_signature     | -        | array  | -    | +        | "[]"              | ["twitter.lang", "twitter.text"] | Match new tweets by signature.                                          |
| match_ttl           | -        | string | -    | +        | "1d"              | "24h"                            | TTL (Time To Live) for matched signatures.                              |
| user_agent          | -        | string | -    | +        | "gosquito v4.1.0" | "webchela 1.0"                   | Custom User-Agent for API access.                                       |


### Flow sample:

```yaml
flow:
  name: "twitter-example"

  input:
    plugin: "twitter"
    params:
      cred: "creds.twitter.default"
      input: ["rianru"]
      force: true
      force_count: 10

  process:
    - id: 0
      plugin: "regexpreplace"
      alias: "clean text"
      params:
        input:   ["twitter.text"]
        output:  ["data.text0"]
        regexp:  ["regexps.urls"]
        replace: [""]

    - id: 1
      plugin: "echo"
      alias: "show original and cleaned text"
      params:
        input: ["twitter.text", "data.text0"]

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
    'https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)'
]
```


