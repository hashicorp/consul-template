package dependency

import (
	"reflect"
	"testing"
)

func TestCatalogServicesFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	consul.AddService("redis", "passing", []string{"master"})

	dep := &CatalogServices{rawKey: "redis"}
	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := results.([]*CatalogService)
	if !ok {
		t.Fatal("could not convert result to []*CatalogService")
	}

	if typed[1].Name != "redis" {
		t.Errorf("expected %q to be %q", typed[1].Name, "redis")
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
