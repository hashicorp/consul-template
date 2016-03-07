package dependency

import (
	"testing"
	"time"
)

func TestCatalogServicesFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseCatalogServices("")
	if err != nil {
		t.Fatal(err)
	}

	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := results.([]*CatalogService); !ok {
		t.Fatal("could not convert result to []*CatalogService")
	}
}

func TestCatalogServicesFetch_stopped(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseCatalogServices("")
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

func TestCatalogServicesHashCode_isUnique(t *testing.T) {
	dep1, err := ParseCatalogServices("")
	if err != nil {
		t.Fatal(err)
	}

	dep2, err := ParseCatalogServices("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseCatalogServices_emptyString(t *testing.T) {
	nd, err := ParseCatalogServices("")
	if err != nil {
		t.Fatal(err)
	}

	if nd.rawKey != "" {
		t.Errorf("expected %q to be %q", nd.rawKey, "")
	}

	if nd.Name != "" {
		t.Errorf("expected %q to be %q", nd.Name, "")
	}

	if nd.DataCenter != "" {
		t.Errorf("expected %q to be %q", nd.DataCenter, "")
	}
}

func TestParseCatalogServices_dataCenter(t *testing.T) {
	nd, err := ParseCatalogServices("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if nd.rawKey != "@nyc1" {
		t.Errorf("expected %q to be %q", nd.rawKey, "@nyc1")
	}

	if nd.Name != "" {
		t.Errorf("expected %q to be %q", nd.Name, "")
	}

	if nd.DataCenter != "nyc1" {
		t.Errorf("expected %q to be %q", nd.DataCenter, "nyc1")
	}
}
