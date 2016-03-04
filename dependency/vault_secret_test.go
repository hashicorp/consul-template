package dependency

import "testing"

func TestVaultSecretFetch(t *testing.T) {
	clients, vault := testVaultServer(t)
	defer vault.Stop()

	vault.CreateSecret("foo/bar", map[string]interface{}{"zip": "zap"})

	dep := &VaultSecret{Path: "secret/foo/bar"}
	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := results.(*Secret)
	if !ok {
		t.Fatal("could not convert result to a *vault/api.Secret")
	}

	if typed.Data["zip"].(string) != "zap" {
		t.Errorf("expected %#v to be %q", typed.Data["zip"], "zap")
	}
}

func TestVaultSecretHashCode_isUnique(t *testing.T) {
	dep1 := &VaultSecret{Path: "secret/foo/foo"}
	dep2 := &VaultSecret{Path: "secret/foo/bar"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}
