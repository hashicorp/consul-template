package dependency

import (
	"reflect"
	"testing"
)

func TestVaultSecretsFetch_empty(t *testing.T) {
	clients, vault := testVaultServer(t)
	defer vault.Stop()

	dep := &VaultSecrets{Path: "secret"}
	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := results.([]string)
	if !ok {
		t.Fatal("could not convert result to []string")
	}

	expected := []string{}
	if !reflect.DeepEqual(typed, expected) {
		t.Errorf("expected %#v to be %#v", typed, expected)
	}
}

func TestVaultSecretsFetch(t *testing.T) {
	clients, vault := testVaultServer(t)
	defer vault.Stop()

	vault.CreateSecret("foo", map[string]interface{}{"a": "b"})
	vault.CreateSecret("bar", map[string]interface{}{"c": "d"})

	dep := &VaultSecrets{Path: "secret"}
	results, _, err := dep.Fetch(clients, nil)
	if err != nil {
		t.Fatal(err)
	}

	typed, ok := results.([]string)
	if !ok {
		t.Fatal("could not convert result to []string")
	}

	expected := []string{"bar", "foo"}
	if !reflect.DeepEqual(typed, expected) {
		t.Errorf("expected %#v to be %#v", typed, expected)
	}
}

func TestVaultSecretsHashCode_isUnique(t *testing.T) {
	dep1 := &VaultSecrets{Path: "secret"}
	dep2 := &VaultSecrets{Path: "postgresql"}
	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseVaultSecrets_trailingSlash(t *testing.T) {
	d, err := ParseVaultSecrets("")
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != "/" {
		t.Errorf("expected %q to be %q", d.Path, "/")
	}

	d, err = ParseVaultSecrets("/")
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != "/" {
		t.Errorf("expected %q to be %q", d.Path, "/")
	}

	d, err = ParseVaultSecrets("secret")
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != "secret/" {
		t.Errorf("expected %q to be %q", d.Path, "/")
	}

	d, err = ParseVaultSecrets("secret/")
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != "secret/" {
		t.Errorf("expected %q to be %q", d.Path, "/")
	}
}
