package dependency

import (
	"strings"
	"testing"
	"time"
)

func TestStoreKeyFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	consul.SetKV("foo", []byte("bar"))

	dep, err := ParseStoreKey("foo")
	if err != nil {
		t.Fatal(err)
	}

	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.(string)
	if !ok {
		t.Fatal("could not convert result to string")
	}
}

func TestStoreKeyFetch_stopped(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	consul.SetKV("foo", []byte("bar"))

	dep, err := ParseStoreKey("foo")
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

func TestStoreKeySetDefault(t *testing.T) {
	dep, err := ParseStoreKey("conns")
	if err != nil {
		t.Fatal(err)
	}
	dep.SetDefault("3")

	if dep.defaultValue != "3" {
		t.Errorf("expected %q to be %q", dep.defaultValue, "3")
	}
}

func TestStoreKeyDisplay_includesDefault(t *testing.T) {
	dep, err := ParseStoreKey("conns")
	if err != nil {
		t.Fatal(err)
	}
	dep.SetDefault("3")

	expected := `"keyOrDefault(conns, "3")"`
	if dep.Display() != expected {
		t.Errorf("expected %q to be %q", dep.Display(), expected)
	}
}

func TestStoreKeyHashCode_isUnique(t *testing.T) {
	dep1, err := ParseStoreKey("config/redis/maxconns")
	if err != nil {
		t.Fatal(err)
	}
	dep2, err := ParseStoreKey("config/redis/minconns")
	if err != nil {
		t.Fatal(err)
	}
	dep3, err := ParseStoreKey("config/redis/minconns")
	if err != nil {
		t.Fatal(err)
	}
	dep3.SetDefault("3")

	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
	if dep2.HashCode() == dep3.HashCode() {
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

	if sd.rawKey != "config/redis/maxconns" {
		t.Errorf("expected %q to be %q", sd.rawKey, "config/redis/maxconns")
	}

	if sd.Path != "config/redis/maxconns" {
		t.Errorf("expected %q to be %q", sd.Path, "config/redis/maxconns")
	}
}

func TestParseStoreKey_nameSpecialCharacters(t *testing.T) {
	sd, err := ParseStoreKey("config/facet:größe-lf-si@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "config/facet:größe-lf-si@nyc1" {
		t.Errorf("expected %q to be %q", sd.rawKey, "config/facet:größe-lf-si@nyc1")
	}

	if sd.Path != "config/facet:größe-lf-si" {
		t.Errorf("expected %q to be %q", sd.Path, "config/facet:größe-lf-si")
	}

	if sd.DataCenter != "nyc1" {
		t.Errorf("expected %q to be %q", sd.DataCenter, "nyc1")
	}
}

func TestParseStoreKey_nameTagDataCenter(t *testing.T) {
	sd, err := ParseStoreKey("config/redis/maxconns@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if sd.rawKey != "config/redis/maxconns@nyc1" {
		t.Errorf("expected %q to be %q", sd.rawKey, "config/redis/maxconns@nyc1")
	}

	if sd.Path != "config/redis/maxconns" {
		t.Errorf("expected %q to be %q", sd.Path, "config/redis/maxconns")
	}

	if sd.DataCenter != "nyc1" {
		t.Errorf("expected %q to be %q", sd.DataCenter, "nyc1")
	}
}
