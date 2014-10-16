package main

import (
	"fmt"
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

func TestWatch_propagatesNewWatchDataError(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}
	w.Watch()

	select {
	case err := <-w.ErrCh:
		expected := "watchdata: missing Dependency"
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("expected %q to contain %q", err.Error(), expected)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected error, but nothing was returned")
	}
}

func TestWatch_propagatesDependencyFetchError(t *testing.T) {
	dependencies := []Dependency{
		&FakeDependencyFetchError{name: "tester"},
	}
	w, err := NewWatcher(&api.Client{}, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	w.Watch()

	select {
	case err := <-w.ErrCh:
		expected := "failed to contact server"
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("expected %q to contain %q", err.Error(), expected)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected error, but nothing was returned")
	}
}

func TestWatch_fetchesData(t *testing.T) {
	dependencies := []Dependency{
		&FakeDependency{name: "tester"},
	}
	w, err := NewWatcher(&api.Client{}, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	w.Watch()

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
	case <-time.After(5 * time.Second):
		t.Fatal("expected data, but nothing was returned")
	}

	w.Stop()
}

func TestStop_stopsWatch(t *testing.T) {
	dependencies := []Dependency{
		&FakeDependency{name: "tester"},
	}
	w, err := NewWatcher(&api.Client{}, dependencies)
	if err != nil {
		t.Fatal(err)
	}

	w.Watch()
	w.Stop()

	select {
	case <-w.stopCh:
		break
	case <-time.After(5 * time.Second):
		t.Fatal("expected stop, but nothing was returned")
	}
}

/// ------------------------- ///

type FakeDependencyFetchError struct {
	name string
}

func (d *FakeDependencyFetchError) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	return nil, nil, fmt.Errorf("failed to contact server")
}

func (d *FakeDependencyFetchError) GoString() string {
	return fmt.Sprintf("%#v", d)
}

func (d *FakeDependencyFetchError) HashCode() string {
	return fmt.Sprintf("FakeDependencyFetchError|%s", d.name)
}

func (d *FakeDependencyFetchError) Key() string {
	return d.name
}

/// ------------------------- ///

type FakeDependency struct {
	name string
}

func (d *FakeDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	data := "this is some data"
	qm := &api.QueryMeta{LastIndex: 1}
	return data, qm, nil
}

func (d *FakeDependency) GoString() string {
	return fmt.Sprintf("%#v", d)
}

func (d *FakeDependency) HashCode() string {
	return fmt.Sprintf("FakeDependency|%s", d.name)
}

func (d *FakeDependency) Key() string {
	return d.name
}
