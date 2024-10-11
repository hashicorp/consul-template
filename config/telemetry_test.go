package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromConfigParsing(t *testing.T) {
	configStr := "telemetry {" +
		"prometheus_port = 9110" +
		"prometheus_retention_time = \"120s\"" +
		"allowed_prefixes = [\"keep\"]" +
		"blocked_prefixes = [\"dont_keep\"]" +
		"metrics_prefix = \"consul_template\"" +
		"}"

	config, err := Parse(configStr)
	require.NoError(t, err)

	require.Equal(t, 9110, config.Telemetry.PrometheusPort)
	require.Equal(t, 120*time.Second, config.Telemetry.PrometheusRetentionTime)
	require.Equal(t, "consul_template", config.Telemetry.MetricsPrefix)
	require.Equal(t, "consul_template", config.Telemetry.MetricsPrefix)
	require.ElementsMatch(t, []string{"keep"}, config.Telemetry.AllowedPrefixes)
	require.ElementsMatch(t, []string{"dont_keep"}, config.Telemetry.BlockedPrefixes)

	config.Finalize()
	require.Equal(t, 9110, config.Telemetry.PrometheusPort)
	require.Equal(t, 120*time.Second, config.Telemetry.PrometheusRetentionTime)
}

func TestTelemetryNilEmptyConfigMerge(t *testing.T) {
	var nilConfig *TelemetryConfig
	require.Nil(t, nilConfig.Merge(nil))

	emptyConfig := &TelemetryConfig{}
	require.Equal(t, emptyConfig, nilConfig.Merge(emptyConfig))
	require.Equal(t, emptyConfig, emptyConfig.Merge(nil))
}

func TestTelemetryPartialConfigMerge(t *testing.T) {
	// Partial configuration merge test
	partialConfigA := &TelemetryConfig{
		MetricsPrefix:   "prefix",
		Disable:         true,
		AllowedPrefixes: []string{"allowedPrefixA"},
		StatsdAddr:      "statsA",
	}

	partialConfigB := &TelemetryConfig{
		MetricsPrefix:   "new_prefix",
		Disable:         false,
		BlockedPrefixes: []string{"prefix"},
	}

	configC := partialConfigA.Merge(partialConfigB)
	require.NotEqual(t, configC, partialConfigB)

	require.Equal(t, "new_prefix", configC.MetricsPrefix)
	require.False(t, configC.Disable)
	require.Equal(t, []string{"allowedPrefixA"}, configC.AllowedPrefixes)
	require.Equal(t, []string{"prefix"}, configC.BlockedPrefixes)
	require.Equal(t, "statsA", configC.StatsdAddr)
}

func TestTelemetryFullConfigMerge(t *testing.T) {
	configA := &TelemetryConfig{
		Disable:                            false,
		CirconusAPIApp:                     "appA",
		CirconusAPIToken:                   "tokenA",
		CirconusAPIURL:                     "apiUrlA",
		CirconusBrokerID:                   "brokerA",
		CirconusBrokerSelectTag:            "brokerTagA",
		CirconusCheckDisplayName:           "A",
		CirconusCheckForceMetricActivation: false,
		CirconusCheckID:                    "idA",
		CirconusCheckInstanceID:            "instanceA",
		CirconusCheckSearchTag:             "searchTagA",
		CirconusCheckTags:                  "tagA",
		CirconusSubmissionInterval:         "1ms",
		CirconusSubmissionURL:              "urlA",
		DisableHostname:                    false,
		DogstatsdAddr:                      "addrA",
		DogstatsdTags:                      []string{"dsTagA1", "dsTagA2"},
		FilterDefault:                      false,
		AllowedPrefixes:                    []string{"allowedPrefixA"},
		BlockedPrefixes:                    []string{"blockedPrefixA"},
		MetricsPrefix:                      "prefixA",
		StatsdAddr:                         "statsA",
		StatsiteAddr:                       "statsiteA",
		PrometheusPort:                     8080,
		PrometheusRetentionTime:            2 * time.Hour,
	}

	configB := &TelemetryConfig{
		Disable:                            true,
		CirconusAPIApp:                     "appB",
		CirconusAPIToken:                   "tokenB",
		CirconusAPIURL:                     "apiUrlB",
		CirconusBrokerID:                   "brokerB",
		CirconusBrokerSelectTag:            "brokerTagB",
		CirconusCheckDisplayName:           "B",
		CirconusCheckForceMetricActivation: true,
		CirconusCheckID:                    "idB",
		CirconusCheckInstanceID:            "instanceB",
		CirconusCheckSearchTag:             "searchTagB",
		CirconusCheckTags:                  "tagB",
		CirconusSubmissionInterval:         "1ms",
		CirconusSubmissionURL:              "urlB",
		DisableHostname:                    true,
		DogstatsdAddr:                      "addrB",
		DogstatsdTags:                      []string{"dsTagB3"},
		FilterDefault:                      true,
		AllowedPrefixes:                    []string{"allowedPrefixB"},
		BlockedPrefixes:                    []string{"blockedPrefixB"},
		MetricsPrefix:                      "prefixB",
		StatsdAddr:                         "statsB",
		StatsiteAddr:                       "statsiteB",
		PrometheusPort:                     9090,
		PrometheusRetentionTime:            10 * time.Minute,
	}

	assert.Equal(t, configB, configA.Merge(configB))
}

func TestTelemetryConfigGoString(t *testing.T) {
	config := &TelemetryConfig{
		PrometheusRetentionTime: 1 * time.Minute,
	}
	expected := "&TelemetryConfig{Disable:false, CirconusAPIApp:, CirconusAPIToken:<empty>, CirconusAPIURL:, CirconusBrokerID:, CirconusBrokerSelectTag:, CirconusCheckDisplayName:, CirconusCheckForceMetricActivation:false, CirconusCheckID:, CirconusCheckInstanceID:, CirconusCheckSearchTag:, CirconusCheckTags:, CirconusSubmissionInterval:, CirconusSubmissionURL:, DisableHostname:false, DogstatsdAddr:, DogstatsdTags:[], FilterDefault:false, AllowedPrefixes:[], BlockedPrefixes:[], MetricsPrefix:, StatsdAddr:, StatsiteAddr:, PrometheusPort:0, PrometheusRetentionTime:1m0s}"

	assert.Equal(t, expected, config.GoString())
}
