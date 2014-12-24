package util

import (
	"reflect"
	"testing"

	"github.com/hashicorp/consul-template/test"
)

func TestKeyPrefixFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
	dep := &KeyPrefix{
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

func TestKeyPrefixHashCode_isUnique(t *testing.T) {
	dep1 := &KeyPrefix{rawKey: "config/redis"}
	dep2 := &KeyPrefix{rawKey: "config/consul"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseKeyPrefix_emptyString(t *testing.T) {
	kpd, err := ParseKeyPrefix("")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefix{}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestParseKeyPrefix_name(t *testing.T) {
	kpd, err := ParseKeyPrefix("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefix{
		rawKey: "config/redis",
		Prefix: "config/redis",
	}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestParseKeyPrefix_nameColon(t *testing.T) {
	sd, err := ParseKeyPrefix("config/redis:magic:80")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefix{
		rawKey: "config/redis:magic:80",
		Prefix: "config/redis:magic:80",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseKeyPrefix_nameTagDataCenter(t *testing.T) {
	kpd, err := ParseKeyPrefix("config/redis@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefix{
		rawKey:     "config/redis@nyc1",
		Prefix:     "config/redis",
		DataCenter: "nyc1",
	}

	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestParseKeyPrefix_dataCenter(t *testing.T) {
	kpd, err := ParseKeyPrefix("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyPrefix{
		rawKey:     "@nyc1",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}
