package dependency

import (
	"testing"

	"github.com/hashicorp/vault/api"
)

func TestVaultTokenFetch(t *testing.T) {
	clients, server := testVaultServer(t)
	defer server.Stop()

	vault, err := clients.Vault()
	if err != nil {
		t.Fatal(err)
	}

	// Create a new token - the default token is a root token and is therefore
	// not renewable
	token, err := vault.Auth().Token().Create(&api.TokenCreateRequest{
		Lease: "1h",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Set the new token on the client so we can try to renew
	newToken := token.Auth.ClientToken
	vault.SetToken(newToken)

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
