package config

import (
	"fmt"
	"time"
)

const (
	// DefaultReportingInterval is the default period to emit metrics.
	DefaultReportingInterval time.Duration = time.Minute
	// DefaultPrometheusPort is the default port for HTTP service to bind on.
	DefaultPrometheusPort uint = 8888
)

// TelemetryConfig is the configuration for telemetry.
type TelemetryConfig struct {
	Stdout     *StdoutConfig     `mapstructure:"stdout"`
	DogStatsD  *DogStatsDConfig  `mapstructure:"dogstatsd"`
	Prometheus *PrometheusConfig `mapstructure:"prometheus"`
}

// StdoutConfig is the configuration for emitting metrics to stdout.
type StdoutConfig struct {
	ReportingInterval *time.Duration `mapstructure:"reporting_interval"`
	PrettyPrint       *bool          `mapstructure:"pretty_print"`
	DoNotPrintTime    *bool          `mapstructure:"do_not_print_time"`
}

// DogStatsDConfig is the configuration for emitting metrics to dogstatsd.
type DogStatsDConfig struct {
	Address           *string        `mapstructure:"address"`
	ReportingInterval *time.Duration `mapstructure:"reporting_interval"`
}

// PrometheusConfig is the configuration for emitting metrics to Prometheus.
type PrometheusConfig struct {
	Port              *uint          `mapstructure:"port"`
	ReportingInterval *time.Duration `mapstructure:"reporting_interval"`
}

// DefaultTelemetryConfig returns a configuration that is populated with the
// default values.
func DefaultTelemetryConfig() *TelemetryConfig {
	return &TelemetryConfig{}
}

