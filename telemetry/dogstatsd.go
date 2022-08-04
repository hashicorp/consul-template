package telemetry

import (
	"github.com/armon/go-metrics"
	"github.com/armon/go-metrics/datadog"
	"github.com/hashicorp/consul-template/config"
)

/*
  methods extracted from Consul telemetry:
    https://github.com/hashicorp/consul/blob/main/lib/telemetry.go#L248
*/

func dogstatdSink(cfg *config.TelemetryConfig, hostname string) (metrics.MetricSink, error) {
	addr := cfg.DogstatsdAddr
	if addr == "" {
		return nil, nil
	}
	sink, err := datadog.NewDogStatsdSink(addr, hostname)
	if err != nil {
		return nil, err
	}
	sink.SetTags(cfg.DogstatsdTags)
	return sink, nil
}
