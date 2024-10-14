package telemetry

import (
	"github.com/armon/go-metrics"
	"github.com/armon/go-metrics/circonus"
	"github.com/hashicorp/consul-template/config"
)

/*
  methods extracted from Consul telemetry:
    https://github.com/hashicorp/consul/blob/main/lib/telemetry.go#L274
*/

func circonusSink(cfg *config.TelemetryConfig, _ string) (metrics.MetricSink, error) {
	token := cfg.CirconusAPIToken
	url := cfg.CirconusSubmissionURL
	if token == "" && url == "" {
		return nil, nil
	}

	conf := &circonus.Config{}
	conf.Interval = cfg.CirconusSubmissionInterval
	conf.CheckManager.API.TokenKey = token
	conf.CheckManager.API.TokenApp = cfg.CirconusAPIApp
	conf.CheckManager.API.URL = cfg.CirconusAPIURL
	conf.CheckManager.Check.SubmissionURL = url
	conf.CheckManager.Check.ID = cfg.CirconusCheckID
	conf.CheckManager.Check.ForceMetricActivation = cfg.GetCirconusCheckForceMetricActivation()
	conf.CheckManager.Check.InstanceID = cfg.CirconusCheckInstanceID
	conf.CheckManager.Check.SearchTag = cfg.CirconusCheckSearchTag
	conf.CheckManager.Check.DisplayName = cfg.CirconusCheckDisplayName
	conf.CheckManager.Check.Tags = cfg.CirconusCheckTags
	conf.CheckManager.Broker.ID = cfg.CirconusBrokerID
	conf.CheckManager.Broker.SelectTag = cfg.CirconusBrokerSelectTag

	if conf.CheckManager.Check.DisplayName == "" {
		conf.CheckManager.Check.DisplayName = "Consul"
	}

	if conf.CheckManager.API.TokenApp == "" {
		conf.CheckManager.API.TokenApp = "consul"
	}

	if conf.CheckManager.Check.SearchTag == "" {
		conf.CheckManager.Check.SearchTag = "service:consul"
	}

	sink, err := circonus.NewCirconusSink(conf)
	if err != nil {
		return nil, err
	}
	sink.Start()
	return sink, nil
}
