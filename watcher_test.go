package main

import (
	"reflect"
	"strings"
	"testing"
	"time"

	api "github.com/armon/consul-api"
)

func TestNewWatcher_noClient(t *testing.T) {
	_, err := NewWatcher(nil, make([]Dependency, 1))
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "watcher: missing Consul API client"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestNewWatcher_noDependencies(t *testing.T) {
	_, err := NewWatcher(&api.Client{}, nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "watcher: must supply at least one Dependency"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestNewWatcher_setsClient(t *testing.T) {
	client := &api.Client{}
	w, err := NewWatcher(client, make([]Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(w.client, client) {
		t.Errorf("expected %q to equal %q", w.client, client)
	}
}

func TestNewWatcher_setsDependencies(t *testing.T) {
	dependencies := []Dependency{
		&ServiceDependency{},
		&ServiceDependency{},
	}
	w, err := NewWatcher(&api.Client{}, dependencies)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(w.dependencies, dependencies) {
		t.Errorf("expected %q to equal %q", w.dependencies, dependencies)
	}
}

func TestNewWatcher_makesDataCh(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if w.DataCh == nil {
		t.Errorf("expected DataCh to exist")
	}
}

func TestNewWatcher_makesErrCh(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if w.ErrCh == nil {
		t.Errorf("expected ErrCh to exist")
	}
}

func TestNewWatcher_makesstopCh(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if w.stopCh == nil {
		t.Errorf("expected stopCh to exist")
	}
}

func TestWatch_propagatesDependencyFetchError(t *testing.T) {
	dependencies := []Dependency{
		&fakeDependencyFetchError{name: "tester"},
	}
	w, err := NewWatcher(&api.Client{}, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	go w.Watch(true)

	select {
	case data := <-w.DataCh:
		t.Fatalf("expected no data, but got %v", data)
	case err := <-w.ErrCh:
		expected := "failed to contact server"
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected %q to contain %q", err.Error(), expected)
		}
	case <-w.FinishCh:
		t.Fatalf("watcher finished prematurely")
	case <-time.After(1 * time.Second):
		t.Fatal("expected error, but nothing was returned")
	}
}

func TestWatch_fetchesData(t *testing.T) {
	dependencies := []Dependency{
		&fakeDependency{name: "tester"},
	}
	w, err := NewWatcher(&api.Client{}, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	go w.Watch(true)

	select {
	case data := <-w.DataCh:
		if !reflect.DeepEqual(data.dependency, dependencies[0]) {
			t.Error("did not get the correct dependency")
		}
		if data.lastIndex != 1 {
			t.Errorf("expected %d to equal %d", data.lastIndex, 1)
		}
		s, expected := data.data.(string), "this is some data"
		if s != expected {
			t.Errorf("expected %q to equal %q", s, expected)
		}
	case err := <-w.ErrCh:
		t.Fatal(err)
	case <-w.FinishCh:
		t.Fatalf("watcher finished prematurely")
	case <-time.After(1 * time.Second):
		t.Fatal("expected error, but nothing was returned")
	}
}
