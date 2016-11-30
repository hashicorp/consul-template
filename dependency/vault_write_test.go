package dependency

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
)

func TestNewVaultWriteQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		d    map[string]interface{}
		exp  *VaultWriteQuery
		err  bool
	}{
		{
			"empty",
			"",
			nil,
			nil,
			true,
		},
		{
			"path",
			"path",
			nil,
			&VaultWriteQuery{
				path:     "path",
				data:     nil,
				dataHash: "da39a3ee",
			},
			false,
		},
		{
			"data",
			"data",
			map[string]interface{}{
				"foo": "bar",
			},
			&VaultWriteQuery{
				path: "data",
				data: map[string]interface{}{
					"foo": "bar",
				},
				dataHash: "ab03a894",
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewVaultWriteQuery(tc.i, tc.d)
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

func TestVaultWriteQuery_Fetch(t *testing.T) {
	t.Parallel()

	clients, vault := testVaultServer(t)
	defer vault.Stop()

	if err := clients.Vault().Sys().Mount("transit", &api.MountInput{
		Type: "transit",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := clients.Vault().Logical().Write("transit/keys/test", nil); err != nil {
		t.Fatal(err)
	}

	b64 := func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}

	cases := []struct {
		name string
		i    string
		d    map[string]interface{}
		exp  interface{}
		err  bool
	}{
		{
			"encrypt",
			"transit/encrypt/test",
			map[string]interface{}{
				"plaintext": b64("test"),
			},
			&Secret{
				Data: map[string]interface{}{
					"ciphertext": "",
				},
			},
			false,
		},
		{
			"no_exist",
			"not/a/real/path/like/ever",
			nil,
			nil,
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewVaultWriteQuery(tc.i, tc.d)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(clients, nil)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if act != nil {
				act.(*Secret).LeaseID = ""
				act.(*Secret).LeaseDuration = 0
				act.(*Secret).Renewable = false
				if act.(*Secret).Data["ciphertext"] != "" {
					act.(*Secret).Data["ciphertext"] = ""
				}
			}

			assert.Equal(t, tc.exp, act)
		})
	}

	t.Run("stops", func(t *testing.T) {
		d, err := NewVaultWriteQuery("transit/encrypt/test", map[string]interface{}{
			"plaintext": b64("test"),
		})
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
		case <-time.After(100 * time.Millisecond):
			t.Errorf("did not stop")
		}
	})

	t.Run("fires_changes", func(t *testing.T) {
		d, err := NewVaultWriteQuery("transit/encrypt/test", map[string]interface{}{
			"plaintext": b64("test"),
		})
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

func TestVaultWriteQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		d    map[string]interface{}
		exp  string
	}{
		{
			"path_nil_data",
			"path",
			nil,
			"vault.write(path -> da39a3ee)",
		},
		{
			"path_data",
			"path",
			map[string]interface{}{
				"foo": "bar",
			},
			"vault.write(path -> ab03a894)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewVaultWriteQuery(tc.i, tc.d)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
