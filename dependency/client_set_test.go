package dependency

import (
	"testing"

	"github.com/hashicorp/vault/api"
)

func TestClientSet_unwrapVaultToken(t *testing.T) {
	// Don't use t.Parallel() here as the SetWrappingLookupFunc is a global
	// setting and breaks other tests if run in parallel

	vault := testClients.Vault()

	// Create a wrapped token
	vault.SetWrappingLookupFunc(func(operation, path string) string {
		return "30s"
	})
	defer vault.SetWrappingLookupFunc(nil)

	wrappedToken, err := vault.Auth().Token().Create(&api.TokenCreateRequest{
		Lease: "1h",
	})
	if err != nil {
		t.Fatal(err)
	}

	token := vault.Token()

	if token == wrappedToken.WrapInfo.Token {
		t.Errorf("expected %q to not be %q", token,
			wrappedToken.WrapInfo.Token)
	}

	if _, err := vault.Auth().Token().LookupSelf(); err != nil {
		t.Fatal(err)
	}
}
