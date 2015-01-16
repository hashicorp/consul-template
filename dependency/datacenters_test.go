package dependency

import (
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
)

func TestDatacentersFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
	dep := &Datacenters{rawKey: ""}

	results, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.([]string)
	if !ok {
		t.Fatal("could not convert result to []string")
	}
}

func TestDatacentersFetch_blocks(t *testing.T) {
	client, options := test.DemoConsulClient(t)
	dep := &Datacenters{rawKey: ""}

	_, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan struct{})
	go func() {
		dep.Fetch(client, options)
	}()

	select {
	case <-dataCh:
		t.Errorf("expected query to block")
	case <-time.After(500 * time.Millisecond):
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
