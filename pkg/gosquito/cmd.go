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
	"time"
)

var (
	flowError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_error",
			Help: "",
		},
		[]string{"flow", "plugin"},
	)

	flowExpire = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_expire",
			Help: "",
		},
		[]string{"flow", "plugin"},
	)

	flowNoData = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_nodata",
			Help: "",
		},
		[]string{"flow", "plugin"},
	)

	flowReceive = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_receive",
			Help: "",
		},
		[]string{"flow", "plugin"},
	)

	flowSend = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosquito_flow_send",
			Help: "",
		},
		[]string{"flow", "plugin"},
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
	config := core.GetConfig()

	// Set maximum number of threads.
	runtime.GOMAXPROCS(config.GetInt(core.VIPER_DEFAULT_PROC_MAX))

	// Set log level.
	ll, _ := log.ParseLevel(config.GetString(core.VIPER_DEFAULT_LOG_LEVEL))
	log.SetLevel(ll)

	// Metrics.
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(config.GetString(core.VIPER_DEFAULT_EXPORTER_LISTEN), nil)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error(core.ERROR_EXPORTER_LISTEN)

			os.Exit(1)
		}
	}()

	// Get user-defined flows.
	flows := getFlow(config)
	flowsStates := make(map[uuid.UUID]time.Time, len(flows))

	if len(flows) > 0 {
		for {
			currentTime := time.Now()

			for _, flow := range flows {
				lastTime := flowsStates[flow.UUID]

				if (currentTime.Unix()-lastTime.Unix()) > flow.Interval && flow.GetNumber() < flow.Number {
					flowsStates[flow.UUID] = currentTime
					go runFlow(config, flow)

				} else if flow.GetNumber() == 0 {
					labels := prometheus.Labels{
						"flow":   flow.Name,
						"plugin": flow.InputPlugin.GetName(),
					}

					flowError.With(labels).Add(float64(flow.MetricError))
					flowExpire.With(labels).Add(float64(flow.MetricExpire))
					flowNoData.With(labels).Add(float64(flow.MetricNoData))
					flowReceive.With(labels).Add(float64(flow.MetricReceive))
					flowSend.With(labels).Add(float64(flow.MetricSend))

					flow.ResetMetric()
				}
			}

			time.Sleep(core.DEFAULT_LOOP_SLEEP * time.Millisecond)
		}

	} else {
		log.WithFields(log.Fields{
			"path": config.GetString(core.VIPER_DEFAULT_FLOW_DIR),
		}).Error(core.ERROR_NO_VALID_FLOW)

		os.Exit(1)
	}
}
