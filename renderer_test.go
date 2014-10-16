package main

import (
	"bytes"
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

	var buff = new(bytes.Buffer)
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

func TestMaybeRender(t *testing.T) {

}
