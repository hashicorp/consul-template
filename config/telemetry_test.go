package config

import (
	"testing"
	"time"

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
