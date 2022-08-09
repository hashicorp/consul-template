package config

/*
  Config structure based on Consul telemetry config:
    https://github.com/hashicorp/consul/blob/main/lib/telemetry.go#L29
*/

import (
	"fmt"
	"time"
)

const (
	defaultMetricsPrefix = "consul_template"
)

// TelemetryConfig is embedded in config.RuntimeConfig and holds the
// configuration variables for go-metrics. It is a separate struct to allow it
// to be exported as JSON and passed to other process like managed connect
// proxies so they can inherit the agent's telemetry config.
//
// It is in lib package rather than agent/config because we need to use it in
// the shared InitTelemetry functions below, but we can't import agent/config
// due to a dependency cycle.
type TelemetryConfig struct {
	// Disable may be set to true to have InitTelemetry to skip initialization
	// and return a nil MetricsSink.
	Disable bool

	// Circonus*: see https://github.com/circonus-labs/circonus-gometrics
	// for more details on the various configuration options.
	// Valid configuration combinations:
	//    - CirconusAPIToken
	//      metric management enabled (search for existing check or create a new one)
	//    - CirconusSubmissionUrl
	//      metric management disabled (use check with specified submission_url,
	//      broker must be using a public SSL certificate)
	//    - CirconusAPIToken + CirconusCheckSubmissionURL
	//      metric management enabled (use check with specified submission_url)
	//    - CirconusAPIToken + CirconusCheckID
	//      metric management enabled (use check with specified id)

	// CirconusAPIApp is an app name associated with API token.
	// Default: "consul"
	//
	// hcl: telemetry { circonus_api_app = string }
	CirconusAPIApp string `json:"circonus_api_app,omitempty" mapstructure:"circonus_api_app"`

	// CirconusAPIToken is a valid API Token used to create/manage check. If provided,
	// metric management is enabled.
	// Default: none
	//
	// hcl: telemetry { circonus_api_token = string }
	CirconusAPIToken string `json:"circonus_api_token,omitempty" mapstructure:"circonus_api_token"`

	// CirconusAPIURL is the base URL to use for contacting the Circonus API.
	// Default: "https://api.circonus.com/v2"
	//
	// hcl: telemetry { circonus_api_url = string }
	CirconusAPIURL string `json:"circonus_apiurl,omitempty" mapstructure:"circonus_apiurl"`

	// CirconusBrokerID is an explicit broker to use when creating a new check. The numeric portion
	// of broker._cid. If metric management is enabled and neither a Submission URL nor Check ID
	// is provided, an attempt will be made to search for an existing check using Instance ID and
	// Search Tag. If one is not found, a new HTTPTRAP check will be created.
	// Default: use Select Tag if provided, otherwise, a random Enterprise Broker associated
	// with the specified API token or the default Circonus Broker.
	// Default: none
	//
	// hcl: telemetry { circonus_broker_id = string }
	CirconusBrokerID string `json:"circonus_broker_id,omitempty" mapstructure:"circonus_broker_id"`

	// CirconusBrokerSelectTag is a special tag which will be used to select a broker when
	// a Broker ID is not provided. The best use of this is to as a hint for which broker
	// should be used based on *where* this particular instance is running.
	// (e.g. a specific geo location or datacenter, dc:sfo)
	// Default: none
	//
	// hcl: telemetry { circonus_broker_select_tag = string }
	CirconusBrokerSelectTag string `json:"circonus_broker_select_tag,omitempty" mapstructure:"circonus_broker_select_tag"`

	// CirconusCheckDisplayName is the name for the check which will be displayed in the Circonus UI.
	// Default: value of CirconusCheckInstanceID
	//
	// hcl: telemetry { circonus_check_display_name = string }
	CirconusCheckDisplayName string `json:"circonus_check_display_name,omitempty" mapstructure:"circonus_check_display_name"`

	// CirconusCheckForceMetricActivation will force enabling metrics, as they are encountered,
	// if the metric already exists and is NOT active. If check management is enabled, the default
	// behavior is to add new metrics as they are encountered. If the metric already exists in the
	// check, it will *NOT* be activated. This setting overrides that behavior.
	// Default: "false"
	//
	// hcl: telemetry { circonus_check_metrics_activation = (true|false)
	CirconusCheckForceMetricActivation string `json:"circonus_check_force_metric_activation,omitempty" mapstructure:"circonus_check_force_metric_activation"`

	// CirconusCheckID is the check id (not check bundle id) from a previously created
	// HTTPTRAP check. The numeric portion of the check._cid field.
	// Default: none
	//
	// hcl: telemetry { circonus_check_id = string }
	CirconusCheckID string `json:"circonus_check_id,omitempty" mapstructure:"circonus_check_id"`

	// CirconusCheckInstanceID serves to uniquely identify the metrics coming from this "instance".
	// It can be used to maintain metric continuity with transient or ephemeral instances as
	// they move around within an infrastructure.
	// Default: hostname:app
	//
	// hcl: telemetry { circonus_check_instance_id = string }
	CirconusCheckInstanceID string `json:"circonus_check_instance_id,omitempty" mapstructure:"circonus_check_instance_id"`

	// CirconusCheckSearchTag is a special tag which, when coupled with the instance id, helps to
	// narrow down the search results when neither a Submission URL or Check ID is provided.
	// Default: service:app (e.g. service:consul)
	//
	// hcl: telemetry { circonus_check_search_tag = string }
	CirconusCheckSearchTag string `json:"circonus_check_search_tag,omitempty" mapstructure:"circonus_check_search_tag"`

	// CirconusCheckSearchTag is a special tag which, when coupled with the instance id, helps to
	// narrow down the search results when neither a Submission URL or Check ID is provided.
	// Default: service:app (e.g. service:consul)
	//
	// hcl: telemetry { circonus_check_tags = string }
	CirconusCheckTags string `json:"circonus_check_tags,omitempty" mapstructure:"circonus_check_tags"`

	// CirconusSubmissionInterval is the interval at which metrics are submitted to Circonus.
	// Default: 10s
	//
	// hcl: telemetry { circonus_submission_interval = "duration" }
	CirconusSubmissionInterval string `json:"circonus_submission_interval,omitempty" mapstructure:"circonus_submission_interval"`

	// CirconusCheckSubmissionURL is the check.config.submission_url field from a
	// previously created HTTPTRAP check.
	// Default: none
	//
	// hcl: telemetry { circonus_submission_url = string }
	CirconusSubmissionURL string `json:"circonus_submission_url,omitempty" mapstructure:"circonus_submission_url"`

	// DisableHostname will disable hostname prefixing for all metrics.
	//
	// hcl: telemetry { disable_hostname = (true|false)
	DisableHostname bool `json:"disable_hostname,omitempty" mapstructure:"disable_hostname"`

	// DogStatsdAddr is the address of a dogstatsd instance. If provided,
	// metrics will be sent to that instance
	//
	// hcl: telemetry { dogstatsd_addr = string }
	DogstatsdAddr string `json:"dogstatsd_addr,omitempty" mapstructure:"dogstatsd_addr"`

	// DogStatsdTags are the global tags that should be sent with each packet to dogstatsd
	// It is a list of strings, where each string looks like "my_tag_name:my_tag_value"
	//
	// hcl: telemetry { dogstatsd_tags = []string }
	DogstatsdTags []string `json:"dogstatsd_tags,omitempty" mapstructure:"dogstatsd_tags"`

	// FilterDefault is the default for whether to allow a metric that's not
	// covered by the filter.
	//
	// hcl: telemetry { filter_default = (true|false) }
	FilterDefault bool `json:"filter_default,omitempty" mapstructure:"filter_default"`

	// AllowedPrefixes is a list of filter rules to apply for allowing metrics
	// by prefix. Use the 'prefix_filter' option and prefix rules with '+' to be
	// included.
	//
	// hcl: telemetry { allowed_prefixes = ["<expr>", "<expr>", ...] }
	AllowedPrefixes []string `json:"allowed_prefixes,omitempty" mapstructure:"allowed_prefixes"`

	// BlockedPrefixes is a list of filter rules to apply for blocking metrics
	// by prefix. Use the 'prefix_filter' option and prefix rules with '-' to be
	// excluded.
	//
	// hcl: telemetry { blocked_prefixes = ["<expr>", "<expr>", ...] }
	BlockedPrefixes []string `json:"blocked_prefixes,omitempty" mapstructure:"blocked_prefixes"`

	// MetricsPrefix is the prefix used to write stats values to.
	// Default: "consul_template."
	//
	// hcl: telemetry { metrics_prefix = string }
	MetricsPrefix string `json:"metrics_prefix,omitempty" mapstructure:"metrics_prefix"`

	// StatsdAddr is the address of a statsd instance. If provided,
	// metrics will be sent to that instance.
	//
	// hcl: telemetry { statsd_address = string }
	StatsdAddr string `json:"statsd_address,omitempty" mapstructure:"statsd_address"`

	// StatsiteAddr is the address of a statsite instance. If provided,
	// metrics will be streamed to that instance.
	//
	// hcl: telemetry { statsite_address = string }
	StatsiteAddr string `json:"statsite_address,omitempty" mapstructure:"statsite_address"`

	// PrometheusRetentionTime is the time before a prometheus metric expires.
	//
	// hcl: telemetry { prometheus_retention_time = "duration" }
	PrometheusRetentionTime time.Duration `json:"prometheus_retention_time,omitempty" mapstructure:"prometheus_retention_time"`

	// PrometheusPort is the REST port under which the metrics can be queried.
	//
	// hcl: telemetry { prometheus_port = int }
	PrometheusPort int `json:"prometheus_port,omitempty" mapstructure:"prometheus_port"`
}

