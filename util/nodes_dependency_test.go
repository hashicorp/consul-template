package util

import (
	"reflect"
	"testing"

	"github.com/hashicorp/consul-template/test"
)

func TestNodesDependencyFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
	dep := &NodesDependency{
		rawKey: "global",
	}

	results, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.([]*Node)
	if !ok {
		t.Fatal("could not convert result to []*Node")
	}
}

func TestNodesDependencyHashCode_isUnique(t *testing.T) {
	dep1 := &NodesDependency{rawKey: ""}
	dep2 := &NodesDependency{rawKey: "@nyc1"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseNodesDependency_emptyString(t *testing.T) {
	nd, err := ParseNodesDependency("")
	if err != nil {
		t.Fatal(err)
	}

	expected := &NodesDependency{}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}

func TestParseNodesDependency_dataCenter(t *testing.T) {
	nd, err := ParseNodesDependency("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &NodesDependency{
		rawKey:     "@nyc1",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}
