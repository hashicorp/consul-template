package dependency

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
)

func TestNewVaultReadQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  *VaultReadQuery
		err  bool
	}{
		{
			"empty",
			"",
			nil,
			true,
		},
		{
			"path",
			"path",
			&VaultReadQuery{
				rawPath:     "path",
				queryValues: url.Values{},
			},
			false,
		},
		{
			"leading_slash",
			"/leading/slash",
			&VaultReadQuery{
				rawPath:     "leading/slash",
				queryValues: url.Values{},
			},
			false,
		},
		{
			"trailing_slash",
			"trailing/slash/",
			&VaultReadQuery{
				rawPath:     "trailing/slash",
				queryValues: url.Values{},
			},
			false,
		},
		{
			"query_param",
			"path?version=3",
			&VaultReadQuery{
				rawPath: "path",
				queryValues: url.Values{
					"version": []string{"3"},
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewVaultReadQuery(tc.i)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if act != nil {
				act.stopCh = nil
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestVaultReadQuery_Fetch_KVv1(t *testing.T) {
	t.Parallel()

	clients, vault := testVaultServer(t, "read_fetch_v1", "1")
	secretsPath := vault.secretsPath
	// Enable v1 kv for versioned secrets
	vc := clients.Vault()
	if err := vc.Sys().TuneMount(secretsPath, api.MountConfigInput{
		Options: map[string]string{
			"version": "1",
		},
	}); err != nil {
		t.Fatalf("Error tuning secrets engine: %s", err)
	}

	err := vault.CreateSecret("foo/bar", map[string]interface{}{
		"ttl": "100ms", // explicitly make this a short duration for testing
		"zip": "zap",
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		i    string
		exp  interface{}
		err  bool
	}{
		{
			"exists",
			secretsPath + "/foo/bar",
			&Secret{
				Data: map[string]interface{}{
					"ttl": "100ms",
					"zip": "zap",
				},
			},
			false,
		},
		{
			"no_exist",
			"not/a/real/path/like/ever",
			nil,
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewVaultReadQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(clients, nil)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if act != nil {
				act.(*Secret).RequestID = ""
				act.(*Secret).LeaseID = ""
				act.(*Secret).LeaseDuration = 0
				act.(*Secret).Renewable = false
			}

			assert.Equal(t, tc.exp, act)
		})
	}

	t.Run("stops", func(t *testing.T) {
		d, err := NewVaultReadQuery(secretsPath + "/foo/bar")
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			for {
				data, _, err := d.Fetch(clients, nil)
				if err != nil {
					errCh <- err
					return
				}
				dataCh <- data
			}
		}()

		select {
		case err := <-errCh:
			t.Fatal(err)
		case <-dataCh:
		}

		d.Stop()

		select {
		case err := <-errCh:
			if err != ErrStopped {
				t.Fatal(err)
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("did not stop")
		}
	})

	t.Run("fires_changes", func(t *testing.T) {
		d, err := NewVaultReadQuery(secretsPath + "/foo/bar")
		if err != nil {
			t.Fatal(err)
		}

		_, qm, err := d.Fetch(clients, nil)
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			for {
				data, _, err := d.Fetch(clients, &QueryOptions{WaitIndex: qm.LastIndex})
				if err != nil {
					errCh <- err
					return
				}
				dataCh <- data
				return
			}
		}()

		select {
		case err := <-errCh:
			t.Fatal(err)
		case <-dataCh:
		}
	})
}

func TestVaultReadQuery_Fetch_KVv2(t *testing.T) {
	t.Parallel()

	clients, vault := testVaultServer(t, "read_fetch_v2", "2")
	secretsPath := vault.secretsPath

	// Write an initial value to the secret path
	err := vault.CreateSecret("data/foo/bar", map[string]interface{}{
		"ttl": "100ms", // explicitly make this a short duration for testing
		"zip": "zap",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Write a new value to increment the version
	err = vault.CreateSecret("data/foo/bar", map[string]interface{}{
		"ttl": "100ms", // explicitly make this a short duration for testing
		"zip": "zop",
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		i    string
		exp  interface{}
		err  bool
	}{
		{
			"exists",
			secretsPath + "/foo/bar",
			&Secret{
				Data: map[string]interface{}{
					"data": map[string]interface{}{
						"ttl": "100ms", // explicitly make this a short duration for testing
						"zip": "zop",
					},
				},
			},
			false,
		},
		{
			"/data in path",
			secretsPath + "/data/foo/bar",
			&Secret{
				Data: map[string]interface{}{
					"data": map[string]interface{}{
						"ttl": "100ms", // explicitly make this a short duration for testing
						"zip": "zop",
					},
				},
			},
			false,
		},
		{
			"version=1",
			secretsPath + "/foo/bar?version=1",
			&Secret{
				Data: map[string]interface{}{
					"data": map[string]interface{}{
						"ttl": "100ms", // explicitly make this a short duration for testing
						"zip": "zap",
					},
				},
			},
			false,
		},
		{
			"no_exist",
			"not/a/real/path/like/ever",
			nil,
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewVaultReadQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(clients, nil)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if act != nil {
				act.(*Secret).RequestID = ""
				act.(*Secret).LeaseID = ""
				act.(*Secret).LeaseDuration = 0
				act.(*Secret).Renewable = false
				tc.exp.(*Secret).Data["metadata"] = act.(*Secret).Data["metadata"]
			}

			assert.Equal(t, tc.exp, act)
		})
	}

	t.Run("stops", func(t *testing.T) {
		d, err := NewVaultReadQuery(secretsPath + "/foo/bar")
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			for {
				data, _, err := d.Fetch(clients, nil)
				if err != nil {
					errCh <- err
					return
				}
				dataCh <- data
			}
		}()

		select {
		case err := <-errCh:
			t.Fatal(err)
		case <-dataCh:
		}

		d.Stop()

		select {
		case err := <-errCh:
			if err != ErrStopped {
				t.Fatal(err)
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("did not stop")
		}
	})

	for _, dataPrefix := range []string{"", "/data"} {
		t.Run(fmt.Sprintf("fires_changes%s", dataPrefix), func(t *testing.T) {
			d, err := NewVaultReadQuery(fmt.Sprintf("%s%s/foo/bar",
				secretsPath, dataPrefix))
			if err != nil {
				t.Fatal(err)
			}

			_, qm, err := d.Fetch(clients, nil)
			if err != nil {
				t.Fatal(err)
			}

			dataCh := make(chan interface{}, 1)
			errCh := make(chan error, 1)
			go func() {
				for {
					data, _, err := d.Fetch(clients, &QueryOptions{WaitIndex: qm.LastIndex})
					if err != nil {
						errCh <- err
						return
					}
					dataCh <- data
					return
				}
			}()

			select {
			case err := <-errCh:
				t.Fatal(err)
			case <-dataCh:
			}
		})
	}
}

// TestVaultReadQuery_Fetch_PKI_Anonymous asserts that vault.read can fetch a
// pki ca public cert even even when running unauthenticated client.
func TestVaultReadQuery_Fetch_PKI_Anonymous(t *testing.T) {
	t.Parallel()
	clients := testClients

	err := clients.Vault().Sys().Mount("pki", &api.MountInput{
		Type: "pki",
	})
	if err != nil {
		t.Fatal(err)
	}

	vc := clients.Vault()
	_, err = vc.Logical().Write("sys/policies/acl/secrets-only",
		map[string]interface{}{
			"policy": `path "secret/*" { capabilities = ["create", "read"] }`,
		})
	if err != nil {
		t.Fatal(err)
	}

	_, err = vc.Logical().Write("pki/root/generate/internal",
		map[string]interface{}{
			"common_name": "example.com",
			"ttl":         "24h",
		})

	anonClient := NewClientSet()
	anonClient.CreateVaultClient(&CreateVaultClientInput{
		Address: vaultAddr,
		Token:   "",
	})
	_, err = anonClient.vault.client.Auth().Token().LookupSelf()
	if err == nil || !strings.Contains(err.Error(), "missing client token") {
		// check environment for VAULT_TOKEN
		t.Fatalf("expected a missing client token error but found: %v", err)
	}

	d, err := NewVaultReadQuery("pki/cert/ca")
	if err != nil {
		t.Fatal(err)
	}

	act, _, err := d.Fetch(anonClient, nil)
	if err != nil {
		t.Fatal(err)
	}

	sec, ok := act.(*Secret)
	if !ok {
		t.Fatalf("expected secret but found %v", reflect.TypeOf(act))
	}
	cert, ok := sec.Data["certificate"].(string)
	if !ok || !strings.Contains(cert, "BEGIN") {
		t.Fatalf("expected a cert but found: %v", cert)
	}
}

// TestVaultReadQuery_Fetch_NonSecrets asserts that vault.read can fetch a
// non-secret
func TestVaultReadQuery_Fetch_NonSecrets(t *testing.T) {
	t.Parallel()

	var err error

	clients := testClients

	vc := clients.Vault()

	err = vc.Sys().EnableAuth("approle", "approle", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = vc.Logical().Write("auth/approle/role/my-approle", nil)
	if err != nil {
		t.Fatal(err)
	}

	// create restricted token
	_, err = vc.Logical().Write("sys/policies/acl/operator",
		map[string]interface{}{
			"policy": `path "auth/approle/role/my-approle/role-id" { capabilities = ["read"] }`,
		})
	secret, err := vc.Auth().Token().Create(&api.TokenCreateRequest{
		Policies: []string{"operator"},
	})
	if err != nil {
		t.Fatal(err)
	}

	anonClient := NewClientSet()
	anonClient.CreateVaultClient(&CreateVaultClientInput{
		Address: vaultAddr,
		Token:   secret.Auth.ClientToken,
	})
	_, err = anonClient.vault.client.Auth().Token().LookupSelf()
	if err != nil {
		t.Fatal(err)
	}

	d, err := NewVaultReadQuery("auth/approle/role/my-approle/role-id")
	if err != nil {
		t.Fatal(err)
	}

	act, _, err := d.Fetch(anonClient, nil)
	if err != nil {
		t.Fatal(err)
	}

	sec, ok := act.(*Secret)
	if !ok {
		t.Fatalf("expected secret but found %v", reflect.TypeOf(act))
	}
	if _, ok := sec.Data["role_id"]; !ok {
		t.Fatalf("expected to find role_id but found: %v", sec.Data)
	}
}

func TestVaultReadQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  string
	}{
		{
			"path",
			"path",
			"vault.read(path)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewVaultReadQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
