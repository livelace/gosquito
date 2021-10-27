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
	"strings"
	tmpl "text/template"
)

const (
	PLUGIN_NAME = "kafka"

	DEFAULT_COMPRESSION    = "none"
	DEFAULT_CONFLUENT_AVRO = false
	DEFAULT_MESSAGE_KEY    = ""
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
	DEFAULT_SCHEMA_SUBJECT_STRATEGY = "TopicName"
	DEFAULT_TIMEOUT                 = 3
)

var (
	ERROR_SCHEMA_CREATE            = errors.New("schema create error: %s")
	ERROR_SCHEMA_ERROR             = errors.New("schema error: %s")
	ERROR_SCHEMA_NOT_SET           = errors.New("schema not set")
	ERROR_SUBJECT_STRATEGY_UNKNOWN = errors.New("schema subject strategy unknown: %s")

	SUBJECT_STRATEGIES = []string{"TOPICNAME", "RECORDNAME", "TOPICRECORDNAME"}
)

func genSchema(p *Plugin, schema *map[string]interface{}) (string, error) {
	var buffer bytes.Buffer
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
				schemaItem = "{\"name\": \"%s\", \"type\": {\"type\": \"array\", \"items\": \"string\"}}"
			default:
				schemaItem = "{\"name\": \"%s\", \"type\": \"string\"}"
			}

			fields = append(fields, fmt.Sprintf(schemaItem, field))
		} else {
			fields = append(fields, fmt.Sprintf("{\"name\": \"%s\", \"type\": \"string\"}", field))
		}
	}

	// Populate schema.
	type schemaData struct {
		Name      string
		Namespace string
		Fields    []string
	}

	data := schemaData{
		Name:      p.OptionSchemaRecordName,
		Namespace: p.OptionSchemaRecordNamespace,
		Fields:    fields,
	}

	if err := template.Execute(&buffer, data); err != nil {
		return "", err
	}

	return buffer.String(), nil
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

func upsertSchema(p *Plugin, subject string) (*srclient.Schema, error) {
	registrySchema, _ := p.OptionSchemaRegistryClient.GetLatestSchema(subject, false)

	if registrySchema == nil {
		return p.OptionSchemaRegistryClient.CreateSchema(subject, p.OptionSchema, srclient.Avro, false)

	} else {
		if registrySchema.Codec().CanonicalSchema() != p.OptionSchemaCodec.CanonicalSchema() {
			return p.OptionSchemaRegistryClient.CreateSchema(subject, p.OptionSchema, srclient.Avro, false)
		} else {
			return registrySchema, nil
		}
	}
}

type Plugin struct {
	Flow *core.Flow

	KafkaConfig *kafka.ConfigMap

	LogFields log.Fields

	PluginName string
	PluginType string

	OptionBrokers               string
	OptionClientId              string
	OptionConfluentAvro         bool
	OptionCompress              string
	OptionMessageKey            string
	OptionOutput                []string
	OptionSchema                string
	OptionSchemaCodec           *goavro.Codec
	OptionSchemaRecordName      string
	OptionSchemaRecordNamespace string
	OptionSchemaNative          map[string]interface{}
	OptionSchemaRegistry        string
	OptionSchemaRegistryClient  *srclient.SchemaRegistryClient
	OptionSchemaSubjectStrategy string
	OptionTimeout               int
}

