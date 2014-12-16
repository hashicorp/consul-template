package util

import (
	"reflect"
	"testing"

	"github.com/hashicorp/consul-template/test"
)

func TestKeyPrefixDependencyFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
	dep := &KeyPrefixDependency{
		rawKey: "global",
		Prefix: "global",
	}

	results, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.([]*KeyPair)
	if !ok {
		t.Fatal("could not convert result to []*KeyPair")
	}
}

func TestKeyPrefixDependencyHashCode_isUnique(t *testing.T) {
	dep1 := &KeyPrefixDependency{rawKey: "config/redis"}
	dep2 := &KeyPrefixDependency{rawKey: "config/consul"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseKeyPrefixDependency_emptyString(t *testing.T) {
	kpd, err := ParseKeyPrefixDependency("")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefixDependency{}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestParseKeyPrefixDependency_name(t *testing.T) {
	kpd, err := ParseKeyPrefixDependency("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefixDependency{
		rawKey: "config/redis",
		Prefix: "config/redis",
	}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestParseKeyPrefixDependency_nameColon(t *testing.T) {
	sd, err := ParseKeyPrefixDependency("config/redis:magic:80")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefixDependency{
		rawKey: "config/redis:magic:80",
		Prefix: "config/redis:magic:80",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseKeyPrefixDependency_nameTagDataCenter(t *testing.T) {
	kpd, err := ParseKeyPrefixDependency("config/redis@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefixDependency{
		rawKey:     "config/redis@nyc1",
		Prefix:     "config/redis",
		DataCenter: "nyc1",
	}

	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestParseKeyPrefixDependency_dataCenter(t *testing.T) {
	kpd, err := ParseKeyPrefixDependency("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefixDependency{
		rawKey:     "@nyc1",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}
