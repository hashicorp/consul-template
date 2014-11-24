package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	util "github.com/hashicorp/consul-template/util"
	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
)

// Config is used to configure Consul Template
type Config struct {
	// Path is the path to this configuration file on disk. This value is not
	// read from disk by rather dynamically populated by the code so the Config
	// has a reference to the path to the file on disk that created it.
	Path string `mapstructure:"-"`

	// Consul is the location of the Consul instance to query (may be an IP
	// address or FQDN) with port.
	Consul string `mapstructure:"consul"`

	// ConfigTemplates is a slice of the ConfigTemplate objects in the config.
	ConfigTemplates []*ConfigTemplate `mapstructure:"template"`

	// Token is the Consul API token.
	Token string `mapstructure:"token"`

	// Wait
	Wait    *util.Wait `mapstructure:"-"`
	WaitRaw string     `mapstructure:"wait" json:""`
}

// Merge merges the values in config into this config object. Values in the
// config object overwrite the values in c.
func (c *Config) Merge(config *Config) {
	if config.Consul != "" {
		c.Consul = config.Consul
	}

	if len(config.ConfigTemplates) > 0 {
		c.ConfigTemplates = make([]*ConfigTemplate, 0, 1)
		for _, template := range config.ConfigTemplates {
			c.ConfigTemplates = append(c.ConfigTemplates, &ConfigTemplate{
				Source:      template.Source,
				Destination: template.Destination,
				Command:     template.Command,
			})
		}
	}

	if config.Token != "" {
		c.Token = config.Token
	}

	if config.Wait != nil {
		c.Wait = &util.Wait{
			Min: config.Wait.Min,
			Max: config.Wait.Max,
		}
		c.WaitRaw = config.WaitRaw
	}
}

// ParseConfig reads the configuration file at the given path and returns a new
// Config struct with the data populated.
func ParseConfig(path string) (*Config, error) {
	errs := NewErrorList("parsing the config")

	// Read the contents of the file
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		errs.Append(err)
		return nil, errs.GetError()
	}

	// Parse the file (could be HCL or JSON)
	var parsed interface{}
	if err := hcl.Decode(&parsed, string(contents)); err != nil {
		errs.Append(err)
		return nil, errs.GetError()
	}

	// Create a new, empty config
	config := &Config{}

	// Use mapstructure to populate the basic config fields
	if err := mapstructure.Decode(parsed, config); err != nil {
		errs.Append(err)
		return nil, errs.GetError()
	}

	// Store a reference to the path where this config was read from
	config.Path = path

	// Parse the Wait component
	if raw := config.WaitRaw; raw != "" {
		wait, err := util.ParseWait(raw)

		if err == nil {
			config.Wait = wait
		} else {
			errs.Append(fmt.Errorf("wait invalid: %v", err))
		}
	}

	return config, errs.GetError()
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
	parts := strings.Split(s, ":")

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
