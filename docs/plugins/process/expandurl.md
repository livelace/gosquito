### Description:

**expandurl** process plugin is intended for expanding short URLs into
full URLs.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |
| timeout |    -     |  int  |    3    |    2    |


### Plugin parameters:

| Param      | Required |  Type  |      Default      |     Example      | Description                          |
|:-----------|:--------:|:------:|:-----------------:|:----------------:|:-------------------------------------|
| depth      |    -     |  int   |         3         |        10        | Maximum depth of HTTP redirects.     |
| **input**  |    +     | array  |        []         | ["twitter.urls"] | List of DataItem fields with URLs.   |
| **output** |    +     | array  |        []         | ["data.array0"]  | List of target DataItem fields.      |
| user_agent |    -     | string | "gosquito v1.0.0" |  "webchela 1.0"  | Custom User-Agent for HTTP requests. |

### Config sample:

```toml

```

### Flow sample:

```yaml
```

