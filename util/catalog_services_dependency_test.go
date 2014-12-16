package util

import (
	"reflect"
	"testing"

	"github.com/hashicorp/consul-template/test"
)

func TestCatalogServicesDependencyFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
	dep := &CatalogServicesDependency{
		rawKey: "global",
	}

	results, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.([]*CatalogService)
	if !ok {
		t.Fatal("could not convert result to []*CatalogService")
	}
}

func TestCatalogServicesDependencyHashCode_isUnique(t *testing.T) {
	dep1 := &CatalogServicesDependency{rawKey: ""}
	dep2 := &CatalogServicesDependency{rawKey: "@nyc1"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseCatalogServicesDependency_emptyString(t *testing.T) {
	nd, err := ParseCatalogServicesDependency("")
	if err != nil {
		t.Fatal(err)
	}

	expected := &CatalogServicesDependency{}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}

func TestParseCatalogServicesDependency_dataCenter(t *testing.T) {
	nd, err := ParseCatalogServicesDependency("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &CatalogServicesDependency{
		rawKey:     "@nyc1",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}
