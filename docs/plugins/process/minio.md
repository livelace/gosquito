### Description:

**minio** process plugin is intended for uploading data into S3 buckets.


### Generic parameters:

| Param   | Required | Type  | Default | Example | Description |
|:--------|:--------:|:-----:|:-------:|:-------:|:------------|
| include |    -     | bool  |  true   |  false  |             |
| require |    -     | array |   []    | [1, 2]  |             |


### Plugin parameters:

| Param      | Required |  Type  | Cred | Template | Default |       Example       | Description |
|:-----------|:--------:|:------:|:----:|:--------:|:-------:|:-------------------:|:------------|
| access_key |    +     | string |  +   |    -     |   ""    |         ""          |             |
| action     |    +     | string |  -   |    +     |   ""    |        "put"        |             |
| bucket     |    +     | string |  -   |    +     |   ""    |       "news"        |             |
| input      |    +     | array  |  -   |    +     |   []    |         []          |             |
| output     |    +     | array  |  -   |    +     |   []    |         []          |             |
| secret_key |    +     | string |  +   |    -     |   ""    |         ""          |             |
| server     |    +     | string |  +   |    -     |   ""    | "minio.example.com" |             |
| ssl        |    -     |  bool  |  -   |    +     |  true   |        false        |             |
| timeout    |    -     |  int   |  -   |    +     |   60    |         300         |             |


### Config sample:

```toml

```

### Flow sample:

```yaml
```