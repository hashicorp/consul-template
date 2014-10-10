package main

import (
	"os"
	"path"
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

// Test that the config reads the "consul" key
func TestParseConfig_readsConsulKey(t *testing.T) {
	configFile := createTempfile([]byte(`
    consul = "nyc1.demo.consul.io"
  `), t)
	defer deleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "nyc1.demo.consul.io"
	if config.Consul != expected {
		t.Fatalf("expected config.Consul to be %q, was %q", expected, config.Consul)
	}
}

// Test that the config reads the "template" keys
func TestParseConfig_readsTemplateKeys(t *testing.T) {
	configFile := createTempfile([]byte(`
    template {
      source = "nginx.conf.ctmpl"
      destination  = "/etc/nginx/nginx.conf"
    }

    template {
      source = "redis.conf.ctmpl"
      destination  = "/etc/redis/redis.conf"
      command = "service restart redis"
    }
  `), t)
	defer deleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(config.Instructions) != 2 {
		t.Fatalf("expected 3 Instructions, but slice had %d", len(config.Instructions))
	}

	nginx := config.Instructions[0]
	expected := "nginx.conf.ctmpl"
	if nginx.Source != expected {
		t.Errorf("expected nginx.Source to be %q, was %q", expected, nginx.Source)
	}
	expected = "/etc/nginx/nginx.conf"
	if nginx.Destination != expected {
		t.Errorf("expected nginx.Destination to be %q, was %q", expected, nginx.Destination)
	}
	expected = ""
	if nginx.Command != expected {
		t.Errorf("expected nginx.Command to be %q, was %q", expected, nginx.Command)
	}

	redis := config.Instructions[1]
	expected = "redis.conf.ctmpl"
	if redis.Source != expected {
		t.Errorf("expected redis.Source to be %q, was %q", expected, redis.Source)
	}
	expected = "/etc/redis/redis.conf"
	if redis.Destination != expected {
		t.Errorf("expected redis.Destination to be %q, was %q", expected, redis.Destination)
	}
	expected = "service restart redis"
	if redis.Command != expected {
		t.Errorf("expected redis.Command to be %q, was %q", expected, redis.Command)
	}
}

// Test that the config reads the "token" key
func TestParseConfig_readsTokenKey(t *testing.T) {
	configFile := createTempfile([]byte(`
    token = "abcd1234"
  `), t)
	defer deleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "abcd1234"
	if config.Token != expected {
		t.Fatalf("expected config.Token to be %q, was %q", expected, config.Token)
	}
}

// Test that mapstructure does not propagate the "path" key from the config
// and properly sets it to the given path argument.
func TestParseConfig_setsPath(t *testing.T) {
	configFile := createTempfile([]byte(`
    path = "/path/to/config.json"
  `), t)
	defer deleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := configFile.Name()
	if config.Path != expected {
		t.Fatalf("expected config.Path to be %q, was %q", expected, config.Token)
	}
}

// Test that the config reads the "wait" key
func TestParseConfig_readsWaitKey(t *testing.T) {
	configFile := createTempfile([]byte(`
    wait = "5s:10s"
  `), t)
	defer deleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	wait := config.Wait

	expectedMin := time.Duration(5) * time.Second
	if wait.Min != expectedMin {
		t.Errorf("expected %q to equal %q", wait.Min, expectedMin)
	}

	expectedMax := time.Duration(10) * time.Second
	if wait.Max != expectedMax {
		t.Errorf("expected %q to equal %q", wait.Max, expectedMin)
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

// Test that the config reads the "dry" key
func TestParseConfig_readsDryKey(t *testing.T) {
	configFile := createTempfile([]byte(`
    dry = true
  `), t)
	defer deleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Dry != expected {
		t.Fatalf("expected config.Dry to be %q, was %q", expected, config.Dry)
	}
}

// Test that the config reads the "once" key
func TestParseConfig_readsOnceKey(t *testing.T) {
	configFile := createTempfile([]byte(`
    once = true
  `), t)
	defer deleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Once != expected {
		t.Fatalf("expected config.Once to be %q, was %q", expected, config.Once)
	}
}
