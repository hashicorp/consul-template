package telemetry

/*
  methods based on Consul telemetry:
    https://github.com/hashicorp/consul/blob/main/lib/telemetry.go
*/

import (
	"context"
	"net/http"
	"sync"
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
	mu       sync.Mutex
	cancelFn context.CancelFunc
}

func (tel *Telemetry) Stop() {
	tel.mu.Lock()
	defer tel.mu.Unlock()

	if tel.cancelFn != nil {
		tel.cancelFn()
	}
}

func ConfigureSinks(cfg *config.TelemetryConfig, memSink metrics.MetricSink) (metrics.FanoutSink, error) {
	metricsConf := metrics.DefaultConfig(cfg.MetricsPrefix)
	metricsConf.EnableHostname = !cfg.DisableHostname
	metricsConf.FilterDefault = cfg.FilterDefault
	metricsConf.AllowedPrefixes = cfg.AllowedPrefixes
	metricsConf.BlockedPrefixes = cfg.BlockedPrefixes

	var sinks metrics.FanoutSink
	var errors *multierror.Error
	addSink := func(fn func(*config.TelemetryConfig, string) (metrics.MetricSink, error)) {
		s, err := fn(cfg, metricsConf.HostName)
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
	addSink(circonusSink)
	addSink(PrometheusSink)

	if len(sinks) > 0 {
		sinks = append(sinks, memSink)
		_, err := metrics.NewGlobal(metricsConf, sinks)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	} else {
		metricsConf.EnableHostname = false
		_, err := metrics.NewGlobal(metricsConf, memSink)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}

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

	telemetry := &Telemetry{
		Handler: memSink,
	}

	_, errs := ConfigureSinks(cfg, memSink)

	if errs != nil {
		return nil, errs
	}

	return telemetry, nil
}
