package dependency

import (
	"reflect"
	"testing"
	"time"
)

func TestDatacentersFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep := &Datacenters{rawKey: ""}
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

	dep := &Datacenters{rawKey: ""}
	_, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan struct{})
	go func() {
		dep.Fetch(clients, nil)
	}()

	select {
	case <-dataCh:
		t.Errorf("expected query to block")
	case <-time.After(50 * time.Millisecond):
		// Test pases
	}
}

func TestParseDatacenters_noArgs(t *testing.T) {
	nd, err := ParseDatacenters()
	if err != nil {
		t.Fatal(err)
	}

	expected := &Datacenters{rawKey: ""}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}
