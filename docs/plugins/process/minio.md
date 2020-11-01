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
| **input**      |    +     | array  |  -   |    +     |   []    |   ["data.array0"]   | List of DataItem fields with files paths.                                     |
| **output**     |    +     | array  |  -   |    +     |   []    |         []          | List of target DataItem fields.                                               |
| **secret_key** |    +     | string |  +   |    -     |   ""    |         ""          | [Minio Admin Guide](https://docs.min.io/docs/minio-admin-complete-guide.html) |
| **server**     |    +     | string |  +   |    -     |   ""    | "minio.example.com" | Minio server.                                                                 |
| ssl            |    -     |  bool  |  -   |    +     |  true   |        false        | Use SSL for connection.                                                       |


### Config sample:

```toml

```

### Flow sample:

```yaml
```