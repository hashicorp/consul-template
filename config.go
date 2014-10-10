package main

import (
	"fmt"
	"io/ioutil"

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
	Wait    *Wait  `mapstructure:"-"`
	WaitRaw string `mapstructure:"wait" json:""`

	// Once runs once and exit as opposed to the default behavior of starting a
	// daemon.
	Once bool `mapstructure:"once"`
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
		wait, err := ParseWait(raw)

		if err == nil {
			config.Wait = wait
		} else {
			errs.Append(fmt.Errorf("Wait invalid: %v", err))
		}
	}

	return config, errs.GetError()
}

// ConfigTemplate is the representation of an input template, output location, and
// optional command to execute when rendered
type ConfigTemplate struct {
	Source      string `mapstructure:"source"`
	Destination string `mapstructure:"destination"`
	Command     string `mapstructure:"command"`
}
