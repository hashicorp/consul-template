package main

import (
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"
)

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
	configFile := createTempfile([]byte(`
    invalid file in here
  `), t)
	defer deleteTempfile(configFile, t)

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
	configFile := createTempfile([]byte(`
    consul = true
  `), t)
	defer deleteTempfile(configFile, t)

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
	configFile := createTempfile([]byte(`
    consul = "nyc1.demo.consul.io"
    token = "abcd1234"
    wait = "5s:10s"
    once = true

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
	defer deleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := &Config{
		Path:   configFile.Name(),
		Consul: "nyc1.demo.consul.io",
		Token:  "abcd1234",
		Wait: &Wait{
			Min: time.Second * 5,
			Max: time.Second * 10,
		},
		WaitRaw: "5s:10s",
		Once:    true,
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
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v", expected, config)
	}
}

// Test that ParseWait errors are propagated up
func TestParseConfig_parseWaitError(t *testing.T) {
	configFile := createTempfile([]byte(`
    wait = "not_valid:duration"
  `), t)
	defer deleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "invalid duration not_valid"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}
