package dependency

import (
	"fmt"
	"net/url"
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
				path:        "path",
				queryValues: url.Values{},
			},
			false,
		},
		{
			"leading_slash",
			"/leading/slash",
			&VaultReadQuery{
				path:        "leading/slash",
				queryValues: url.Values{},
			},
			false,
		},
		{
			"trailing_slash",
			"trailing/slash/",
			&VaultReadQuery{
				path:        "trailing/slash",
				queryValues: url.Values{},
			},
			false,
		},
		{
			"query_param",
			"path?version=3",
			&VaultReadQuery{
				path: "path",
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

func TestVaultReadQuery_Fetch(t *testing.T) {
	t.Parallel()

	clients, vault := testVaultServer(t)
	defer vault.Stop()

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
			"secret/foo/bar",
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
		d, err := NewVaultReadQuery("secret/foo/bar")
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
		d, err := NewVaultReadQuery("secret/foo/bar")
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

	clients, vault := testVaultServer(t)
	defer vault.Stop()

	// Enable v2 kv for versioned secrets
	vc := clients.Vault()
	if err := vc.Sys().TuneMount("secret", api.MountConfigInput{
		Options: map[string]string{
			"version": "2",
		},
	}); err != nil {
		t.Fatalf("Error tuning secrets engine: %s", err)
	}

	// Write an initial value to the secret path
	err := vault.CreateSecret("data/foo/bar", map[string]interface{}{
		"data": map[string]interface{}{
			"ttl": "100ms", // explicitly make this a short duration for testing
			"zip": "zap",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Write a new value to increment the version
	err = vault.CreateSecret("data/foo/bar", map[string]interface{}{
		"data": map[string]interface{}{
			"ttl": "100ms", // explicitly make this a short duration for testing
			"zip": "zop",
		},
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
			"secret/foo/bar",
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
			"secret/foo/bar?version=1",
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
		d, err := NewVaultReadQuery("secret/foo/bar")
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
		d, err := NewVaultReadQuery("secret/foo/bar")
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
