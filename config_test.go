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

// Test that an empty config does nothing
func TestMerge_emptyConfig(t *testing.T) {
	consul := "consul.io:8500"
	config := &Config{Consul: consul}
	config.Merge(&Config{})

	if config.Consul != consul {
		t.Fatalf("expected %q to equal %q", config.Consul, consul)
	}
}

// Test that simple values are merged
func TestMerge_simpleConfig(t *testing.T) {
	config, newConsul := &Config{Consul: "consul.io:8500"}, "packer.io:7300"
	config.Merge(&Config{Consul: newConsul})

	if config.Consul != newConsul {
		t.Fatalf("expected %q to equal %q", config.Consul, newConsul)
	}
}

// Test that complex values are merged, and that ConfigTemplates are additive
func TestMerge_complexConfig(t *testing.T) {
	templates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      "a",
			Destination: "b",
		},
		&ConfigTemplate{
			Source:      "c",
			Destination: "d",
			Command:     "e",
		},
		&ConfigTemplate{
			Source:      "f",
			Destination: "g",
			Command:     "h",
		},
		&ConfigTemplate{
			Source:      "i",
			Destination: "j",
		},
	}

	config := &Config{
		ConfigTemplates: templates[:2],
		Retry:           5 * time.Second,
		Token:           "abc123",
		MaxStale:        3 * time.Second,
		Wait:            &watch.Wait{Min: 5 * time.Second, Max: 10 * time.Second},
		LogLevel:        "WARN",
	}
	otherConfig := &Config{
		ConfigTemplates: templates[2:],
		Retry:           15 * time.Second,
		Token:           "def456",
		Wait:            &watch.Wait{Min: 25 * time.Second, Max: 50 * time.Second},
		LogLevel:        "ERR",
	}

	config.Merge(otherConfig)

	expected := &Config{
		ConfigTemplates: templates,
		Retry:           15 * time.Second,
		Token:           "def456",
		MaxStale:        3 * time.Second,
		Wait:            &watch.Wait{Min: 25 * time.Second, Max: 50 * time.Second},
		LogLevel:        "ERR",
	}

	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("expected %#v to equal %#v", config, expected)
	}
}

// Test that the flags for HTTPS are properly merged
func TestMerge_HttpsOptions(t *testing.T) {
	config := &Config{
		SSL: &SSL{
			Enabled: false,
			Verify:  false,
		},
	}
	otherConfig := &Config{
		SSL: &SSL{
			Enabled: true,
			Verify:  true,
			Cert: "c1.pem",
			CaCert: "c2.pem",
		},
	}
	config.Merge(otherConfig)

	if config.SSL.Enabled != true {
		t.Errorf("expected enabled to be true")
	}

	if config.SSL.Verify != true {
		t.Errorf("expected SSL verify to be true")
	}

	if config.SSL.Cert != "c1.pem" {
		t.Errorf("expected SSL cert to be c1.pem")
	}

	if config.SSL.CaCert != "c2.pem" {
		t.Errorf("expected SSL ca cert to be c2.pem")
	}

	config = &Config{
		SSL: &SSL{
			Enabled: true,
			Verify:  true,
			Cert: "c1.pem",
			CaCert: "c2.pem",
		},
	}
	otherConfig = &Config{
		SSL: &SSL{
			Enabled: false,
			Verify:  false,
		},
	}
	config.Merge(otherConfig)

	if config.SSL.Enabled != false {
		t.Errorf("expected enabled to be false")
	}

	if config.SSL.Verify != false {
		t.Errorf("expected SSL verify to be false")
	}

	if config.SSL.Cert != "" {
		t.Errorf("expected SSL cert to be empty string")
	}

	if config.SSL.CaCert != "" {
		t.Errorf("expected SSL ca cert to be empty string")
	}
}

func TestMerge_AuthOptions(t *testing.T) {
	config := &Config{
		Auth: &Auth{Username: "user", Password: "pass"},
	}
	otherConfig := &Config{
		Auth: &Auth{Username: "newUser", Password: ""},
	}
	config.Merge(otherConfig)

	if config.Auth.Username != "newUser" {
		t.Errorf("expected %q to be %q", config.Auth.Username, "newUser")
	}
}

func TestMerge_SyslogOptions(t *testing.T) {
	config := &Config{
		Syslog: &Syslog{Enabled: false, Facility: "LOCAL0"},
	}
	otherConfig := &Config{
		Syslog: &Syslog{Enabled: true, Facility: "LOCAL1"},
	}
	config.Merge(otherConfig)

	if config.Syslog.Enabled != true {
		t.Errorf("expected %t to be %t", config.Syslog.Enabled, true)
	}

	if config.Syslog.Facility != "LOCAL1" {
		t.Errorf("expected %q to be %q", config.Syslog.Facility, "LOCAL1")
	}
}

func TestAuthString_disabled(t *testing.T) {
	a := &Auth{Enabled: false}
	expected := ""
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabledNoPassword(t *testing.T) {
	a := &Auth{Enabled: true, Username: "username"}
	expected := "username"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabled(t *testing.T) {
	a := &Auth{Enabled: true, Username: "username", Password: "password"}
	expected := "username:password"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

// Test that file read errors are propagated up
func TestParseConfig_readFileError(t *testing.T) {
	_, err := ParseConfig(path.Join(os.TempDir(), "config.json"))
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "no such file or directory"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that parser errors are propagated up
func TestParseConfig_parseFileError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    invalid file in here
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "syntax error"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that mapstructure errors are propagated up
func TestParseConfig_mapstructureError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    consul = true
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "nconvertible type 'bool'"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that mapstructure errors on extra kes
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

// Test that the config is parsed correctly
func TestParseConfig_correctValues(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    consul = "nyc1.demo.consul.io"
    max_stale = "5s"
    token = "abcd1234"
    wait = "5s:10s"
    retry = "10s"
    log_level = "warn"

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
		Path:        configFile.Name(),
		Consul:      "nyc1.demo.consul.io",
		MaxStale:    time.Second * 5,
		MaxStaleRaw: "5s",
		Auth: &Auth{
			Enabled:  true,
			Username: "test",
			Password: "test",
		},
		AuthRaw: []*Auth{
			&Auth{
				Enabled:  true,
				Username: "test",
				Password: "test",
			},
		},
		SSL: &SSL{
			Enabled: true,
			Verify:  false,
			Cert: "c1.pem",
			CaCert: "c2.pem",
		},
		SSLRaw: []*SSL{
			&SSL{
				Enabled: true,
				Verify:  false,
				Cert: "c1.pem",
				CaCert: "c2.pem",
			},
		},
		Syslog: &Syslog{
			Enabled:  true,
			Facility: "LOCAL5",
		},
		SyslogRaw: []*Syslog{
			&Syslog{
				Enabled:  true,
				Facility: "LOCAL5",
			},
		},
		Token: "abcd1234",
		Wait: &watch.Wait{
			Min: time.Second * 5,
			Max: time.Second * 10,
		},
		WaitRaw:  "5s:10s",
		Retry:    10 * time.Second,
		RetryRaw: "10s",
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
		t.Fatalf("expected %#v to be %#v", config, expected)
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

	expectedErr := "retry invalid"
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

	expectedErr := "wait invalid"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that an error is returned when the empty string is given
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

// Test that an error is returned when a string with spaces is given
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

// Test that an error is returned when there are too many arguments
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

// Test that we properly parse Windows drive paths
func TestParseConfigurationTemplate_windowsDrives(t *testing.T) {
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

// Test that a source value is correctly used
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

// Test that a destination wait value is correctly used
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

// Test that a command wait value is correctly used
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
