package dependency

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSVListQuery(t *testing.T) {

	cases := []struct {
		name string
		i    string
		exp  *SVListQuery
		err  bool
	}{
		{
			"empty",
			"",
			&SVListQuery{},
			false,
		},
		{
			"prefix",
			"prefix",
			&SVListQuery{
				prefix: "prefix",
			},
			false,
		},
		{
			"dots",
			"prefix.with.dots",
			&SVListQuery{
				prefix: "prefix.with.dots",
			},
			false,
		},
		{
			"slashes",
			"prefix/with/slashes",
			&SVListQuery{
				prefix: "prefix/with/slashes",
			},
			false,
		},
		{
			"dashes",
			"prefix-with-dashes",
			&SVListQuery{
				prefix: "prefix-with-dashes",
			},
			false,
		},
		{
			"leading_slash",
			"/leading/slash",
			&SVListQuery{
				prefix: "leading/slash",
			},
			false,
		},
		{
			"trailing_slash",
			"trailing/slash/",
			&SVListQuery{
				prefix: "trailing/slash/",
			},
			false,
		},
		{
			"underscores",
			"prefix_with_underscores",
			&SVListQuery{
				prefix: "prefix_with_underscores",
			},
			false,
		},
		{
			"special_characters",
			"config/facet:größe-lf-si",
			&SVListQuery{
				prefix: "config/facet:größe-lf-si",
			},
			false,
		},
		{
			"splat",
			"config/*/timeouts/",
			&SVListQuery{
				prefix: "config/*/timeouts/",
			},
			false,
		},
		{
			"slash",
			"/",
			&SVListQuery{
				prefix: "/",
			},
			false,
		},
		{
			"slash-slash",
			"//",
			&SVListQuery{
				prefix: "/",
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewSVListQuery(tc.i)
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

func TestSVListQuery_Fetch(t *testing.T) {

	type svmap map[string]string
	_ = testNomad.CreateSecureVariable("test-kv-list/prefix/foo", svmap{"bar": "barp"})
	_ = testNomad.CreateSecureVariable("test-kv-list/prefix/zip", svmap{"zap": "zapp"})
	_ = testNomad.CreateSecureVariable("test-kv-list/prefix/wave/ocean", svmap{"sleek": "sleekp"})

	cases := []struct {
		name string
		i    string
		exp  []*NomadSVMeta
	}{
		{
			"exists",
			"test-kv-list/prefix",
			[]*NomadSVMeta{
				{Namespace: "default", Path: "test-kv-list/prefix/foo"},
				{Namespace: "default", Path: "test-kv-list/prefix/wave/ocean"},
				{Namespace: "default", Path: "test-kv-list/prefix/zip"},
			},
		},
		{
			"trailing",
			"test-kv-list/prefix/",
			[]*NomadSVMeta{
				{Namespace: "default", Path: "test-kv-list/prefix/foo"},
				{Namespace: "default", Path: "test-kv-list/prefix/wave/ocean"},
				{Namespace: "default", Path: "test-kv-list/prefix/zip"},
			},
		},
		{
			"no_exist",
			"test-kv-list/not/a/real/prefix/like/ever",
			[]*NomadSVMeta{},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewSVListQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(testClients, nil)
			if err != nil {
				t.Fatal(err)
			}

			for _, p := range act.([]*NomadSVMeta) {
				p.CreateIndex = 0
				p.CreateTime = 0
				p.ModifyIndex = 0
				p.ModifyTime = 0
			}

			assert.ElementsMatch(t, tc.exp, act)
		})
	}

	t.Run("stops", func(t *testing.T) {
		d, err := NewSVListQuery("test-kv-list/prefix")
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			for {
				data, _, err := d.Fetch(testClients, nil)
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
		case <-time.After(250 * time.Millisecond):
			t.Errorf("did not stop")
		}
	})

	t.Run("fires_changes", func(t *testing.T) {
		d, err := NewSVListQuery("test-kv-list/prefix/")
		if err != nil {
			t.Fatal(err)
		}

		_, qm, err := d.Fetch(testClients, nil)
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			for {
				data, _, err := d.Fetch(testClients, &QueryOptions{WaitIndex: qm.LastIndex})
				if err != nil {
					errCh <- err
					return
				}
				dataCh <- data
				return
			}
		}()

		_ = testNomad.CreateSecureVariable("test-kv-list/prefix/foo", svmap{"new-bar": "new-barp"})

		select {
		case err := <-errCh:
			t.Fatal(err)
		case data := <-dataCh:
			svs := data.([]*NomadSVMeta)
			if len(svs) == 0 {
				t.Fatal("bad length")
			}

			// Zero out the dynamic elements
			act := svs[0]
			act.CreateIndex = 0
			act.CreateTime = 0
			act.ModifyIndex = 0
			act.ModifyTime = 0

			exp := &NomadSVMeta{Namespace: "default", Path: "test-kv-list/prefix/foo"}
			assert.Equal(t, exp, act)
		}
	})
}

func TestSVListQuery_String(t *testing.T) {

	cases := []struct {
		name string
		i    string
		exp  string
	}{
		{
			"prefix",
			"prefix",
			"nomad.secure_variables.list(prefix)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewSVListQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
