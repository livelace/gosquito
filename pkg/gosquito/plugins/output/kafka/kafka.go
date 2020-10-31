package kafkaOut

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/linkedin/goavro/v2"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"reflect"
	tmpl "text/template"
)

const (
	DEFAULT_COMPRESSION = "none"
	DEFAULT_SCHEMA_BASE = `
{
  "type": "record",
  "name": "DataRecord",
  "fields": [
	{{range  $i, $e := .}}{{$e}}{{if last $i $ | not}}, 
	{{end}}{{end}}
  ]
}
`
	DEFAULT_TIMEOUT = 3
)

var (
	ERROR_SCHEMA_NOT_SET = errors.New("schema not set")
	ERROR_SCHEMA_ERROR   = errors.New("schema error: %s")
)

func genSchema(schema *map[string]interface{}) (string, error) {
	var b bytes.Buffer
	fields := make([]string, 0)

	// Parse base schema.
	getLast := tmpl.FuncMap{
		"last": func(x int, a interface{}) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
	}

	template, err := tmpl.New("schema").Funcs(getLast).Parse(DEFAULT_SCHEMA_BASE)
	if err != nil {
		return "", err
	}

	// Try to detect/expand data fields and/or use field as string.
	for k, v := range *schema {
		if fieldType, err := core.GetDataFieldType(v); err == nil {
			var schemaItem string

			switch fieldType {
			case reflect.Slice:
				schemaItem = "{\"name\": \"%s\", \"type\": \"array\", \"items\": \"string\"}"
			default:
				schemaItem = "{\"name\": \"%s\", \"type\": \"string\"}"
			}

			fields = append(fields, fmt.Sprintf(schemaItem, k))
		} else {
			fields = append(fields, fmt.Sprintf("{\"name\": \"%s\", \"type\": \"string\"}", k))
		}
	}

	// Populate schema.
	if err := template.Execute(&b, fields); err != nil {
		return "", err
	}

	return b.String(), nil
}

func sendData(p *Plugin, messages []*kafka.Message) error {
	producer, err := kafka.NewProducer(p.KafkaConfig)
	if err != nil {
		return err
	}

	for _, message := range messages {
		c := make(chan error)

		go func() {
			for e := range producer.Events() {
				switch ev := e.(type) {
				case *kafka.Message:
					m := ev
					if m.TopicPartition.Error != nil {
						c <- m.TopicPartition.Error
					} else {
						c <- nil
					}
					return
				}
			}
		}()

		producer.ProduceChannel() <- message

		if err := <-c; err != nil {
			return err
		}
	}

	producer.Close()

	return nil
}

type Plugin struct {
	Hash string
	Flow string

	File string
	Name string
	Type string

	Brokers      string
	ClientId     string
	Compress     string
	Output       []string
	Schema       string
	SchemaCodec  *goavro.Codec
	SchemaNative map[string]interface{}
	Timeout      int

	KafkaConfig *kafka.ConfigMap
}

func (p *Plugin) Send(data []*core.DataItem) error {
	// Generate and send messages for every provided topic.
	for _, topic := range p.Output {
		messages := make([]*kafka.Message, 0)

		for _, item := range data {
			// Populate schema with data.
			schema := make(map[string]interface{}, 0)

			for k, v := range p.SchemaNative {
				// Try to detect/expand data fields and/or use value as string.
				if rv, err := core.ReflectDataField(item, v); err == nil {
					switch rv.Kind() {
					case reflect.String:
						schema[k] = rv.Interface()
					case reflect.Slice:
						schema[k] = rv.Interface()
					default:
						// Some special fields like: UUID, Time, etc.
						schema[k] = fmt.Sprintf("%v", rv)
					}
				} else {
					schema[k] = fmt.Sprintf("%v", v)
				}
			}

			// Convert data into Avro binary.
			binary, err := p.SchemaCodec.BinaryFromNative(nil, schema)
			if err != nil {
				return err
			}

			// Assemble Kafka message.
			message := kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          binary,
			}

			messages = append(messages, &message)
		}

		// Send messages to topic.
		err := sendData(p, messages)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) GetFile() string {
	return p.File
}

func (p *Plugin) GetName() string {
	return p.Name
}

func (p *Plugin) GetOutput() []string {
	return p.Output
}

