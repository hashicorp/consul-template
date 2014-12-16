package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/util"
)

func TestNewRunner_noDependencies(t *testing.T) {
	runner, err := NewRunner(nil)
	if err != nil {
		t.Fatal(err)
	}

	if runner.configTemplates == nil {
		t.Errorf("expected to be initialized")
	}
}

func TestNewRunner_setsOutStream(t *testing.T) {
	runner, err := NewRunner(nil)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	runner.SetOutStream(buff)

	if runner.outStream != buff {
		t.Errorf("expected %q to equal %q", runner.outStream, buff)
	}
}

func TestNewRunner_setsErrStream(t *testing.T) {
	runner, err := NewRunner(nil)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	runner.SetErrStream(buff)

	if runner.errStream != buff {
		t.Errorf("expected %q to equal %q", runner.errStream, buff)
	}
}

func TestNewRunner_singleConfigTemplate(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if len(runner.dependencies) != 1 {
		t.Errorf("expected 1 Dependency, got %d", len(runner.dependencies))
	}

	if len(runner.templates) != 1 {
		t.Errorf("expected 1 Template, got %d", len(runner.templates))
	}

	if len(runner.configTemplates) != 1 {
		t.Errorf("expected 1 ConfigTemplate, got %d", len(runner.configTemplates))
	}
}

func TestNewRunner_multipleConfigTemplate(t *testing.T) {
	inTemplate1 := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate1, t)

	inTemplate2 := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc2"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate2, t)

	inTemplate3 := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc3"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate3, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate1.Name()},
		&ConfigTemplate{Source: inTemplate2.Name()},
		&ConfigTemplate{Source: inTemplate3.Name()},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(runner.dependencies); num != 3 {
		t.Errorf("expected 3 Dependency, got %d", num)
	}

	if num := len(runner.templates); num != 3 {
		t.Errorf("expected 3 Template, got %d", num)
	}

	if num := len(runner.configTemplates); num != 3 {
		t.Errorf("expected 3 ConfigTemplate, got %d", num)
	}
}

func TestNewRunner_templateWithMultipleDependency(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
    {{ range service "consul@nyc2"}}{{end}}
    {{ range service "consul@nyc3"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(runner.dependencies); num != 3 {
		t.Errorf("expected 3 Dependency, got %d", num)
	}

	if num := len(runner.templates); num != 1 {
		t.Errorf("expected 1 Template, got %d", num)
	}

	if num := len(runner.configTemplates); num != 1 {
		t.Errorf("expected 1 ConfigTemplate, got %d", num)
	}
}

func TestNewRunner_templatesWithDuplicateDependency(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
    {{ range service "consul@nyc1"}}{{end}}
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(runner.dependencies); num != 1 {
		t.Errorf("expected 1 Dependency, got %d", num)
	}

	if num := len(runner.templates); num != 1 {
		t.Errorf("expected 1 Template, got %d", num)
	}

	if num := len(runner.configTemplates); num != 1 {
		t.Errorf("expected 1 ConfigTemplate, got %d", num)
	}
}

func TestNewRunner_multipleTemplatesWithDuplicateDependency(t *testing.T) {
	inTemplate1 := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate1, t)
	inTemplate2 := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate2, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate1.Name()},
		&ConfigTemplate{Source: inTemplate2.Name()},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(runner.dependencies); num != 1 {
		t.Errorf("expected 1 Dependency, got %d", num)
	}

	if num := len(runner.templates); num != 2 {
		t.Errorf("expected 2 Template, got %d", num)
	}

	if num := len(runner.configTemplates); num != 2 {
		t.Errorf("expected 2 ConfigTemplate, got %d", num)
	}
}