// Copy returns a deep copy of this configuration.
func (c *TelemetryConfig) Copy() *TelemetryConfig {
	if c == nil {
		return nil
	}

	var o TelemetryConfig
	if c.Stdout != nil {
		o.Stdout = c.Stdout.Copy()
	}
	if c.DogStatsD != nil {
		o.DogStatsD = c.DogStatsD.Copy()
	}
	if c.Prometheus != nil {
		o.Prometheus = c.Prometheus.Copy()
	}

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality
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

	if o.Stdout != nil {
		r.Stdout = o.Stdout.Copy()
	}

	if o.DogStatsD != nil {
		r.DogStatsD = o.DogStatsD.Copy()
	}

	if o.Prometheus != nil {
		r.Prometheus = o.Prometheus.Copy()
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *TelemetryConfig) Finalize() {
	if c == nil {
		return
	}

	c.Stdout.Finalize()
	c.DogStatsD.Finalize()
	c.Prometheus.Finalize()
}

// GoString defines the printable version of this struct.
func (c *TelemetryConfig) GoString() string {
	if c == nil {
		return "(*TelemetryConfig)(nil)"
	}

	// TODO
	return fmt.Sprintf("&TelemetryConfig{"+
		"Stdout:%s, "+
		"DogStatsD:%s, "+
		"Prometheus:%s, "+
		"}",
		c.Stdout.GoString(),
		c.DogStatsD.GoString(),
		c.Prometheus.GoString(),
	)
}

// DefaultStdoutConfig returns a configuration that is populated with the
// default values.
func DefaultStdoutConfig() *StdoutConfig {
	return &StdoutConfig{
		ReportingInterval: TimeDuration(DefaultReportingInterval),
		PrettyPrint:       Bool(false),
		DoNotPrintTime:    Bool(false),
	}
}

// Copy returns a deep copy of this configuration.
func (c *StdoutConfig) Copy() *StdoutConfig {
	if c == nil {
		return nil
	}

	return &StdoutConfig{
		ReportingInterval: TimeDurationCopy(c.ReportingInterval),
		PrettyPrint:       BoolCopy(c.PrettyPrint),
		DoNotPrintTime:    BoolCopy(c.DoNotPrintTime),
	}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *StdoutConfig) Merge(o *StdoutConfig) *StdoutConfig {
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

	if o.ReportingInterval != nil {
		r.ReportingInterval = TimeDurationCopy(o.ReportingInterval)
	}

	if o.PrettyPrint != nil {
		r.PrettyPrint = BoolCopy(o.PrettyPrint)
	}

	if o.DoNotPrintTime != nil {
		r.DoNotPrintTime = BoolCopy(o.DoNotPrintTime)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *StdoutConfig) Finalize() {
	if c == nil {
		return
	}

	d := DefaultStdoutConfig()

	if c.ReportingInterval == nil {
		c.ReportingInterval = d.ReportingInterval
	}

	if c.PrettyPrint == nil {
		c.PrettyPrint = d.PrettyPrint
	}

	if c.DoNotPrintTime == nil {
		c.DoNotPrintTime = d.DoNotPrintTime
	}
}

// GoString defines the printable version of this struct.
func (c *StdoutConfig) GoString() string {
	if c == nil {
		return "(*StdoutConfig)(nil)"
	}

	return fmt.Sprintf("&StdoutConfig{"+
		"ReportingInterval:%s, "+
		"PrettyPrint:%s, "+
		"DoNotPrintTime:%s, "+
		"}",
		TimeDurationGoString(c.ReportingInterval),
		BoolGoString(c.PrettyPrint),
		BoolGoString(c.DoNotPrintTime),
	)
}

// DefaultDogStatsDConfig returns a configuration that is populated with the
// default values.
func DefaultDogStatsDConfig() *DogStatsDConfig {
	return &DogStatsDConfig{
		Address:           String("udp://127.0.0.1:8125"),
		ReportingInterval: TimeDuration(DefaultReportingInterval),
	}
}

// Copy returns a deep copy of this configuration.
func (c *DogStatsDConfig) Copy() *DogStatsDConfig {
	if c == nil {
		return nil
	}

	return &DogStatsDConfig{
		Address:           StringCopy(c.Address),
		ReportingInterval: TimeDurationCopy(c.ReportingInterval),
	}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *DogStatsDConfig) Merge(o *DogStatsDConfig) *DogStatsDConfig {
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

	if o.Address != nil {
		r.Address = StringCopy(o.Address)
	}

	if o.ReportingInterval != nil {
		r.ReportingInterval = TimeDurationCopy(o.ReportingInterval)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *DogStatsDConfig) Finalize() {
	if c == nil {
		return
	}

	d := DefaultDogStatsDConfig()

	if c.Address == nil {
		c.Address = d.Address
	}

	if c.ReportingInterval == nil {
		c.ReportingInterval = d.ReportingInterval
	}
}

// GoString defines the printable version of this struct.
func (c *DogStatsDConfig) GoString() string {
	if c == nil {
		return "(*DogStatsDConfig)(nil)"
	}

	return fmt.Sprintf("&DogStatsDConfig{"+
		"Address:%s, "+
		"ReportingInterval:%s, "+
		"}",
		StringGoString(c.Address),
		TimeDurationGoString(c.ReportingInterval),
	)
}

// DefaultPrometheusConfig returns a configuration that is populated with the
// default values.
func DefaultPrometheusConfig() *PrometheusConfig {
	return &PrometheusConfig{
		Port:              Uint(DefaultPrometheusPort),
		ReportingInterval: TimeDuration(DefaultReportingInterval),
	}
}

// Copy returns a deep copy of this configuration.
func (c *PrometheusConfig) Copy() *PrometheusConfig {
	if c == nil {
		return nil
	}

	return &PrometheusConfig{
		Port:              UintCopy(c.Port),
		ReportingInterval: TimeDurationCopy(c.ReportingInterval),
	}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *PrometheusConfig) Merge(o *PrometheusConfig) *PrometheusConfig {
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

	if o.Port != nil {
		r.Port = UintCopy(o.Port)
	}

	if o.ReportingInterval != nil {
		r.ReportingInterval = TimeDurationCopy(o.ReportingInterval)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *PrometheusConfig) Finalize() {
	if c == nil {
		return
	}

	d := DefaultPrometheusConfig()

	if c.Port == nil {
		c.Port = d.Port
	}

	if c.ReportingInterval == nil {
		c.ReportingInterval = d.ReportingInterval
	}
}

// GoString defines the printable version of this struct.
func (c *PrometheusConfig) GoString() string {
	if c == nil {
		return "(*PrometheusConfig)(nil)"
	}

	return fmt.Sprintf("&PrometheusConfig{"+
		"Port:%s, "+
		"ReportingInterval:%s, "+
		"}",
		UintGoString(c.Port),
		TimeDurationGoString(c.ReportingInterval),
	)
}
