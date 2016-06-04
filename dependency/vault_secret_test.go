package dependency

import (
	"testing"
	"time"
)

func TestVaultSecretFetch(t *testing.T) {
	clients, vault := testVaultServer(t)
	defer vault.Stop()

	vault.CreateSecret("foo/bar", map[string]interface{}{"zip": "zap"})

	dep, err := ParseVaultSecret("secret/foo/bar")
	if err != nil {
		t.Fatal(err)
	}

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

func TestVaultSecretFetch_stopped(t *testing.T) {
	clients, vault := testVaultServer(t)
	defer vault.Stop()

	vault.CreateSecret("foo/bar", map[string]interface{}{"zip": "zap"})

	dep, err := ParseVaultSecret("secret/foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	// Attach a secret to make it appear like we already requested once.
	dep.secret = &Secret{
		LeaseDuration: 5,
		Renewable:     true,
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

func TestVaultSecretHashCode_isUnique(t *testing.T) {
	dep1, err := ParseVaultSecret("secret/foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	dep2, err := ParseVaultSecret("secret/foo/foo")
	if err != nil {
		t.Fatal(err)
	}

	if dep1.HashCode() == dep2.HashCode() {
		t.Errorf("expected HashCode to be unique")
	}
}

func TestParseVaultSecret_nil(t *testing.T) {
	_, err := ParseVaultSecret()
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "expected 1 or more arguments, got 0"
	if err.Error() != expected {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestParseVaultSecret_empty(t *testing.T) {
	_, err := ParseVaultSecret("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "vault path must be at least one character"
	if err.Error() != expected {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestParseVaultSecret_single(t *testing.T) {
	v, err := ParseVaultSecret("secret/foo")
	if err != nil {
		t.Fatal(err)
	}

	if v.Path != "secret/foo" {
		t.Errorf("expected %q to be %q", v.Path, "secret/foo")
	}
}

func TestParseVaultSecret_multi(t *testing.T) {
	v, err := ParseVaultSecret("secret/foo", " a=b", "c = d")
	if err != nil {
		t.Fatal(err)
	}

	if v.Path != "secret/foo" {
		t.Errorf("expected %q to be %q", v.Path, "secret/foo")
	}

	if v.data == nil {
		t.Fatal("missing data")
	}

	if d, _ := v.data["a"]; d != "b" {
		t.Errorf("expected %q to be %q", d, "b")
	}

	if d, _ := v.data["c"]; d != "d" {
		t.Errorf("expected %q to be %q", d, "d")
	}
}

func TestParseVaultSecret_multiBad(t *testing.T) {
	_, err := ParseVaultSecret("secret/foo", "a")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `invalid value "a" - must be key=value`
	if err.Error() != expected {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}
