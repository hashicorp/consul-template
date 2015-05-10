package main

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/watch"
)

func TestMerge_emptyConfig(t *testing.T) {
	config := DefaultConfig()
	config.Merge(&Config{})

	expected := DefaultConfig()
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \b\b%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_ignoresPath(t *testing.T) {
	config := &Config{Path: "/path/1"}
	config.Merge(&Config{Path: "/path/2"})

	expected := "/path/1"
	if config.Path != expected {
		t.Errorf("expected %q to be %q", config.Path, expected)
	}
}

func TestMerge_topLevel(t *testing.T) {
	config1 := &Config{
		Consul:   "consul-1",
		Token:    "token-1",
		MaxStale: 1 * time.Second,
		Retry:    1 * time.Second,
		LogLevel: "log_level-1",
	}
	config2 := &Config{
		Consul:   "consul-2",
		Token:    "token-2",
		MaxStale: 2 * time.Second,
		Retry:    2 * time.Second,
		LogLevel: "log_level-2",
	}
	config1.Merge(config2)

	if !reflect.DeepEqual(config1, config2) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config1, config2)
	}
}

func TestMerge_vault(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "1.1.1.1",
			Token:   "1",
		},
	}
	config.Merge(&Config{
		Vault: &VaultConfig{
			Address: "2.2.2.2",
		},
	})

	expected := &Config{
		Vault: &VaultConfig{
			Address: "2.2.2.2",
			Token:   "1",
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_vaultSSL(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			SSL: &SSLConfig{
				Enabled: BoolTrue,
				Verify:  BoolTrue,
				Cert:    "1.pem",
				CaCert:  "ca-1.pem",
			},
		},
	}
	config.Merge(&Config{
		Vault: &VaultConfig{
			SSL: &SSLConfig{
				Enabled: BoolFalse,
			},
		},
	})

	expected := &Config{
		Vault: &VaultConfig{
			SSL: &SSLConfig{
				Enabled: BoolFalse,
				Verify:  BoolTrue,
				Cert:    "1.pem",
				CaCert:  "ca-1.pem",
			},
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_auth(t *testing.T) {
	config := &Config{
		Auth: &AuthConfig{
			Enabled:  BoolTrue,
			Username: "1",
			Password: "1",
		},
	}
	config.Merge(&Config{
		Auth: &AuthConfig{
			Password: "2",
		},
	})

	expected := &Config{
		Auth: &AuthConfig{
			Enabled:  BoolTrue,
			Username: "1",
			Password: "2",
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_SSL(t *testing.T) {
	config := &Config{
		SSL: &SSLConfig{
			Enabled: BoolTrue,
			Verify:  BoolTrue,
			Cert:    "1.pem",
			CaCert:  "ca-1.pem",
		},
	}
	config.Merge(&Config{
		SSL: &SSLConfig{
			Enabled: BoolFalse,
		},
	})

	expected := &Config{
		SSL: &SSLConfig{
			Enabled: BoolFalse,
			Verify:  BoolTrue,
			Cert:    "1.pem",
			CaCert:  "ca-1.pem",
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_syslog(t *testing.T) {
	config := &Config{
		Syslog: &SyslogConfig{
			Enabled:  BoolTrue,
			Facility: "1",
		},
	}
	config.Merge(&Config{
		Syslog: &SyslogConfig{
			Facility: "2",
		},
	})

	expected := &Config{
		Syslog: &SyslogConfig{
			Enabled:  BoolTrue,
			Facility: "2",
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_configTemplates(t *testing.T) {
	config := &Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:      "1",
				Destination: "1",
				Command:     "1",
			},
		},
	}
	config.Merge(&Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:      "2",
				Destination: "2",
				Command:     "2",
			},
		},
	})

	expected := &Config{
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:      "1",
				Destination: "1",
				Command:     "1",
			},
			&ConfigTemplate{
				Source:      "2",
				Destination: "2",
				Command:     "2",
			},
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_wait(t *testing.T) {
	config := &Config{
		Wait: &watch.Wait{
			Min: 10 * time.Second,
			Max: 20 * time.Second,
		},
	}
	config.Merge(&Config{
		Wait: &watch.Wait{
			Min: 40 * time.Second,
		},
	})

	expected := &Config{
		Wait: &watch.Wait{
			Min: 40 * time.Second,
			Max: 0 * time.Second, // Verify a full overwrite
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestParseConfig_readFileError(t *testing.T) {
	_, err := ParseConfig(path.Join(os.TempDir(), "config.json"))
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "no such file or directory"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_parseFileError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    invalid file in here
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "syntax error"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestParseConfig_consul(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		consul = "testing"
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Consul != expected {
		t.Errorf("expected %q to be %q", config.Consul, expected)
	}
}

func TestParseConfig_consulBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		consul = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "consul: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_token(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		token = "testing"
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Token != expected {
		t.Errorf("expected %q to be %q", config.Token, expected)
	}
}

func TestParseConfig_tokenBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		token = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "token: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_vault(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		vault {
			address = "testing"
		}
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Vault.Address != expected {
		t.Errorf("expected %q to be %q", config.Vault.Address, expected)
	}
}

func TestParseConfig_vaultMultiple(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		vault {
			address = "one"
		}

		vault {
			address = "two"
		}
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "two"
	if config.Vault.Address != expected {
		t.Errorf("expected %q to be %q", config.Vault.Address, expected)
	}
}

func TestParseConfig_vaultSSL(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		vault {
			address = "testing"

			ssl {
				enabled = true
				cert = "custom.pem"
			}
		}
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Vault.Address != expected {
		t.Errorf("expected %q to be %q", config.Vault.Address, expected)
	}

	if config.Vault.SSL.Enabled != BoolTrue {
		t.Errorf("expected vault ssl to be enabled")
	}

	expected = "custom.pem"
	if config.Vault.SSL.Cert != expected {
		t.Errorf("expected %q to be %q", config.Vault.SSL.Cert, expected)
	}
}

func TestParseConfig_vaultSSLMultiple(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		vault {
			address = "testing"

			ssl {
				enabled = false
				cert = "one.pem"
			}

			ssl {
				enabled = true
				cert = "two.pem"
			}
		}
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Vault.Address != expected {
		t.Errorf("expected %q to be %q", config.Vault.Address, expected)
	}

	if config.Vault.SSL.Enabled != BoolTrue {
		t.Errorf("expected vault ssl to be enabled")
	}

	expected = "two.pem"
	if config.Vault.SSL.Cert != expected {
		t.Errorf("expected %q to be %q", config.Vault.SSL.Cert, expected)
	}
}

func TestParseConfig_vaultSSLBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		vault {
			ssl = true
		}
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "vault: ssl: cannot convert bool to []map[string]interface{}"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_vaultBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		vault = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "vault: cannot convert bool to map[string]interface{}"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_auth(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		auth {
			username = "testing"
		}
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Auth.Username != expected {
		t.Errorf("expected %q to be %q", config.Auth.Username, expected)
	}
}

func TestParseConfig_authBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		auth = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "auth: cannot convert bool to map[string]interface{}"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_ssl(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		ssl {
			enabled = true
		}
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	if config.SSL.Enabled != BoolTrue {
		t.Errorf("expected ssl to be enabled")
	}
}

func TestParseConfig_sslBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		ssl = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "ssl: cannot convert bool to map[string]interface{}"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_syslog(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		syslog {
			facility = "testing"
		}
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Syslog.Facility != expected {
		t.Errorf("expected %q to be %q", config.Syslog.Facility, expected)
	}
}

func TestParseConfig_syslogBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		syslog = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "syslog: cannot convert bool to map[string]interface{}"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_maxstale(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		max_stale = "10s"
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := 10 * time.Second
	if config.MaxStale != expected {
		t.Errorf("expected %q to be %q", config.MaxStale, expected)
	}
}

func TestParseConfig_maxstaleBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		max_stale = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "max_stale: cannot covnert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_retry(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		retry = "10s"
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := 10 * time.Second
	if config.Retry != expected {
		t.Errorf("expected %q to be %q", config.Retry, expected)
	}
}

func TestParseConfig_retryBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		retry = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "retry: cannot covnert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_retryParseError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    retry = "bacon pants"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "retry invalid"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseConfig_wait(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		wait = "10s:20s"
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := &watch.Wait{Min: 10 * time.Second, Max: 20 * time.Second}
	if !reflect.DeepEqual(config.Wait, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Wait, expected)
	}
}

func TestParseConfig_waitBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		wait = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "wait: cannot covnert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_waitParseError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    wait = "not_valid:duration"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "wait invalid"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseConfig_loglevel(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		log_level = "testing"
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.LogLevel != expected {
		t.Errorf("expected %q to be %q", config.LogLevel, expected)
	}
}

func TestParseConfig_loglevelBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		log_level = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "log_level: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_template(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		template {
			source = "testing"
		}
	`), t)
	defer test.DeleteTempfile(file, t)

	config, err := ParseConfig(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.ConfigTemplates[0].Source != expected {
		t.Errorf("expected %q to be %q", config.ConfigTemplates[0].Source, expected)
	}
}

func TestParseConfig_templateBadType(t *testing.T) {
	file := test.CreateTempfile([]byte(`
		template = true
	`), t)
	defer test.DeleteTempfile(file, t)

	_, err := ParseConfig(file.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "template: cannot convert bool to []map[string]interface{}"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_extraKeys(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
		foo = "nope"
		bar = "never"
	`), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error")
	}

	expected := `config: unknown field(s): "bar", "foo"`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestParseConfig_correctValues(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    consul = "nyc1.demo.consul.io"
    max_stale = "5s"
    token = "abcd1234"
    wait = "5s:10s"
    retry = "10s"
    log_level = "warn"

    vault {
			address = "vault.service.consul"
			token = "efgh5678"
			ssl {
				enabled = false
			}
    }

    auth {
    	enabled = true
    	username = "test"
    	password = "test"
    }

    ssl {
			enabled = true
			verify = false
			cert = "c1.pem"
			ca_cert = "c2.pem"
    }

    syslog {
    	enabled = true
    	facility = "LOCAL5"
    }

    template {
      source = "nginx.conf.ctmpl"
      destination  = "/etc/nginx/nginx.conf"
    }

    template {
      source = "redis.conf.ctmpl"
      destination  = "/etc/redis/redis.conf"
      command = "service redis restart"
    }
  `), t)
	defer test.DeleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := &Config{
		Path:     configFile.Name(),
		Consul:   "nyc1.demo.consul.io",
		MaxStale: time.Second * 5,
		Vault: &VaultConfig{
			Address: "vault.service.consul",
			Token:   "efgh5678",
			SSL: &SSLConfig{
				Enabled: BoolFalse,
				Verify:  BoolUnset,
				Cert:    "",
				CaCert:  "",
			},
		},
		Auth: &AuthConfig{
			Enabled:  BoolTrue,
			Username: "test",
			Password: "test",
		},
		SSL: &SSLConfig{
			Enabled: BoolTrue,
			Verify:  BoolFalse,
			Cert:    "c1.pem",
			CaCert:  "c2.pem",
		},
		Syslog: &SyslogConfig{
			Enabled:  BoolTrue,
			Facility: "LOCAL5",
		},
		Token: "abcd1234",
		Wait: &watch.Wait{
			Min: time.Second * 5,
			Max: time.Second * 10,
		},
		Retry:    10 * time.Second,
		LogLevel: "warn",
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:      "nginx.conf.ctmpl",
				Destination: "/etc/nginx/nginx.conf",
			},
			&ConfigTemplate{
				Source:      "redis.conf.ctmpl",
				Destination: "/etc/redis/redis.conf",
				Command:     "service redis restart",
			},
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("expected \n%#v\n to be \n%#v\n", config, expected)
	}
}

func TestDecodeAuthConfig_nil(t *testing.T) {
	config, err := DecodeAuthConfig(nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := &AuthConfig{
		Enabled:  BoolUnset,
		Username: "",
		Password: "",
	}
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestDecodeAuthConfig_username(t *testing.T) {
	config, err := DecodeAuthConfig(map[string]interface{}{
		"username": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Username != expected {
		t.Errorf("expected %q to be %q", config.Username, expected)
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeAuthConfig_usernameBadType(t *testing.T) {
	_, err := DecodeAuthConfig(map[string]interface{}{
		"username": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "auth: username: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeAuthConfig_password(t *testing.T) {
	config, err := DecodeAuthConfig(map[string]interface{}{
		"password": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Password != expected {
		t.Errorf("expected %q to be %q", config.Password, expected)
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeAuthConfig_passwordBadType(t *testing.T) {
	_, err := DecodeAuthConfig(map[string]interface{}{
		"password": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "auth: password: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeAuthConfig_enabledTrue(t *testing.T) {
	config, err := DecodeAuthConfig(map[string]interface{}{
		"enabled": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeAuthConfig_enabledFalse(t *testing.T) {
	config, err := DecodeAuthConfig(map[string]interface{}{
		"enabled": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Enabled != BoolFalse {
		t.Errorf("expected auth to be disabled")
	}
}

func TestDecodeAuthConfig_enabledBadType(t *testing.T) {
	_, err := DecodeAuthConfig(map[string]interface{}{
		"enabled": 1,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "auth: enabled: cannot convert int to bool"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeAuthConfig_extraFields(t *testing.T) {
	_, err := DecodeAuthConfig(map[string]interface{}{
		"username": "",
		"foo":      "",
		"bar":      "",
	})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `auth: unknown field(s): "bar", "foo"`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestAuthString_disabled(t *testing.T) {
	a := &AuthConfig{Enabled: BoolFalse}
	expected := ""
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabledNoPassword(t *testing.T) {
	a := &AuthConfig{Enabled: BoolTrue, Username: "username"}
	expected := "username"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabled(t *testing.T) {
	a := &AuthConfig{Enabled: BoolTrue, Username: "username", Password: "password"}
	expected := "username:password"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestDecodeSSLConfig_nil(t *testing.T) {
	config, err := DecodeSSLConfig(nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := &SSLConfig{
		Enabled: BoolUnset,
		Verify:  BoolUnset,
		Cert:    "",
		CaCert:  "",
	}
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestDecodeSSLConfig_cert(t *testing.T) {
	config, err := DecodeSSLConfig(map[string]interface{}{
		"cert": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Cert != expected {
		t.Errorf("expected %q to be %q", config.Cert, expected)
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeSSLConfig_certBadType(t *testing.T) {
	_, err := DecodeSSLConfig(map[string]interface{}{
		"cert": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "ssl: cert: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeSSLConfig_ca_cert(t *testing.T) {
	config, err := DecodeSSLConfig(map[string]interface{}{
		"ca_cert": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.CaCert != expected {
		t.Errorf("expected %q to be %q", config.CaCert, expected)
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeSSLConfig_ca_certBadType(t *testing.T) {
	_, err := DecodeSSLConfig(map[string]interface{}{
		"ca_cert": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "ssl: ca_cert: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeSSLConfig_verifyTrue(t *testing.T) {
	config, err := DecodeSSLConfig(map[string]interface{}{
		"verify": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Verify != BoolTrue {
		t.Errorf("expected ssl to verify")
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeSSLConfig_verifyFalse(t *testing.T) {
	config, err := DecodeSSLConfig(map[string]interface{}{
		"verify": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Verify != BoolFalse {
		t.Errorf("expected ssl to not verify")
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeSSLConfig_verifyBadType(t *testing.T) {
	_, err := DecodeSSLConfig(map[string]interface{}{
		"verify": "testing",
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "ssl: verify: cannot convert string to bool"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeSSLConfig_enabledTrue(t *testing.T) {
	config, err := DecodeSSLConfig(map[string]interface{}{
		"enabled": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeSSLConfig_enabledFalse(t *testing.T) {
	config, err := DecodeSSLConfig(map[string]interface{}{
		"enabled": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Enabled != BoolFalse {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeSSLConfig_enabledBadType(t *testing.T) {
	_, err := DecodeSSLConfig(map[string]interface{}{
		"enabled": "testing",
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "ssl: enabled: cannot convert string to bool"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeSSLConfig_extraFields(t *testing.T) {
	_, err := DecodeSSLConfig(map[string]interface{}{
		"cert": "",
		"foo":  "",
		"bar":  "",
	})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `ssl: unknown field(s): "bar", "foo"`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeSyslogConfig_nil(t *testing.T) {
	config, err := DecodeSyslogConfig(nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := &SyslogConfig{
		Enabled:  BoolUnset,
		Facility: "",
	}
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestDecodeSyslogConfig_facility(t *testing.T) {
	config, err := DecodeSyslogConfig(map[string]interface{}{
		"facility": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Facility != expected {
		t.Errorf("expected %q to be %q", config.Facility, expected)
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeSyslogConfig_facilityBadType(t *testing.T) {
	_, err := DecodeSyslogConfig(map[string]interface{}{
		"facility": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "syslog: facility: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeSyslogConfig_enabledTrue(t *testing.T) {
	config, err := DecodeSyslogConfig(map[string]interface{}{
		"enabled": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Enabled != BoolTrue {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeSyslogConfig_enabledFalse(t *testing.T) {
	config, err := DecodeSyslogConfig(map[string]interface{}{
		"enabled": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Enabled != BoolFalse {
		t.Errorf("expected auth to be enabled")
	}
}

func TestDecodeSyslogConfig_enabledBadType(t *testing.T) {
	_, err := DecodeSyslogConfig(map[string]interface{}{
		"enabled": "testing",
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "syslog: enabled: cannot convert string to bool"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeSyslogConfig_extraFields(t *testing.T) {
	_, err := DecodeSyslogConfig(map[string]interface{}{
		"facility": "",
		"foo":      "",
		"bar":      "",
	})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `syslog: unknown field(s): "bar", "foo"`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeConfigTemplate_nil(t *testing.T) {
	_, err := DecodeConfigTemplate(nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "template: missing source"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeConfigTemplate_source(t *testing.T) {
	config, err := DecodeConfigTemplate(map[string]interface{}{
		"source": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Source != expected {
		t.Errorf("expected %q to be %q", config.Source, expected)
	}
}

func TestDecodeConfigTemplate_sourceBadType(t *testing.T) {
	_, err := DecodeConfigTemplate(map[string]interface{}{
		"source": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "template: source: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeConfigTemplate_destination(t *testing.T) {
	config, err := DecodeConfigTemplate(map[string]interface{}{
		"source":      "_",
		"destination": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Destination != expected {
		t.Errorf("expected %q to be %q", config.Destination, expected)
	}
}

func TestDecodeConfigTemplate_destinationBadType(t *testing.T) {
	_, err := DecodeConfigTemplate(map[string]interface{}{
		"source":      "_",
		"destination": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "template: destination: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeConfigTemplate_command(t *testing.T) {
	config, err := DecodeConfigTemplate(map[string]interface{}{
		"source":  "_",
		"command": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Command != expected {
		t.Errorf("expected %q to be %q", config.Command, expected)
	}
}

func TestDecodeConfigTemplate_commandBadType(t *testing.T) {
	_, err := DecodeConfigTemplate(map[string]interface{}{
		"source":  "_",
		"command": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "template: command: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeConfigTemplate_extraFields(t *testing.T) {
	_, err := DecodeConfigTemplate(map[string]interface{}{
		"source": "",
		"foo":    "",
		"bar":    "",
	})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `template: unknown field(s): "bar", "foo"`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeVaultConfig_nil(t *testing.T) {
	config, err := DecodeVaultConfig(nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := &VaultConfig{
		Address: "",
		Token:   "",
		SSL:     nil,
	}
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%v\n\n to be \n\n%v\n\n", config, expected)
	}
}

func TestDecodeVaultConfig_address(t *testing.T) {
	config, err := DecodeVaultConfig(map[string]interface{}{
		"address": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Address != expected {
		t.Errorf("expected %q to be %q", config.Address, expected)
	}
}

func TestDecodeVaultConfig_addressBadType(t *testing.T) {
	_, err := DecodeVaultConfig(map[string]interface{}{
		"address": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "vault: address: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeVaultConfig_token(t *testing.T) {
	config, err := DecodeVaultConfig(map[string]interface{}{
		"token": "testing",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "testing"
	if config.Token != expected {
		t.Errorf("expected %q to be %q", config.Token, expected)
	}
}

func TestDecodeVaultConfig_tokenBadType(t *testing.T) {
	_, err := DecodeVaultConfig(map[string]interface{}{
		"token": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "vault: token: cannot convert bool to string"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeVaultConfig_ssl(t *testing.T) {
	config, err := DecodeVaultConfig(map[string]interface{}{
		"ssl": []map[string]interface{}{
			map[string]interface{}{"enabled": true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := &SSLConfig{
		Enabled: BoolTrue,
		Verify:  BoolUnset,
		Cert:    "",
		CaCert:  "",
	}
	if !reflect.DeepEqual(config.SSL, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.SSL, expected)
	}
}

func TestDecodeVaultConfig_sslBadType(t *testing.T) {
	_, err := DecodeVaultConfig(map[string]interface{}{
		"ssl": true,
	})
	if err == nil {
		t.Fatal(err)
	}

	expected := "vault: ssl: cannot convert bool to []map[string]interface{}"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestDecodeVaultConfig_extraFields(t *testing.T) {
	_, err := DecodeVaultConfig(map[string]interface{}{
		"address": "",
		"foo":     "",
		"bar":     "",
	})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `vault: unknown field(s): "bar", "foo"`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfigTemplate_emptyStringArgs(t *testing.T) {
	_, err := ParseConfigTemplate("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify empty template declaration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigTemplate_stringWithSpacesArgs(t *testing.T) {
	_, err := ParseConfigTemplate("  ")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify empty template declaration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigTemplate_tooManyArgs(t *testing.T) {
	_, err := ParseConfigTemplate("foo:bar:blitz:baz")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "invalid template declaration format"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigTemplate_windowsDrives(t *testing.T) {
	ct, err := ParseConfigTemplate(`C:\abc\123:D:\xyz\789:some command`)
	if err != nil {
		t.Fatalf("failed parsing windows drive letters: %s", err)
	}

	expected := &ConfigTemplate{
		Source:      `C:\abc\123`,
		Destination: `D:\xyz\789`,
		Command:     "some command",
	}

	if !reflect.DeepEqual(ct, expected) {
		t.Fatalf("unexpected result parsing windows drives: %#v", ct)
	}
}

func TestParseConfigTemplate_source(t *testing.T) {
	source := "/tmp/config.ctmpl"
	template, err := ParseConfigTemplate(source)
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}
}

func TestParseConfigTemplate_destination(t *testing.T) {
	source, destination := "/tmp/config.ctmpl", "/tmp/out"
	template, err := ParseConfigTemplate(fmt.Sprintf("%s:%s", source, destination))
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}

	if template.Destination != destination {
		t.Errorf("expected %q to equal %q", template.Destination, destination)
	}
}

func TestParseConfigTemplate_command(t *testing.T) {
	source, destination, command := "/tmp/config.ctmpl", "/tmp/out", "reboot"
	template, err := ParseConfigTemplate(fmt.Sprintf("%s:%s:%s", source, destination, command))
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}

	if template.Destination != destination {
		t.Errorf("expected %q to equal %q", template.Destination, destination)
	}

	if template.Command != command {
		t.Errorf("expected %q to equal %q", template.Command, command)
	}
}