func (p *Plugin) GetType() string {
	return p.Type
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Hash: pluginConfig.Hash,
		Flow: pluginConfig.Flow,

		File: pluginConfig.File,
		Name: "kafka",
		Type: "output",
	}

	// -----------------------------------------------------------------------------------------------------------------
	// All available parameters of the plugin:
	// "-1" - not strictly required.
	// "1" - strictly required.
	// "0" - will be set if parameter is set somehow (defaults, template, config etc.).
	availableParams := map[string]int{
		"template": -1,
		"timeout":  -1,

		"brokers":  1,
		"compress": -1,
		"output":   1,
		"schema":   1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	showParam := func(p string, v interface{}) {
		log.WithFields(log.Fields{
			"hash":   plugin.Hash,
			"flow":   plugin.Flow,
			"file":   plugin.File,
			"plugin": plugin.Name,
			"type":   plugin.Type,
			"value":  fmt.Sprintf("%s: %v", p, v),
		}).Debug(core.LOG_SET_VALUE)
	}

	// -----------------------------------------------------------------------------------------------------------------

	template, _ := core.IsString((*pluginConfig.Params)["template"])

	// brokers.
	setServer := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["brokers"] = 0
			plugin.Brokers = v
		}
	}
	setServer(pluginConfig.Config.GetString(fmt.Sprintf("%s.brokers", template)))
	setServer((*pluginConfig.Params)["brokers"])

	// compress.
	setCompress := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["compress"] = 0
			plugin.Compress = v
		}
	}
	setCompress(DEFAULT_COMPRESSION)
	setCompress(pluginConfig.Config.GetString(fmt.Sprintf("%s.compress", template)))
	setCompress((*pluginConfig.Params)["compress"])

	// client_id.
	setClientId := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["client_id"] = 0
			plugin.ClientId = v
		}
	}
	setClientId(pluginConfig.Config.GetString(fmt.Sprintf("%s.client_id", template)))
	setClientId((*pluginConfig.Params)["client_id"])

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.Output = core.ExtractConfigVariableIntoArray(pluginConfig.Config, v)
		}
	}
	setOutput(pluginConfig.Config.GetStringSlice(fmt.Sprintf("%s.output", template)))
	setOutput((*pluginConfig.Params)["output"])
	showParam("output", plugin.Output)

	// schema.
	templateSchema, _ := core.IsMapWithStringAsKey(pluginConfig.Config.GetStringMap(fmt.Sprintf("%s.schema", template)))
	configSchema, _ := core.IsMapWithStringAsKey((*pluginConfig.Params)["schema"])
	mergedSchema := make(map[string]interface{}, 0)

	// config schema has higher priority over template schema.
	for k, v := range templateSchema {
		mergedSchema[k] = v
	}

	for k, v := range configSchema {
		mergedSchema[k] = v
	}

	if len(mergedSchema) > 0 {
		if v, err := genSchema(&mergedSchema); err == nil {
			if c, err := goavro.NewCodec(v); err == nil {
				availableParams["schema"] = 0
				plugin.Schema = v
				plugin.SchemaCodec = c
				plugin.SchemaNative = mergedSchema
			} else {
				return &Plugin{}, fmt.Errorf(ERROR_SCHEMA_ERROR.Error(), err)
			}
		} else {
			return &Plugin{}, fmt.Errorf(ERROR_SCHEMA_ERROR.Error(), err)
		}
	} else {
		return &Plugin{}, ERROR_SCHEMA_NOT_SET
	}

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.Timeout = v
		}
	}
	setTimeout(DEFAULT_TIMEOUT)
	setTimeout(pluginConfig.Config.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.Params)["timeout"])
	showParam("timeout", plugin.Timeout)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.Params); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Kafka.

	// Set client id for identification (really ?! where ?!).
	var clientId string
	if plugin.ClientId == "" {
		clientId = plugin.Flow
	} else {
		clientId = plugin.ClientId
	}

	kafkaConfig := kafka.ConfigMap{
		"bootstrap.servers":           plugin.Brokers,
		"client.id":                   clientId,
		"compression.type":            plugin.Compress,
		"message.timeout.ms":          plugin.Timeout * 1000,
		"metadata.request.timeout.ms": plugin.Timeout * 1000,
		"socket.timeout.ms":           plugin.Timeout * 1000,
	}

	plugin.KafkaConfig = &kafkaConfig

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
