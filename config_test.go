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
		Wait:            &watch.Wait{Min: 5 * time.Second, Max: 10 * time.Second},
	}
	otherConfig := &Config{
		ConfigTemplates: templates[2:],
		Retry:           15 * time.Second,
		Token:           "def456",
		Wait:            &watch.Wait{Min: 25 * time.Second, Max: 50 * time.Second},
	}

	config.Merge(otherConfig)

	expected := &Config{
		ConfigTemplates: templates,
		Retry:           15 * time.Second,
		Token:           "def456",
		Wait:            &watch.Wait{Min: 25 * time.Second, Max: 50 * time.Second},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("expected %q to equal %q", config, expected)
	}
}

// Test that the flags for HTTPS are properly merged
func TestMerge_HttpsOptions(t *testing.T) {
	{
		// True merges over false
		config := &Config{
			SSL:         false,
			SSLNoVerify: false,
		}
		otherConfig := &Config{
			SSL:         true,
			SSLNoVerify: true,
		}
		config.Merge(otherConfig)
		if !config.SSL || !config.SSLNoVerify {
			t.Fatalf("bad: %#v", config)
		}
	}

	{
		// False does not merge over true
		config := &Config{
			SSL:         true,
			SSLNoVerify: true,
		}
		otherConfig := &Config{
			SSL:         false,
			SSLNoVerify: false,
		}
		config.Merge(otherConfig)
		if !config.SSL || !config.SSLNoVerify {
			t.Fatalf("bad: %#v", config)
		}
	}
}

func TestMerge_BasicAuthOptions(t *testing.T) {
	{
		// If username is present it merges in
		httpAuth := HttpAuth{
			Username: "TestUser",
			Password: "",
		}
		config := &Config{
			HttpAuth: httpAuth,
		}
		otherHttpAuth := HttpAuth{
			Username: "",
			Password: "",
		}
		otherConfig := &Config{
			HttpAuth: otherHttpAuth,
		}
		config.Merge(otherConfig)
		if config.HttpAuth.Username != "TestUser" {
			t.Fatalf("bad %#v", config)
		}
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

// Test that the config is parsed correctly
func TestParseConfig_correctValues(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    consul = "nyc1.demo.consul.io"
    ssl = true
    ssl_no_verify = true
    token = "abcd1234"
    wait = "5s:10s"
    retry = "10s"

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
		SSL:         true,
		SSLNoVerify: true,
		Token:       "abcd1234",
		Wait: &watch.Wait{
			Min: time.Second * 5,
			Max: time.Second * 10,
		},
		WaitRaw:  "5s:10s",
		Retry:    10 * time.Second,
		RetryRaw: "10s",
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
		t.Fatalf("expected %+v to be %+v", config, expected)
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