func (p *Plugin) FlowLog(message interface{}) {
	f := make(map[string]interface{}, len(p.LogFields))

	for k, v := range p.LogFields {
		f[k] = v
	}

	_, ok := message.(error)

	if ok {
		f["error"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Warn(core.LOG_FLOW_WARN)
	} else {
		f["data"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Debug(core.LOG_FLOW_STAT)
	}
}

func (p *Plugin) GetFile() string {
	return p.Flow.FlowFile
}

func (p *Plugin) GetName() string {
	return p.PluginName
}

func (p *Plugin) GetOutput() []string {
	return p.OptionOutput
}

func (p *Plugin) GetType() string {
	return p.PluginType
}

func (p *Plugin) Send(data []*core.DataItem) error {
	// Generate and send messages for every provided topic.
	for _, topic := range p.OptionOutput {
		messages := make([]*kafka.Message, 0)

		for _, item := range data {
			// Populate schema with data.
			schema := make(map[string]interface{}, 0)

			for k, v := range p.OptionSchemaNative {
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
			avroBinary, err := p.OptionSchemaCodec.BinaryFromNative(nil, schema)
			if err != nil {
				return err
			}

			// Assemble Kafka message.
			message := kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Key:            []byte(p.OptionMessageKey),
			}

			// Create confluent avro (magic + version + message) or vanilla avro message.
			if p.OptionConfluentAvro {
				var subject string

				switch p.OptionSchemaSubjectStrategy {
				case "TOPICNAME":
					subject = topic
				case "RECORDNAME":
					subject = p.OptionSchemaRecordName
				case "TOPICRECORDNAME":
					subject = fmt.Sprintf("%s-%s", topic, p.OptionSchemaRecordName)
				}

				registrySchema, err := upsertSchema(p, subject)
				if err != nil {
					return fmt.Errorf(ERROR_SCHEMA_CREATE.Error(), err)
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

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Flow: pluginConfig.Flow,
		LogFields: log.Fields{
			"hash":   pluginConfig.Flow.FlowHash,
			"flow":   pluginConfig.Flow.FlowName,
			"file":   pluginConfig.Flow.FlowFile,
			"plugin": PLUGIN_NAME,
			"type":   pluginConfig.PluginType,
		},
		PluginName: PLUGIN_NAME,
		PluginType: pluginConfig.PluginType,
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
		"schema_subject_strategy": -1,
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// -----------------------------------------------------------------------------------------------------------------

	// brokers.
	setServer := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["brokers"] = 0
			plugin.OptionBrokers = v
		}
	}
	setServer(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.brokers", template)))
	setServer((*pluginConfig.PluginParams)["brokers"])
	core.ShowPluginParam(plugin.LogFields, "brokers", plugin.OptionBrokers)

	// compress.
	setCompress := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["compress"] = 0
			plugin.OptionCompress = v
		}
	}
	setCompress(DEFAULT_COMPRESSION)
	setCompress(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.compress", template)))
	setCompress((*pluginConfig.PluginParams)["compress"])
	core.ShowPluginParam(plugin.LogFields, "compress", plugin.OptionCompress)

	// confluent_avro.
	setConfluentAvro := func(p interface{}) {
		if v, b := core.IsBool(p); b {
			availableParams["confluent_avro"] = 0
			plugin.OptionConfluentAvro = v
		}
	}
	setConfluentAvro(DEFAULT_CONFLUENT_AVRO)
	setConfluentAvro(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.confluent_avro", template)))
	setConfluentAvro((*pluginConfig.PluginParams)["confluent_avro"])
	core.ShowPluginParam(plugin.LogFields, "confluent_avro", plugin.OptionConfluentAvro)

	// client_id.
	setClientId := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["client_id"] = 0
			plugin.OptionClientId = v
		}
	}
	setClientId(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.client_id", template)))
	setClientId((*pluginConfig.PluginParams)["client_id"])
	core.ShowPluginParam(plugin.LogFields, "client_id", plugin.OptionClientId)

	// message_key.
	setMessageKey := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["message_key"] = 0
			plugin.OptionMessageKey = v
		}
	}
	setMessageKey(DEFAULT_MESSAGE_KEY)
	setMessageKey(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.message_key", template)))
	setMessageKey((*pluginConfig.PluginParams)["message_key"])
	core.ShowPluginParam(plugin.LogFields, "message_key", plugin.OptionMessageKey)

	// output.
	setOutput := func(p interface{}) {
		if v, b := core.IsSliceOfString(p); b {
			availableParams["output"] = 0
			plugin.OptionOutput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
		}
	}
	setOutput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.output", template)))
	setOutput((*pluginConfig.PluginParams)["output"])
	core.ShowPluginParam(plugin.LogFields, "output", plugin.OptionOutput)

	// schema_record_name.
	setSchemaRecordName := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["schema_record_name"] = 0
			plugin.OptionSchemaRecordName = v
		}
	}
	setSchemaRecordName(DEFAULT_SCHEMA_RECORD_NAME)
	setSchemaRecordName(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.schema_record_name", template)))
	setSchemaRecordName((*pluginConfig.PluginParams)["schema_record_name"])
	core.ShowPluginParam(plugin.LogFields, "schema_record_name", plugin.OptionSchemaRecordName)

	// schema_record_namespace.
	setSchemaRecordNamespace := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["schema_record_namespace"] = 0
			plugin.OptionSchemaRecordNamespace = v
		}
	}
	setSchemaRecordNamespace(DEFAULT_SCHEMA_RECORD_NAMESPACE)
	setSchemaRecordNamespace(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.schema_record_namespace", template)))
	setSchemaRecordNamespace((*pluginConfig.PluginParams)["schema_record_namespace"])
	core.ShowPluginParam(plugin.LogFields, "schema_record_namespace", plugin.OptionSchemaRecordNamespace)

	// schema_registry.
	setSchemaRegistry := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["schema_registry"] = 0
			plugin.OptionSchemaRegistry = v
			plugin.OptionSchemaRegistryClient = srclient.CreateSchemaRegistryClient(v)
		}
	}
	setSchemaRegistry(DEFAULT_SCHEMA_REGISTRY)
	setSchemaRegistry(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.schema_registry", template)))
	setSchemaRegistry((*pluginConfig.PluginParams)["schema_registry"])
	core.ShowPluginParam(plugin.LogFields, "schema_registry", plugin.OptionSchemaRegistry)

	// schema_subject_strategy.
	setSchemaSubjectStrategy := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["schema_subject_strategy"] = 0
			plugin.OptionSchemaSubjectStrategy = v
		}
	}
	setSchemaSubjectStrategy(DEFAULT_SCHEMA_SUBJECT_STRATEGY)
	setSchemaSubjectStrategy(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.schema_subject_strategy", template)))
	setSchemaSubjectStrategy((*pluginConfig.PluginParams)["schema_subject_strategy"])
	core.ShowPluginParam(plugin.LogFields, "schema_subject_strategy", plugin.OptionSchemaSubjectStrategy)

	// schema.
	templateSchema, _ := core.IsMapWithStringAsKey(pluginConfig.AppConfig.GetStringMap(fmt.Sprintf("%s.schema", template)))
	configSchema, _ := core.IsMapWithStringAsKey((*pluginConfig.PluginParams)["schema"])
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
				plugin.OptionSchema = v
				plugin.OptionSchemaCodec = c
				plugin.OptionSchemaNative = mergedSchema
			} else {
				return &Plugin{}, fmt.Errorf(ERROR_SCHEMA_ERROR.Error(), err)
			}
		} else {
			return &Plugin{}, fmt.Errorf(ERROR_SCHEMA_ERROR.Error(), err)
		}
	} else {
		return &Plugin{}, ERROR_SCHEMA_NOT_SET
	}

	core.ShowPluginParam(plugin.LogFields, "schema", plugin.OptionSchemaCodec.CanonicalSchema())

	// timeout.
	setTimeout := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["timeout"] = 0
			plugin.OptionTimeout = v
		}
	}
	setTimeout(DEFAULT_TIMEOUT)
	setTimeout(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.timeout", template)))
	setTimeout((*pluginConfig.PluginParams)["timeout"])
	core.ShowPluginParam(plugin.LogFields, "timeout", plugin.OptionTimeout)

	// -----------------------------------------------------------------------------------------------------------------
	// Check required and unknown parameters.

	if err := core.CheckPluginParams(&availableParams, pluginConfig.PluginParams); err != nil {
		return &Plugin{}, err
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Additional checks.

	if !core.IsValueInSlice(strings.ToUpper(plugin.OptionSchemaSubjectStrategy), &SUBJECT_STRATEGIES) {
		return &Plugin{}, fmt.Errorf(ERROR_SUBJECT_STRATEGY_UNKNOWN.Error(), plugin.OptionSchemaSubjectStrategy)
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Kafka.

	// Set client id for identification.
	var clientId string
	if plugin.OptionClientId == "" {
		clientId = plugin.Flow.FlowName
	} else {
		clientId = plugin.OptionClientId
	}

	kafkaConfig := kafka.ConfigMap{
		"bootstrap.servers":           plugin.OptionBrokers,
		"client.id":                   clientId,
		"compression.type":            plugin.OptionCompress,
		"message.timeout.ms":          plugin.OptionTimeout * 1000,
		"metadata.request.timeout.ms": plugin.OptionTimeout * 1000,
		"socket.timeout.ms":           plugin.OptionTimeout * 1000,
	}

	plugin.KafkaConfig = &kafkaConfig

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
