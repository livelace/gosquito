### Description:

**kafka** output plugin is for sending data to Kafka topics.


### Generic parameters:

| Param   | Required |  Type   | Template | Default | Description |
|:--------|:--------:|:-------:|:--------:|:-------:|:------------|
| timeout |    -     | seconds |    +     |    3    |             |



### Plugin parameters:

| Param    | Required |  Type  | Template | Default |                Example                 | Description |
|:---------|:--------:|:------:|:--------:|:-------:|:--------------------------------------:|:------------|
| brokers  |    +     | string |    +     |   ""    | "127.0.0.1:9092,host.example.com:1111" |             |
| compress |    -     | string |    +     | "none"  |                 "zstd"                 |             |
| output   |    +     | array  |    +     |   []    |                ["news"]                |             |
| schema   |    +     |  map   |    +     |   ""    |              see example               |             |


### Config sample:

```toml

```

### Flow sample:

```yaml
```