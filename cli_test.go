package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
)

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

func TestRun_versionFlag(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("consul-template -version", " ")

	status := cli.Run(args)
	if status != ExitCodeOK {
		t.Errorf("expected %q to eq %q", status, ExitCodeOK)
	}

	expected := fmt.Sprintf("consul-template v%s", Version)
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("expected %q to eq %q", errStream.String(), expected)
	}
}

func TestRun_parseError(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("consul-template -bacon delicious", " ")

	status := cli.Run(args)
	if status != ExitCodeParseFlagsError {
		t.Errorf("expected %q to eq %q", status, ExitCodeParseFlagsError)
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(errStream.String(), expected) {
		t.Fatalf("expected %q to contain %q", errStream.String(), expected)
	}
}

func TestRun_waitFlagError(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("consul-template -wait=watermelon:bacon", " ")

	status := cli.Run(args)
	if status != ExitCodeParseWaitError {
		t.Errorf("expected %q to eq %q", status, ExitCodeParseWaitError)
	}

	expected := "time: invalid duration watermelon"
	if !strings.Contains(errStream.String(), expected) {
		t.Fatalf("expected %q to contain %q", errStream.String(), expected)
	}
}

func TestRun_onceFlag(t *testing.T) {
	template := test.CreateTempfile([]byte(`
	{{range service "consul"}}{{.Name}}{{end}}
  `), t)
	defer test.DeleteTempfile(template, t)

	out := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(out, t)

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}

	command := fmt.Sprintf("consul-template -consul demo.consul.io -template %s:%s -once", template.Name(), out.Name())
	args := strings.Split(command, " ")

	ch := make(chan int, 1)

	go func() {
		ch <- cli.Run(args)
	}()

	select {
	case status := <-ch:
		if status != ExitCodeOK {
			t.Errorf("expected %d to eq %d", status, ExitCodeOK)
			t.Errorf("stderr: %s", errStream.String())
		}
	case <-time.After(5 * time.Second):
		t.Errorf("expected data, but nothing was returned")
	}

}

func TestQuiescence(t *testing.T) {
	t.Skip("TODO")
}

func TestRun_configDir(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
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

	args := strings.Split("consul-template -config "+configDir, " ")

	status := cli.Run(args)
	if status == ExitCodeOK {
		t.Fatal("expected not OK exit code")
	}
}

func TestCLI_BuildConfig(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
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

	config := new(Config)

	cli.BuildConfig(config, configDir)

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
