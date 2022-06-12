package kafkaMulti

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	tmpl "text/template"
	"time"

	"github.com/google/uuid"
	"github.com/linkedin/goavro/v2"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/riferrei/srclient"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

const (
	PLUGIN_NAME = "kafka"

	DEFAULT_BATCH_SIZE     = 1000
	DEFAULT_COMPRESSION    = "none"
	DEFAULT_CONFLUENT_AVRO = true
	DEFAULT_LOG_LEVEL      = 0
	DEFAULT_MATCH_TTL      = "1d"
	DEFAULT_OFFSET         = "earliest"
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
	DEFAULT_SCHEMA_RECORD_NAMESPACE = "ru.livelace.gosquito"
	DEFAULT_SCHEMA_REGISTRY         = "http://127.0.0.1:8081"
	DEFAULT_SCHEMA_SUBJECT_STRATEGY = "TopicRecordName"
	DEFAULT_SEND_DELAY              = "1ms"
	DEFAULT_TIMEOUT                 = 3
)

var (
	ERROR_OFFSET_UNKNOWN           = errors.New("offset unknown: %s")
	ERROR_SCHEMA_CREATE            = errors.New("schema create error: %s")
	ERROR_SCHEMA_ERROR             = errors.New("schema error: %s")
	ERROR_SCHEMA_NOT_SET           = errors.New("schema not set")
	ERROR_SEND_ERROR               = errors.New("cannot send data: %v")
	ERROR_SUBJECT_STRATEGY_UNKNOWN = errors.New("schema subject strategy unknown: %s")
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

		if fieldType, err := core.GetDatumFieldType(fieldValue); err == nil {
			var schemaItem string

			switch fieldType {
			case reflect.Slice:
				schemaItem = "{\"name\": \"%s\", \"type\": {\"type\": \"array\", \"items\": \"string\"}}"
			default:
				schemaItem = "{\"name\": \"%s\", \"type\": \"string\"}"
			}

			fields = append(fields, fmt.Sprintf(schemaItem, field))

		} else {
			// Allow user to set arbitrary string data inside fields.
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
	registrySchema, _ := p.SchemaRegistryClient.GetLatestSchema(subject, false)

	if registrySchema == nil || registrySchema.Codec().CanonicalSchema() != p.SchemaCodec.CanonicalSchema() {
		return p.SchemaRegistryClient.CreateSchema(subject, p.OptionSchema, srclient.Avro, false)
	}

	return registrySchema, nil
}

type Plugin struct {
	m sync.Mutex

	Flow *core.Flow

	KafkaConfig *kafka.ConfigMap

	LogFields log.Fields

	PluginName string
	PluginType string

	SchemaCache          map[uint32]*goavro.Codec
	SchemaCodec          *goavro.Codec
	SchemaNative         map[string]interface{}
	SchemaRegistryClient *srclient.SchemaRegistryClient

	OptionBrokers               string
	OptionClientId              string
	OptionCompress              string
	OptionConfluentAvro         bool
	OptionExpireAction          []string
	OptionExpireActionDelay     int64
	OptionExpireActionTimeout   int
	OptionExpireInterval        int64
	OptionExpireLast            int64
	OptionForce                 bool
	OptionForceCount            int
	OptionGroupId               string
	OptionInput                 []string
	OptionLogLevel              int
	OptionMatchSignature        []string
	OptionMatchTTL              time.Duration
	OptionMessageKey            string
	OptionOffset                string
	OptionSendDelay             time.Duration
	OptionOutput                []string
	OptionSchema                string
	OptionSchemaRecordName      string
	OptionSchemaRecordNamespace string
	OptionSchemaRegistry        string
	OptionSchemaSubjectStrategy string
	OptionTimeFormat            string
	OptionTimeFormatA           string
	OptionTimeFormatB           string
	OptionTimeFormatC           string
	OptionTimeZone              *time.Location
	OptionTimeZoneA             *time.Location
	OptionTimeZoneB             *time.Location
	OptionTimeZoneC             *time.Location
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

func (p *Plugin) GetInput() []string {
	return p.OptionInput
}

func (p *Plugin) GetName() string {
	return p.PluginName
}

func (p *Plugin) GetOutput() []string {
	return p.OptionOutput
}

func (p *Plugin) LoadState() (map[string]time.Time, error) {
	p.m.Lock()
	defer p.m.Unlock()

	data := make(map[string]time.Time, 0)

	if err := core.PluginLoadState(p.Flow.FlowStateDir, &data); err != nil {
		return data, err
	}

	return data, nil
}

func (p *Plugin) Receive() ([]*core.Datum, error) {
	temp := make([]*core.Datum, 0)
	p.LogFields["run"] = p.Flow.GetRunID()

	// Load flow sources' states.
	flowStates, err := p.LoadState()
	if err != nil {
		return temp, err
	}
	core.LogInputPlugin(p.LogFields, "all", fmt.Sprintf("states loaded: %d", len(flowStates)))

	// Create consumer.
	consumer, err := kafka.NewConsumer(p.KafkaConfig)
	if err != nil {
		return temp, err
	}

	// Subscribe to topics.
	err = consumer.SubscribeTopics(p.OptionInput, nil)
	if err != nil {
		return temp, err
	}

	// Source stat.
	sourceFailStat := make(map[string]int32)
	sourceNewStat := make(map[string]int32)
	sourceTotalStat := make(map[string]int32)

	// Consume messages.
	for {
		// Break if there is no new data.
		message, err := consumer.ReadMessage(time.Duration(p.OptionTimeout) * time.Second)
		if message == nil && err.(kafka.Error).Code() == kafka.ErrTimedOut {
			break
		} else if err != nil {
			return temp, err
		}

		// Update source overwall (valid and invalid messages) stat.
		sourceTotalStat[*message.TopicPartition.Topic] += 1

		// Try to decode message.
		var messageData interface{}
		var schemaId uint32
		isConfluentAvro := p.OptionConfluentAvro && len(message.Value) > 5 && message.Value[0] == 0

		if isConfluentAvro {
			schemaId = binary.BigEndian.Uint32(message.Value[1:5])

			if _, ok := p.SchemaCache[schemaId]; !ok {
				if registrySchema, err := p.SchemaRegistryClient.GetSchema(int(schemaId)); err == nil {
					p.SchemaCache[schemaId] = registrySchema.Codec()
				} else {
					core.LogInputPlugin(p.LogFields, "schema", fmt.Errorf("skip message: %v", err))
					continue
				}
			}

			messageData, _, err = p.SchemaCache[schemaId].NativeFromBinary(message.Value[5:])
		} else {
			messageData, _, err = p.SchemaCodec.NativeFromBinary(message.Value)
		}

		if err != nil {
			sourceFailStat[*message.TopicPartition.Topic] += 1

			if isConfluentAvro {
				core.LogInputPlugin(p.LogFields, "decode", fmt.Errorf("schema id: %d, skip message: %v", schemaId, err))
			} else {
				core.LogInputPlugin(p.LogFields, "decode", fmt.Errorf("skip message: %v", err))
			}
			continue
		}

		// Form new dataItem.
		if messageMap, messageMapValid := core.IsMapWithStringAsKey(messageData); messageMapValid {
			var currentTime = time.Now().UTC()
			var itemNew = false
			var itemSignature string
			var itemSignatureHash string
			var u, _ = uuid.NewRandom()

			item := core.Datum{
				FLOW:        p.Flow.FlowName,
				PLUGIN:      p.PluginName,
				SOURCE:      *message.TopicPartition.Topic,
				TIME:        currentTime,
				TIMEFORMAT:  currentTime.In(p.OptionTimeZone).Format(p.OptionTimeFormat),
				TIMEFORMATA: currentTime.In(p.OptionTimeZoneA).Format(p.OptionTimeFormatA),
				TIMEFORMATB: currentTime.In(p.OptionTimeZoneB).Format(p.OptionTimeFormatB),
				TIMEFORMATC: currentTime.In(p.OptionTimeZoneC).Format(p.OptionTimeFormatC),
				UUID:        u,
			}

			// Map message data into item fields.
			for fieldName, fieldValue := range p.SchemaNative {
				ri := reflect.ValueOf(messageMap[fieldName])
				ro, _ := core.ReflectDatumField(&item, fieldValue)

				// Handle absence schema's key in message data.
				// Handle in/out type mismatch.
				// Fill with empty data.
				// BTW. Output is always right!
				if _, ok := messageMap[fieldName]; !ok || ri.Kind() != ro.Kind() {
					switch ro.Kind() {
					case reflect.String:
						ro.SetString("")
					case reflect.Slice:
						ro.Set(reflect.Append(ro, reflect.ValueOf("")))
					}
					continue
				}

				// Populate datum with field data.
				switch ri.Kind() {
				case reflect.String:
					ro.SetString(ri.String())
				case reflect.Slice:
					for i := 0; i < ri.Len(); i++ {
						ro.Set(reflect.Append(ro, reflect.ValueOf(ri.Index(i).Interface())))
					}
				}
			}

			// Process only new items. Two methods:
			// 1. Match item by user provided signature.
			// 2. Compare item timestamp with source timestamp.
			if len(p.OptionMatchSignature) > 0 {
				for _, v := range p.OptionMatchSignature {
					switch v {
					case "DATA.TEXTA":
						itemSignature += item.DATA.TEXTA
					case "DATA.TEXTB":
						itemSignature += item.DATA.TEXTB
					case "DATA.TEXTC":
						itemSignature += item.DATA.TEXTC
					case "DATA.TEXTD":
						itemSignature += item.DATA.TEXTD
					case "DATA.TEXTE":
						itemSignature += item.DATA.TEXTE
					case "DATA.TEXTF":
						itemSignature += item.DATA.TEXTF
					case "DATA.TEXTG":
						itemSignature += item.DATA.TEXTG
					case "DATA.TEXTH":
						itemSignature += item.DATA.TEXTH
					case "DATA.TEXTI":
						itemSignature += item.DATA.TEXTI
					case "DATA.TEXTJ":
						itemSignature += item.DATA.TEXTJ
					case "DATA.TEXTK":
						itemSignature += item.DATA.TEXTK
					case "DATA.TEXTL":
						itemSignature += item.DATA.TEXTL
					case "DATA.TEXTM":
						itemSignature += item.DATA.TEXTM
					case "DATA.TEXTN":
						itemSignature += item.DATA.TEXTN
					case "DATA.TEXTO":
						itemSignature += item.DATA.TEXTO
					case "DATA.TEXTP":
						itemSignature += item.DATA.TEXTP
					case "DATA.TEXTQ":
						itemSignature += item.DATA.TEXTQ
					case "DATA.TEXTR":
						itemSignature += item.DATA.TEXTR
					case "DATA.TEXTS":
						itemSignature += item.DATA.TEXTS
					case "DATA.TEXTT":
						itemSignature += item.DATA.TEXTT
					case "DATA.TEXTU":
						itemSignature += item.DATA.TEXTU
					case "DATA.TEXTV":
						itemSignature += item.DATA.TEXTV
					case "DATA.TEXTW":
						itemSignature += item.DATA.TEXTW
					case "DATA.TEXTX":
						itemSignature += item.DATA.TEXTX
					case "DATA.TEXTY":
						itemSignature += item.DATA.TEXTY
					case "DATA.TEXTZ":
						itemSignature += item.DATA.TEXTZ
					}
				}

				// set default value for signature if user provided wrong values.
				if len(itemSignature) == 0 {
					itemSignature += item.SOURCE
				}

				itemSignatureHash = core.HashString(&itemSignature)

				if _, ok := flowStates[itemSignatureHash]; !ok || p.OptionForce {
					// save item signature hash to state.
					flowStates[itemSignatureHash] = currentTime
					itemNew = true
				}
			} else {
				itemNew = true
			}

			// Update stat and append item to results.
			if itemNew {
				flowStates[item.SOURCE] = currentTime
				sourceNewStat[item.SOURCE] += 1
				temp = append(temp, &item)
			}

			// Stop processing messages if force is set.
			if p.OptionForce && len(temp) >= p.OptionForceCount {
				break
			}
		}
	}

	// Close consumer.
	err = consumer.Close()

	// Show source (topics) statistics.
	for _, source := range p.OptionInput {
		core.LogInputPlugin(p.LogFields, source, fmt.Sprintf("last update: %s, received: %d, new: %d, skipped: %d",
			flowStates[source], sourceTotalStat[source], sourceNewStat[source], sourceFailStat[source]))
	}

	// Save updated flow states.
	if err := p.SaveState(flowStates); err != nil {
		return temp, err
	}

	// Check every source for expiration.
	sourcesExpired := false

	// Check if any source is expired.
	currentTime := time.Now().UTC()

	for source, sourceTime := range flowStates {
		if (currentTime.Unix() - sourceTime.Unix()) > p.OptionExpireInterval/1000 {
			sourcesExpired = true

			// Execute command if expire delay exceeded.
			// ExpireLast keeps last execution timestamp.
			if (currentTime.Unix() - p.OptionExpireLast) > p.OptionExpireActionDelay/1000 {
				p.OptionExpireLast = currentTime.Unix()

				// Execute command with args.
				// We don't worry about command return code.
				if len(p.OptionExpireAction) > 0 {
					cmd := p.OptionExpireAction[0]
					args := []string{p.Flow.FlowName, source, fmt.Sprintf("%v", sourceTime.Unix())}
					args = append(args, p.OptionExpireAction[1:]...)

					output, err := core.ExecWithTimeout(cmd, args, p.OptionExpireActionTimeout)

					core.LogInputPlugin(p.LogFields, source, fmt.Sprintf(
						"expire_action: command: %s, arguments: %v, output: %s, error: %v",
						cmd, args, output, err))
				}
			}
		}
	}

	// Inform about expiration.
	if sourcesExpired {
		return temp, core.ERROR_FLOW_EXPIRE
	}

	return temp, err
}

func (p *Plugin) SaveState(data map[string]time.Time) error {
	p.m.Lock()
	defer p.m.Unlock()

	return core.PluginSaveState(p.Flow.FlowStateDir, &data, p.OptionMatchTTL)
}

func (p *Plugin) Send(data []*core.Datum) error {
	p.LogFields["run"] = p.Flow.GetRunID()
	sendStatus := true

	// Generate and send messages for every provided topic.
	for _, topic := range p.OptionOutput {
		messages := make([]*kafka.Message, 0)

		for _, item := range data {
			// Populate schema with data.
			schema := make(map[string]interface{}, 0)

			for k, v := range p.SchemaNative {
				// Try to detect/expand data fields and/or use value as string.
				if rv, err := core.ReflectDatumField(item, v); err == nil {
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
			avroBinary, err := p.SchemaCodec.BinaryFromNative(nil, schema)
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
					subject = fmt.Sprintf("%s.%s", 
                        p.OptionSchemaRecordNamespace, p.OptionSchemaRecordName)
				case "TOPICRECORDNAME":
					subject = fmt.Sprintf("%s-%s.%s", 
                        topic, p.OptionSchemaRecordNamespace, p.OptionSchemaRecordName)
				default:
					subject = topic
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
			sendStatus = false
			core.LogOutputPlugin(p.LogFields, "send", 
                fmt.Errorf(ERROR_SEND_ERROR.Error(), err))
		}

		time.Sleep(p.OptionSendDelay)
	}
	
    if !sendStatus {
		return core.ERROR_SEND_FAIL
	}

	return nil
}

func Init(pluginConfig *core.PluginConfig) (*Plugin, error) {
	// -----------------------------------------------------------------------------------------------------------------

	plugin := Plugin{
		Flow: pluginConfig.Flow,
		LogFields: log.Fields{
			"hash":   pluginConfig.Flow.FlowHash,
			"run":    pluginConfig.Flow.GetRunID(),
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
		"confluent_avro":          -1,
		"log_level":               -1,
		"schema":                  1,
		"schema_record_name":      -1,
		"schema_record_namespace": -1,
		"schema_registry":         -1,
	}

	switch pluginConfig.PluginType {
	case "input":
		availableParams["expire_action"] = -1
		availableParams["expire_action_timeout"] = -1
		availableParams["expire_delay"] = -1
		availableParams["expire_interval"] = -1
		availableParams["force"] = -1
		availableParams["group_id"] = -1
		availableParams["input"] = 1
		availableParams["match_signature"] = -1
		availableParams["match_ttl"] = -1
		availableParams["offset"] = -1
		availableParams["send_delay"] = -1
		availableParams["time_format"] = -1
		availableParams["time_format_a"] = -1
		availableParams["time_format_b"] = -1
		availableParams["time_format_c"] = -1
		availableParams["time_zone"] = -1
		availableParams["time_zone_a"] = -1
		availableParams["time_zone_b"] = -1
		availableParams["time_zone_c"] = -1
	case "output":
		availableParams["compress"] = -1
		availableParams["output"] = 1
		availableParams["message_key"] = -1
		availableParams["schema_subject_strategy"] = -1
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Get plugin specific settings.

	template, _ := core.IsString((*pluginConfig.PluginParams)["template"])

	// -----------------------------------------------------------------------------------------------------------------

	switch pluginConfig.PluginType {

	case "input":
		// expire_action.
		setExpireAction := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["expire_action"] = 0
				plugin.OptionExpireAction = v
			}
		}
		setExpireAction(pluginConfig.AppConfig.GetStringSlice(core.VIPER_DEFAULT_EXPIRE_ACTION))
		setExpireAction(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.expire_action", template)))
		setExpireAction((*pluginConfig.PluginParams)["expire_action"])
		core.ShowPluginParam(plugin.LogFields, "expire_action", plugin.OptionExpireAction)

		// expire_action_delay.
		setExpireActionDelay := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["expire_action_delay"] = 0
				plugin.OptionExpireActionDelay = v
			}
		}
		setExpireActionDelay(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_EXPIRE_ACTION_DELAY))
		setExpireActionDelay(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.expire_action_delay", template)))
		setExpireActionDelay((*pluginConfig.PluginParams)["expire_action_delay"])
		core.ShowPluginParam(plugin.LogFields, "expire_action_delay", plugin.OptionExpireActionDelay)

		// expire_action_timeout.
		setExpireActionTimeout := func(p interface{}) {
			if v, b := core.IsInt(p); b {
				availableParams["expire_action_timeout"] = 0
				plugin.OptionExpireActionTimeout = v
			}
		}
		setExpireActionTimeout(pluginConfig.AppConfig.GetInt(core.VIPER_DEFAULT_EXPIRE_ACTION_TIMEOUT))
		setExpireActionTimeout(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.expire_action_timeout", template)))
		setExpireActionTimeout((*pluginConfig.PluginParams)["expire_action_timeout"])
		core.ShowPluginParam(plugin.LogFields, "expire_action_timeout", plugin.OptionExpireActionTimeout)

		// expire_interval.
		setExpireInterval := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["expire_interval"] = 0
				plugin.OptionExpireInterval = v
			}
		}
		setExpireInterval(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_EXPIRE_INTERVAL))
		setExpireInterval(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.expire_interval", template)))
		setExpireInterval((*pluginConfig.PluginParams)["expire_interval"])
		core.ShowPluginParam(plugin.LogFields, "expire_interval", plugin.OptionExpireInterval)

		// force.
		setForce := func(p interface{}) {
			if v, b := core.IsBool(p); b {
				availableParams["force"] = 0
				plugin.OptionForce = v
			}
		}
		setForce(core.DEFAULT_FORCE_INPUT)
		setForce(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.force", template)))
		setForce((*pluginConfig.PluginParams)["force"])
		core.ShowPluginParam(plugin.LogFields, "force", plugin.OptionForce)

		// force_count.
		setForceCount := func(p interface{}) {
			if v, b := core.IsInt(p); b {
				availableParams["force_count"] = 0
				plugin.OptionForceCount = v
			}
		}
		setForceCount(core.DEFAULT_FORCE_COUNT)
		setForceCount(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.force_count", template)))
		setForceCount((*pluginConfig.PluginParams)["force_count"])
		core.ShowPluginParam(plugin.LogFields, "force_count", plugin.OptionForceCount)

		// group_id.
		setGroupId := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["group_id"] = 0
				plugin.OptionGroupId = v
			}
		}
		setGroupId(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.group_id", template)))
		setGroupId((*pluginConfig.PluginParams)["group_id"])
		core.ShowPluginParam(plugin.LogFields, "group_id", plugin.OptionGroupId)

		if plugin.OptionGroupId == "" {
			plugin.OptionGroupId = plugin.Flow.FlowName
		}

		// input.
		setInput := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["input"] = 0
				plugin.OptionInput = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
			}
		}
		setInput(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.input", template)))
		setInput((*pluginConfig.PluginParams)["input"])
		core.ShowPluginParam(plugin.LogFields, "input", plugin.OptionInput)

		// match_signature.
		setMatchSignature := func(p interface{}) {
			if v, b := core.IsSliceOfString(p); b {
				availableParams["match_signature"] = 0
				plugin.OptionMatchSignature = core.ExtractConfigVariableIntoArray(pluginConfig.AppConfig, v)
			}
		}
		setMatchSignature(pluginConfig.AppConfig.GetStringSlice(fmt.Sprintf("%s.match_signature", template)))
		setMatchSignature((*pluginConfig.PluginParams)["match_signature"])
		core.ShowPluginParam(plugin.LogFields, "match_signature", plugin.OptionMatchSignature)
		core.SliceStringToUpper(&plugin.OptionMatchSignature)

		// match_ttl.
		setMatchTTL := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["match_ttl"] = 0
				plugin.OptionMatchTTL = time.Duration(v) * time.Millisecond
			}
		}
		setMatchTTL(DEFAULT_MATCH_TTL)
		setMatchTTL(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.match_ttl", template)))
		setMatchTTL((*pluginConfig.PluginParams)["match_ttl"])
		core.ShowPluginParam(plugin.LogFields, "match_ttl", plugin.OptionMatchTTL)

		// offset.
		setOffset := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["offset"] = 0
				plugin.OptionOffset = v
			}
		}
		setOffset(DEFAULT_OFFSET)
		setOffset(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.offset", template)))
		setOffset((*pluginConfig.PluginParams)["offset"])
		core.ShowPluginParam(plugin.LogFields, "offset", plugin.OptionOffset)

		// send_delay.
		setSendDelay := func(p interface{}) {
			if v, b := core.IsInterval(p); b {
				availableParams["send_delay"] = 0
				plugin.OptionSendDelay = time.Duration(v) * time.Millisecond
			}
		}
		setSendDelay(DEFAULT_SEND_DELAY)
		setSendDelay(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.send_delay", template)))
		setSendDelay((*pluginConfig.PluginParams)["send_delay"])
		core.ShowPluginParam(plugin.LogFields, "send_delay", plugin.OptionSendDelay)

		// time_format.
		setTimeFormat := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["time_format"] = 0
				plugin.OptionTimeFormat = v
			}
		}
		setTimeFormat(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
		setTimeFormat(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format", template)))
		setTimeFormat((*pluginConfig.PluginParams)["time_format"])
		core.ShowPluginParam(plugin.LogFields, "time_format", plugin.OptionTimeFormat)

		// time_format_a.
		setTimeFormatA := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["time_format_a"] = 0
				plugin.OptionTimeFormatA = v
			}
		}
		setTimeFormatA(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
		setTimeFormatA(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format_a", template)))
		setTimeFormatA((*pluginConfig.PluginParams)["time_format_a"])
		core.ShowPluginParam(plugin.LogFields, "time_format_a", plugin.OptionTimeFormatA)

		// time_format_b.
		setTimeFormatB := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["time_format_b"] = 0
				plugin.OptionTimeFormatB = v
			}
		}
		setTimeFormatB(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
		setTimeFormatB(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format_b", template)))
		setTimeFormatB((*pluginConfig.PluginParams)["time_format_b"])
		core.ShowPluginParam(plugin.LogFields, "time_format_b", plugin.OptionTimeFormatB)

		// time_format_c.
		setTimeFormatC := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["time_format_c"] = 0
				plugin.OptionTimeFormatC = v
			}
		}
		setTimeFormatC(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_FORMAT))
		setTimeFormatC(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_format_c", template)))
		setTimeFormatC((*pluginConfig.PluginParams)["time_format_c"])
		core.ShowPluginParam(plugin.LogFields, "time_format_c", plugin.OptionTimeFormatC)

		// time_zone.
		setTimeZone := func(p interface{}) {
			if v, b := core.IsTimeZone(p); b {
				availableParams["time_zone"] = 0
				plugin.OptionTimeZone = v
			}
		}
		setTimeZone(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
		setTimeZone(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone", template)))
		setTimeZone((*pluginConfig.PluginParams)["time_zone"])
		core.ShowPluginParam(plugin.LogFields, "time_zone", plugin.OptionTimeZone)

		// time_zone_a.
		setTimeZoneA := func(p interface{}) {
			if v, b := core.IsTimeZone(p); b {
				availableParams["time_zone_a"] = 0
				plugin.OptionTimeZoneA = v
			}
		}
		setTimeZoneA(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
		setTimeZoneA(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone_a", template)))
		setTimeZoneA((*pluginConfig.PluginParams)["time_zone_a"])
		core.ShowPluginParam(plugin.LogFields, "time_zone_a", plugin.OptionTimeZoneA)

		// time_zone_b.
		setTimeZoneB := func(p interface{}) {
			if v, b := core.IsTimeZone(p); b {
				availableParams["time_zone_b"] = 0
				plugin.OptionTimeZoneB = v
			}
		}
		setTimeZoneB(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
		setTimeZoneB(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone_b", template)))
		setTimeZoneB((*pluginConfig.PluginParams)["time_zone_b"])
		core.ShowPluginParam(plugin.LogFields, "time_zone_b", plugin.OptionTimeZoneB)

		// time_zone_c.
		setTimeZoneC := func(p interface{}) {
			if v, b := core.IsTimeZone(p); b {
				availableParams["time_zone_c"] = 0
				plugin.OptionTimeZoneC = v
			}
		}
		setTimeZoneC(pluginConfig.AppConfig.GetString(core.VIPER_DEFAULT_TIME_ZONE))
		setTimeZoneC(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.time_zone_c", template)))
		setTimeZoneC((*pluginConfig.PluginParams)["time_zone_c"])
		core.ShowPluginParam(plugin.LogFields, "time_zone_c", plugin.OptionTimeZoneC)

	case "output":
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

		// schema_subject_strategy.
		setSchemaSubjectStrategy := func(p interface{}) {
			if v, b := core.IsString(p); b {
				availableParams["schema_subject_strategy"] = 0
				plugin.OptionSchemaSubjectStrategy = strings.ToUpper(v)
			}
		}
		setSchemaSubjectStrategy(DEFAULT_SCHEMA_SUBJECT_STRATEGY)
		setSchemaSubjectStrategy(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.schema_subject_strategy", template)))
		setSchemaSubjectStrategy((*pluginConfig.PluginParams)["schema_subject_strategy"])
		core.ShowPluginParam(plugin.LogFields, "schema_subject_strategy", plugin.OptionSchemaSubjectStrategy)
	}

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

	if plugin.OptionClientId == "" {
		plugin.OptionClientId = plugin.Flow.FlowName
	}

	// log_level.
	setLogLevel := func(p interface{}) {
		if v, b := core.IsInt(p); b {
			availableParams["log_level"] = 0
			plugin.OptionLogLevel = v
		}
	}
	setLogLevel(DEFAULT_LOG_LEVEL)
	setLogLevel(pluginConfig.AppConfig.GetInt(fmt.Sprintf("%s.log_level", template)))
	setLogLevel((*pluginConfig.PluginParams)["log_level"])
	core.ShowPluginParam(plugin.LogFields, "log_level", plugin.OptionLogLevel)

	// schema_record_name.
	setSchemaRecordName := func(p interface{}) {
		if v, b := core.IsString(p); b {
			availableParams["schema_record_name"] = 0
			plugin.OptionSchemaRecordName = v
		}
	}
	setSchemaRecordName(strings.ReplaceAll(plugin.Flow.FlowName, "-", "_"))
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
			plugin.SchemaRegistryClient = srclient.CreateSchemaRegistryClient(v)
		}
	}
	setSchemaRegistry(DEFAULT_SCHEMA_REGISTRY)
	setSchemaRegistry(pluginConfig.AppConfig.GetString(fmt.Sprintf("%s.schema_registry", template)))
	setSchemaRegistry((*pluginConfig.PluginParams)["schema_registry"])
	core.ShowPluginParam(plugin.LogFields, "schema_registry", plugin.OptionSchemaRegistry)

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
				plugin.SchemaCodec = c
				plugin.SchemaNative = mergedSchema
			} else {
				return &Plugin{}, fmt.Errorf(ERROR_SCHEMA_ERROR.Error(), err)
			}
		} else {
			return &Plugin{}, fmt.Errorf(ERROR_SCHEMA_ERROR.Error(), err)
		}

		core.ShowPluginParam(plugin.LogFields, "schema", plugin.SchemaCodec.CanonicalSchema())
	}

	// Init schema cache map.
	plugin.SchemaCache = make(map[uint32]*goavro.Codec)

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

	if plugin.PluginType == "input" {
		if plugin.OptionOffset != "earliest" && plugin.OptionOffset != "latest" && plugin.OptionOffset != "none" {
			return &Plugin{}, fmt.Errorf(ERROR_OFFSET_UNKNOWN.Error(), plugin.OptionOffset)
		}
	}

	// -----------------------------------------------------------------------------------------------------------------
	// Kafka.

	kafkaConfig := kafka.ConfigMap{
		"bootstrap.servers": plugin.OptionBrokers,
		"client.id":         plugin.OptionClientId,
		"socket.timeout.ms": plugin.OptionTimeout * 1000,
		"log_level":         plugin.OptionLogLevel,
	}

	switch plugin.PluginType {
	case "input":
		kafkaConfig["auto.offset.reset"] = plugin.OptionOffset
		kafkaConfig["group.id"] = plugin.OptionGroupId
		break
	case "output":
		kafkaConfig["compression.type"] = plugin.OptionCompress
		kafkaConfig["message.timeout.ms"] = plugin.OptionTimeout * 1000
		break
	}

	plugin.KafkaConfig = &kafkaConfig

	// -----------------------------------------------------------------------------------------------------------------

	return &plugin, nil
}
