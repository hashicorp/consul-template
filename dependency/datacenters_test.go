package dependency

import (
	"testing"
	"time"
)

func TestDatacentersFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseDatacenters()
	if err != nil {
		t.Fatal(err)
	}

	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.([]string)
	if !ok {
		t.Fatal("could not convert result to []string")
	}
}

func TestDatacentersFetch_blocks(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseDatacenters()
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan struct{})
	go func() {
		_, _, err := dep.Fetch(clients, &QueryOptions{WaitIndex: 1})
		if err != nil {
			t.Fatal(err)
		}
		close(dataCh)
	}()

	select {
	case <-dataCh:
		t.Errorf("expected query to block")
	case <-time.After(50 * time.Millisecond):
		// Test pases
	}
}

func TestDatacentersFetch_stopped(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseDatacenters()
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error)
	go func() {
		_, _, err := dep.Fetch(clients, &QueryOptions{WaitIndex: 1})
		errCh <- err
	}()

	dep.Stop()

	select {
	case err := <-errCh:
		if err != ErrStopped {
			t.Errorf("expected %q to be %q", err, ErrStopped)
		}
	case <-time.After(50 * time.Millisecond):
		t.Errorf("did not return in 50ms")
	}
}