func DefaultTelemetryConfig() *TelemetryConfig {
	return &TelemetryConfig{}
}

func (c *TelemetryConfig) Copy() *TelemetryConfig {
	if c == nil {
		return nil
	}

	return &TelemetryConfig{
		Disable:                            c.Disable,
		CirconusAPIApp:                     c.CirconusAPIApp,
		CirconusAPIToken:                   c.CirconusAPIToken,
		CirconusAPIURL:                     c.CirconusAPIURL,
		CirconusBrokerID:                   c.CirconusBrokerID,
		CirconusBrokerSelectTag:            c.CirconusBrokerSelectTag,
		CirconusCheckDisplayName:           c.CirconusCheckDisplayName,
		CirconusCheckForceMetricActivation: c.CirconusCheckForceMetricActivation,
		CirconusCheckID:                    c.CirconusCheckID,
		CirconusCheckInstanceID:            c.CirconusCheckInstanceID,
		CirconusCheckSearchTag:             c.CirconusCheckSearchTag,
		CirconusCheckTags:                  c.CirconusCheckTags,
		CirconusSubmissionInterval:         c.CirconusSubmissionInterval,
		CirconusSubmissionURL:              c.CirconusSubmissionURL,
		DisableHostname:                    c.DisableHostname,
		DogstatsdAddr:                      c.DogstatsdAddr,
		DogstatsdTags:                      c.DogstatsdTags,
		FilterDefault:                      c.FilterDefault,
		AllowedPrefixes:                    c.AllowedPrefixes,
		BlockedPrefixes:                    c.BlockedPrefixes,
		MetricsPrefix:                      c.MetricsPrefix,
		StatsdAddr:                         c.StatsdAddr,
		StatsiteAddr:                       c.StatsiteAddr,
		PrometheusPort:                     c.PrometheusPort,
		PrometheusRetentionTime:            c.PrometheusRetentionTime,
	}
}

