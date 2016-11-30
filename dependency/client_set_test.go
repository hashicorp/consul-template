package dependency

import (
	"testing"

	"github.com/hashicorp/vault/api"
)

func TestClientSet_unwrapVaultToken(t *testing.T) {
	t.Parallel()

	clients, server := testVaultServer(t)
	defer server.Stop()

	vault := clients.vault.client

	// Grab the original token
	originalToken, err := vault.Auth().Token().LookupSelf()
	if err != nil {
		t.Fatal(err)
	}

	// Create a wrapped token
	vault.SetWrappingLookupFunc(func(operation, path string) string {
		return "30s"
	})
	wrappedToken, err := vault.Auth().Token().Create(&api.TokenCreateRequest{
		Lease: "1h",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := clients.CreateVaultClient(&CreateVaultClientInput{
		Address:     server.Address,
		Token:       wrappedToken.WrapInfo.Token,
		UnwrapToken: true,
	}); err != nil {
		t.Fatal(err)
	}

	newToken := clients.vault.client.Token()

	if newToken == originalToken.Data["id"] {
		t.Errorf("expected %q to not be %q", newToken, originalToken.Data["id"])
	}

	if newToken == wrappedToken.WrapInfo.Token {
		t.Errorf("expected %q to not be %q", newToken, wrappedToken.WrapInfo.Token)
	}

	if _, err := vault.Auth().Token().LookupSelf(); err != nil {
		t.Fatal(err)
	}
}