func TestNewRunner_multipleTemplatesWithMultipleDependencies(t *testing.T) {
	inTemplate1 := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
    {{ range service "consul@nyc2"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate1, t)
	inTemplate2 := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc2"}}{{end}}
    {{ range service "consul@nyc3"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate2, t)
	inTemplate3 := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc3"}}{{end}}
    {{ range service "consul@nyc4"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate3, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate1.Name()},
		&ConfigTemplate{Source: inTemplate2.Name()},
		&ConfigTemplate{Source: inTemplate3.Name()},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(runner.dependencies); num != 4 {
		t.Errorf("expected 4 Dependency, got %d", num)
	}

	if num := len(runner.templates); num != 3 {
		t.Errorf("expected 3 Template, got %d", num)
	}

	if num := len(runner.configTemplates); num != 3 {
		t.Errorf("expected 3 ConfigTemplate, got %d", num)
	}
}

func TestNewRunner_multipleConfigTemplateSameTemplate(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if num := len(runner.dependencies); num != 1 {
		t.Errorf("expected 1 Dependency, got %d", num)
	}

	if num := len(runner.templates); num != 1 {
		t.Errorf("expected 1 Template, got %d", num)
	}

	if num := len(runner.configTemplates); num != 2 {
		t.Errorf("expected 2 ConfigTemplate, got %d", num)
	}
}

func TestReceive_addsDependency(t *testing.T) {
	runner, err := NewRunner(nil)
	if err != nil {
		t.Fatal(err)
	}

	dependency, data := &test.FakeDependency{}, "this is some data"
	runner.Receive(dependency, data)

	if !runner.receivedData(dependency) {
		t.Errorf("expected dependency to be in received")
	}
	if data != runner.data(dependency) {
		t.Errorf("expected %q to equal %q", data, runner.data(dependency))
	}
}

func TestReceive_updatesDependency(t *testing.T) {
	runner, err := NewRunner(nil)
	if err != nil {
		t.Fatal(err)
	}

	dependency, data := &test.FakeDependency{}, "this is new data"
	runner.Receive(dependency, "first data")
	runner.Receive(dependency, data)

	if !runner.receivedData(dependency) {
		t.Errorf("expected dependency to be in received")
	}
	if data != runner.data(dependency) {
		t.Errorf("expected %q to equal %q", data, runner.data(dependency))
	}
}

func TestRender_noopIfMissingData(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	runner, err := NewRunner(configTemplates)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	runner.SetOutStream(buff)

	if err := runner.RunAll(true); err != nil {
		t.Fatal(err)
	}

	if num := len(buff.Bytes()); num != 0 {
		t.Errorf("expected %d to be %d", num, 0)
	}
}

func TestRender_dryRender(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: "/out/file.txt",
		},
	}

	runner, err := NewRunner(configTemplates)
	if err != nil {
		t.Fatal(err)
	}

	dependency := runner.Dependencies()[0]
	data := []*util.Service{
		&util.Service{Node: "consul1"},
		&util.Service{Node: "consul2"},
	}
	runner.Receive(dependency, data)

	buff := new(bytes.Buffer)
	runner.SetOutStream(buff)

	if err := runner.RunAll(true); err != nil {
		t.Fatal(err)
	}

	actual := bytes.TrimSpace(buff.Bytes())
	expected := bytes.TrimSpace([]byte(`
    > /out/file.txt

    consul1consul2
  `))
	if !bytes.Equal(actual, expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", actual, expected)
	}
}

func TestRender_sameContentsDoesNotRender(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplate := test.CreateTempfile([]byte(`
    consul1consul2
  `), t)
	defer test.DeleteTempfile(outTemplate, t)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outTemplate.Name(),
		},
	}

	runner, err := NewRunner(configTemplates)
	if err != nil {
		t.Fatal(err)
	}

	dependency := runner.Dependencies()[0]
	data := []*util.Service{
		&util.Service{Node: "consul1"},
		&util.Service{Node: "consul2"},
	}
	runner.Receive(dependency, data)

	rendered, err := runner.render(runner.templates[0], outTemplate.Name(), false)
	if err != nil {
		t.Fatal(err)
	}

	if rendered {
		t.Fatal("expected file to not be rendered")
	}
}

func TestRender_sameContentsDoesNotExecuteCommand(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	inTemplate := test.CreateTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplate := test.CreateTempfile([]byte(`
    consul1consul2
  `), t)
	defer test.DeleteTempfile(outTemplate, t)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outTemplate.Name(),
			Command:     fmt.Sprintf("echo 'foo' > %s", outFile.Name()),
		},
	}

	runner, err := NewRunner(configTemplates)
	if err != nil {
		t.Fatal(err)
	}

	dependency := runner.Dependencies()[0]
	data := []*util.Service{
		&util.Service{Node: "consul1"},
		&util.Service{Node: "consul2"},
	}
	runner.Receive(dependency, data)

	if err := runner.RunAll(false); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(outFile.Name())
	if !os.IsNotExist(err) {
		t.Fatalf("expected command to not be run")
	}
}

func TestRender_containingFolderMissing(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	// Create a TempDir and a TempFile in that TempDir, then remove them to
	// "simulate" a non-existent folder
	outDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)
	outFile, err := ioutil.TempFile(outDir, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(outDir); err != nil {
		t.Fatal(err)
	}

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outFile.Name(),
		},
	}

	runner, err := NewRunner(configTemplates)
	if err != nil {
		t.Fatal(err)
	}

	dependency := runner.Dependencies()[0]
	data := []*util.Service{
		&util.Service{Node: "consul1"},
		&util.Service{Node: "consul2"},
	}
	runner.Receive(dependency, data)

	if err := runner.RunAll(false); err != nil {
		t.Fatal(err)
	}

	actual, err := ioutil.ReadFile(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	actual = bytes.TrimSpace(actual)
	expected := []byte("consul1consul2")
	if !bytes.Equal(actual, expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", actual, expected)
	}
}

