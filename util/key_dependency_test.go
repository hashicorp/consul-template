package util

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/consul-template/test"
)

func TestKeyDependencyFetch(t *testing.T) {
	client, options := test.DemoConsulClient(t)
	dep := &KeyDependency{
		rawKey: "global/time",
		Path:   "global/time",
	}

	results, _, err := dep.Fetch(client, options)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.(string)
	if !ok {
		t.Fatal("could not convert result to string")
	}
}

func TestKeyDependencyHashCode_isUnique(t *testing.T) {
	dep1 := &KeyDependency{rawKey: "config/redis/maxconns"}
	dep2 := &KeyDependency{rawKey: "config/redis/minconns"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseKeyDependency_emptyString(t *testing.T) {
	_, err := ParseKeyDependency("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "cannot specify empty key dependency"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseKeyDependency_name(t *testing.T) {
	sd, err := ParseKeyDependency("config/redis/maxconns")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyDependency{
		rawKey: "config/redis/maxconns",
		Path:   "config/redis/maxconns",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseKeyDependency_nameColon(t *testing.T) {
	sd, err := ParseKeyDependency("config/redis:magic:80/maxconns")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyDependency{
		rawKey: "config/redis:magic:80/maxconns",
		Path:   "config/redis:magic:80/maxconns",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseKeyDependency_nameTagDataCenter(t *testing.T) {
	sd, err := ParseKeyDependency("config/redis/maxconns@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &KeyDependency{
		rawKey:     "config/redis/maxconns@nyc1",
		Path:       "config/redis/maxconns",
		DataCenter: "nyc1",
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}
