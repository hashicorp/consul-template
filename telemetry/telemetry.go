package telemetry

/*
  methods based on Consul telemetry:
    https://github.com/hashicorp/consul/blob/main/lib/telemetry.go
*/

import (
	"context"
	"net/http"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/go-multierror"

	"github.com/hashicorp/consul-template/config"
)

// MetricsHandler provides an http.Handler for displaying metrics.
type MetricsHandler interface {
	DisplayMetrics(resp http.ResponseWriter, req *http.Request) (interface{}, error)
	Stream(ctx context.Context, encoder metrics.Encoder)
}

type Telemetry struct {
	Handler  MetricsHandler
	cancelFn context.CancelFunc
}

func (tel *Telemetry) Stop() {
	if tel.cancelFn != nil {
		tel.cancelFn()
	}
}

func computeMetricsConfig(telemetryConf *config.TelemetryConfig) *metrics.Config {
	metricsConf := metrics.DefaultConfig(telemetryConf.MetricsPrefix)
	metricsConf.EnableHostname = !telemetryConf.DisableHostname
	metricsConf.FilterDefault = telemetryConf.FilterDefault
	metricsConf.AllowedPrefixes = telemetryConf.AllowedPrefixes
	metricsConf.BlockedPrefixes = telemetryConf.BlockedPrefixes
	return metricsConf
}

func setupSinks(telemetryConf *config.TelemetryConfig, hostname string) (metrics.FanoutSink, error) {
	var sinks metrics.FanoutSink
	var errors *multierror.Error
	addSink := func(fn func(*config.TelemetryConfig, string) (metrics.MetricSink, error)) {
		s, err := fn(telemetryConf, hostname)
		if err != nil {
			errors = multierror.Append(errors, err)
			return
		}
		if s != nil {
			sinks = append(sinks, s)
		}
	}

	addSink(statsiteSink)
	addSink(statsdSink)
	addSink(dogstatdSink)
	addSink(circonusSink)
	addSink(PrometheusSink)

	return sinks, errors.ErrorOrNil()
}

// Init configures go-metrics based on map of telemetry config
// values as returned by Runtimecfg.Config().
// Init retries configurating the sinks in case error is retriable
// and retry_failed_connection is set to true.
func Init(cfg *config.TelemetryConfig) (*Telemetry, error) {
	if cfg.Disable {
		return &Telemetry{}, nil
	}

	memSink := metrics.NewInmemSink(10*time.Second, time.Minute)
	metrics.DefaultInmemSignal(memSink)

	metricsConf := computeMetricsConfig(cfg)

	sinks, errs := setupSinks(cfg, metricsConf.HostName)
	if errs != nil {
		return nil, errs
	}
	sinks = append(sinks, memSink)

	metricsServer, err := metrics.NewGlobal(metricsConf, sinks)
	if err != nil {
		return nil, err
	}

	telemetry := &Telemetry{
		Handler:  memSink,
		cancelFn: metricsServer.Shutdown,
	}

	return telemetry, nil
}
