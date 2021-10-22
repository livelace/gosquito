package gosquito

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/livelace/gosquito/pkg/gosquito/core"
	log "github.com/livelace/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"runtime"
	"sort"
	"time"
)

var (
	flowError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_error",
			Help: "",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)

	flowExpire = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_expire",
			Help: "",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)

	flowNoData = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_nodata",
			Help: "",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)

	flowReceive = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_receive",
			Help: "",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)

	flowSend = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_send",
			Help: "",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)
)

func init() {
	prometheus.MustRegister(flowError)
	prometheus.MustRegister(flowExpire)
	prometheus.MustRegister(flowNoData)
	prometheus.MustRegister(flowReceive)
	prometheus.MustRegister(flowSend)

	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: false,
		ForceColors:            true,
		ForceQuote:             true,
		FullTimestamp:          true,
		SortingFunc:            core.SortLogFields,
		TimestampFormat:        core.DEFAULT_LOG_TIME_FORMAT,
		QuoteEmptyFields:       true,
	})
}

func RunApp() {
	// Greetings.
	log.Info(fmt.Sprintf("%s %s", core.APP_NAME, core.APP_VERSION))

	// Get user config.
	appConfig := core.GetAppConfig()

	// Set maximum number of threads.
	runtime.GOMAXPROCS(appConfig.GetInt(core.VIPER_DEFAULT_PROC_NUM))

	// Set log level.
	ll, _ := log.ParseLevel(appConfig.GetString(core.VIPER_DEFAULT_LOG_LEVEL))
	log.SetLevel(ll)

	// Prometheus' metrics.
	go func() {
		http.Handle("/", promhttp.Handler())
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(appConfig.GetString(core.VIPER_DEFAULT_EXPORTER_LISTEN), nil)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error(core.ERROR_EXPORTER_LISTEN)

			os.Exit(1)
		}
	}()

	// Get flows.
	flows := getFlow(appConfig)
	flowLimit := appConfig.GetInt(core.VIPER_DEFAULT_FLOW_LIMIT)
	flowCounter := make(map[uuid.UUID]int64, len(flows))
	flowTimestamp := make(map[uuid.UUID]time.Time, len(flows))

	if len(flows) > 0 {
		for {
			currentTime := time.Now()
			flowCandidates := make(map[*core.Flow]int64, 0)
			flowRunning := 0

			// Analyze flows:
			// 1. Count running flows.
			// 2. Find flow candidates and save their execution counters.
			// 3. Update metrics for non-running flows.
			for _, flow := range flows {
				lastTime := flowTimestamp[flow.FlowUUID]

				if flow.GetInstance() > 0 {
					flowCounter[flow.FlowUUID] += int64(flow.GetInstance())
					flowRunning += flow.GetInstance()

				} else {
					// Save candidates counters.
					if currentTime.Unix()-lastTime.Unix() > flow.FlowInterval {
						flowCandidates[flow] = flowCounter[flow.FlowUUID]
					}

					// Process/output plugins may not be set.
					processPlugins := make([]string, 0)
					if len(flow.ProcessPlugins) > 0 {
						processPlugins = flow.ProcessPluginsNames
					}

					outputPlugin := ""
					outputValues := make([]string, 0)
					if flow.OutputPlugin != nil {
						outputPlugin = flow.OutputPlugin.GetName()
						outputValues = flow.OutputPlugin.GetOutput()
					}

					// Update flow metrics.
					labels := prometheus.Labels{
						"flow":            flow.FlowName,
						"hash":            flow.FlowHash,
						"input_plugin":    flow.InputPlugin.GetName(),
						"input_values":    fmt.Sprintf("%v", flow.InputPlugin.GetInput()),
						"process_plugins": fmt.Sprintf("%v", processPlugins),
						"output_plugin":   outputPlugin,
						"output_values":   fmt.Sprintf("%v", outputValues),
					}

					flowError.With(labels).Add(float64(flow.MetricError))
					flowExpire.With(labels).Add(float64(flow.MetricExpire))
					flowNoData.With(labels).Add(float64(flow.MetricNoData))
					flowReceive.With(labels).Add(float64(flow.MetricReceive))
					flowSend.With(labels).Add(float64(flow.MetricSend))

					flow.ResetMetric()
				}
			}

			// Run flow candidates.
			// 1. No limits.
			// 2. With limits.
			if flowLimit == 0 {
				for flow := range flowCandidates {
					flowTimestamp[flow.FlowUUID] = currentTime
					go runFlow(flow)
				}

			} else {
				// Create and sort slice of candidates.
				// It needs for searching the most infrequent running flows.
				// If we don't respect counter so the frequent (1s) flows will prevent running infrequent (1m) flows.
				// Example (flow_limit = 4):
				// flow1 1s 10
				// flow2 1s 10
				// flow3 1s 10
				// flow4 1s 10
				// flow5 1m 0
				var flowCandidatesSlice []core.FlowCandidate
				for flow, counter := range flowCandidates {
					flowCandidatesSlice = append(flowCandidatesSlice, core.FlowCandidate{Flow: flow, Counter: counter})
				}

				sort.Slice(flowCandidatesSlice, func(i, j int) bool {
					return flowCandidatesSlice[i].Counter < flowCandidatesSlice[j].Counter
				})

				for _, candidate := range flowCandidatesSlice {
					if flowRunning+1 <= flowLimit {
						flowTimestamp[candidate.Flow.FlowUUID] = currentTime
						flowRunning += 1
						go runFlow(candidate.Flow)
					} else {
						break
					}
				}
			}

			time.Sleep(core.DEFAULT_LOOP_SLEEP * time.Millisecond)
		}

	} else {
		log.WithFields(log.Fields{
			"path": appConfig.GetString(core.VIPER_DEFAULT_FLOW_CONF),
		}).Error(core.ERROR_NO_VALID_FLOW)

		os.Exit(1)
	}
}