func TestRender_outputFileMissing(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	// Create a TempDir and a TempFile in that TempDir, then remove the file to
	// "simulate" a non-existent file
	outDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)
	outFile, err := ioutil.TempFile(outDir, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(outFile.Name()); err != nil {
		t.Fatal(err)
	}

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outFile.Name(),
		},
	}
	runner, err := NewRunner(configTemplates)
	if err != nil {
		t.Fatal(err)
	}

	dependency := runner.Dependencies()[0]
	data := []*util.Service{
		&util.Service{Node: "consul1"},
		&util.Service{Node: "consul2"},
	}
	runner.Receive(dependency, data)

	if err := runner.RunAll(false); err != nil {
		t.Fatal(err)
	}

	actual, err := ioutil.ReadFile(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	actual = bytes.TrimSpace(actual)
	expected := []byte("consul1consul2")
	if !bytes.Equal(actual, expected) {
		t.Errorf("expected \n%q\n to equal \n%q\n", actual, expected)
	}
}

func TestRender_outputFileRetainsPermissions(t *testing.T) {
	inTemplate := test.CreateTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	// Create a TempDir and a TempFile in that TempDir, then remove the file to
	// "simulate" a non-existent file
	outDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)
	outFile, err := ioutil.TempFile(outDir, "")
	if err != nil {
		t.Fatal(err)
	}
	os.Chmod(outFile.Name(), 0644)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outFile.Name(),
		},
	}

	runner, err := NewRunner(configTemplates)
	if err != nil {
		t.Fatal(err)
	}

	dependency := runner.Dependencies()[0]
	data := []*util.Service{
		&util.Service{Node: "consul1"},
		&util.Service{Node: "consul2"},
	}
	runner.Receive(dependency, data)

	if err := runner.RunAll(false); err != nil {
		t.Fatal(err)
	}

	stat, err := os.Stat(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := os.FileMode(0644)
	if stat.Mode() != expected {
		t.Errorf("expected %q to be %q", stat.Mode(), expected)
	}
}

func TestExecute_doesNotRunInDry(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplate := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outTemplate.Name(),
			Command:     fmt.Sprintf("echo 'foo' > %s", outFile.Name()),
		},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if err := runner.RunAll(true); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(outFile.Name())
	if !os.IsNotExist(err) {
		t.Fatalf("expected command to not be run")
	}
}

func TestExecute_doesNotExecuteCommandMissingDependencies(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplate := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outTemplate.Name(),
			Command:     fmt.Sprintf("echo 'foo' > %s", outFile.Name()),
		},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	if err := runner.RunAll(false); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(outFile.Name())
	if !os.IsNotExist(err) {
		t.Fatalf("expected command to not be run")
	}
}

func TestExecute_executesCommand(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplate := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplate, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outTemplate.Name(),
			Command:     fmt.Sprintf("echo 'foo' > %s", outFile.Name()),
		},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	serviceDependency, err := util.ParseServiceDependency("consul@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	data := []*util.Service{
		&util.Service{
			Node:    "consul",
			Address: "1.2.3.4",
			ID:      "consul@nyc1",
			Name:    "consul",
		},
	}

	runner.Receive(serviceDependency, data)

	if err := runner.RunAll(false); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute_doesNotExecuteCommandMoreThanOnce(t *testing.T) {
	outFile := test.CreateTempfile(nil, t)
	os.Remove(outFile.Name())
	defer os.Remove(outFile.Name())

	inTemplate := test.CreateTempfile([]byte(`
    {{ range service "consul@nyc1"}}{{end}}
  `), t)
	defer test.DeleteTempfile(inTemplate, t)

	outTemplateA := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplateA, t)

	outTemplateB := test.CreateTempfile(nil, t)
	defer test.DeleteTempfile(outTemplateB, t)

	ctemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outTemplateA.Name(),
			Command:     fmt.Sprintf("echo 'foo' >> %s", outFile.Name()),
		},
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outTemplateB.Name(),
			Command:     fmt.Sprintf("echo 'foo' >> %s", outFile.Name()),
		},
	}

	runner, err := NewRunner(ctemplates)
	if err != nil {
		t.Fatal(err)
	}

	serviceDependency, err := util.ParseServiceDependency("consul@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	data := []*util.Service{
		&util.Service{
			Node:    "consul",
			Address: "1.2.3.4",
			ID:      "consul@nyc1",
			Name:    "consul",
		},
	}

	runner.Receive(serviceDependency, data)

	if err := runner.RunAll(false); err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	output, err := ioutil.ReadFile(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(string(output), "foo") > 1 {
		t.Fatalf("expected command to be run once.")
	}

}
