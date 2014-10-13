package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

//
func TestRun_printsErrors(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("consul-template -bacon delicious", " ")

	status := cli.Run(args)
	if status == ExitCodeOK {
		t.Fatal("expected not OK exit code")
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("expected %q to eq %q", errStream.String(), expected)
	}
}

// Test that the version flag outputs the version and retuns an OK exit code
func TestParse_versionFlag(t *testing.T) {
	var cli CLI
	args := strings.Split("consul-template -version", " ")

	_, status, err := cli.Parse(args)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := fmt.Sprintf("consul-template v%s", Version)
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}

	if status != ExitCodeOK {
		t.Errorf("expected %s to eq %s", status, ExitCodeOK)
	}
}

// Test that parser errors are returned
func TestParse_parseError(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("consul-template -bacon delicious", " ")

	_, status, err := cli.Parse(args)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}

	if status != ExitCodeParseFlagsError {
		t.Errorf("expected %s to eq %s", status, ExitCodeParseFlagsError)
	}
}

// Test wait flag is parsed
func TestParse_waitFlag(t *testing.T) {
	var cli CLI
	args := strings.Split("consul-template -wait=5s:10s", " ")

	config, status, err := cli.Parse(args)
	if err != nil {
		t.Fatal(err)
	}

	expected := &Wait{
		Min: time.Duration(5) * time.Second,
		Max: time.Duration(10) * time.Second,
	}

	if !reflect.DeepEqual(config.Wait, expected) {
		t.Errorf("expected %q to equal %q", config.Wait, expected)
	}

	if status != ExitCodeOK {
		t.Errorf("expected %s to eq %s", status, ExitCodeOK)
	}
}

// Test wait flag error is propagated
func TestParse_waitFlagError(t *testing.T) {
	var cli CLI
	args := strings.Split("consul-template -wait=watermelon:bacon", " ")

	_, status, err := cli.Parse(args)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "time: invalid duration watermelon"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}

	if status != ExitCodeParseWaitError {
		t.Errorf("expected %s to eq %s", status, ExitCodeParseWaitError)
	}
}

// Test that the -config flag is parsed properly
func TestParse_configFlag(t *testing.T) {
	var cli CLI

	configOnDisk := createTempfile([]byte(`
    consul = "nyc1.demo.consul.io"
    wait = "5s:10s"
  `), t)
	defer deleteTempfile(configOnDisk, t)

	cmd := fmt.Sprintf("consul-template -config %s -wait=30s:1m", configOnDisk.Name())
	args := strings.Split(cmd, " ")

	config, status, err := cli.Parse(args)
	if err != nil {
		t.Fatal(err)
	}

	// Test that the config is parsed
	expectedConsul := "nyc1.demo.consul.io"
	if config.Consul != expectedConsul {
		t.Fatalf("expected %q to equal %q", config.Consul, expectedConsul)
	}

	// Test that command line options take precedence over config file options
	expectedWait := &Wait{
		Min: time.Duration(30) * time.Second,
		Max: time.Duration(1) * time.Minute,
	}

	if !reflect.DeepEqual(config.Wait, expectedWait) {
		t.Fatalf("expected %q to equal %q", config.Wait, expectedWait)
	}

	if status != ExitCodeOK {
		t.Errorf("expected %s to eq %s", status, ExitCodeOK)
	}
}
