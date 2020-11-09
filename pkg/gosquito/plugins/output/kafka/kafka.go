package kafkaOut

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/linkedin/goavro/v2"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/riferrei/srclient"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"reflect"
	tmpl "text/template"
)

const (
	DEFAULT_COMPRESSION    = "none"
	DEFAULT_CONFLUENT_AVRO = false
	DEFAULT_MESSAGE_KEY    = "none"
	DEFAULT_SCHEMA_BASE    = `
{
  "type": "record",
  "name": "{{.Name}}",
  "namespace": "{{.Namespace}}",
  "fields": [
	{{range  $i, $e := .Fields}}{{$e}}{{if last $i $.Fields | not}}, 
	{{end}}{{end}}
  ]
}
`
	DEFAULT_SCHEMA_RECORD_NAME      = "DataItem"
	DEFAULT_SCHEMA_RECORD_NAMESPACE = "ru.livelace.gosquito"
	DEFAULT_SCHEMA_REGISTRY         = "http://127.0.0.1:8081"
	DEFAULT_TIMEOUT                 = 3
)

var (
	ERROR_SCHEMA_CREATE  = errors.New("schema create error: %s")
	ERROR_SCHEMA_ERROR   = errors.New("schema error: %s")
	ERROR_SCHEMA_NOT_SET = errors.New("schema not set")
)

func genSchema(p *Plugin, schema *map[string]interface{}) (string, error) {
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

	schemaFields := core.MapKeysToStringSlice(schema)

	// Try to detect/expand data fields and/or use field as string.
	for _, field := range schemaFields {
		fieldValue := (*schema)[field]

		if fieldType, err := core.GetDataFieldType(fieldValue); err == nil {
			var schemaItem string

			switch fieldType {
			case reflect.Slice:
				schemaItem = "{\"name\": \"%s\", \"type\": \"array\", \"items\": \"string\"}"
			default:
				schemaItem = "{\"name\": \"%s\", \"type\": \"string\"}"
			}

			fields = append(fields, fmt.Sprintf(schemaItem, field))
		} else {
			fields = append(fields, fmt.Sprintf("{\"name\": \"%s\", \"type\": \"string\"}", field))
		}
	}

	// Populate schema.
	type data struct {
		Name      string
		Namespace string
		Fields    []string
	}

	d := data{
		Name:      p.SchemaRecordName,
		Namespace: p.SchemaRecordNamespace,
		Fields:    fields,
	}

	if err := template.Execute(&b, d); err != nil {
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

	Brokers               string
	ClientId              string
	ConfluentAvro         bool
	Compress              string
	MessageKey            string
	Output                []string
	Schema                string
	SchemaCodec           *goavro.Codec
	SchemaRecordName      string
	SchemaRecordNamespace string
	SchemaNative          map[string]interface{}
	SchemaRegistry        string
	Timeout               int

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

			// Convert data into Avro avroBinary.
			avroBinary, err := p.SchemaCodec.BinaryFromNative(nil, schema)
			if err != nil {
				return err
			}

			// Assemble Kafka message.
			message := kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Key:            []byte(p.MessageKey),
			}

			if p.ConfluentAvro {
				schemaRegistryClient := srclient.CreateSchemaRegistryClient(p.SchemaRegistry)
				registrySchema, _ := schemaRegistryClient.GetLatestSchema(topic, false)

				if registrySchema == nil {
					registrySchema, err = schemaRegistryClient.CreateSchema(topic, p.Schema, srclient.Avro, false)
					if err != nil {
						return fmt.Errorf(ERROR_SCHEMA_CREATE.Error(), err)
					}
				}

				registrySchemaIDBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(registrySchemaIDBytes, uint32(registrySchema.ID()))

				var resultBinary []byte
				resultBinary = append(resultBinary, byte(0))
				resultBinary = append(resultBinary, registrySchemaIDBytes...)
				resultBinary = append(resultBinary, avroBinary...)

				message.Value = resultBinary
			} else {
				message.Value = avroBinary
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

		"brokers":                 1,
		"client_id":               -1,
		"compress":                -1,
		"confluent_avro":          -1,
		"message_key":             -1,
		"output":                  1,
		"schema":                  1,
		"schema_record_name":      -1,
		"schema_record_namespace": -1,
		"schema_registry":         -1,
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
	showParam("brokers", plugin.Brokers)

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
	showParam("compress", plugin.Compress)

	// confluent_avro.
	setConfluentAvro := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["confluent_avro"] = 0
			plugin.ConfluentAvro = v
		}
	}
	setConfluentAvro(DEFAULT_CONFLUENT_AVRO)
	setConfluentAvro(pluginConfig.Config.GetString(fmt.Sprintf("%s.confluent_avro", template)))
	setConfluentAvro((*pluginConfig.Params)["confluent_avro"])
	showParam("confluent_avro", plugin.ConfluentAvro)

	// client_id.
	setClientId := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["client_id"] = 0
			plugin.ClientId = v
		}
	}
	setClientId(pluginConfig.Config.GetString(fmt.Sprintf("%s.client_id", template)))
	setClientId((*pluginConfig.Params)["client_id"])
	showParam("client_id", plugin.ClientId)

	// message_key.
	setMessageKey := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["message_key"] = 0
			plugin.MessageKey = v
		}
	}
	setMessageKey(DEFAULT_MESSAGE_KEY)
	setMessageKey(pluginConfig.Config.GetString(fmt.Sprintf("%s.message_key", template)))
	setMessageKey((*pluginConfig.Params)["message_key"])
	showParam("message_key", plugin.MessageKey)

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

	// schema_record_name.
	setSchemaRecordName := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["schema_record_name"] = 0
			plugin.SchemaRecordName = v
		}
	}
	setSchemaRecordName(DEFAULT_SCHEMA_RECORD_NAME)
	setSchemaRecordName(pluginConfig.Config.GetString(fmt.Sprintf("%s.schema_record_name", template)))
	setSchemaRecordName((*pluginConfig.Params)["schema_record_name"])
	showParam("schema_record_name", plugin.SchemaRecordName)

	// schema_record_namespace.
	setSchemaRecordNamespace := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["schema_record_namespace"] = 0
			plugin.SchemaRecordNamespace = v
		}
	}
	setSchemaRecordNamespace(DEFAULT_SCHEMA_RECORD_NAMESPACE)
	setSchemaRecordNamespace(pluginConfig.Config.GetString(fmt.Sprintf("%s.schema_record_namespace", template)))
	setSchemaRecordNamespace((*pluginConfig.Params)["schema_record_namespace"])
	showParam("schema_record_namespace", plugin.SchemaRecordNamespace)

	// schema_registry.
	setSchemaRegistry := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["schema_registry"] = 0
			plugin.SchemaRegistry = v
		}
	}
	setSchemaRegistry(DEFAULT_SCHEMA_REGISTRY)
	setSchemaRegistry(pluginConfig.Config.GetString(fmt.Sprintf("%s.schema_registry", template)))
	setSchemaRegistry((*pluginConfig.Params)["schema_registry"])
	showParam("schema_registry", plugin.SchemaRegistry)

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
		if v, err := genSchema(&plugin, &mergedSchema); err == nil {
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

	showParam("schema", plugin.SchemaCodec.CanonicalSchema())

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
