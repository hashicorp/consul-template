package watch

import (
	"reflect"
	"strings"
	"testing"
	"time"

	api "github.com/armon/consul-api"
	"github.com/hashicorp/consul-template/dependency"
)

func TestNewWatcher_noClient(t *testing.T) {
	_, err := NewWatcher(nil, make([]dependency.Dependency, 1))
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
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewWatcher_setsClient(t *testing.T) {
	client := &api.Client{}
	w, err := NewWatcher(client, make([]dependency.Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(w.client, client) {
		t.Errorf("expected %+v to equal %+v", w.client, client)
	}
}

func TestNewWatcher_setsDependencies(t *testing.T) {
	dependencies := []dependency.Dependency{
		&dependency.HealthServices{},
		&dependency.HealthServices{},
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
	w, err := NewWatcher(&api.Client{}, make([]dependency.Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if w.DataCh == nil {
		t.Errorf("expected DataCh to exist")
	}
}

func TestNewWatcher_makesErrCh(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]dependency.Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if w.ErrCh == nil {
		t.Errorf("expected ErrCh to exist")
	}
}

func TestNewWatcher_makesFinishCh(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]dependency.Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if w.FinishCh == nil {
		t.Errorf("expected FinishCh to exist")
	}
}

func TestNewWatcher_makesstopCh(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]dependency.Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if w.stopCh == nil {
		t.Errorf("expected stopCh to exist")
	}
}

func TestNewWatcher_setsRetry(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]dependency.Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	if w.retryFunc == nil {
		t.Errorf("expected retryFunc to exist")
	}
}

func TestSetRetry_setsRetryFunc(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]dependency.Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	retry := 10 * time.Second
	w.SetRetry(retry)
	result := w.retryFunc(0 * time.Second)

	if result != retry {
		t.Errorf("expected %q to be %q", result, retry)
	}
}

func TestSetRetryFunc_setsRetryFunc(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, make([]dependency.Dependency, 1))
	if err != nil {
		t.Fatal(err)
	}

	w.SetRetryFunc(func(current time.Duration) time.Duration {
		return 2 * current
	})

	data := map[time.Duration]time.Duration{
		0 * time.Second: 0 * time.Second,
		1 * time.Second: 2 * time.Second,
		2 * time.Second: 4 * time.Second,
		9 * time.Second: 18 * time.Second,
	}

	for current, expected := range data {
		result := w.retryFunc(current)
		if result != expected {
			t.Errorf("expected %q to be %q", result, expected)
		}
	}
}
