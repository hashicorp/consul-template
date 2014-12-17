package util

import (
	"reflect"
	"testing"
	"time"

	api "github.com/armon/consul-api"
	"github.com/hashicorp/consul-template/test"
)

func TestNewView_noClient(t *testing.T) {
	_, err := NewView(nil, &test.FakeDependency{})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "view: missing Consul API client"
	if err.Error() != expected {
		t.Errorf("expected %q to eq %q", err.Error(), expected)
	}
}

func TestNewView_noDependency(t *testing.T) {
	_, err := NewView(&api.Client{}, nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "view: missing Dependency"
	if err.Error() != expected {
		t.Errorf("expected %q to eq %q", err.Error(), expected)
	}
}

func TestNewView_setsValues(t *testing.T) {
	client, dep := &api.Client{}, &test.FakeDependency{}
	view, err := NewView(client, dep)
	if err != nil {
		t.Fatal(err)
	}

	if view.client != client {
		t.Errorf("expected %+v to be %+v", view.client, client)
	}

	if view.Dependency != dep {
		t.Errorf("expected %+v to be %+v", view.Dependency, dep)
	}
}

func TestPoll_returnsViewCh(t *testing.T) {
	view, err := NewView(&api.Client{}, &test.FakeDependency{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)
	stopCh := make(chan struct{})

	go view.poll(viewCh, errCh, stopCh, defaultRetryFunc)
	defer close(stopCh)

	select {
	case <-viewCh:
		// Got this far, so the test passes
	case err := <-errCh:
		t.Errorf("error while polling: %s", err)
	case <-stopCh:
		t.Errorf("poll received premature stop")
	}
}

func TestPoll_returnsErrCh(t *testing.T) {
	view, err := NewView(&api.Client{}, &test.FakeDependencyFetchError{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)
	stopCh := make(chan struct{})

	go view.poll(viewCh, errCh, stopCh, defaultRetryFunc)
	defer close(stopCh)

	select {
	case <-viewCh:
		t.Errorf("expected error, but received view data")
	case err := <-errCh:
		expected := "failed to contact server"
		if err.Error() != expected {
			t.Fatalf("expected error %q to be %q", err.Error(), expected)
		}
	case <-stopCh:
		t.Errorf("poll received premature stop")
	}
}

func TestPoll_stopsStopCh(t *testing.T) {
	view, err := NewView(&api.Client{}, &test.FakeDependency{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)
	stopCh := make(chan struct{})

	go view.poll(viewCh, errCh, stopCh, defaultRetryFunc)
	close(stopCh)

	select {
	case <-viewCh:
		t.Errorf("expected no data, but received view data")
	case err := <-errCh:
		t.Errorf("error while polling: %s", err)
	case <-stopCh:
	case <-time.After(100 * time.Millisecond):
		// No data was received, test passes
	}
}

func TestPoll_retries(t *testing.T) {
	view, err := NewView(&api.Client{}, &test.FakeDependencyFetchRetry{})
	if err != nil {
		t.Fatal(err)
	}

	viewCh := make(chan *View)
	errCh := make(chan error)
	stopCh := make(chan struct{})

	go view.poll(viewCh, errCh, stopCh, defaultRetryFunc)
	defer close(stopCh)

	select {
	case <-viewCh:
		t.Errorf("expected no data (yet), but received view data")
	case err := <-errCh:
		expected := "failed to contact server (try again)"
		if err.Error() != expected {
			t.Fatalf("expected error %q to be %q", err.Error(), expected)
		}
	case <-stopCh:
		t.Errorf("poll received premature stop")
	}

	select {
	case <-viewCh:
		// Got this far, so the test passes
	case err := <-errCh:
		t.Errorf("error while polling: %s", err)
	case <-stopCh:
		t.Errorf("poll received premature stop")
	}
}

func TestFetch_savesView(t *testing.T) {
	view, err := NewView(&api.Client{}, &test.FakeDependency{})
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
	view, err := NewView(&api.Client{}, &test.FakeDependencyFetchError{})
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
