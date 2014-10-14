package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

func TestNewWatcher_noConfig(t *testing.T) {
	_, err := NewWatcher(nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "cannot specify empty config"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestNewMatcher_setsConfig(t *testing.T) {
	config := &Config{}
	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatal(err)
	}

	if watcher.config != config {
		t.Errorf("expected %q to be %q", watcher.config, config)
	}
}

func TestWatch_fuck(t *testing.T) {
	outFile := createTempfile(nil, t)
	defer deleteTempfile(outFile, t)

	config := &Config{
		Consul: "demo.consul.io",
		Once:   true,
		ConfigTemplates: []*ConfigTemplate{&ConfigTemplate{
			Source:      "test-fixtures/haproxy.cfg.ctmpl",
			Destination: outFile.Name(),
		}},
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatal(err)
	}

	if err := watcher.Watch(); err != nil {
		t.Fatal(err)
	}

	expected, err := ioutil.ReadFile("test-fixtures/haproxy.cfg")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := ioutil.ReadFile(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(actual, expected) {
		t.Errorf("expected \n%s\n\nto eq\n\n%s\n", actual, expected)
	}
}
