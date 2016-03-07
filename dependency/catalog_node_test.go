package dependency

import (
	"sort"
	"testing"
	"time"
)

func TestCatalogNodeFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	// AddService does not let me specify an ID or a port.
	consul.AddService("z", "passing", []string{"baz"})
	consul.AddService("a", "critical", []string{"foo", "bar"})

	dep, err := ParseCatalogNode()
	if err != nil {
		t.Fatal(err)
	}

	result, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := result.(*NodeDetail)
	if !ok {
		t.Fatal("could not convert result to *NodeDetail")
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

	var s *NodeService

	s = typed.Services[0]
	if s.ID != "a" {
		t.Errorf("expecting %q to be \"a\"", s.ID)
	}
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
	if s.ID != "consul" {
		t.Errorf("expecting %q to be \"consul\"", s.ID)
	}
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
	if s.ID != "z" {
		t.Errorf("expecting %q to be \"z\"", s.ID)
	}
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

func TestCatalogNodeFetch_stopped(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	// AddService does not let me specify an ID or a port.
	consul.AddService("z", "passing", []string{"baz"})
	consul.AddService("a", "critical", []string{"foo", "bar"})

	dep, err := ParseCatalogNode()
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error)
	go func() {
		results, _, err := dep.Fetch(clients, &QueryOptions{WaitIndex: 100})
		if results != nil {
			t.Fatalf("should not get results: %#v", results)
		}
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

func TestCatalogNodeFetch_unknownNode(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseCatalogNode("unknownNode")
	if err != nil {
		t.Fatal(err)
	}

	result, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := result.(*NodeDetail)
	if !ok {
		t.Fatal("could not convert result to *NodeDetail")
	}

	if typed != nil {
		t.Fatal("Expecting to get nil for an unknown node")
	}
}

func TestCatalogNodeFetch_nameArgument(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseCatalogNode(consul.Config.NodeName)
	if err != nil {
		t.Fatal(err)
	}

	result, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := result.(*NodeDetail)
	if !ok {
		t.Fatal("could not convert result to *NodeDetail")
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

	var s *NodeService

	s = typed.Services[0]
	if s.ID != "consul" {
		t.Errorf("expecting %q to be \"consul\"", s.ID)
	}
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

func TestCatalogNodeHashCode_isUnique(t *testing.T) {
	dep1, err := ParseCatalogNode("")
	if err != nil {
		t.Fatal(err)
	}

	dep2, err := ParseCatalogNode("node")
	if err != nil {
		t.Fatal(err)
	}

	dep3, err := ParseCatalogNode("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

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

func TestParseCatalogNodeNoArguments(t *testing.T) {
	nd, err := ParseCatalogNode()
	if err != nil {
		t.Fatal(err)
	}

	if nd.rawKey != "" {
		t.Errorf("expected %q to be %q", nd.rawKey, "")
	}

	if nd.dataCenter != "" {
		t.Errorf("expected %q to be %q", nd.dataCenter, "")
	}
}

func TestParseCatalogNodeOneArgument(t *testing.T) {
	nd, err := ParseCatalogNode("node")
	if err != nil {
		t.Fatal(err)
	}

	if nd.rawKey != "node" {
		t.Errorf("expected %q to be %q", nd.rawKey, "node")
	}

	if nd.dataCenter != "" {
		t.Errorf("expected %q to be %q", nd.dataCenter, "")
	}
}

func TestParseCatalogNodeTwoArguments(t *testing.T) {
	nd, err := ParseCatalogNode("node", "@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if nd.rawKey != "node" {
		t.Errorf("expected %q to be %q", nd.rawKey, "node")
	}

	if nd.dataCenter != "nyc1" {
		t.Errorf("expected %q to be %q", nd.dataCenter, "nyc1")
	}
}

func TestNodeServiceListSort(t *testing.T) {
	services := make(NodeServiceList, 0, 2)
	services = append(services, &NodeService{
		ID:      "s-m",
		Service: "z",
		Tags:    make(ServiceTags, 0),
		Port:    3000,
	})
	services = append(services, &NodeService{
		ID:      "s-z",
		Service: "s",
		Tags:    make(ServiceTags, 0),
		Port:    2000,
	})
	services = append(services, &NodeService{
		ID:      "s-a",
		Service: "s",
		Tags:    make(ServiceTags, 0),
		Port:    1000,
	})
	sort.Stable(services)

	var s *NodeService

	s = services[0]
	if s.ID != "s-a" {
		t.Errorf("expecting %q to be \"s-a\"", s.ID)
	}
	if s.Service != "s" {
		t.Errorf("expecting %q to be \"s\"", s.Service)
	}

	s = services[1]
	if s.ID != "s-z" {
		t.Errorf("expecting %q to be \"s-z\"", s.ID)
	}
	if s.Service != "s" {
		t.Errorf("expecting %q to be \"s\"", s.Service)
	}

	s = services[2]
	if s.ID != "s-m" {
		t.Errorf("expecting %q to be \"s-m\"", s.ID)
	}
	if s.Service != "z" {
		t.Errorf("expecting %q to be \"z\"", s.Service)
	}
}
