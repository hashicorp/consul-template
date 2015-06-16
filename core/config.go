package core

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
)

// The pattern to split the config template syntax on
var configTemplateRe = regexp.MustCompile("([a-zA-Z]:)?([^:]+)")

// Config is used to configure Consul Template
type Config struct {
	// Path is the path to this configuration file on disk. This value is not
	// read from disk by rather dynamically populated by the code so the Config
	// has a reference to the path to the file on disk that created it.
	Path string `json:"path" mapstructure:"-"`

	// Consul is the location of the Consul instance to query (may be an IP
	// address or FQDN) with port.
	Consul string `json:"consul" mapstructure:"consul"`

	// Token is the Consul API token.
	Token string `json:"token" mapstructure:"token"`

	// Auth is the HTTP basic authentication for communicating with Consul.
	Auth    *AuthConfig   `json:"auth" mapstructure:"-"`
	AuthRaw []*AuthConfig `json:"-" mapstructure:"auth"`

	// SSL indicates we should use a secure connection while talking to
	// Consul. This requires Consul to be configured to serve HTTPS.
	SSL    *SSLConfig   `json:"ssl" mapstructure:"-"`
	SSLRaw []*SSLConfig `json:"-" mapstructure:"ssl"`

	// Syslog is the configuration for syslog.
	Syslog    *SyslogConfig   `json:"syslog" mapstructure:"-"`
	SyslogRaw []*SyslogConfig `json:"-" mapstructure:"syslog"`

	// MaxStale is the maximum amount of time for staleness from Consul as given
	// by LastContact. If supplied, Consul Template will query all servers instead
	// of just the leader.
	MaxStale    time.Duration `json:"max_stale" mapstructure:"-"`
	MaxStaleRaw string        `json:"-" mapstructure:"max_stale"`

	// ConfigTemplates is a slice of the ConfigTemplate objects in the config.
	ConfigTemplates []*ConfigTemplate `json:"templates" mapstructure:"template"`

	// Retry is the duration of time to wait between Consul failures.
	Retry    time.Duration `json:"retry" mapstructure:"-"`
	RetryRaw string        `json:"-" mapstructure:"retry" json:""`

	// Wait is the quiescence timers.
	Wait    *watch.Wait `json:"wait" mapstructure:"-"`
	WaitRaw string      `json:"-" mapstructure:"wait" json:""`

	// LogLevel is the level with which to log for this config.
	LogLevel string `json:"log_level" mapstructure:"log_level"`
}

// Merge merges the values in config into this config object. Values in the
// config object overwrite the values in c.
func (c *Config) Merge(config *Config) {
	if config.Consul != "" {
		c.Consul = config.Consul
	}

	if config.Token != "" {
		c.Token = config.Token
	}

	if config.Auth != nil {
		c.Auth = &AuthConfig{
			Enabled:  config.Auth.Enabled,
			Username: config.Auth.Username,
			Password: config.Auth.Password,
		}
	}

	if config.SSL != nil {
		c.SSL = &SSLConfig{
			Enabled: config.SSL.Enabled,
			Verify:  config.SSL.Verify,
			Cert:    config.SSL.Cert,
			CaCert:  config.SSL.CaCert,
		}
	}

	if config.Syslog != nil {
		c.Syslog = &SyslogConfig{
			Enabled:  config.Syslog.Enabled,
			Facility: config.Syslog.Facility,
		}
	}

	if config.MaxStale != 0 {
		c.MaxStale = config.MaxStale
		c.MaxStaleRaw = config.MaxStaleRaw
	}

	if len(config.ConfigTemplates) > 0 {
		if c.ConfigTemplates == nil {
			c.ConfigTemplates = make([]*ConfigTemplate, 0, 1)
		}
		for _, template := range config.ConfigTemplates {
			c.ConfigTemplates = append(c.ConfigTemplates, &ConfigTemplate{
				Source:         template.Source,
				Destination:    template.Destination,
				StartCommand:   template.StartCommand,
				RestartCommand: template.RestartCommand,
				StopCommand:    template.StopCommand,
				First:          template.First,
			})
		}
	}

	if config.Retry != 0 {
		c.Retry = config.Retry
		c.RetryRaw = config.RetryRaw
	}

	if config.Wait != nil {
		c.Wait = &watch.Wait{
			Min: config.Wait.Min,
			Max: config.Wait.Max,
		}
		c.WaitRaw = config.WaitRaw
	}

	if config.LogLevel != "" {
		c.LogLevel = config.LogLevel
	}
}

