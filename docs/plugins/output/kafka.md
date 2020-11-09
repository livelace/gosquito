### Description:

**kafka** output plugin is intended for sending data to Kafka topics.
Kafka messages are generated in
[Avro](https://en.wikipedia.org/wiki/Apache_Avro) format with an
arbitrary schema (flat, no nested objects).


### Generic parameters:

| Param   | Required | Type | Template | Default |
|:--------|:--------:|:----:|:--------:|:-------:|
| timeout |    -     | int  |    +     |    3    |


### Plugin parameters:

| Param                   | Required |  Type  | Template |         Default         |                Example                 | Description                        |
|:------------------------|:--------:|:------:|:--------:|:-----------------------:|:--------------------------------------:|:-----------------------------------|
| **brokers**             |    +     | string |    +     |           ""            | "127.0.0.1:9092,host.example.com:1111" | List of Kafka brokers.             |
| client_id               |    -     | string |    +     |       <FLOW_NAME>       |               "gosquito"               | Client identification.             |
| compress                |    -     | string |    +     |         "none"          |                 "zstd"                 | Compression algorithm.             |
| confluent_avro          |    -     |  bool  |    +     |          false          |                  true                  | Send Confluent Avro.               |
| message_key             |    -     | string |    +     |         "none"          |               "partkey1"               | Message partition key.             |
| **output**              |    +     | array  |    +     |           []            |                ["news"]                | List of Kafka topics.              |
| **schema**              |    +     |  map   |    +     |          map[]          |              see example               | Dynamic schema for Kafka messages. |
| schema_record_name      |    -     | string |    +     |       "DataItem"        |                "event"                 | Avro record name.                  |
| schema_record_namespace |    -     | string |    +     | "ru.livelace.gosquito"  |             "com.example"              | Avro record namespace.             |
| schema_registry         |    -     | string |    +     | "http://127.0.0.1:8081" |     "https://schemas.example.com"      | Confluent schema registry.         |


### Flow sample:

```yaml
flow:
  name: "kafka-example"

  input:
    plugin: "rss"
    params:
      input: ["http://tass.ru/rss/v2.xml"]
      force: true
      force_count: 10

  output:
    plugin: "kafka"
    params:
      template: "templates.rss.kafka.default"
      output: ["kafka-example"]
      
      # These fields have higher priority and will be merged with 
      # template fields and sorted alphabetically. 
      schema:
        content: "rss.content"
        title: "rss.title"
        foo: "bar"
```

### Config sample:

```toml
[templates.rss.kafka.default]
brokers = "127.0.0.1:9092"

[templates.rss.kafka.default.schema]
flow     = "flow"
plugin   = "plugin"
source   = "source"
time     = "time"
uuid     = "uuid"
```

