package watch

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul/api"
)

func TestNewWatcher_noClient(t *testing.T) {
	_, err := NewWatcher(nil, false)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "watcher: missing Consul API client"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestNewWatcher_setsClient(t *testing.T) {
	client := &api.Client{}
	w, err := NewWatcher(client, false)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(w.client, client) {
		t.Errorf("expected %+v to equal %+v", w.client, client)
	}
}

func TestNewWatcher_setsOnce(t *testing.T) {
	client := &api.Client{}
	w, err := NewWatcher(client, true)
	if err != nil {
		t.Fatal(err)
	}

	if !w.once {
		t.Errorf("expected once to be true")
	}
}

func TestNewWatcher_makesDataCh(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	if w.DataCh == nil {
		t.Errorf("expected DataCh to exist")
	}
}

func TestNewWatcher_makesErrCh(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	if w.ErrCh == nil {
		t.Errorf("expected ErrCh to exist")
	}
}

func TestNewWatcher_makesFinishCh(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	if w.FinishCh == nil {
		t.Errorf("expected FinishCh to exist")
	}
}

func TestNewWatcher_setsRetry(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	if w.retryFunc == nil {
		t.Errorf("expected retryFunc to exist")
	}
}

func TestNewWatcher_makesdepViewMap(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	if w.depViewMap == nil {
		t.Errorf("expected depViewMap to exist")
	}
}

func TestAddDependency_exists(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	dep := &test.FakeDependency{}
	w.depViewMap[dep.HashCode()] = &View{}

	added, err := w.AddDependency(dep)
	if err != nil {
		t.Fatal(err)
	}

	if added != false {
		t.Errorf("expected AddDependency to return false")
	}
}

func TestAddDependency_error(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	// Set the client to nil to force the view to return an error
	w.client = nil

	added, err := w.AddDependency(&test.FakeDependency{})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "view: missing Consul API client"
	if err.Error() != expected {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}

	if added != false {
		t.Errorf("expected AddDependency to return false")
	}
}

func TestAddDependency_startsViewPoll(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	added, err := w.AddDependency(&test.FakeDependency{})
	if err != nil {
		t.Fatal(err)
	}

	if added != true {
		t.Errorf("expected AddDependency to return true")
	}

	select {
	case err := <-w.ErrCh:
		t.Fatal(err)
	case <-w.DataCh:
		// Got data, which means the poll was started
	}
}

func TestRemoveDependency_exists(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	dep := &test.FakeDependency{}

	if _, err := w.AddDependency(dep); err != nil {
		t.Fatal(err)
	}

	removed := w.RemoveDependency(dep)
	if removed == true {
		t.Error("expected RemoveDependency to return true")
	}

	if _, ok := w.depViewMap[dep.HashCode()]; ok {
		t.Error("expected dependency to be removed")
	}
}

func TestRemoveDependency_doesNotExist(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
	if err != nil {
		t.Fatal(err)
	}

	removed := w.RemoveDependency(&test.FakeDependency{})
	if removed != false {
		t.Fatal("expected RemoveDependency to return false")
	}
}

func TestSetRetry_setsRetryFunc(t *testing.T) {
	w, err := NewWatcher(&api.Client{}, false)
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
	w, err := NewWatcher(&api.Client{}, false)
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
