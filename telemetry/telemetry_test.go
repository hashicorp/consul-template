package telemetry

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/config"
	"github.com/stretchr/testify/require"
)

func newCfg() *config.TelemetryConfig {
	return &config.TelemetryConfig{
		StatsdAddr:    "statsd.host:1234",
		StatsiteAddr:  "statsite.host:1234",
		DogstatsdAddr: "mydog.host:8125",
	}
}

func TestConfigureSinks(t *testing.T) {
	cfg := newCfg()
	sinks, err := setupSinks(cfg, "")
	require.Error(t, err)
	// 3 sinks: statsd, statsite, inmem
	require.Equal(t, 2, len(sinks))

	cfg = &config.TelemetryConfig{
		DogstatsdAddr: "",
	}
	_, err = setupSinks(cfg, "")
	require.NoError(t, err)

}

func TestPrometheusMetrics(t *testing.T) {

	// expected metrics based on the first metric-related PR:
	//  https://github.com/hashicorp/consul-template/pull/1378/files#diff-b335630551682c19a781afebcf4d07bf978fb1f8ac04c6bf87428ed5106870f5R2680
	expectedMetrics := []string{
		"# HELP consul_template_commands_exec The number of commands executed with labels status=(success|error)",
		"# TYPE consul_template_commands_exec counter",
		"consul_template_commands_exec{status=\"error\"} 0",
		"consul_template_commands_exec{status=\"success\"} 1",
		"# HELP consul_template_dependencies_received A counter of dependencies received with labels type=(consul|vault|local) and id=dependencyString",
		"# TYPE consul_template_dependencies_received counter",
		"consul_template_dependencies_received{id=\"kv.block(hello)\",type=\"consul\"} 1",
		"# HELP consul_template_runner_actions A count of runner actions with labels action=(start|stop|run)",
		"# TYPE consul_template_runner_actions counter",
		"consul_template_runner_actions{action=\"run\"} 2",
		"consul_template_runner_actions{action=\"start\"} 1",
		"# HELP consul_template_templates_rendered A counter of templates rendered with labels id=templateID and status=(rendered|would|quiescence)",
		"# TYPE consul_template_templates_rendered counter",
		"consul_template_templates_rendered{id=\"aadcafd7f28f1d9fc5e76ab2e029f844\",status=\"rendered\"} 1",
	}

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	address := l.Addr().String()
	portStr := address[strings.LastIndex(address, ":")+1:]

	err = l.Close()
	require.NoError(t, err)

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	cfg := config.TelemetryConfig{
		PrometheusRetentionTime: 60 * time.Second,
		PrometheusPort:          port,
	}

	_, err = Init(&cfg)
	require.NoError(t, err)

	CounterCommandExecs.Add(0, NewLabel("status", "error"))
	CounterCommandExecs.Add(1, NewLabel("status", "success"))
	CounterDependenciesReceived.Add(1, NewLabel("id", "kv.block(hello)"), NewLabel("type", "consul"))
	CounterActions.Add(2, NewLabel("action", "run"))
	CounterActions.Add(1, NewLabel("action", "start"))
	CounterTemplatesRendered.Add(1, NewLabel("id", "aadcafd7f28f1d9fc5e76ab2e029f844"), NewLabel("status", "rendered"))

	httpClient := http.DefaultClient

	resp, err := httpClient.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	require.NoError(t, err)

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	actualMetrics := strings.Split(string(b), "\n")

	missingActualMetrics := []string{}

	prefixes := []string{"# HELP go_", "# TYPE go_", "go_", "# HELP process_", "# TYPE process_", "process_"}
	for _, actualMetric := range actualMetrics {
		if slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(actualMetric, p) }) ||
			actualMetric == "" {
			continue
		}

		contained := false
		for _, expectedMetric := range expectedMetrics {
			if actualMetric == expectedMetric {
				contained = true
				break
			}
		}

		if !contained {
			missingActualMetrics = append(missingActualMetrics, actualMetric)
		}
	}

	t.Log(len(missingActualMetrics))
	require.Emptyf(t, missingActualMetrics, "The following metrics are missing:\n - %s", strings.Join(missingActualMetrics, "\n - "))

}

func TestInitWithEmptyConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Finalize()
	_, err := Init(cfg.Telemetry)
	require.NoError(t, err)

}
