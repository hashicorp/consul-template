package dependency

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewKVListQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  *KVListQuery
		err  bool
	}{
		{
			"empty",
			"",
			&KVListQuery{},
			false,
		},
		{
			"dc_only",
			"@dc1",
			nil,
			true,
		},
		{
			"prefix",
			"prefix",
			&KVListQuery{
				prefix: "prefix",
			},
			false,
		},
		{
			"dc",
			"prefix@dc1",
			&KVListQuery{
				prefix: "prefix",
				dc:     "dc1",
			},
			false,
		},
		{
			"dots",
			"prefix.with.dots",
			&KVListQuery{
				prefix: "prefix.with.dots",
			},
			false,
		},
		{
			"slashes",
			"prefix/with/slashes",
			&KVListQuery{
				prefix: "prefix/with/slashes",
			},
			false,
		},
		{
			"dashes",
			"prefix-with-dashes",
			&KVListQuery{
				prefix: "prefix-with-dashes",
			},
			false,
		},
		{
			"leading_slash",
			"/leading/slash",
			&KVListQuery{
				prefix: "leading/slash",
			},
			false,
		},
		{
			"trailing_slash",
			"trailing/slash/",
			&KVListQuery{
				prefix: "trailing/slash/",
			},
			false,
		},
		{
			"underscores",
			"prefix_with_underscores",
			&KVListQuery{
				prefix: "prefix_with_underscores",
			},
			false,
		},
		{
			"special_characters",
			"config/facet:größe-lf-si",
			&KVListQuery{
				prefix: "config/facet:größe-lf-si",
			},
			false,
		},
		{
			"splat",
			"config/*/timeouts/",
			&KVListQuery{
				prefix: "config/*/timeouts/",
			},
			false,
		},
		{
			"slash",
			"/",
			&KVListQuery{
				prefix: "/",
			},
			false,
		},
		{
			"slash-slash",
			"//",
			&KVListQuery{
				prefix: "/",
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewKVListQuery(tc.i)
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

func TestKVListQuery_Fetch(t *testing.T) {
	t.Parallel()

	clients, consul := testConsulServer(t)
	defer consul.Stop()

	consul.SetKV("prefix/foo", []byte("bar"))
	consul.SetKV("prefix/zip", []byte("zap"))
	consul.SetKV("prefix/wave/ocean", []byte("sleek"))

	cases := []struct {
		name string
		i    string
		exp  []*KeyPair
	}{
		{
			"exists",
			"prefix",
			[]*KeyPair{
				&KeyPair{
					Path:  "prefix/foo",
					Key:   "foo",
					Value: "bar",
				},
				&KeyPair{
					Path:  "prefix/wave/ocean",
					Key:   "wave/ocean",
					Value: "sleek",
				},
				&KeyPair{
					Path:  "prefix/zip",
					Key:   "zip",
					Value: "zap",
				},
			},
		},
		{
			"trailing",
			"prefix/",
			[]*KeyPair{
				&KeyPair{
					Path:  "prefix/foo",
					Key:   "foo",
					Value: "bar",
				},
				&KeyPair{
					Path:  "prefix/wave/ocean",
					Key:   "wave/ocean",
					Value: "sleek",
				},
				&KeyPair{
					Path:  "prefix/zip",
					Key:   "zip",
					Value: "zap",
				},
			},
		},
		{
			"no_exist",
			"not/a/real/prefix/like/ever",
			[]*KeyPair{},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVListQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(clients, nil)
			if err != nil {
				t.Fatal(err)
			}

			for _, p := range act.([]*KeyPair) {
				p.CreateIndex = 0
				p.ModifyIndex = 0
			}

			assert.Equal(t, tc.exp, act)
		})
	}

	t.Run("stops", func(t *testing.T) {
		d, err := NewKVListQuery("prefix")
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
		d, err := NewKVListQuery("prefix/")
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

		consul.SetKV("prefix/foo", []byte("new-bar"))

		select {
		case err := <-errCh:
			t.Fatal(err)
		case data := <-dataCh:
			typed := data.([]*KeyPair)
			if len(typed) == 0 {
				t.Fatal("bad length")
			}

			act := typed[0]
			act.CreateIndex = 0
			act.ModifyIndex = 0

			exp := &KeyPair{
				Path:  "prefix/foo",
				Key:   "foo",
				Value: "new-bar",
			}

			assert.Equal(t, exp, act)
		}
	})
}

func TestKVListQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  string
	}{
		{
			"prefix",
			"prefix",
			"kv.list(prefix)",
		},
		{
			"dc",
			"prefix@dc1",
			"kv.list(prefix@dc1)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVListQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
