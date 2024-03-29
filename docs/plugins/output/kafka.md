### Description:

**kafka** output plugin is intended for sending data to [Kafka](https://kafka.apache.org/) topics.  

Kafka messages are generated in [Avro](https://en.wikipedia.org/wiki/Apache_Avro) format with an arbitrary schema (flat, no nested objects).


### Generic parameters:

| Param   | Required | Type | Template | Default |
|:--------|:--------:|:----:|:--------:|:-------:|
| timeout |    -     | int  |    +     |    3    |


### Plugin parameters:

| Param                   | Required | Type   | Template | Default                 | Example                                            | Description                                                                                                                                    |
|:------------------------|:--------:|:------:|:--------:|:-----------------------:|:--------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------|
| **brokers**             | +        | string | +        | ""                      | "127.0.0.1:9092,host:1111"                         | List of Kafka brokers.                                                                                                                         |
| client_id               | -        | string | +        | <FLOW_NAME>             | "gosquito"                                         | Client identification.                                                                                                                         |
| compress                | -        | string | +        | "none"                  | "zstd"                                             | Compression algorithm.                                                                                                                         |
| confluent_avro          | -        | bool   | +        | true                    | false                                              | Send [Confluent Avro (magic byte + schema)](https://docs.confluent.io/platform/current/schema-registry/serdes-develop/index.html#wire-format). |
| log_level               | -        | int    | +        | 0                       | 7                                                  | librdkafka log level.                                                                                                                          |
| message_key             | -        | string | +        | "none"                  | "partkey1"                                         | Message partition key.                                                                                                                         |
| **output**              | +        | array  | +        | []                      | ["news"]                                           | List of Kafka topics.                                                                                                                          |
| **schema**              | +        | map    | +        | map[]                   | see example                                        | Dynamic schema for Kafka messages.                                                                                                             |
| schema_record_name      | -        | string | +        | <FLOW_NAME>             | "event"                                            | [Avro record name](http://avro.apache.org/docs/current/spec.html).                                                                             |
| schema_record_namespace | -        | string | +        | "ru.livelace.gosquito"  | "com.example"                                      | [Avro record namespace](http://avro.apache.org/docs/current/spec.html).                                                                        |
| schema_registry         | -        | string | +        | "http://127.0.0.1:8081" | "https://host.example.com"                         | [Confluent schema registry](https://docs.confluent.io/current/schema-registry/index.html).                                                     |
| schema_subject_strategy | -        | string | +        | "TopicRecordName"       | "TopicName",<br>"RecordName",<br>"TopicRecordName" | [Subject name strategy](https://docs.confluent.io/current/schema-registry/serdes-develop/index.html#subject-name-strategy).                    |
| send_delay              | -        | string | +        | "1ms"                   | "1s"                                               | Delay between sending.                                                                                                                         |


### Flow sample:

```yaml
flow:
  name: "kafka-output-example"

  input:
    plugin: "rss"
    params:
      input: ["https://tass.ru/rss/v2.xml"]
      force: true
      force_count: 10

  output:
    plugin: "kafka"
    params:
      template: "templates.kafka.output.default"
      output: ["test"]
      
      # These fields have higher priority over template fields.
      # Fields will be merged and sorted alphabetically. 
      schema:
        content: "rss.content"
        title: "rss.title"
        foo: "bar"
```

### Config sample:

```toml
[templates.kafka.output.default]
brokers = "127.0.0.1:9092"

[templates.kafka.output.default.schema]
flow     = "flow"
plugin   = "plugin"
source   = "source"
time     = "time"
uuid     = "uuid"
```

