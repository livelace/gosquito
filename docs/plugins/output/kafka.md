### Description:

**kafka** output plugin is intended for sending data to Kafka topics.


### Generic parameters:

| Param   | Required | Type | Template | Default |
|:--------|:--------:|:----:|:--------:|:-------:|
| timeout |    -     | int  |    +     |    3    |



### Plugin parameters:

| Param       | Required |  Type  | Template | Default |                Example                 | Description                        |
|:------------|:--------:|:------:|:--------:|:-------:|:--------------------------------------:|:-----------------------------------|
| **brokers** |    +     | string |    +     |   ""    | "127.0.0.1:9092,host.example.com:1111" | List of Kafka brokers.             |
| compress    |    -     | string |    +     | "none"  |                 "zstd"                 | Compression algorithm.             |
| **output**  |    +     | array  |    +     |   []    |                ["news"]                | List of Kafka topics.              |
| **schema**  |    +     |  map   |    +     |  map[]  |              see example               | Dynamic schema for Kafka messages. |


### Config sample:

```toml

```

### Flow sample:

```yaml
```