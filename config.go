package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/consul-template/watch"
	multierror "github.com/hashicorp/go-multierror"
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
	Path string `mapstructure:"-"`

	// Consul is the location of the Consul instance to query (may be an IP
	// address or FQDN) with port.
	Consul string `mapstructure:"consul"`

	// SSL indicates we should use a secure connection while talking to
	// Consul. This requires Consul to be configured to serve HTTPS.
	SSL bool `mapstructure:"ssl"`

	// Authentication is the HTTP basic authentication to use when communicating
	// with Consul.
	Auth *Auth `mapstructure:"auth"`

	// MaxStale is the maximum amount of time for staleness from Consul as given
	// by LastContact. If supplied, Consul Template will query all servers instead
	// of just the leader.
	MaxStale    time.Duration `mapstructure:"-"`
	MaxStaleRaw string        `mapstructure:"max_stale"`

	// SSLNoVerify determines if we should skip certificate warnings
	SSLNoVerify bool `mapstructure:"ssl_no_verify"`

	// ConfigTemplates is a slice of the ConfigTemplate objects in the config.
	ConfigTemplates []*ConfigTemplate `mapstructure:"template"`

	// Retry is the duration of time to wait between Consul failures.
	Retry    time.Duration `mapstructure:"-"`
	RetryRaw string        `mapstructure:"retry" json:""`

	// Token is the Consul API token.
	Token string `mapstructure:"token"`

	// Wait
	Wait    *watch.Wait `mapstructure:"-"`
	WaitRaw string      `mapstructure:"wait" json:""`
}

// Merge merges the values in config into this config object. Values in the
// config object overwrite the values in c.
func (c *Config) Merge(config *Config) {
	if config.Consul != "" {
		c.Consul = config.Consul
	}

	if config.SSL {
		c.SSL = true
	}

	if config.SSLNoVerify {
		c.SSLNoVerify = true
	}

	if config.Auth != nil {
		if config.Auth.Username != "" || config.Auth.Password != "" {
			c.Auth = &Auth{
				Username: config.Auth.Username,
				Password: config.Auth.Password,
			}
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
				Source:      template.Source,
				Destination: template.Destination,
				Command:     template.Command,
			})
		}
	}

	if config.Retry != 0 {
		c.Retry = config.Retry
		c.RetryRaw = config.RetryRaw
	}

	if config.Token != "" {
		c.Token = config.Token
	}

	if config.Wait != nil {
		c.Wait = &watch.Wait{
			Min: config.Wait.Min,
			Max: config.Wait.Max,
		}
		c.WaitRaw = config.WaitRaw
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
	return &Config{
		Retry: 5 * time.Second,
	}
}

// Auth is the HTTP basic authentication data.
type Auth struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// ConfigTemplate is the representation of an input template, output location,
// and optional command to execute when rendered
type ConfigTemplate struct {
	Source      string `mapstructure:"source"`
	Destination string `mapstructure:"destination"`
	Command     string `mapstructure:"command"`
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

	return &ConfigTemplate{source, destination, command}, nil
}
