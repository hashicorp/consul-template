package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
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
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("consul-template -version", " ")

	status := cli.Run(args)
	if status != ExitCodeOK {
		t.Errorf("expected %s to eq %s", status, ExitCodeOK)
	}

	expected := fmt.Sprintf("consul-template v%s", Version)
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("expected %q to eq %q", errStream.String(), expected)
	}
}

// Test that parser errors are returned
func TestParse_parseError(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("consul-template -bacon delicious", " ")

	status := cli.Run(args)
	if status != ExitCodeParseFlagsError {
		t.Errorf("expected %s to eq %s", status, ExitCodeParseFlagsError)
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(errStream.String(), expected) {
		t.Fatalf("expected %q to contain %q", errStream.String(), expected)
	}
}

// Test wait flag error is propagated
func TestParse_waitFlagError(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("consul-template -wait=watermelon:bacon", " ")

	status := cli.Run(args)
	if status != ExitCodeParseWaitError {
		t.Errorf("expected %s to eq %s", status, ExitCodeParseWaitError)
	}

	expected := "time: invalid duration watermelon"
	if !strings.Contains(errStream.String(), expected) {
		t.Fatalf("expected %q to contain %q", errStream.String(), expected)
	}
}
