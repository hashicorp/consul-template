package dependency

import (
	"reflect"
	"testing"

	"github.com/marouenj/consul-template/test"
)

func TestCatalogServicesFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
	dep := &CatalogServices{
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

func TestCatalogServicesHashCode_isUnique(t *testing.T) {
	dep1 := &CatalogServices{rawKey: ""}
	dep2 := &CatalogServices{rawKey: "@nyc1"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseCatalogServices_emptyString(t *testing.T) {
	nd, err := ParseCatalogServices("")
	if err != nil {
		t.Fatal(err)
	}

	expected := &CatalogServices{}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}

func TestParseCatalogServices_dataCenter(t *testing.T) {
	nd, err := ParseCatalogServices("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &CatalogServices{
		rawKey:     "@nyc1",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}
