package telemetry

import (
	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul-template/config"
)

/*
  methods extracted from Consul telemetry:
    https://github.com/hashicorp/consul/blob/main/lib/telemetry.go#L240
*/

func statsdSink(cfg *config.TelemetryConfig, hostname string) (metrics.MetricSink, error) {
	addr := cfg.StatsdAddr
	if addr == "" {
		return nil, nil
	}
	return metrics.NewStatsdSink(addr)
}