func (c *TelemetryConfig) Merge(o *TelemetryConfig) *TelemetryConfig {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}
	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	r.Disable = o.Disable

	if o.CirconusAPIApp != "" {
		r.CirconusAPIApp = o.CirconusAPIApp
	}
	if o.CirconusAPIToken != "" {
		r.CirconusAPIToken = o.CirconusAPIToken
	}
	if o.CirconusAPIURL != "" {
		r.CirconusAPIURL = o.CirconusAPIURL
	}
	if o.CirconusBrokerID != "" {
		r.CirconusBrokerID = o.CirconusBrokerID
	}
	if o.CirconusBrokerSelectTag != "" {
		r.CirconusBrokerSelectTag = o.CirconusBrokerSelectTag
	}
	if o.CirconusCheckDisplayName != "" {
		r.CirconusCheckDisplayName = o.CirconusCheckDisplayName
	}
	if o.CirconusCheckForceMetricActivation != "" {
		r.CirconusCheckForceMetricActivation = o.CirconusCheckForceMetricActivation
	}
	if o.CirconusCheckID != "" {
		r.CirconusCheckID = o.CirconusCheckID
	}
	if o.CirconusCheckInstanceID != "" {
		r.CirconusCheckInstanceID = o.CirconusCheckInstanceID
	}
	if o.CirconusCheckSearchTag != "" {
		r.CirconusCheckSearchTag = o.CirconusCheckSearchTag
	}
	if o.CirconusCheckTags != "" {
		r.CirconusCheckTags = o.CirconusCheckTags
	}
	if o.CirconusSubmissionInterval != "" {
		r.CirconusSubmissionInterval = o.CirconusSubmissionInterval
	}
	if o.CirconusSubmissionURL != "" {
		r.CirconusSubmissionURL = o.CirconusSubmissionURL
	}
	r.DisableHostname = o.DisableHostname
	if o.DogstatsdAddr != "" {
		r.DogstatsdAddr = o.DogstatsdAddr
	}
	if len(o.DogstatsdTags) != 0 {
		r.DogstatsdTags = o.DogstatsdTags
	}
	r.FilterDefault = o.FilterDefault
	if len(o.AllowedPrefixes) != 0 {
		r.AllowedPrefixes = o.AllowedPrefixes
	}
	if len(o.BlockedPrefixes) != 0 {
		r.BlockedPrefixes = o.BlockedPrefixes
	}
	if o.MetricsPrefix != "" {
		r.MetricsPrefix = o.MetricsPrefix
	}
	if o.StatsdAddr != "" {
		r.StatsdAddr = o.StatsdAddr
	}
	if o.StatsiteAddr != "" {
		r.StatsiteAddr = o.StatsiteAddr
	}

	if o.PrometheusRetentionTime.Nanoseconds() > 0 {
		r.PrometheusRetentionTime = o.PrometheusRetentionTime
	}
	if o.PrometheusPort != 0 {
		r.PrometheusPort = o.PrometheusPort
	}

	return r
}

