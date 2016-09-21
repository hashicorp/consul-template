package dependency

import (
	"testing"
	"time"
)

func TestStoreKeyPrefixFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	consul.SetKV("foo/bar", []byte("zip"))
	consul.SetKV("foo/zip", []byte("zap"))

	dep, err := ParseStoreKeyPrefix("foo")
	if err != nil {
		t.Fatal(err)
	}

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

func TestStoreKeyPrefixFetch_stopped(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	consul.SetKV("foo/bar", []byte("zip"))
	consul.SetKV("foo/zip", []byte("zap"))

	dep, err := ParseStoreKeyPrefix("foo")
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

func TestStoreKeyPrefixHashCode_isUnique(t *testing.T) {
	dep1, err := ParseStoreKeyPrefix("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	dep2, err := ParseStoreKeyPrefix("config/consul")
	if err != nil {
		t.Fatal(err)
	}

	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseStoreKeyPrefix_emptyString(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("")
	if err != nil {
		t.Fatal(err)
	}

	if kpd.rawKey != "" {
		t.Errorf("expected %q to be %q", kpd.rawKey, "")
	}

	if kpd.Prefix != "/" {
		t.Errorf("expected %q to be %q", kpd.Prefix, "")
	}

	if kpd.DataCenter != "" {
		t.Errorf("expected %q to be %q", kpd.DataCenter, "")
	}
}

func TestParseStoreKeyPrefix_name(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	if kpd.rawKey != "config/redis" {
		t.Errorf("expected %q to be %q", kpd.rawKey, "config/redis")
	}

	if kpd.Prefix != "config/redis" {
		t.Errorf("expected %q to be %q", kpd.Prefix, "config/redis")
	}

	if kpd.DataCenter != "" {
		t.Errorf("expected %q to be %q", kpd.DataCenter, "")
	}
}

func TestParseStoreKeyPrefix_nameColon(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("config/redis:magic:80")
	if err != nil {
		t.Fatal(err)
	}

	if kpd.rawKey != "config/redis:magic:80" {
		t.Errorf("expected %q to be %q", kpd.rawKey, "config/redis:magic:80")
	}

	if kpd.Prefix != "config/redis:magic:80" {
		t.Errorf("expected %q to be %q", kpd.Prefix, "config/redis:magic:80")
	}

	if kpd.DataCenter != "" {
		t.Errorf("expected %q to be %q", kpd.DataCenter, "")
	}
}

func TestParseStoreKeyPrefix_nameTagDataCenter(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("config/redis@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if kpd.rawKey != "config/redis@nyc1" {
		t.Errorf("expected %q to be %q", kpd.rawKey, "config/redis@nyc1")
	}

	if kpd.Prefix != "config/redis" {
		t.Errorf("expected %q to be %q", kpd.Prefix, "config/redis")
	}

	if kpd.DataCenter != "nyc1" {
		t.Errorf("expected %q to be %q", kpd.DataCenter, "nyc1")
	}
}

func TestParseStoreKeyPrefix_dataCenter(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if kpd.rawKey != "@nyc1" {
		t.Errorf("expected %q to be %q", kpd.rawKey, "@nyc1")
	}

	if kpd.Prefix != "/" {
		t.Errorf("expected %q to be %q", kpd.Prefix, "/")
	}

	if kpd.DataCenter != "nyc1" {
		t.Errorf("expected %q to be %q", kpd.DataCenter, "nyc1")
	}
}

func TestParseStoreKeyPrefix_leadingSlash(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("/config")
	if err != nil {
		t.Fatal(err)
	}

	if kpd.rawKey != "/config" {
		t.Errorf("expected %q to be %q", kpd.rawKey, "/config")
	}

	if kpd.Prefix != "config" {
		t.Errorf("expected %q to be %q", kpd.Prefix, "config")
	}
}

func TestParseStoreKeyPrefix_trailngSlashExist(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("config/")
	if err != nil {
		t.Fatal(err)
	}

	if kpd.rawKey != "config/" {
		t.Errorf("expected %q to be %q", kpd.rawKey, "config/")
	}

	if kpd.Prefix != "config/" {
		t.Errorf("expected %q to be %q", kpd.Prefix, "config/")
	}
}

func TestParseStoreKeyPrefix_slash(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("/")
	if err != nil {
		t.Fatal(err)
	}

	if kpd.rawKey != "/" {
		t.Errorf("expected %q to be %q", kpd.rawKey, "/")
	}

	if kpd.Prefix != "/" {
		t.Errorf("expected %q to be %q", kpd.Prefix, "/")
	}
}

func TestParseStoreKeyPrefix_slashSlash(t *testing.T) {
	kpd, err := ParseStoreKeyPrefix("//")
	if err != nil {
		t.Fatal(err)
	}

	if kpd.rawKey != "//" {
		t.Errorf("expected %q to be %q", kpd.rawKey, "//")
	}

	if kpd.Prefix != "/" {
		t.Errorf("expected %q to be %q", kpd.Prefix, "/")
	}
}
