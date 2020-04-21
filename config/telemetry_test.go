package config

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestTelemetryConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *TelemetryConfig
	}{
		{
			"nil",
			nil,
		},
		{
			"empty",
			&TelemetryConfig{},
		},
		{
			"stdout",
			&TelemetryConfig{
				Stdout: &StdoutConfig{},
			},
		},
		{
			"dogstatsd",
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{
					ReportingInterval: TimeDuration(time.Second * 5),
				},
			},
		},
		{
			"prometheus",
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{
					Port:              Uint(8888),
					ReportingInterval: TimeDuration(time.Second * 5),
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Copy()
			if !reflect.DeepEqual(tc.a, r) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.a, r)
			}
		})
	}
}

func TestTelemetryConfig_Merge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a    *TelemetryConfig
		b    *TelemetryConfig
		r    *TelemetryConfig
	}{
		{
			"nil_a",
			nil,
			&TelemetryConfig{},
			&TelemetryConfig{},
		},
		{
			"nil_b",
			&TelemetryConfig{},
			nil,
			&TelemetryConfig{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&TelemetryConfig{},
			&TelemetryConfig{},
			&TelemetryConfig{},
		},
		{
			"stdout_overrides",
			&TelemetryConfig{
				Stdout: &StdoutConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{
				Stdout: &StdoutConfig{
					ReportingInterval: TimeDuration(time.Minute),
					DoNotPrintTime:    Bool(true),
				},
			},
			&TelemetryConfig{
				Stdout: &StdoutConfig{
					ReportingInterval: TimeDuration(time.Minute),
					DoNotPrintTime:    Bool(true),
				},
			},
		},
		{
			"stdout_empty_one",
			&TelemetryConfig{
				Stdout: &StdoutConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{},
			&TelemetryConfig{
				Stdout: &StdoutConfig{ReportingInterval: TimeDuration(time.Second)},
			},
		},
		{
			"stdout_empty_two",
			&TelemetryConfig{
				Stdout: &StdoutConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{},
			&TelemetryConfig{
				Stdout: &StdoutConfig{ReportingInterval: TimeDuration(time.Second)},
			},
		},
		{
			"stdout_same",
			&TelemetryConfig{
				Stdout: &StdoutConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{
				Stdout: &StdoutConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{
				Stdout: &StdoutConfig{ReportingInterval: TimeDuration(time.Second)},
			},
		},
		{
			"dogstatsd_overrides",
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{
					Address:           String("udp://127.0.0.1:8888"),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{
					Address:           String("udp://127.0.0.1:8888"),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
		},
		{
			"dogstatsd_empty_one",
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{},
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{ReportingInterval: TimeDuration(time.Second)},
			},
		},
		{
			"dogstatsd_empty_two",
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{},
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{ReportingInterval: TimeDuration(time.Second)},
			},
		},
		{
			"dogstatsd_same",
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{
					Address:           String("udp://127.0.0.1:8888"),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{
					Address:           String("udp://127.0.0.1:8888"),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{
					Address:           String("udp://127.0.0.1:8888"),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
		},
		{
			"prometheus_overrides",
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{
					Port:              Uint(8080),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{
					Port:              Uint(8080),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
		},
		{
			"prometheus_empty_one",
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{},
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{ReportingInterval: TimeDuration(time.Second)},
			},
		},
		{
			"prometheus_empty_two",
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{ReportingInterval: TimeDuration(time.Second)},
			},
			&TelemetryConfig{},
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{ReportingInterval: TimeDuration(time.Second)},
			},
		},
		{
			"prometheus_same",
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{
					Port:              Uint(8080),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{
					Port:              Uint(8080),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{
					Port:              Uint(8080),
					ReportingInterval: TimeDuration(time.Minute),
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			if !reflect.DeepEqual(tc.r, r) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.r, r)
			}
		})
	}
}

func TestTelemetryConfig_Finalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    *TelemetryConfig
		r    *TelemetryConfig
	}{
		{
			"empty",
			&TelemetryConfig{},
			&TelemetryConfig{},
		},
		{
			"with_stdout",
			&TelemetryConfig{
				Stdout: &StdoutConfig{
					ReportingInterval: TimeDuration(time.Minute * 5),
				},
			},
			&TelemetryConfig{
				Stdout: &StdoutConfig{
					ReportingInterval: TimeDuration(time.Minute * 5),
					PrettyPrint:       Bool(false),
					DoNotPrintTime:    Bool(false),
				},
			},
		},
		{
			"with_dogstatsd",
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{
					ReportingInterval: TimeDuration(time.Minute * 5),
				},
			},
			&TelemetryConfig{
				DogStatsD: &DogStatsDConfig{
					Address:           String("udp://127.0.0.1:8125"),
					ReportingInterval: TimeDuration(time.Minute * 5),
				},
			},
		},
		{
			"with_prometheus",
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{
					Port: Uint(80),
				},
			},
			&TelemetryConfig{
				Prometheus: &PrometheusConfig{
					Port:              Uint(80),
					ReportingInterval: TimeDuration(DefaultReportingInterval),
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			tc.i.Finalize()
			if !reflect.DeepEqual(tc.r, tc.i) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.r, tc.i)
			}
		})
	}
}
