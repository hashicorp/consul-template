package telemetry

import (
	"fmt"
	"log"

	"github.com/hashicorp/consul-template/config"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
)

// MeterName the name of the global meter
const MeterName = "consul.template"

// Telemetry manages the telemetry sinks and abstracts the caller from the
// which provider is configured.
type Telemetry struct {
	controller *push.Controller

	Metric metric.Provider
	Meter  metric.Meter
}

// GlobalMeter is a wrapper to fetch the global meter
func GlobalMeter() metric.Meter {
	return global.Meter(MeterName)
}

// Init initializes metrics reporting. If no sink is configured, the no-op
// provider is used.
func Init(c *config.TelemetryConfig) (*Telemetry, error) {
	var ctrl *push.Controller
	var provider metric.Provider
	var err error

	// If multiple providers are configured, the last provider listed below
	// with be used. We're not requiring only one provider to be configured
	// just yet to allow flexibility later when tracing may be supported.
	switch {
	case c.Stdout != nil:
		ctrl, err = NewStdout(c.Stdout)

	case c.DogStatsD != nil:
		ctrl, err = NewDogStatsD(c.DogStatsD)

	case c.Prometheus != nil:
		ctrl, err = NewPrometheus(c.Prometheus)

	default:
		log.Printf("[DEBUG] (telemetry) no metric sink configured, using no-op provider")
		provider = &metric.NoopProvider{}
	}
	if err != nil {
		return nil, fmt.Errorf("unable to initialize telemetry: %s", err)
	}

	if ctrl != nil {
		provider = metric.Provider(ctrl)
	}
	global.SetMeterProvider(provider)

	return &Telemetry{
		controller: ctrl,
		Metric:     provider,
		Meter:      global.Meter(MeterName),
	}, nil
}

// Stop propagates stop to the controller and waits for the background
// go routine and exports metrics one last time before returning.
func (t *Telemetry) Stop() {
	if t.controller != nil {
		t.controller.Stop()
	}
}

// NewLabel is a helper function to create a OpenTelemetry KeyValue of strings
func NewLabel(k, v string) core.KeyValue {
	return core.KeyValue{
		Key:   core.Key(k),
		Value: core.String(v),
	}
}
