package dependency

import (
	"reflect"
	"testing"
	"time"
)

func TestVaultSecretsFetch_empty(t *testing.T) {
	clients, vault := testVaultServer(t)
	defer vault.Stop()

	dep, err := ParseVaultSecrets("secret")
	if err != nil {
		t.Fatal(err)
	}

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

	dep, err := ParseVaultSecrets("secret")
	if err != nil {
		t.Fatal(err)
	}

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

func TestVaultSecretsFetch_stopped(t *testing.T) {
	clients, vault := testVaultServer(t)
	defer vault.Stop()

	dep, err := ParseVaultSecrets("secret")
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

func TestVaultSecretsHashCode_isUnique(t *testing.T) {
	dep1, err := ParseVaultSecrets("secret")
	if err != nil {
		t.Fatal(err)
	}

	dep2, err := ParseVaultSecrets("postgresql")
	if err != nil {
		t.Fatal(err)
	}

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
