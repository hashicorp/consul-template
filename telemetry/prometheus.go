package telemetry

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/consul-template/config"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
)

// NewPrometheus instantiates a metric sink for Prometheus with an HTTP server
// serving a `/metrics` endpoint for scraping.
func NewPrometheus(c *config.PrometheusConfig) (*push.Controller, error) {
	log.Printf("[DEBUG] (telemetry) configuring Prometheus sink")

	controller, hf, err := prometheus.NewExportPipeline(prometheus.Config{},
		*c.ReportingInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prometheus exporter %v", err)
	}
	global.SetMeterProvider(controller)
	http.HandleFunc("/metrics", hf)

	addr := fmt.Sprintf(":%d", *c.Port)
	go func(addr string) {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("failed to run Prometheus /metrics endpoint: %v", err)
		}
	}(addr)

	return controller, nil
}
