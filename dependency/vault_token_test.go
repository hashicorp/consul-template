package dependency

import (
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

func TestVaultTokenFetch(t *testing.T) {
	clients, server := testVaultServer(t)
	defer server.Stop()

	// Create a new token - the default token is a root token and is therefore
	// not renewable
	secret, err := clients.vault.Auth().Token().Create(&api.TokenCreateRequest{
		Lease: "1h",
	})
	if err != nil {
		t.Fatal(err)
	}
	clients.vault.SetToken(secret.Auth.ClientToken)

	dep, err := ParseVaultToken()
	if err != nil {
		t.Fatal(err)
	}

	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.(*Secret)
	if !ok {
		t.Fatal("could not convert result to a *vault/api.Secret")
	}
}

func TestVaultTokenFetch_stoped(t *testing.T) {
	clients, server := testVaultServer(t)
	defer server.Stop()

	// Create a new token - the default token is a root token and is therefore
	// not renewable
	secret, err := clients.vault.Auth().Token().Create(&api.TokenCreateRequest{
		Lease: "1h",
	})
	if err != nil {
		t.Fatal(err)
	}
	clients.vault.SetToken(secret.Auth.ClientToken)

	dep, err := ParseVaultToken()
	if err != nil {
		t.Fatal(err)
	}

	// Attach a lease to make it appear like we already requested once.
	dep.leaseDuration = 5

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
