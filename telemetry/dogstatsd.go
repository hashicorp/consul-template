package telemetry

import (
	"fmt"
	"log"

	"github.com/hashicorp/consul-template/config"
	"go.opentelemetry.io/contrib/exporters/metric/dogstatsd"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
)

// NewDogStatsD sets up a dogstatsd exporter.
func NewDogStatsD(c *config.DogStatsDConfig) (*push.Controller, error) {
	log.Printf("[DEBUG] (telemetry) configuring dogstatsd sink")

	if c == nil || c.Address == nil {
		return nil, fmt.Errorf("address is required")
	}

	cfg := dogstatsd.Config{URL: *c.Address}
	controller, err := dogstatsd.NewExportPipeline(cfg, *c.ReportingInterval)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] (telemetry) dogstatsd initialized, reporting to %s", *c.Address)
	return controller, nil
}
