package dependency

import (
	"reflect"
	"testing"
)

func TestStoreKeyPrefixFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	consul.SetKV("foo/bar", []byte("zip"))
	consul.SetKV("foo/zip", []byte("zap"))

	dep := &StoreKeyPrefix{rawKey: "foo", Prefix: "foo"}
	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := results.([]*KeyPair)
	if !ok {
		t.Fatal("could not convert result to []*KeyPair")
	}

	if typed[0].Value != "zip" {
		t.Errorf("expected %q to be %q", typed[0].Value, "zip")
	}

	if typed[1].Value != "zap" {
		t.Errorf("expected %q to be %q", typed[0].Value, "zap")
	}
}

func TestStoreKeyPrefixHashCode_isUnique(t *testing.T) {
	dep1 := &StoreKeyPrefix{rawKey: "config/redis"}
	dep2 := &StoreKeyPrefix{rawKey: "config/consul"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseStoreKeyPrefix_emptyString(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("")
	if err != nil {
		t.Fatal(err)
	}

	expected := &StoreKeyPrefix{}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestParseStoreKeyPrefix_name(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	expected := &StoreKeyPrefix{
		rawKey: "config/redis",
		Prefix: "config/redis",
	}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestParseStoreKeyPrefix_nameColon(t *testing.T) {
	sd, err := ParseStoreKeyPrefix("config/redis:magic:80")
	if err != nil {
		t.Fatal(err)
	}

	expected := &StoreKeyPrefix{
		rawKey: "config/redis:magic:80",
		Prefix: "config/redis:magic:80",
	}
	if !reflect.DeepEqual(sd, expected) {
		t.Errorf("expected %+v to equal %+v", sd, expected)
	}
}

func TestParseStoreKeyPrefix_nameTagDataCenter(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("config/redis@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &StoreKeyPrefix{
		rawKey:     "config/redis@nyc1",
		Prefix:     "config/redis",
		DataCenter: "nyc1",
	}

	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}

func TestParseStoreKeyPrefix_dataCenter(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &StoreKeyPrefix{
		rawKey:     "@nyc1",
		DataCenter: "nyc1",
	}
	if !reflect.DeepEqual(kpd, expected) {
		t.Errorf("expected %+v to equal %+v", kpd, expected)
	}
}