// ParseConfig reads the configuration file at the given path and returns a new
// Config struct with the data populated.
func ParseConfig(path string) (*Config, error) {
	var errs *multierror.Error

	// Read the contents of the file
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		errs = multierror.Append(errs, err)
		return nil, errs.ErrorOrNil()
	}

	// Parse the file (could be HCL or JSON)
	var parsed interface{}
	if err := hcl.Decode(&parsed, string(contents)); err != nil {
		errs = multierror.Append(errs, err)
		return nil, errs.ErrorOrNil()
	}

	// Create a new, empty config
	config := &Config{}

	// Use mapstructure to populate the basic config fields
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Metadata:    nil,
		Result:      config,
	})
	if err != nil {
		errs = multierror.Append(errs, err)
		return nil, errs.ErrorOrNil()
	}

	if err := decoder.Decode(parsed); err != nil {
		errs = multierror.Append(errs, err)
		return nil, errs.ErrorOrNil()
	}

	// Store a reference to the path where this config was read from
	config.Path = path

	// Parse the MaxStale component
	if raw := config.MaxStaleRaw; raw != "" {
		stale, err := time.ParseDuration(raw)

		if err == nil {
			config.MaxStale = stale
		} else {
			errs = multierror.Append(errs, fmt.Errorf("max_stale invalid: %v", err))
		}
	}

	// Extract the last Auth block
	if len(config.AuthRaw) > 0 {
		config.Auth = config.AuthRaw[len(config.AuthRaw)-1]
	}

	// Extract the last SSL block
	if len(config.SSLRaw) > 0 {
		config.SSL = config.SSLRaw[len(config.SSLRaw)-1]
	}

	// Extract the last Syslog block
	if len(config.SyslogRaw) > 0 {
		config.Syslog = config.SyslogRaw[len(config.SyslogRaw)-1]
	}

	// Parse the Retry component
	if raw := config.RetryRaw; raw != "" {
		retry, err := time.ParseDuration(raw)

		if err == nil {
			config.Retry = retry
		} else {
			errs = multierror.Append(errs, fmt.Errorf("retry invalid: %v", err))
		}
	}

	// Parse the Wait component
	if raw := config.WaitRaw; raw != "" {
		wait, err := watch.ParseWait(raw)

		if err == nil {
			config.Wait = wait
		} else {
			errs = multierror.Append(errs, fmt.Errorf("wait invalid: %v", err))
		}
	}

	return config, errs.ErrorOrNil()
}

// DefaultConfig returns the default configuration struct.
func DefaultConfig() *Config {
	logLevel := os.Getenv("CONSUL_TEMPLATE_LOG")
	if logLevel == "" {
		logLevel = "WARN"
	}

	return &Config{
		Auth: &AuthConfig{
			Enabled: false,
		},
		SSL: &SSLConfig{
			Enabled: false,
			Verify:  true,
		},
		Syslog: &SyslogConfig{
			Enabled:  false,
			Facility: "LOCAL0",
		},
		ConfigTemplates: []*ConfigTemplate{},
		Retry:           5 * time.Second,
		Wait:            &watch.Wait{},
		LogLevel:        logLevel,
	}
}

// AuthConfig is the HTTP basic authentication data.
type AuthConfig struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`
}

// String is the string representation of this authentication. If authentication
// is not enabled, this returns the empty string. The username and password will
// be separated by a colon.
func (a *AuthConfig) String() string {
	if !a.Enabled {
		return ""
	}

	if a.Password != "" {
		return fmt.Sprintf("%s:%s", a.Username, a.Password)
	}

	return a.Username
}

// SSLConfig is the configuration for SSL.
type SSLConfig struct {
	Enabled bool   `json:"enabled" mapstructure:"enabled"`
	Verify  bool   `json:"verify" mapstructure:"verify"`
	Cert    string `json:"cert" mapstructure:"cert"`
	CaCert  string `json:"ca_cert" mapstructure:"ca_cert"`
}

// SyslogConfig is the configuration for syslog.
type SyslogConfig struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
	Facility string `json:"facility" mapstructure:"facility"`
}

// ConfigTemplate is the representation of an input template, output location,
// and optional command to execute when rendered
type ConfigTemplate struct {
	Source         string `json:"source" mapstructure:"source"`
	Destination    string `json:"destination" mapstructure:"destination"`
	StartCommand   string `json:"startcommand" mapstructure:"startcommand"`
	RestartCommand string `json:"restartcommand" mapstructure:"restartcommand"`
	StopCommand    string `json:"stopcommand" mapstructure:"stopcommand"`
	// A binary value on whether the service has been executed
	// at least once or not.
	// This is to know which command (StartCommand or RestartCommand)
	// to pick next (upon a new rendering of the template)
	First          bool
}

// ParseConfigTemplate parses a string into a ConfigTemplate struct
func ParseConfigTemplate(s string) (*ConfigTemplate, error) {
	if len(strings.TrimSpace(s)) < 1 {
		return nil, errors.New("cannot specify empty template declaration")
	}

	var source, destination, command string
	parts := configTemplateRe.FindAllString(s, -1)

	switch len(parts) {
	case 1:
		source = parts[0]
	case 2:
		source, destination = parts[0], parts[1]
	case 3:
		source, destination, command = parts[0], parts[1], parts[2]
	default:
		return nil, errors.New("invalid template declaration format")
	}

	return &ConfigTemplate{source, destination, "", command, "", false}, nil
}
