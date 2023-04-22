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
	flowMetricError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_error",
			Help: "How many errors raised during flow executions.",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)

	flowMetricExpire = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_expire",
			Help: "",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)

	flowMetricNoData = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_nodata",
			Help: "How many times flow received previously seen data.",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)

	flowMetricReceive = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_receive",
			Help: "How much new data flow has received.",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)

	flowMetricRun = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_run",
			Help: "How many times flow has executed.",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)

	flowMetricSend = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_send",
			Help: "How much data flow sent.",
		},
		[]string{"flow", "hash", "input_plugin", "input_values", "process_plugins", "output_plugin", "output_values"},
	)
)

func init() {
	prometheus.MustRegister(flowMetricError)
	prometheus.MustRegister(flowMetricExpire)
	prometheus.MustRegister(flowMetricNoData)
	prometheus.MustRegister(flowMetricReceive)
	prometheus.MustRegister(flowMetricRun)
	prometheus.MustRegister(flowMetricSend)

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

	// Get app config.
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

	if len(flows) == 0 {
		log.WithFields(log.Fields{
			"path": appConfig.GetString(core.VIPER_DEFAULT_FLOW_CONF),
		}).Error(core.ERROR_NO_VALID_FLOW)
		os.Exit(1)
	}

	// Main loop.
	flowLimit := appConfig.GetInt(core.VIPER_DEFAULT_FLOW_LIMIT)
	flowCounter := make(map[uuid.UUID]int64, len(flows))
	flowTimestamp := make(map[uuid.UUID]time.Time, len(flows))

	for {
		currentTime := time.Now()
		flowCandidates := make(map[*core.Flow]int64, 0)
		flowRunning := 0

		// 1. Analyze all flows:
		for _, flow := range flows {
			lastTime := flowTimestamp[flow.FlowUUID]

			// Count running flows.
			if flow.GetInstance() > 0 {
				flowCounter[flow.FlowUUID] += int64(flow.GetInstance())
				flowRunning += flow.GetInstance()
			}

			// Update metrics for non-running flows.
			if flow.GetInstance() == 0 {
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

				flowMetricError.With(labels).Add(float64(flow.MetricError))
				flowMetricExpire.With(labels).Add(float64(flow.MetricExpire))
				flowMetricNoData.With(labels).Add(float64(flow.MetricNoData))
				flowMetricReceive.With(labels).Add(float64(flow.MetricReceive))
				flowMetricRun.With(labels).Add(float64(flow.MetricRun))
				flowMetricSend.With(labels).Add(float64(flow.MetricSend))

				flow.ResetMetric()
			}

			// Find flow candidates and save their execution counters.
			if flow.GetInstance() < flow.FlowInstance {
				// Save candidates counters.
				if currentTime.Unix()-lastTime.Unix() > flow.FlowInterval/1000 {
					flowCandidates[flow] = flowCounter[flow.FlowUUID]
				}
			}
		}

		// 2. Run flow candidates.
		// a. No limits.
		// b. With limits.
		if flowLimit == 0 {
			for flow := range flowCandidates {
				flowTimestamp[flow.FlowUUID] = currentTime

				for i := flow.GetInstance(); i < flow.FlowInstance; i++ {
					go runFlow(flow)
					time.Sleep(1 * time.Millisecond)
				}
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
				if flowRunning >= flowLimit {
					break
				}

				for flowRunning < flowLimit && candidate.Flow.GetInstance() < candidate.Flow.FlowInstance {
					flowTimestamp[candidate.Flow.FlowUUID] = currentTime
					flowRunning += 1

					go runFlow(candidate.Flow)
				}
			}
		}

		time.Sleep(core.DEFAULT_LOOP_SLEEP * time.Millisecond)
	}
}
