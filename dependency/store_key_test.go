package dependency

import (
	"reflect"
	"strings"
	"testing"
)

func TestStoreKeyFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	consul.SetKV("foo", []byte("bar"))

	dep := &StoreKey{rawKey: "foo", Path: "foo"}
	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.(string)
	if !ok {
		t.Fatal("could not convert result to string")
	}
}

func TestStoreKeyHashCode_isUnique(t *testing.T) {
	dep1 := &StoreKey{rawKey: "config/redis/maxconns"}
	dep2 := &StoreKey{rawKey: "config/redis/minconns"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseStoreKey_emptyString(t *testing.T) {
	_, err := ParseStoreKey("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "cannot specify empty key dependency"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error %q to contain %q", err.Error(), expected)
	}
}

func TestParseStoreKey_name(t *testing.T) {
	sd, err := ParseStoreKey("config/redis/maxconns")
	if err != nil {
		t.Fatal(err)
	}

	expected := &StoreKey{
		rawKey: "config/redis/maxconns",
		Path:   "config/redis/maxconns",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseStoreKey_nameSpecialCharacters(t *testing.T) {
	sd, err := ParseStoreKey("config/facet:größe-lf-si@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &StoreKey{
		rawKey:     "config/facet:größe-lf-si@nyc1",
		Path:       "config/facet:größe-lf-si",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseStoreKey_nameTagDataCenter(t *testing.T) {
	sd, err := ParseStoreKey("config/redis/maxconns@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &StoreKey{
		rawKey:     "config/redis/maxconns@nyc1",
		Path:       "config/redis/maxconns",
		DataCenter: "nyc1",
	}

	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}
