package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl"
)

// The pattern to split the config template syntax on
var configTemplateRe = regexp.MustCompile("([a-zA-Z]:)?([^:]+)")

// Config is used to configure Consul Template
type Config struct {
	// Path is the path to this configuration file on disk. This value is not
	// read from disk by rather dynamically populated by the code so the Config
	// has a reference to the path to the file on disk that created it.
	Path string `json:"path"`

	// Consul is the location of the Consul instance to query (may be an IP
	// address or FQDN) with port.
	Consul string `json:"consul"`

	// Token is the Consul API token.
	Token string `json:"-"`

	// Vault is the configuration for connecting to a vault server.
	Vault *VaultConfig `json:"vault"`

	// Auth is the HTTP basic authentication for communicating with Consul.
	Auth *AuthConfig `json:"auth"`

	// SSL indicates we should use a secure connection while talking to
	// Consul. This requires Consul to be configured to serve HTTPS.
	SSL *SSLConfig `json:"ssl"`

	// Syslog is the configuration for syslog.
	Syslog *SyslogConfig `json:"syslog"`

	// MaxStale is the maximum amount of time for staleness from Consul as given
	// by LastContact. If supplied, Consul Template will query all servers instead
	// of just the leader.
	MaxStale time.Duration `json:"max_stale"`

	// Retry is the duration of time to wait between Consul failures.
	Retry time.Duration `json:"retry"`

	// Wait is the quiescence timers.
	Wait *watch.Wait `json:"wait"`

	// LogLevel is the level with which to log for this config.
	LogLevel string `json:"log_level"`

	// ConfigTemplates is a slice of the ConfigTemplate objects in the config.
	ConfigTemplates []*ConfigTemplate `json:"templates"`
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

	if config.Vault != nil {
		if c.Vault == nil {
			c.Vault = &VaultConfig{}
		}

		if config.Vault.Address != "" {
			c.Vault.Address = config.Vault.Address
		}

		if config.Vault.Token != "" {
			c.Vault.Token = config.Vault.Token
		}

		if config.Vault.SSL != nil {
			if c.Vault.SSL == nil {
				c.Vault.SSL = &SSLConfig{}
			}

			if config.Vault.SSL.Enabled != BoolUnset {
				c.Vault.SSL.Enabled = config.Vault.SSL.Enabled
			}

			if config.Vault.SSL.Verify != BoolUnset {
				c.Vault.SSL.Verify = config.Vault.SSL.Verify
			}

			if config.Vault.SSL.Cert != "" {
				c.Vault.SSL.Cert = config.Vault.SSL.Cert
			}

			if config.Vault.SSL.CaCert != "" {
				c.Vault.SSL.CaCert = config.Vault.SSL.CaCert
			}
		}
	}

	if config.Auth != nil {
		if c.Auth == nil {
			c.Auth = &AuthConfig{}
		}

		if config.Auth.Enabled != BoolUnset {
			c.Auth.Enabled = config.Auth.Enabled
		}

		if config.Auth.Username != "" {
			c.Auth.Username = config.Auth.Username
		}

		if config.Auth.Password != "" {
			c.Auth.Password = config.Auth.Password
		}
	}

	if config.SSL != nil {
		if c.SSL == nil {
			c.SSL = &SSLConfig{}
		}

		if config.SSL.Enabled != BoolUnset {
			c.SSL.Enabled = config.SSL.Enabled
		}

		if config.SSL.Verify != BoolUnset {
			c.SSL.Verify = config.SSL.Verify
		}

		if config.SSL.Cert != "" {
			c.SSL.Cert = config.SSL.Cert
		}

		if config.SSL.CaCert != "" {
			c.SSL.CaCert = config.SSL.CaCert
		}
	}

	if config.Syslog != nil {
		if c.Syslog == nil {
			c.Syslog = &SyslogConfig{}
		}

		if config.Syslog.Enabled != BoolUnset {
			c.Syslog.Enabled = config.Syslog.Enabled
		}

		if config.Syslog.Facility != "" {
			c.Syslog.Facility = config.Syslog.Facility
		}
	}

	if config.MaxStale != 0 {
		c.MaxStale = config.MaxStale
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
	}

	if config.Wait != nil {
		c.Wait = &watch.Wait{
			Min: config.Wait.Min,
			Max: config.Wait.Max,
		}
	}

	if config.LogLevel != "" {
		c.LogLevel = config.LogLevel
	}
}

