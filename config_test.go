package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/watch"
)

func testConfig(contents string, t *testing.T) *Config {
	f, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.Write([]byte(contents))
	if err != nil {
		t.Fatal(err)
	}

	config, err := ParseConfig(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestMerge_emptyConfig(t *testing.T) {
	config := DefaultConfig()
	config.Merge(&Config{})

	expected := DefaultConfig()
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_topLevel(t *testing.T) {
	config1 := testConfig(`
		consul = "consul-1"
		token = "token-1"
		max_stale = "1s"
		retry = "1s"
		wait = "1s"
		pid_file = "/pid-1"
		log_level = "log_level-1"
	`, t)
	config2 := testConfig(`
		consul = "consul-2"
		token = "token-2"
		max_stale = "2s"
		retry = "2s"
		wait = "2s"
		pid_file = "/pid-2"
		log_level = "log_level-2"
	`, t)
	config1.Merge(config2)

	if !reflect.DeepEqual(config1, config2) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config1, config2)
	}
}

func TestMerge_vault(t *testing.T) {
	config := testConfig(`
		vault {
			address = "1.1.1.1"
			token = "1"
			renew = true
		}
	`, t)
	config.Merge(testConfig(`
		vault {
			address = "2.2.2.2"
			renew = false
		}
	`, t))

	expected := &VaultConfig{
		Address: "2.2.2.2",
		Token:   "1",
		Renew:   false,
		SSL: &SSLConfig{
			Enabled: true,
			Verify:  true,
			Cert:    "",
			CaCert:  "",
		},
	}

	if !reflect.DeepEqual(config.Vault, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Vault, expected)
	}
}

func TestMerge_vaultSSL(t *testing.T) {
	config := testConfig(`
		vault {
			ssl {
				enabled = true
				verify = true
				cert = "1.pem"
				ca_cert = "ca-1.pem"
			}
		}
	`, t)
	config.Merge(testConfig(`
		vault {
			ssl {
				enabled = false
			}
		}
	`, t))

	expected := &VaultConfig{
		SSL: &SSLConfig{
			Enabled: false,
			Verify:  true,
			Cert:    "1.pem",
			CaCert:  "ca-1.pem",
		},
	}

	if !reflect.DeepEqual(config.Vault.SSL, expected.SSL) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Vault.SSL, expected.SSL)
	}
}

func TestMerge_auth(t *testing.T) {
	config := testConfig(`
		auth {
			enabled = true
			username = "1"
			password = "1"
		}
	`, t)
	config.Merge(testConfig(`
		auth {
			password = "2"
		}
	`, t))

	expected := &AuthConfig{
		Enabled:  true,
		Username: "1",
		Password: "2",
	}

	if !reflect.DeepEqual(config.Auth, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Auth, expected)
	}
}

func TestMerge_SSL(t *testing.T) {
	config := testConfig(`
		ssl {
			enabled = true
			verify = true
			cert = "1.pem"
			ca_cert = "ca-1.pem"
		}
	`, t)
	config.Merge(testConfig(`
		ssl {
			enabled = false
		}
	`, t))

	expected := &SSLConfig{
		Enabled: false,
		Verify:  true,
		Cert:    "1.pem",
		CaCert:  "ca-1.pem",
	}

	if !reflect.DeepEqual(config.SSL, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.SSL, expected)
	}
}

func TestMerge_syslog(t *testing.T) {
	config := testConfig(`
		syslog {
			enabled = true
			facility = "1"
		}
	`, t)
	config.Merge(testConfig(`
		syslog {
			facility = "2"
		}
	`, t))

	expected := &SyslogConfig{
		Enabled:  true,
		Facility: "2",
	}

	if !reflect.DeepEqual(config.Syslog, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Syslog, expected)
	}
}

func TestMerge_configTemplates(t *testing.T) {
	config := testConfig(`
		template {
			source = "1"
			destination = "1"
			command = "1"
			perms = 0600
			backup = false
		}
	`, t)
	config.Merge(testConfig(`
		template {
			source = "2"
			destination = "2"
			command = "2"
			perms = 0755
			backup = true
		}
	`, t))

	expected := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      "1",
			Destination: "1",
			Command:     "1",
			Perms:       0600,
			Backup:      false,
		},
		&ConfigTemplate{
			Source:      "2",
			Destination: "2",
			Command:     "2",
			Perms:       0755,
			Backup:      true,
		},
	}

	if !reflect.DeepEqual(config.ConfigTemplates, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.ConfigTemplates[0], expected[0])
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.ConfigTemplates[1], expected[1])
	}
}

