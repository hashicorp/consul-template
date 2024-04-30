package telemetry

import (
	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul-template/config"
)

/*
  methods extracted from Consul telemetry:
    https://github.com/hashicorp/consul/blob/main/lib/telemetry.go#L232
*/

func statsiteSink(cfg *config.TelemetryConfig, hostname string) (metrics.MetricSink, error) {
	addr := cfg.StatsiteAddr
	if addr == "" {
		return nil, nil
	}
	return metrics.NewStatsiteSink(addr)
}