// ParseConfig reads the configuration file at the given path and returns a new
// Config struct with the data populated.
func ParseConfig(path string) (*Config, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: error opening %q: %s", path, err)
	}

	var shadow interface{}
	if err := hcl.Decode(&shadow, string(contents)); err != nil {
		return nil, fmt.Errorf("config: error decoding %q: %s", path, err)
	}

	parsed, ok := shadow.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config: invalid format")
	}

	config := &Config{Path: path}
	var errs *multierror.Error

	if raw, ok := parsed["consul"]; ok {
		delete(parsed, "consul")
		config.Consul, ok = raw.(string)
		if !ok {
			err := fmt.Errorf("consul: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := parsed["token"]; ok {
		delete(parsed, "token")
		config.Token, ok = raw.(string)
		if !ok {
			err := fmt.Errorf("token: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := parsed["vault"]; ok {
		delete(parsed, "vault")

		typed, ok := raw.([]map[string]interface{})
		if !ok {
			err := fmt.Errorf("vault: cannot convert %T to map[string]interface{}", raw)
			errs = multierror.Append(errs, err)
		}

		if last := lastConfig(typed); last != nil {
			if vault, err := DecodeVaultConfig(last); err == nil {
				config.Vault = vault
			} else {
				errs = multierror.Append(errs, err.(*multierror.Error).Errors...)
			}
		}
	}

	if raw, ok := parsed["auth"]; ok {
		delete(parsed, "auth")

		typed, ok := raw.([]map[string]interface{})
		if !ok {
			err := fmt.Errorf("auth: cannot convert %T to map[string]interface{}", raw)
			errs = multierror.Append(errs, err)
		}

		if last := lastConfig(typed); last != nil {
			if auth, err := DecodeAuthConfig(last); err == nil {
				config.Auth = auth
			} else {
				errs = multierror.Append(errs, err.(*multierror.Error).Errors...)
			}
		}
	}

	if raw, ok := parsed["ssl"]; ok {
		delete(parsed, "ssl")

		typed, ok := raw.([]map[string]interface{})
		if !ok {
			err := fmt.Errorf("ssl: cannot convert %T to map[string]interface{}", raw)
			errs = multierror.Append(errs, err)
		}

		if last := lastConfig(typed); last != nil {
			if ssl, err := DecodeSSLConfig(last); err == nil {
				config.SSL = ssl
			} else {
				errs = multierror.Append(errs, err.(*multierror.Error).Errors...)
			}
		}
	}

	if raw, ok := parsed["syslog"]; ok {
		delete(parsed, "syslog")

		typed, ok := raw.([]map[string]interface{})
		if !ok {
			err := fmt.Errorf("syslog: cannot convert %T to map[string]interface{}", raw)
			errs = multierror.Append(errs, err)
		}

		if last := lastConfig(typed); last != nil {
			if syslog, err := DecodeSyslogConfig(last); err == nil {
				config.Syslog = syslog
			} else {
				errs = multierror.Append(errs, err.(*multierror.Error).Errors...)
			}
		}
	}

	if raw, ok := parsed["max_stale"]; ok {
		delete(parsed, "max_stale")

		typed, ok := raw.(string)
		if !ok {
			err := fmt.Errorf("max_stale: cannot covnert %T to string", raw)
			errs = multierror.Append(errs, err)
		}

		stale, err := time.ParseDuration(typed)
		if err == nil {
			config.MaxStale = stale
		} else {
			errs = multierror.Append(errs, fmt.Errorf("max_stale invalid: %s", err))
		}
	}

	if raw, ok := parsed["retry"]; ok {
		delete(parsed, "retry")

		typed, ok := raw.(string)
		if !ok {
			err := fmt.Errorf("retry: cannot covnert %T to string", raw)
			errs = multierror.Append(errs, err)
		}

		retry, err := time.ParseDuration(typed)
		if err == nil {
			config.Retry = retry
		} else {
			errs = multierror.Append(errs, fmt.Errorf("retry invalid: %v", err))
		}
	}

	if raw, ok := parsed["wait"]; ok {
		delete(parsed, "wait")

		typed, ok := raw.(string)
		if !ok {
			err := fmt.Errorf("wait: cannot covnert %T to string", raw)
			errs = multierror.Append(errs, err)
		}

		wait, err := watch.ParseWait(typed)
		if err == nil {
			config.Wait = wait
		} else {
			errs = multierror.Append(errs, fmt.Errorf("wait invalid: %s", err))
		}
	}

	if raw, ok := parsed["log_level"]; ok {
		delete(parsed, "log_level")
		config.LogLevel, ok = raw.(string)
		if !ok {
			err := fmt.Errorf("log_level: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := parsed["template"]; ok {
		delete(parsed, "template")
		typed, ok := raw.([]map[string]interface{})
		if !ok {
			err := fmt.Errorf("template: cannot convert %T to []map[string]interface{}", raw)
			errs = multierror.Append(errs, err)
		}

		if len(config.ConfigTemplates) == 0 {
			config.ConfigTemplates = make([]*ConfigTemplate, 0, len(typed))
		}

		for _, data := range typed {
			if template, err := DecodeConfigTemplate(data); err == nil {
				config.ConfigTemplates = append(config.ConfigTemplates, template)
			} else {
				errs = multierror.Append(errs, err.(*multierror.Error).Errors...)
			}
		}
	}

	if len(parsed) > 0 {
		err := fmt.Errorf("config: unknown field(s): %s", keysToList(parsed))
		errs = multierror.Append(errs, err)
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
		Vault: &VaultConfig{
			SSL: &SSLConfig{
				Enabled: BoolTrue,
				Verify:  BoolTrue,
			},
		},
		Auth: &AuthConfig{
			Enabled: BoolFalse,
		},
		SSL: &SSLConfig{
			Enabled: BoolFalse,
			Verify:  BoolTrue,
		},
		Syslog: &SyslogConfig{
			Enabled:  BoolFalse,
			Facility: "LOCAL0",
		},
		ConfigTemplates: []*ConfigTemplate{},
		Retry:           5 * time.Second,
		Wait:            &watch.Wait{},
		LogLevel:        logLevel,
	}
}

// EmptyConfig is a blank config that has all the associations setup.
func EmptyConfig() *Config {
	logLevel := os.Getenv("CONSUL_TEMPLATE_LOG")
	if logLevel == "" {
		logLevel = "WARN"
	}

	return &Config{
		Vault: &VaultConfig{
			SSL: &SSLConfig{},
		},
		Auth:            &AuthConfig{},
		SSL:             &SSLConfig{},
		Syslog:          &SyslogConfig{},
		ConfigTemplates: []*ConfigTemplate{},
		Retry:           0 * time.Second,
		Wait:            &watch.Wait{},
	}
}

// AuthConfig is the HTTP basic authentication data.
type AuthConfig struct {
	Enabled  Bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// DecodeAuthConfig decodes the given map into an AuthConfig.
func DecodeAuthConfig(data map[string]interface{}) (*AuthConfig, error) {
	auth := &AuthConfig{}
	var errs *multierror.Error

	if raw, ok := data["username"]; ok {
		delete(data, "username")
		if typed, ok := raw.(string); ok {
			auth.Enabled = BoolTrue
			auth.Username = typed
		} else {
			err := fmt.Errorf("auth: username: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := data["password"]; ok {
		delete(data, "password")
		if typed, ok := raw.(string); ok {
			auth.Enabled = BoolTrue
			auth.Password = typed
		} else {
			err := fmt.Errorf("auth: password: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := data["enabled"]; ok {
		delete(data, "enabled")
		if typed, ok := raw.(bool); ok {
			if typed == true {
				auth.Enabled = BoolTrue
			} else {
				auth.Enabled = BoolFalse
			}
		} else {
			err := fmt.Errorf("auth: enabled: cannot convert %T to bool", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if len(data) > 0 {
		err := fmt.Errorf("auth: unknown field(s): %s", keysToList(data))
		errs = multierror.Append(errs, err)
	}

	return auth, errs.ErrorOrNil()
}

// String is the string representation of this authentication. If authentication
// is not enabled, this returns the empty string. The username and password will
// be separated by a colon.
func (a *AuthConfig) String() string {
	if a.Enabled != BoolTrue {
		return ""
	}

	if a.Password != "" {
		return fmt.Sprintf("%s:%s", a.Username, a.Password)
	}

	return a.Username
}

// SSLConfig is the configuration for SSL.
type SSLConfig struct {
	Enabled Bool   `json:"enabled"`
	Verify  Bool   `json:"verify"`
	Cert    string `json:"cert"`
	CaCert  string `json:"ca_cert"`
}

// DecodeSSLConfig decodes the given map into an SSLConfig.
func DecodeSSLConfig(data map[string]interface{}) (*SSLConfig, error) {
	ssl := &SSLConfig{}
	var errs *multierror.Error

	if raw, ok := data["cert"]; ok {
		delete(data, "cert")
		if typed, ok := raw.(string); ok {
			ssl.Cert = typed
			ssl.Enabled = BoolTrue
		} else {
			err := fmt.Errorf("ssl: cert: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := data["ca_cert"]; ok {
		delete(data, "ca_cert")
		if typed, ok := raw.(string); ok {
			ssl.CaCert = typed
			ssl.Enabled = BoolTrue
		} else {
			err := fmt.Errorf("ssl: ca_cert: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := data["verify"]; ok {
		delete(data, "verify")
		if typed, ok := raw.(bool); ok {
			if typed == true {
				ssl.Verify = BoolTrue
				ssl.Enabled = BoolTrue
			} else {
				ssl.Verify = BoolFalse
				ssl.Enabled = BoolTrue
			}
		} else {
			err := fmt.Errorf("ssl: verify: cannot convert %T to bool", raw)
			errs = multierror.Append(err)
		}
	}

	if raw, ok := data["enabled"]; ok {
		delete(data, "enabled")
		if typed, ok := raw.(bool); ok {
			if typed == true {
				ssl.Enabled = BoolTrue
			} else {
				ssl.Enabled = BoolFalse
			}
		} else {
			err := fmt.Errorf("ssl: enabled: cannot convert %T to bool", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if len(data) > 0 {
		err := fmt.Errorf("ssl: unknown field(s): %s", keysToList(data))
		errs = multierror.Append(err)
	}

	return ssl, errs.ErrorOrNil()
}

// SyslogConfig is the configuration for syslog.
type SyslogConfig struct {
	Enabled  Bool   `json:"enabled"`
	Facility string `json:"facility"`
}

// DecodeSyslogConfig decodes the given map into an SyslogConfig.
func DecodeSyslogConfig(data map[string]interface{}) (*SyslogConfig, error) {
	syslog := &SyslogConfig{}
	var errs *multierror.Error

	if raw, ok := data["facility"]; ok {
		delete(data, "facility")
		if typed, ok := raw.(string); ok {
			syslog.Enabled = BoolTrue
			syslog.Facility = typed
		} else {
			err := fmt.Errorf("syslog: facility: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := data["enabled"]; ok {
		delete(data, "enabled")
		if typed, ok := raw.(bool); ok {
			if typed == true {
				syslog.Enabled = BoolTrue
			} else {
				syslog.Enabled = BoolFalse
			}
		} else {
			err := fmt.Errorf("syslog: enabled: cannot convert %T to bool", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if len(data) > 0 {
		err := fmt.Errorf("syslog: unknown field(s): %s", keysToList(data))
		errs = multierror.Append(errs, err)
	}

	return syslog, errs.ErrorOrNil()
}

// ConfigTemplate is the representation of an input template, output location,
// and optional command to execute when rendered
type ConfigTemplate struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Command     string `json:"command"`
}

// DecodeConfigTemplate decodes the given map into an ConfigTemplate.
func DecodeConfigTemplate(data map[string]interface{}) (*ConfigTemplate, error) {
	template := &ConfigTemplate{}
	var errs *multierror.Error

	if raw, ok := data["source"]; ok {
		delete(data, "source")
		if typed, ok := raw.(string); ok {
			template.Source = typed
		} else {
			err := fmt.Errorf("template: source: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := data["destination"]; ok {
		delete(data, "destination")
		if typed, ok := raw.(string); ok {
			template.Destination = typed
		} else {
			err := fmt.Errorf("template: destination: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := data["command"]; ok {
		delete(data, "command")
		if typed, ok := raw.(string); ok {
			template.Command = typed
		} else {
			err := fmt.Errorf("template: command: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if template.Source == "" {
		err := fmt.Errorf("template: missing source")
		errs = multierror.Append(errs, err)
	}

	if len(data) > 0 {
		err := fmt.Errorf("template: unknown field(s): %s", keysToList(data))
		errs = multierror.Append(errs, err)
	}

	return template, errs.ErrorOrNil()
}

// VaultConfig is the configuration for connecting to a vault server.
type VaultConfig struct {
	Address string `json:"address"`
	Token   string `json:"-"`

	// SSL indicates we should use a secure connection while talking to Vault.
	SSL *SSLConfig `json:"ssl"`
}

// DecodeVaultConfig decodes the given map into an VaultConfig.
func DecodeVaultConfig(data map[string]interface{}) (*VaultConfig, error) {
	vault := &VaultConfig{}
	var errs *multierror.Error

	if raw, ok := data["address"]; ok {
		delete(data, "address")
		if typed, ok := raw.(string); ok {
			vault.Address = typed
		} else {
			err := fmt.Errorf("vault: address: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := data["token"]; ok {
		delete(data, "token")
		if typed, ok := raw.(string); ok {
			vault.Token = typed
		} else {
			err := fmt.Errorf("vault: token: cannot convert %T to string", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if raw, ok := data["ssl"]; ok {
		delete(data, "ssl")
		if typed, ok := raw.([]map[string]interface{}); ok {
			if last := lastConfig(typed); last != nil {
				if ssl, err := DecodeSSLConfig(last); err == nil {
					vault.SSL = ssl
				} else {
					for _, e := range err.(*multierror.Error).Errors {
						errs = multierror.Append(errs, fmt.Errorf("vault: %s", e))
					}
				}
			}
		} else {
			err := fmt.Errorf("vault: ssl: cannot convert %T to []map[string]interface{}", raw)
			errs = multierror.Append(errs, err)
		}
	}

	if len(data) > 0 {
		err := fmt.Errorf("vault: unknown field(s): %s", keysToList(data))
		errs = multierror.Append(errs, err)
	}

	return vault, errs.ErrorOrNil()
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

// lastConfig gets the last item in the map for config purposes. If the map is
// empty or nil, nil is returned.
func lastConfig(list []map[string]interface{}) map[string]interface{} {
	if len(list) == 0 {
		return nil
	}
	return list[len(list)-1]
}

// keysToList gets the list of keys from the given map and converts them to a
// comma-separated list.
func keysToList(data map[string]interface{}) string {
	list := make([]string, 0, len(data))
	for k, _ := range data {
		list = append(list, fmt.Sprintf("%q", k))
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}
