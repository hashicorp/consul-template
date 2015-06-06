package dependency

import (
	"reflect"
	"testing"
)

func TestCatalogSingleNodeFetchForUnknownNode(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep := &CatalogSingleNode{rawKey: "unknownNode"}
	result, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := result.(*CatalogNode)
	if !ok {
		t.Fatal("could not convert result to *CatalogNode")
	}

	if typed != nil {
		t.Fatal("Expecting to get nil for an unknown node")
	}

}

func TestCatalogSingleNodeFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	// AddService does not let me specify a port.
	consul.AddService("z", "passing", []string{"baz"})
	consul.AddService("a", "critical", []string{"foo", "bar"})

	dep := &CatalogSingleNode{}
	result, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := result.(*CatalogNode)
	if !ok {
		t.Fatal("could not convert result to *CatalogNode")
	}

	if typed == nil {
		t.Fatal("Not expecting to get nil for a known node")
	}

	if typed.Node.Node != consul.Config.NodeName {
		t.Errorf("expected %q to be %q", typed.Node.Node, consul.Config.NodeName)
	}
	if typed.Node.Address != "127.0.0.1" {
		t.Errorf("expected %q to be %q", typed.Node.Address, "127.0.0.1")
	}
	if len(typed.Services) != 3 {
		t.Fatalf("expected 3 services got %d", len(typed.Services))
	}

	var s *CatalogNodeService

	s = typed.Services[0]
	if s.Service != "a" {
		t.Errorf("expecting %q to be \"a\"", s.Service)
	}
	if s.Port != 0 {
		t.Errorf("expecting %d to be 0", s.Port)
	}
	if len(s.Tags) != 2 {
		t.Fatalf("expecting %d to be 2", len(s.Tags))
	}
	if s.Tags[0] != "bar" {
		t.Errorf("expecting %q to be \"bar\"", s.Tags[0])
	}
	if s.Tags[1] != "foo" {
		t.Errorf("expecting %q to be \"foo\"", s.Tags[1])
	}

	s = typed.Services[1]
	if s.Service != "consul" {
		t.Errorf("expecting %q to be \"consul\"", s.Service)
	}
	if s.Port != consul.Config.Ports.Server {
		t.Errorf("expecting %d to be %d", s.Port, consul.Config.Ports.Server)
	}
	if len(s.Tags) != 0 {
		t.Fatalf("expecting %d to be 0", len(s.Tags))
	}

	s = typed.Services[2]
	if s.Service != "z" {
		t.Errorf("expecting %q to be \"z\"", s.Service)
	}
	if s.Port != 0 {
		t.Errorf("expecting %d to be 0", s.Port)
	}
	if len(s.Tags) != 1 {
		t.Fatalf("expecting %d to be 1", len(s.Tags))
	}
	if s.Tags[0] != "baz" {
		t.Errorf("expecting %q to be \"baz\"", s.Tags[0])
	}

}

func TestCatalogSingleNodeFetchWithNameArgument(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep := &CatalogSingleNode{
		rawKey: consul.Config.NodeName,
	}
	result, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := result.(*CatalogNode)
	if !ok {
		t.Fatal("could not convert result to *CatalogNode")
	}

	if typed == nil {
		t.Fatal("Not expecting to get nil for a known node")
	}

	if typed.Node.Node != consul.Config.NodeName {
		t.Errorf("expected %q to be %q", typed.Node.Node, consul.Config.NodeName)
	}
	if typed.Node.Address != "127.0.0.1" {
		t.Errorf("expected %q to be %q", typed.Node.Address, "127.0.0.1")
	}
	if len(typed.Services) != 1 {
		t.Fatalf("expected 1 services got %d", len(typed.Services))
	}

	var s *CatalogNodeService

	s = typed.Services[0]
	if s.Service != "consul" {
		t.Errorf("expecting %q to be \"consul\"", s.Service)
	}
	if s.Port != consul.Config.Ports.Server {
		t.Errorf("expecting %d to be %d", s.Port, consul.Config.Ports.Server)
	}
	if len(s.Tags) != 0 {
		t.Fatalf("expecting %d to be 0", len(s.Tags))
	}

}

func TestCatalogSingleNodeHashCode_isUnique(t *testing.T) {
	dep1 := &CatalogSingleNode{rawKey: ""}
	dep2 := &CatalogSingleNode{rawKey: "node"}
	dep3 := &CatalogSingleNode{rawKey: "", dataCenter: "@nyc1"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
	if dep1.HashCode() == dep3.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
	if dep2.HashCode() == dep3.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseCatalogSingleNodeNoArguments(t *testing.T) {
	nd, err := ParseCatalogSingleNode()
	if err != nil {
		t.Fatal(err)
	}

	expected := &CatalogSingleNode{}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}

func TestParseCatalogSingleNodeOneArgument(t *testing.T) {
	nd, err := ParseCatalogSingleNode("node")
	if err != nil {
		t.Fatal(err)
	}

	expected := &CatalogSingleNode{
		rawKey: "node",
	}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}

func TestParseCatalogSingleNodeTwoArguments(t *testing.T) {
	nd, err := ParseCatalogSingleNode("node", "@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &CatalogSingleNode{
		rawKey:     "node",
		dataCenter: "nyc1",
	}
	if !reflect.DeepEqual(nd, expected) {
		t.Errorf("expected %+v to equal %+v", nd, expected)
	}
}
