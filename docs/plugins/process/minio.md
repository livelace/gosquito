### Description:

**minio** process plugin is intended for uploading data into S3 buckets.


### Generic parameters:

| Param   | Required | Type  | Default | Example |
|:--------|:--------:|:-----:|:-------:|:-------:|
| include |    -     | bool  |  true   |  false  |
| require |    -     | array |   []    | [1, 2]  |
| timeout |    -     |  int  |   60    |   300   |

### Plugin parameters:

| Param          | Required |  Type  | Cred | Template | Default |       Example       | Description                                                                   |
|:---------------|:--------:|:------:|:----:|:--------:|:-------:|:-------------------:|:------------------------------------------------------------------------------|
| **access_key** |    +     | string |  +   |    -     |   ""    |         ""          | [Minio Admin Guide](https://docs.min.io/docs/minio-admin-complete-guide.html) |
| **action**     |    +     | string |  -   |    +     |   ""    |        "put"        | Perform action ("put" - implemented, "get" - TODO).                           |
| **bucket**     |    +     | string |  -   |    +     |   ""    |       "news"        | Bucket name.                                                                  |
| **input**      |    +     | array  |  -   |    +     |   []    |   ["data.array0"]   | List of [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields with files paths.                                     |
| **output**     |    +     | array  |  -   |    +     |   []    |         []          | List of target [DataItem](https://github.com/livelace/gosquito/blob/master/docs/data.md) fields.                                               |
| **secret_key** |    +     | string |  +   |    -     |   ""    |         ""          | [Minio Admin Guide](https://docs.min.io/docs/minio-admin-complete-guide.html) |
| **server**     |    +     | string |  +   |    -     |   ""    | "host.example.com" | Minio server.                                                                 |
| ssl            |    -     |  bool  |  -   |    +     |  true   |        false        | Use SSL for connection.                                                       |

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
      alias: "fetch media"
      plugin: "fetch"
      params:
        input:  ["twitter.media"]
        output: ["data.array0"]

    - id: 1
      alias: "save media"
      plugin: "minio"
      params:
        cred:   "creds.minio.default"
        template: "templates.minio.default"
        input:   ["data.array0"]
        output:  ["data.array1"]
        
    - id: 2
      plugin: "echo"
      params:
        input:  ["data.array1"]
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

