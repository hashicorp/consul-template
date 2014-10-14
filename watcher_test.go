package main

import (
	"errors"
	"io/ioutil"
	"strings"
	"testing"
	"time"
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

func TestWatch_polls(t *testing.T) {
	t.Parallel()

	config := &Config{
		Consul: "demo.consul.io",
	}
	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error)
	go func() {
		errCh <- watcher.Watch()
		errCh <- errors.New("returned too soon")
	}()

	// This test is a little weird, so I will explain it. We are checking to make
	// sure that the Watch() function runs continuously so we put it in a go
	// routine and if it does not return within a certain number of seconds, we
	// call that "OK".
	select {
	case e := <-errCh:
		if e != nil {
			t.Fatal(err)
		}
	case <-time.After(1 * time.Second):
		// OK
	}
}

func TestWatch_once(t *testing.T) {
	t.Parallel()

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

	bytes, err := ioutil.ReadFile(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	actual := string(bytes)

	expected := "global\n    maxconn 256"
	if !strings.Contains(actual, expected) {
		t.Errorf("expected %q to contain %q", actual, expected)
	}

	expected = "backend app\n    server consul"
	if !strings.Contains(actual, expected) {
		t.Errorf("expected %q to contain %q", actual, expected)
	}
}
