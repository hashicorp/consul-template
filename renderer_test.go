package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNewRenderer_noDependencies(t *testing.T) {
	_, err := NewRenderer(nil, false)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "renderer: must supply at least one Dependency"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestNewRenderer_setsDependencies(t *testing.T) {
	dependencies := []Dependency{
		&FakeDependency{},
		&FakeDependency{},
	}

	renderer, err := NewRenderer(dependencies, false)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(renderer.dependencies, dependencies) {
		t.Errorf("expected %q to equal %q", renderer.dependencies, dependencies)
	}
}

func TestNewRenderer_setsDry(t *testing.T) {
	renderer, err := NewRenderer(make([]Dependency, 1), true)
	if err != nil {
		t.Fatal(err)
	}

	if renderer.dry != true {
		t.Errorf("expected %q to equal %q", renderer.dry, true)
	}
}

func TestNewRenderer_createsDependencyDataMap(t *testing.T) {
	renderer, err := NewRenderer(make([]Dependency, 1), false)
	if err != nil {
		t.Fatal(err)
	}

	if renderer.dependencyDataMap == nil {
		t.Errorf("expected dependencyDataMap")
	}
}

func TestSetDryStream_setsStream(t *testing.T) {
	renderer, err := NewRenderer(make([]Dependency, 1), false)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	renderer.SetDryStream(buff)

	if renderer.dryStream != buff {
		t.Errorf("expected %q to equal %q", renderer.dryStream, buff)
	}
}

func TestReceive_addsDependency(t *testing.T) {
	renderer, err := NewRenderer(make([]Dependency, 1), false)
	if err != nil {
		t.Fatal(err)
	}

	dependency, data := &FakeDependency{}, "this is some data"
	renderer.Receive(dependency, data)

	storedData, ok := renderer.dependencyDataMap[dependency]
	if !ok {
		t.Errorf("expected dependency to be in map")
	}
	if data != storedData {
		t.Errorf("expected %q to equal %q", data, storedData)
	}
}

func TestReceive_updatesDependency(t *testing.T) {
	renderer, err := NewRenderer(make([]Dependency, 1), false)
	if err != nil {
		t.Fatal(err)
	}

	dependency, data := &FakeDependency{}, "this is new data"
	renderer.Receive(dependency, "first data")
	renderer.Receive(dependency, data)

	storedData, ok := renderer.dependencyDataMap[dependency]
	if !ok {
		t.Errorf("expected dependency to be in map")
	}
	if data != storedData {
		t.Errorf("expected %q to equal %q", data, storedData)
	}
}

func TestMaybeRender_noopIfMissingData(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{range service "consul@nyc1"}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	renderer, err := NewRenderer(template.Dependencies(), true)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	renderer.SetDryStream(buff)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{Source: inTemplate.Name()},
	}

	if err := renderer.MaybeRender(template, configTemplates); err != nil {
		t.Fatal(err)
	}

	if num := len(buff.Bytes()); num != 0 {
		t.Errorf("expected %d to be %d", num, 0)
	}
}

func TestMaybeRender_dryRender(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate, t)

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	renderer, err := NewRenderer(template.Dependencies(), true)
	if err != nil {
		t.Fatal(err)
	}

	dependency := template.Dependencies()[0]
	data := []*Service{
		&Service{Node: "consul1"},
		&Service{Node: "consul2"},
	}
	renderer.Receive(dependency, data)

	buff := new(bytes.Buffer)
	renderer.SetDryStream(buff)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: "/out/file.txt",
		},
	}

	if err := renderer.MaybeRender(template, configTemplates); err != nil {
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

func TestMaybeRender_containingFolderMissing(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate, t)

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

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	renderer, err := NewRenderer(template.Dependencies(), false)
	if err != nil {
		t.Fatal(err)
	}

	dependency := template.Dependencies()[0]
	data := []*Service{
		&Service{Node: "consul1"},
		&Service{Node: "consul2"},
	}
	renderer.Receive(dependency, data)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outFile.Name(),
		},
	}

	if err := renderer.MaybeRender(template, configTemplates); err != nil {
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

func TestMaybeRender_outputFileMissing(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate, t)

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

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	renderer, err := NewRenderer(template.Dependencies(), false)
	if err != nil {
		t.Fatal(err)
	}

	dependency := template.Dependencies()[0]
	data := []*Service{
		&Service{Node: "consul1"},
		&Service{Node: "consul2"},
	}
	renderer.Receive(dependency, data)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outFile.Name(),
		},
	}

	if err := renderer.MaybeRender(template, configTemplates); err != nil {
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

func TestMaybeRender_outputFileRetainsPermissions(t *testing.T) {
	inTemplate := createTempfile([]byte(`
    {{range service "consul@nyc1"}}{{.Node}}{{end}}
  `), t)
	defer deleteTempfile(inTemplate, t)

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

	template, err := NewTemplate(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	renderer, err := NewRenderer(template.Dependencies(), false)
	if err != nil {
		t.Fatal(err)
	}

	dependency := template.Dependencies()[0]
	data := []*Service{
		&Service{Node: "consul1"},
		&Service{Node: "consul2"},
	}
	renderer.Receive(dependency, data)

	configTemplates := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      inTemplate.Name(),
			Destination: outFile.Name(),
		},
	}

	if err := renderer.MaybeRender(template, configTemplates); err != nil {
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
