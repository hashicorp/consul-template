package dependency

import (
	"testing"
	"time"
)

func TestCatalogNodesFetch(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseCatalogNodes("")
	if err != nil {
		t.Fatal(err)
	}

	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := results.([]*Node)
	if !ok {
		t.Fatal("could not convert result to []*Node")
	}

	if typed[0].Address != "127.0.0.1" {
		t.Errorf("expected %q to be %q", typed[0].Address, "127.0.0.1")
	}
}

func TestCatalogNodesFetch_stopped(t *testing.T) {
	clients, consul := testConsulServer(t)
	defer consul.Stop()

	dep, err := ParseCatalogNodes("")
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

func TestCatalogNodesHashCode_isUnique(t *testing.T) {
	dep1, err := ParseCatalogNodes("")
	if err != nil {
		t.Fatal(err)
	}

	dep2, err := ParseCatalogNodes("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseCatalogNodes_emptyString(t *testing.T) {
	nd, err := ParseCatalogNodes("")
	if err != nil {
		t.Fatal(err)
	}

	if nd.rawKey != "" {
		t.Errorf("expected %q to be %q", nd.rawKey, "")
	}

	if nd.DataCenter != "" {
		t.Errorf("expected %q to be %q", nd.DataCenter, "")
	}
}

func TestParseCatalogNodes_dataCenter(t *testing.T) {
	nd, err := ParseCatalogNodes("@nyc1")
	if err != nil {
		t.Fatal(err)
	}

	if nd.rawKey != "@nyc1" {
		t.Errorf("expected %q to be %q", nd.rawKey, "@nyc1")
	}

	if nd.DataCenter != "nyc1" {
		t.Errorf("expected %q to be %q", nd.DataCenter, "nyc1")
	}
}
