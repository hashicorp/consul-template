package dependency

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewKVGetQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  *KVGetQuery
		err  bool
	}{
		{
			"empty",
			"",
			&KVGetQuery{},
			false,
		},

		{
			"dc_only",
			"@dc1",
			nil,
			true,
		},
		{
			"key",
			"key",
			&KVGetQuery{
				key: "key",
			},
			false,
		},
		{
			"dc",
			"key@dc1",
			&KVGetQuery{
				key: "key",
				dc:  "dc1",
			},
			false,
		},
		{
			"dots",
			"key.with.dots",
			&KVGetQuery{
				key: "key.with.dots",
			},
			false,
		},
		{
			"slashes",
			"key/with/slashes",
			&KVGetQuery{
				key: "key/with/slashes",
			},
			false,
		},
		{
			"dashes",
			"key-with-dashes",
			&KVGetQuery{
				key: "key-with-dashes",
			},
			false,
		},
		{
			"leading_slash",
			"/leading/slash",
			&KVGetQuery{
				key: "leading/slash",
			},
			false,
		},
		{
			"trailing_slash",
			"trailing/slash/",
			&KVGetQuery{
				key: "trailing/slash/",
			},
			false,
		},
		{
			"underscores",
			"key_with_underscores",
			&KVGetQuery{
				key: "key_with_underscores",
			},
			false,
		},
		{
			"special_characters",
			"config/facet:größe-lf-si",
			&KVGetQuery{
				key: "config/facet:größe-lf-si",
			},
			false,
		},
		{
			"splat",
			"config/*/timeouts/",
			&KVGetQuery{
				key: "config/*/timeouts/",
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewKVGetQuery(tc.i)
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

func TestKVGetQuery_Fetch(t *testing.T) {
	t.Parallel()

	clients, consul := testConsulServer(t)
	defer consul.Stop()

	consul.SetKV("key", []byte("value"))
	consul.SetKV("key_empty", []byte(""))

	cases := []struct {
		name string
		i    string
		exp  interface{}
	}{
		{
			"exists",
			"key",
			"value",
		},
		{
			"exists_empty_string",
			"key_empty",
			"",
		},
		{
			"no_exist",
			"not/a/real/key/like/ever",
			nil,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVGetQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(clients, nil)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.exp, act)
		})
	}

	t.Run("stops", func(t *testing.T) {
		d, err := NewKVGetQuery("key")
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
		d, err := NewKVGetQuery("key")
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

		consul.SetKV("key", []byte("new-value"))

		select {
		case err := <-errCh:
			t.Fatal(err)
		case data := <-dataCh:
			assert.Equal(t, data, "new-value")
		}
	})
}

func TestKVGetQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  string
	}{
		{
			"key",
			"key",
			"kv.get(key)",
		},
		{
			"dc",
			"key@dc1",
			"kv.get(key@dc1)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVGetQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
