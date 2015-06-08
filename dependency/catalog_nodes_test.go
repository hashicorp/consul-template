package dependency

import (
	"reflect"
	"testing"
)

func TestCatalogNodesFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep := &CatalogNodes{rawKey: "global"}
	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := results.([]*Node)
	if !ok {
		t.Fatal("could not convert result to []*Node")
	}

	if typed[0].Address != "127.0.0.1" {
		t.Errorf("expected %q to be %q", typed[0].Address, "127.0.0.1")
	}
}

func TestCatalogNodesHashCode_isUnique(t *testing.T) {
	dep1 := &CatalogNodes{rawKey: ""}
	dep2 := &CatalogNodes{rawKey: "@nyc1"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseCatalogNodes_emptyString(t *testing.T) {
	nd, err := ParseCatalogNodes("")
	if err != nil {
		t.Fatal(err)
	}

	expected := &CatalogNodes{}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}

func TestParseCatalogNodes_dataCenter(t *testing.T) {
	nd, err := ParseCatalogNodes("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &CatalogNodes{
		rawKey:     "@nyc1",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}
