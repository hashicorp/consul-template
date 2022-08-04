package telemetry

import (
	"fmt"
	"log"
	"net/http"

	"github.com/armon/go-metrics"
	"github.com/armon/go-metrics/prometheus"
	"github.com/hashicorp/consul-template/config"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

/*
  methods based on Consul telemetry:
    https://github.com/hashicorp/consul/blob/main/lib/telemetry.go#L261
*/

func PrometheusSink(cfg *config.TelemetryConfig, hostname string) (metrics.MetricSink, error) {
	if cfg.PrometheusPort == 0 {
		return nil, nil
	}

	sink, err := prometheus.NewPrometheusSinkFrom(prometheus.PrometheusOpts{
		Expiration: cfg.PrometheusRetentionTime,
	})

	if err != nil {
		return nil, err
	}

	runPrometheusMetricServer(cfg.PrometheusPort)

	return sink, nil
}

func runPrometheusMetricServer(prometheusPort int) {
	handlerOptions := promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
	}

	go func() {
		log.Println("[INFO] (prometheus) running prom server")
		handler := promhttp.HandlerFor(prom.DefaultGatherer, handlerOptions)
		http.Handle("/metrics", handler)
		err := http.ListenAndServe(fmt.Sprintf(":%d", prometheusPort), nil)
		if err != nil {
			log.Printf("[ERROR] (prometheus) error thrown by the metric server: %v", err)
		}
	}()
}
