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
		"}"

	config, err := Parse(configStr)
	require.NoError(t, err)

	require.Equal(t, 9110, config.Telemetry.PrometheusPort)
	require.Equal(t, 120*time.Second, config.Telemetry.PrometheusRetentionTime)

	config.Finalize()
	require.Equal(t, 9110, config.Telemetry.PrometheusPort)
	require.Equal(t, 120*time.Second, config.Telemetry.PrometheusRetentionTime)
}
