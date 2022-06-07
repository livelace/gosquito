### Description:

**kafka** input plugin is intended for receiving data from [Kafka](https://kafka.apache.org/) topics.  

### Data structure:

There is no static structure of inbound Kafka messages. Schema may specefied by user or fetched from schema registry.
Schema's fields are just copied to Datum fields:

```go
type Data struct {
    ARRAY0  []string
    ...
    ARRAY15 []string

    ARRAYA  []string
    ...
    ARRAYP  []string
	
    TEXT0   string
    ...
    TEXT15  string

    TEXTA*  string
    ...
    TEXTZ*  string
}
```

&ast; - field may be used with **match_signature** parameter.

### Generic parameters:

| Param                 | Required |  Type  | Template |        Default        |
|:----------------------|:--------:|:------:|:--------:|:---------------------:|
| expire_action         |    -     | array  |    +     |          []           |
| expire_action_delay   |    -     | string |    +     |         "1d"          |
| expire_action_timeout |    -     |  int   |    +     |          30           |
| expire_interval       |    -     | string |    +     |         "7d"          |
| force                 |    -     |  bool  |    +     |         false         |
| force_count           |    -     |  int   |    +     |          100          |
| timeout               |    -     |  int   |    +     |          60           |
| time_format           |    -     | string |    +     | "15:04:05 02.01.2006" |
| time_zone             |    -     | string |    +     |         "UTC"         |

### Generic parameters:

| Param   | Required | Type | Template | Default |
|:--------|:--------:|:----:|:--------:|:-------:|
| timeout | -        | int  | +        | 3       |


### Plugin parameters:

| Param                   | Required | Type   | Template | Default                 | Example                      | Description                                                                                                                                                                                                         |
|:------------------------|:--------:|:------:|:--------:|:-----------------------:|:----------------------------:|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **brokers**             | +        | string | +        | ""                      | "127.0.0.1:9092,host:1111"   | List of Kafka brokers.                                                                                                                                                                                              |
| client_id               | -        | string | +        | <FLOW_NAME>             | "gosquito"                   | Client identification.                                                                                                                                                                                              |
| confluent_avro          | -        | bool   | +        | false                   | true                         | Get [Confluent Avro](https://docs.confluent.io/platform/current/schema-registry/serdes-develop/index.html#wire-format) schema from [schema registry](https://docs.confluent.io/current/schema-registry/index.html). |
| group_id                | -        | string | +        | <FLOW_NAME>             | "gosquito"                   | Group identification.                                                                                                                                                                                               |
| **input**               | +        | array  | +        | []                      | ["news"]                     | List of Kafka topics.                                                                                                                                                                                               |
| log_level               | -        | int    | +        | 0                       | 7                            | librdkafka log level.                                                                                                                                                                                               |
| match_signature         | -        | array  | +        | "[]"                    | ["data.textA", "data.textP"] | Match new messages by signature.                                                                                                                                                                                    |
| match_ttl               | -        | string | +        | "1d"                    | "24h"                        | TTL (Time To Live) for matched signatures.                                                                                                                                                                          |
| offset                  | -        | string | +        | "earliest"              | "latest"                     | Offset to start consuming from.                                                                                                                                                                                     |
| **schema**              | *        | map    | +        | map[]                   | see example                  | Dynamic schema for Kafka messages.                                                                                                                                                                                  |
| schema_record_name      | -        | string | +        | "Datum"              | "event"                      | [Avro record name](http://avro.apache.org/docs/current/spec.html).                                                                                                                                                  |
| schema_record_namespace | -        | string | +        | "ru.livelace.gosquito"  | "com.example"                | [Avro record namespace](http://avro.apache.org/docs/current/spec.html).                                                                                                                                             |
| schema_registry         | -        | string | +        | "http://127.0.0.1:8081" | "https://host.example.com"   | [Confluent schema registry](https://docs.confluent.io/current/schema-registry/index.html).                                                                                                                          |

### Flow sample:

```yaml
flow:
  name: "kafka-input-example"

  input:
    plugin: "kafka"
    params:
      template: "templates.kafka.input.default"
      input: ["news"]
      schema:
        title: "data.text0"
  
  process:
    - id: 0
      plugin: "echo"
      params:
        input: [
          "data.text0", 
          "data.text1", 
          "---"
        ]
```

### Config sample:

```toml
[templates.kafka.input.default]
brokers = "127.0.0.1:9092"

[templates.kafka.input.default.schema]
content = "data.text1"
```