func (c *TelemetryConfig) GoString() string {
	if c == nil {
		return "(*TelemetryConfig)(nil)"
	}
	return fmt.Sprintf("&TelemetryConfig{"+
		"Disable:%v, "+
		"CirconusAPIApp:%s, "+
		"CirconusAPIToken:%s, "+
		"CirconusAPIURL:%s, "+
		"CirconusBrokerID:%s, "+
		"CirconusBrokerSelectTag:%s, "+
		"CirconusCheckDisplayName:%s, "+
		"CirconusCheckForceMetricActivation:%s, "+
		"CirconusCheckID:%s, "+
		"CirconusCheckInstanceID:%s, "+
		"CirconusCheckSearchTag:%s, "+
		"CirconusCheckTags:%s, "+
		"CirconusSubmissionInterval:%s, "+
		"CirconusSubmissionURL:%s, "+
		"DisableHostname:%v, "+
		"DogstatsdAddr:%s, "+
		"DogstatsdTags:%v, "+
		"FilterDefault:%v, "+
		"AllowedPrefixes:%s, "+
		"BlockedPrefixes:%s, "+
		"MetricsPrefix:%s, "+
		"StatsdAddr:%s, "+
		"StatsiteAddr:%s, "+
		"PrometheusPort:%d, "+
		"PrometheusRetentionTime:%s}",
		c.Disable,
		c.CirconusAPIApp,
		c.CirconusAPIToken,
		c.CirconusAPIURL,
		c.CirconusBrokerID,
		c.CirconusBrokerSelectTag,
		c.CirconusCheckDisplayName,
		c.CirconusCheckForceMetricActivation,
		c.CirconusCheckID,
		c.CirconusCheckInstanceID,
		c.CirconusCheckSearchTag,
		c.CirconusCheckTags,
		c.CirconusSubmissionInterval,
		c.CirconusSubmissionURL,
		c.DisableHostname,
		c.DogstatsdAddr,
		c.DogstatsdTags,
		c.FilterDefault,
		c.AllowedPrefixes,
		c.BlockedPrefixes,
		c.MetricsPrefix,
		c.StatsdAddr,
		c.StatsiteAddr,
		c.PrometheusPort,
		c.PrometheusRetentionTime,
	)
}

func (c *TelemetryConfig) Finalize() {
	if c == nil {
		return
	}
	if c.MetricsPrefix == "" {
		c.MetricsPrefix = defaultMetricsPrefix
	}

	c.AllowedPrefixes = append(c.AllowedPrefixes, c.MetricsPrefix)

	if c.PrometheusRetentionTime.Nanoseconds() < 1 {
		c.PrometheusRetentionTime = 60 * time.Second
	}
}
