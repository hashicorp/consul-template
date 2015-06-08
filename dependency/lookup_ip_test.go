package dependency

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestLookupIPFetch(t *testing.T) {
	dep := &LookupIP{
		spec:   "LookupIP",
		rawKey: "some.host",
		lookup: func(string) ([]net.IP, error) {
			return []net.IP{net.IPv6loopback, net.IPv4(127, 0, 0, 1)}, nil
		},
	}

	results, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	ip, ok := results.([]net.IP)
	if !ok {
		t.Fatal("could not convert result to []net.IP")
	}

	if len(ip) != 2 {
		t.Fatalf("Expecting ip slice to contain 2 items, got %v.", ip)
	}

	if !bytes.Equal(ip[0], net.IPv6loopback) {
		t.Fatalf("expected %v to be %v", ip[0], net.IPv6loopback)
	}
	if !bytes.Equal(ip[1], net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("expected %v to be %v", ip[1], net.IPv4(127, 0, 0, 1))
	}
}

func TestLookupIPFetch_waits(t *testing.T) {
	dep := &LookupIP{
		spec:   "LookupIP",
		rawKey: "some.host",
		lookup: func(string) ([]net.IP, error) {
			return []net.IP{net.IPv6loopback, net.IPv4(127, 0, 0, 1)}, nil
		},
	}

	_, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan interface{})
	go func(c chan<- interface{}) {
		data, _, err := dep.Fetch(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		c <- data
	}(dataCh)

	select {
	case <-dataCh:
		t.Fatal("received data, but should not have")
	case <-time.After(1000 * time.Nanosecond):
		return
	}
}

func TestLookupIPFetch_firesChanges(t *testing.T) {
	dep := &LookupIP{
		spec:   "LookupIP",
		rawKey: "some.host",
		lookup: func(string) ([]net.IP, error) {
			return []net.IP{net.IPv6loopback, net.IPv4(127, 0, 0, 1)}, nil
		},
	}

	_, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan interface{})
	go func(c chan<- interface{}) {
		data, _, err := dep.Fetch(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		c <- data
	}(dataCh)

	dep.lookup = func(string) ([]net.IP, error) {
		return []net.IP{net.IPv4(127, 0, 0, 1)}, nil
	}

	select {
	case d := <-dataCh:

		ip, ok := d.([]net.IP)
		if !ok {
			t.Fatal("could not convert result to []net.IP")
		}

		if len(ip) != 1 {
			t.Fatalf("Expecting ip slice to contain 1 item, got %v.", ip)
		}

		if !bytes.Equal(ip[0], net.IPv4(127, 0, 0, 1)) {
			t.Fatalf("expected %v to be %v", ip[0], net.IPv4(127, 0, 0, 1))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("did not receive data from ip lookup changes")
	}
}

func TestLookupIPFetch_doesNotFireChangeWhenIPAreInDifferentOrder(t *testing.T) {
	dep := &LookupIP{
		spec:   "LookupIP",
		rawKey: "some.host",
		lookup: func(string) ([]net.IP, error) {
			return []net.IP{net.IPv6loopback, net.IPv4(127, 0, 0, 1)}, nil
		},
	}

	_, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan interface{})
	go func(c chan<- interface{}) {
		data, _, err := dep.Fetch(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		c <- data
	}(dataCh)

	dep.lookup = func(string) ([]net.IP, error) {
		return []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}, nil
	}

	select {
	case <-dataCh:
		t.Fatal("received data from ip lookup changes")
	case <-time.After(1 * time.Second):
		return
	}
}

func TestOnlyIPv4ReturnsCorrectItems(t *testing.T) {
	ip, err := onlyIPv4([]net.IP{net.IPv6loopback, net.IPv4(127, 0, 0, 1)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ip) != 1 {
		t.Fatalf("Expecting ip slice to contain 1 item, got %v.", ip)
	}
	if !bytes.Equal(ip[0], net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("Expecting ip slice to only contain IPv4 items, got %v.", ip)
	}
}

func TestOnlyIPv6ReturnsCorrectItems(t *testing.T) {
	ip, err := onlyIPv6([]net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ip) != 1 {
		t.Fatalf("Expecting ip slice to contain 1 item, got %v.", ip)
	}
	if !bytes.Equal(ip[0], net.IPv6loopback) {
		t.Fatalf("Expecting ip slice to only contain IPv6 items, got %v.", ip)
	}
}
