### Description:

**minio** process plugin is intended for uploading data into S3 buckets.


### Generic parameters:

| Param   | Required | Type  | Template | Default | Example |
|:--------|:--------:|:-----:|:--------:|:-------:|:-------:|
| include | -        | bool  | +        | false   | true    |
| require | -        | array | +        | []      | [1, 2]  |
| timeout | -        | int   | +        | 60      | 300     |

### Plugin parameters:

| Param          | Required | Type   | Cred | Template | Default | Example            | Description                                                                   |
|:---------------|:--------:|:------:|:----:|:--------:|:-------:|:------------------:|:------------------------------------------------------------------------------|
| **access_key** | +        | string | +    | -        | ""      | ""                 | [Minio Admin Guide](https://docs.min.io/docs/minio-admin-complete-guide.html) |
| **action**     | +        | string | -    | +        | ""      | "put"              | Available actions: get, put.                                                  |
| **bucket**     | +        | string | -    | +        | ""      | "news"             | Bucket name.                                                                  |
| **input**      | +        | array  | -    | +        | []      | ["data.array0"]    | List of [Datum](../../concept.md) fields with files paths.                    |
| **output**     | +        | array  | -    | +        | []      | ["data.array1"]    | List of target [Datum](../../concept.md) fields.                              |
| **secret_key** | +        | string | +    | -        | ""      | ""                 | [Minio Admin Guide](https://docs.min.io/docs/minio-admin-complete-guide.html) |
| **server**     | +        | string | +    | -        | ""      | "host.example.com" | Minio server.                                                                 |
| source_delete  | -        | bool   | -    | +        | false   | true               | Delete source file after get/put.                                             |
| ssl            | -        | bool   | -    | +        | true    | false              | Use SSL for connection.                                                       |

### Flow sample:

```yaml
flow:
  name: "minio-example"

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
      alias: "fetch media"
      params:
        input:  ["twitter.media"]
        output: ["data.array0"]

    - id: 1
      plugin: "regexpfind"
      params:
        input:  ["data.array0"]
        output: ["data.array1"]
        regexp: ["process/fetch/([a-z0-9\\-]+)/([a-zA-Z0-9.?=_\\-]+)"]
        group:  [[1, 2]]
        group_join: ["/"]

    - id: 2
      plugin: "minio"
      alias: "save media"
      params:
        cred:   "creds.minio.default"
        template: "templates.minio.default"
        input:   ["data.array0"]
        output:  ["data.array1"]
        
    - id: 3
      plugin: "echo"
      alias: "echo local and remote files"
      params:
        input:  ["data.array0", "data.array1"]
```

### Config sample:

```toml
[creds.twitter.default]
access_token = "<ACCESS_TOKEN>"
access_secret = "<ACCESS_SECRET>"
consumer_key = "<CONSUMER_KEY>"
consumer_secret = "<CONSUMER_SECRET>"

[creds.minio.default]
server = "<SERVER>"
access_key = "<ACCESS_KEY>"
secret_key = "<SECRET_KEY>"

[templates.minio.default]
action = "put"
bucket = "<BUCKET>"
```