func TestMerge_wait(t *testing.T) {
	config := testConfig(`
		wait = "1s:1s"
	`, t)
	config.Merge(testConfig(`
		wait = "2s:2s"
	`, t))

	expected := &watch.Wait{
		Min: 2 * time.Second,
		Max: 2 * time.Second,
	}

	if !reflect.DeepEqual(config.Wait, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Wait, expected)
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

func TestParseConfig_correctValues(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
		consul = "nyc1.demo.consul.io"
		max_stale = "5s"
		token = "abcd1234"
		wait = "5s:10s"
		retry = "10s"
		pid_file = "/var/run/ct"
		log_level = "warn"

		vault {
			address = "vault.service.consul"
			token = "efgh5678"
			renew = true
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
			perms = 0755
		}
  `), t)
	defer test.DeleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := &Config{
		Path:     configFile.Name(),
		PidFile:  "/var/run/ct",
		Consul:   "nyc1.demo.consul.io",
		MaxStale: time.Second * 5,
		Vault: &VaultConfig{
			Address: "vault.service.consul",
			Token:   "efgh5678",
			Renew:   true,
			SSL: &SSLConfig{
				Enabled: false,
				Verify:  true,
				Cert:    "",
				CaCert:  "",
			},
		},
		Auth: &AuthConfig{
			Enabled:  true,
			Username: "test",
			Password: "test",
		},
		SSL: &SSLConfig{
			Enabled: true,
			Verify:  false,
			Cert:    "c1.pem",
			CaCert:  "c2.pem",
		},
		Syslog: &SyslogConfig{
			Enabled:  true,
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
				Perms:       0644,
			},
			&ConfigTemplate{
				Source:      "redis.conf.ctmpl",
				Destination: "/etc/redis/redis.conf",
				Command:     "service redis restart",
				Perms:       0755,
			},
		},
		setKeys: config.setKeys,
	}

	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("expected \n%#v\n to be \n%#v\n", config, expected)
	}
}

func TestParseConfig_mapstructureError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    consul = true
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "unconvertible type 'bool'"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfig_extraKeys(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
		fake_key = "nope"
		another_fake_key = "never"
	`), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error")
	}

	expected := "invalid keys: another_fake_key, fake_key"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestParseConfig_parseMaxStaleError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    max_stale = "bacon pants"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfig_parseRetryError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    retry = "bacon pants"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfig_parseWaitError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    wait = "not_valid:duration"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestConfigFromPath_singleFile(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
		consul = "127.0.0.1"
	`), t)
	defer test.DeleteTempfile(configFile, t)

	config, err := ConfigFromPath(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "127.0.0.1"
	if config.Consul != expected {
		t.Errorf("expected %q to be %q", config.Consul, expected)
	}
}

func TestConfigFromPath_NonExistentDirectory(t *testing.T) {
	// Create a directory and then delete it
	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(configDir); err != nil {
		t.Fatal(err)
	}

	_, err = ConfigFromPath(configDir)
	if err == nil {
		t.Fatalf("expected error, but nothing was returned")
	}

	expected := "missing file/folder"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestConfigFromPath_EmptyDirectory(t *testing.T) {
	// Create a directory with no files
	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(configDir)

	_, err = ConfigFromPath(configDir)
	if err != nil {
		t.Fatalf("empty directories are allowed")
	}
}

func TestConfigFromPath_configDir(t *testing.T) {
	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	configFile1, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	config1 := []byte(`
		consul = "127.0.0.1:8500"
	`)
	_, err = configFile1.Write(config1)
	if err != nil {
		t.Fatal(err)
	}
	configFile2, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	config2 := []byte(`
		template {
		  source = "/path/on/disk/to/template"
		  destination = "/path/on/disk/where/template/will/render"
		  command = "optional command to run when the template is updated"
		}
	`)
	_, err = configFile2.Write(config2)
	if err != nil {
		t.Fatal(err)
	}

	config, err := ConfigFromPath(configDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedConfig := Config{
		Consul: "127.0.0.1:8500",
		ConfigTemplates: []*ConfigTemplate{{
			Source:      "/path/on/disk/to/template",
			Destination: "/path/on/disk/where/template/will/render",
			Command:     "optional command to run when the template is updated",
		}},
	}
	if expectedConfig.Consul != config.Consul {
		t.Fatalf("Config files failed to combine. Expected Consul to be %s but got %s", expectedConfig.Consul, config.Consul)
	}
	if len(config.ConfigTemplates) != len(expectedConfig.ConfigTemplates) {
		t.Fatalf("Expected %d ConfigTemplate but got %d", len(expectedConfig.ConfigTemplates), len(config.ConfigTemplates))
	}
	for i, expectTemplate := range expectedConfig.ConfigTemplates {
		actualTemplate := config.ConfigTemplates[i]
		if actualTemplate.Source != expectTemplate.Source {
			t.Fatalf("Expected template Source to be %s but got %s", expectTemplate.Source, actualTemplate.Source)
		}
		if actualTemplate.Destination != expectTemplate.Destination {
			t.Fatalf("Expected template Destination to be %s but got %s", expectTemplate.Destination, actualTemplate.Destination)
		}
		if actualTemplate.Command != expectTemplate.Command {
			t.Fatalf("Expected template Command to be %s but got %s", expectTemplate.Command, actualTemplate.Command)
		}
	}
}

func TestAuthString_disabled(t *testing.T) {
	a := &AuthConfig{Enabled: false}
	expected := ""
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabledNoPassword(t *testing.T) {
	a := &AuthConfig{Enabled: true, Username: "username"}
	expected := "username"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabled(t *testing.T) {
	a := &AuthConfig{Enabled: true, Username: "username", Password: "password"}
	expected := "username:password"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
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

func TestParseConfigurationTemplate_tooManyArgs(t *testing.T) {
	_, err := ParseConfigTemplate("foo:bar:blitz:baz")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "invalid template declaration format"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigurationTemplate_windowsDrives(t *testing.T) {
	ct, err := ParseConfigTemplate(`C:\abc\123:D:\xyz\789:some command`)
	if err != nil {
		t.Fatalf("failed parsing windows drive letters: %s", err)
	}

	expected := &ConfigTemplate{
		Source:      `C:\abc\123`,
		Destination: `D:\xyz\789`,
		Command:     "some command",
		Perms:       0644,
	}

	if !reflect.DeepEqual(ct, expected) {
		t.Fatalf("unexpected result parsing windows drives: %#v", ct)
	}
}

func TestParseConfigurationTemplate_source(t *testing.T) {
	source := "/tmp/config.ctmpl"
	template, err := ParseConfigTemplate(source)
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}
}

func TestParseConfigurationTemplate_destination(t *testing.T) {
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

func TestParseConfigurationTemplate_command(t *testing.T) {
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
