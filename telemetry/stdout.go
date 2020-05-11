package telemetry

import (
	"fmt"
	"log"

	"github.com/hashicorp/consul-template/config"
	"go.opentelemetry.io/otel/exporters/metric/stdout"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
)

// NewStdout instantiates a metric sink to stdout.
func NewStdout(c *config.StdoutConfig) (*push.Controller, error) {
	log.Printf("[DEBUG] (telemetry) configuring stdout sink")

	cfg := stdout.Config{
		PrettyPrint:    *c.PrettyPrint,
		DoNotPrintTime: *c.DoNotPrintTime,
	}
	controller, err := stdout.NewExportPipeline(cfg, *c.ReportingInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stdout exporter %v", err)
	}

	log.Printf("[DEBUG] (telemetry) stdout initialized")
	return controller, nil
}
