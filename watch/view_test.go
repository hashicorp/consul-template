package watch

import (
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
)

// testRetryFunc is a function specifically for tests that has a 0-time retry.
var testRetryFunc = func(time.Duration) time.Duration { return 0 }

func TestNewView_noConfig(t *testing.T) {
	_, err := NewView(nil, &test.FakeDependency{})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "view: missing config"
	if err.Error() != expected {
		t.Errorf("expected %q to eq %q", err.Error(), expected)
	}
}

func TestNewView_noDependency(t *testing.T) {
	_, err := NewView(defaultWatcherConfig, nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "view: missing dependency"
	if err.Error() != expected {
		t.Errorf("expected %q to eq %q", err.Error(), expected)
	}
}

func TestNewView_setsValues(t *testing.T) {
	config, dep := defaultWatcherConfig, &test.FakeDependency{}
	view, err := NewView(config, dep)
	if err != nil {
		t.Fatal(err)
	}

	if view.config != config {
		t.Errorf("expected %+v to be %+v", view.config, config)
	}

	if view.Dependency != dep {
		t.Errorf("expected %+v to be %+v", view.Dependency, dep)
	}
}

func TestPoll_returnsViewCh(t *testing.T) {
	view, err := NewView(defaultWatcherConfig, &test.FakeDependency{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)

	go view.poll(viewCh, errCh)
	defer view.stop()

	select {
	case <-viewCh:
		// Got this far, so the test passes
	case err := <-errCh:
		t.Errorf("error while polling: %s", err)
	case <-view.stopCh:
		t.Errorf("poll received premature stop")
	}
}

func TestPoll_noReturnErrCh(t *testing.T) {
	view, err := NewView(defaultWatcherConfig, &test.FakeDependencyFetchError{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)

	go view.poll(viewCh, errCh)
	defer view.stop()

	select {
	case data := <-viewCh:
		t.Errorf("expected no data, but got %+v", data)
	case err := <-errCh:
		t.Errorf("expected no error, but got %s", err)
	case <-view.stopCh:
		t.Errorf("poll received premature stop")
	case <-time.After(20 * time.Millisecond):
		// No data was received, test passes
	}
}

func TestPoll_stopsViewStopCh(t *testing.T) {
	view, err := NewView(defaultWatcherConfig, &test.FakeDependency{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)

	go view.poll(viewCh, errCh)
	view.stop()

	select {
	case <-viewCh:
		t.Errorf("expected no data, but received view data")
	case err := <-errCh:
		t.Errorf("error while polling: %s", err)
	case <-time.After(20 * time.Millisecond):
		// No data was received, test passes
	}
}

func TestPoll_once(t *testing.T) {
	view, err := NewView(defaultWatcherConfig, &test.FakeDependency{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)

	go view.poll(viewCh, errCh)
	defer view.stop()

	select {
	case <-viewCh:
		// Got this far, so the test passes
	case err := <-errCh:
		t.Errorf("error while polling: %s", err)
	case <-view.stopCh:
		t.Errorf("poll received premature stop")
	}

	select {
	case <-viewCh:
		t.Errorf("expected no data (should have stopped), but received view data")
	case err := <-errCh:
		t.Errorf("error while polling: %s", err)
	case <-view.stopCh:
		t.Errorf("poll received premature stop")
	case <-time.After(20 * time.Millisecond):
		// No data in 0.2s, so the test passes
	}
}

func TestPoll_retries(t *testing.T) {
	view, err := NewView(defaultWatcherConfig, &test.FakeDependencyFetchRetry{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)

	go view.poll(viewCh, errCh)
	defer view.stop()

	select {
	case <-viewCh:
		// Got this far, so the test passes
	case err := <-errCh:
		t.Errorf("error while polling: %s", err)
	case <-view.stopCh:
		t.Errorf("poll received premature stop")
	}
}

func TestFetch_maxStale(t *testing.T) {
	config := defaultWatcherConfig
	config.MaxStale = 10 * time.Millisecond

	view, err := NewView(config, &test.FakeDependencyStale{})
	if err != nil {
		t.Fatal(err)
	}

	doneCh := make(chan struct{})
	errCh := make(chan error)

	go view.fetch(doneCh, errCh)

	select {
	case <-doneCh:
		expected := "this is some fresh data"
		if !reflect.DeepEqual(view.Data, expected) {
			t.Errorf("expected %q to be %q", view.Data, expected)
		}
	case err := <-errCh:
		t.Errorf("error while fetching: %s", err)
	}
}

func TestFetch_savesView(t *testing.T) {
	view, err := NewView(defaultWatcherConfig, &test.FakeDependency{})
	if err != nil {
		t.Fatal(err)
	}

	doneCh := make(chan struct{})
	errCh := make(chan error)

	go view.fetch(doneCh, errCh)

	select {
	case <-doneCh:
		expected := "this is some data"
		if !reflect.DeepEqual(view.Data, expected) {
			t.Errorf("expected %q to be %q", view.Data, expected)
		}
	case err := <-errCh:
		t.Errorf("error while fetching: %s", err)
	}
}

func TestFetch_returnsErrCh(t *testing.T) {
	view, err := NewView(defaultWatcherConfig, &test.FakeDependencyFetchError{})
	if err != nil {
		t.Fatal(err)
	}

	doneCh := make(chan struct{})
	errCh := make(chan error)

	go view.fetch(doneCh, errCh)

	select {
	case <-doneCh:
		t.Errorf("expected error, but received doneCh")
	case err := <-errCh:
		expected := "failed to contact server"
		if err.Error() != expected {
			t.Fatalf("expected error %q to be %q", err.Error(), expected)
		}
	}
}

func TestStop_stopsPolling(t *testing.T) {
	view, err := NewView(defaultWatcherConfig, &test.FakeDependency{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)

	go view.poll(viewCh, errCh)
	view.stop()

	select {
	case view := <-viewCh:
		t.Errorf("got unexpected view: %#v", view)
	case err := <-errCh:
		t.Error(err)
	case <-view.stopCh:
		// Successfully stopped
	}
}
