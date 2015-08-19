package watch

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
)

var defaultWatcherConfig = &WatcherConfig{
	Clients:   dep.NewClientSet(),
	Once:      true,
	RetryFunc: func(time.Duration) time.Duration { return 0 },
}

func TestNewWatcher_noConfig(t *testing.T) {
	_, err := NewWatcher(nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "watcher: missing config"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestNewWatcher_defaultValues(t *testing.T) {
	w, err := NewWatcher(&WatcherConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if w.config.RetryFunc == nil {
		t.Errorf("expected RetryFunc to not be nil")
	}

	if w.DataCh == nil {
		t.Errorf("expected DataCh to exist")
	}

	if size := cap(w.DataCh); size != dataBufferSize {
		t.Errorf("expected DataCh to have %d buffer, but was %d", dataBufferSize, size)
	}

	if w.ErrCh == nil {
		t.Errorf("expected ErrCh to exist")
	}

	if w.FinishCh == nil {
		t.Errorf("expected FinishCh to exist")
	}

	if w.depViewMap == nil {
		t.Errorf("expected depViewMap to exist")
	}
}

func TestNewWatcher_values(t *testing.T) {
	clients := dep.NewClientSet()

	w, err := NewWatcher(&WatcherConfig{
		Clients: clients,
		Once:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(w.config.Clients, clients) {
		t.Errorf("expected %#v to be %#v", w.config.Clients, clients)
	}

	if w.config.Once != true {
		t.Errorf("expected w.config.Once to be true")
	}
}

func TestNewWatcher_renewVault(t *testing.T) {
	clients := dep.NewClientSet()

	w, err := NewWatcher(&WatcherConfig{
		Clients:    clients,
		Once:       true,
		RenewVault: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if !w.Watching(new(dep.VaultToken)) {
		t.Errorf("expected watcher to be renewing vault token")
	}
}

func TestAdd_updatesMap(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	d := &dep.Test{}
	if _, err := w.Add(d); err != nil {
		t.Fatal(err)
	}

	_, exists := w.depViewMap[d.HashCode()]
	if !exists {
		t.Errorf("expected Add to append to map")
	}
}

func TestAdd_exists(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	d := &dep.Test{}
	w.depViewMap[d.HashCode()] = &View{}

	added, err := w.Add(d)
	if err != nil {
		t.Fatal(err)
	}

	if added != false {
		t.Errorf("expected Add to return false")
	}
}

func TestAdd_error(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	// Set the client to nil to force the view to return an error
	w.config = nil

	added, err := w.Add(&dep.Test{})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "view: missing config"
	if err.Error() != expected {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}

	if added != false {
		t.Errorf("expected Add to return false")
	}
}

func TestAdd_startsViewPoll(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	added, err := w.Add(&dep.Test{})
	if err != nil {
		t.Fatal(err)
	}

	if added != true {
		t.Errorf("expected Add to return true")
	}

	select {
	case err := <-w.ErrCh:
		t.Fatal(err)
	case <-w.DataCh:
		// Got data, which means the poll was started
	}
}

func TestWatching_notExists(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	d := &dep.Test{}
	if w.Watching(d) == true {
		t.Errorf("expected to not be watching")
	}
}

func TestWatching_exists(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	d := &dep.Test{}
	if _, err := w.Add(d); err != nil {
		t.Fatal(err)
	}

	if w.Watching(d) == false {
		t.Errorf("expected to be watching")
	}
}

func TestRemove_exists(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	d := &dep.Test{}
	if _, err := w.Add(d); err != nil {
		t.Fatal(err)
	}

	removed := w.Remove(d)
	if removed != true {
		t.Error("expected Remove to return true")
	}

	if _, ok := w.depViewMap[d.HashCode()]; ok {
		t.Error("expected dependency to be removed")
	}
}

func TestRemove_doesNotExist(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	removed := w.Remove(&dep.Test{})
	if removed != false {
		t.Fatal("expected Remove to return false")
	}
}

func TestSize_empty(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	if w.Size() != 0 {
		t.Errorf("expected %d to be %d", w.Size(), 0)
	}
}

func TestSize_returnsNumViews(t *testing.T) {
	w, err := NewWatcher(defaultWatcherConfig)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		d := &dep.Test{Name: fmt.Sprintf("%d", i)}
		if _, err := w.Add(d); err != nil {
			t.Fatal(err)
		}
	}

	if w.Size() != 10 {
		t.Errorf("expected %d to be %d", w.Size(), 10)
	}
}
