package dependency

import (
	"testing"

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

	dep := new(VaultToken)
	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := results.(*Secret)
	if !ok {
		t.Fatal("could not convert result to a *vault/api.Secret")
	}
}
