// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"fmt"
	"github.com/hashicorp/consul-template/test"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewKVKeysQuery(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  *KVKeysQuery
		err  bool
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("empty", tenancy),
				"",
				&KVKeysQuery{},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_only", tenancy),
				"@dc1",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("query_only", tenancy),
				fmt.Sprintf("?ns=%s", tenancy.Namespace),
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("invalid query param (unsupported key)", tenancy),
				"prefix?unsupported=foo",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("prefix", tenancy),
				"prefix",
				&KVKeysQuery{
					prefix: "prefix",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc", tenancy),
				"prefix@dc1",
				&KVKeysQuery{
					prefix: "prefix",
					dc:     "dc1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition", tenancy),
				fmt.Sprintf("prefix?partition=%s", tenancy.Partition),
				&KVKeysQuery{
					prefix:    "prefix",
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace", tenancy),
				fmt.Sprintf("prefix?ns=%s", tenancy.Namespace),
				&KVKeysQuery{
					prefix:    "prefix",
					namespace: tenancy.Namespace,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace_and_partition", tenancy),
				"prefix?ns=foo&partition=bar",
				&KVKeysQuery{
					prefix:    "prefix",
					namespace: "foo",
					partition: "bar",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_namespace_and_partition", tenancy),
				fmt.Sprintf("prefix?ns=%s&partition=%s@dc1", tenancy.Namespace, tenancy.Partition),
				&KVKeysQuery{
					prefix:    "prefix",
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
					dc:        "dc1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("empty_query", tenancy),
				"prefix?ns=&partition=",
				&KVKeysQuery{
					prefix:    "prefix",
					namespace: "",
					partition: "",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dots", tenancy),
				"prefix.with.dots",
				&KVKeysQuery{
					prefix: "prefix.with.dots",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("slashes", tenancy),
				"prefix/with/slashes",
				&KVKeysQuery{
					prefix: "prefix/with/slashes",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dashes", tenancy),
				"prefix-with-dashes",
				&KVKeysQuery{
					prefix: "prefix-with-dashes",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("leading_slash", tenancy),
				"/leading/slash",
				&KVKeysQuery{
					prefix: "leading/slash",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("trailing_slash", tenancy),
				"trailing/slash/",
				&KVKeysQuery{
					prefix: "trailing/slash/",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("underscores", tenancy),
				"prefix_with_underscores",
				&KVKeysQuery{
					prefix: "prefix_with_underscores",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("special_characters", tenancy),
				"config/facet:größe-lf-si",
				&KVKeysQuery{
					prefix: "config/facet:größe-lf-si",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("splat", tenancy),
				"config/*/timeouts/",
				&KVKeysQuery{
					prefix: "config/*/timeouts/",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("slash", tenancy),
				"/",
				&KVKeysQuery{
					prefix: "/",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("slash-slash", tenancy),
				"//",
				&KVKeysQuery{
					prefix: "/",
				},
				false,
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewKVKeysQuery(tc.i)
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

func TestKVKeysQuery_Fetch(t *testing.T) {
	for _, tenancy := range tenancyHelper.TestTenancies() {
		fooKey := fmt.Sprintf("test-kv-keys/prefix/foo-%s-%s", tenancy.Partition, tenancy.Namespace)
		if tenancyHelper.IsConsulEnterprise() {
			fooKey += fmt.Sprintf("?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace)
		}
		testConsul.SetKVString(t, fooKey, "bar")

		zipKey := fmt.Sprintf("test-kv-keys/prefix/zip-%s-%s", tenancy.Partition, tenancy.Namespace)
		if tenancyHelper.IsConsulEnterprise() {
			zipKey += fmt.Sprintf("?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace)
		}
		testConsul.SetKVString(t, zipKey, "zap")

		oceanKey := fmt.Sprintf("test-kv-keys/prefix/wave/ocean-%s-%s", tenancy.Partition, tenancy.Namespace)
		if tenancyHelper.IsConsulEnterprise() {
			oceanKey += fmt.Sprintf("?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace)
		}
		testConsul.SetKVString(t, oceanKey, "sleek")
	}

	type testCase struct {
		name string
		i    string
		exp  []string
	}

	cases := tenancyHelper.GenerateNonDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("exists", tenancy),
				fmt.Sprintf("test-kv-keys/prefix?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				[]string{
					fmt.Sprintf("foo-%s-%s", tenancy.Partition, tenancy.Namespace),
					fmt.Sprintf("wave/ocean-%s-%s", tenancy.Partition, tenancy.Namespace),
					fmt.Sprintf("zip-%s-%s", tenancy.Partition, tenancy.Namespace),
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("trailing", tenancy),
				fmt.Sprintf("test-kv-keys/prefix/?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				[]string{
					fmt.Sprintf("foo-%s-%s", tenancy.Partition, tenancy.Namespace),
					fmt.Sprintf("wave/ocean-%s-%s", tenancy.Partition, tenancy.Namespace),
					fmt.Sprintf("zip-%s-%s", tenancy.Partition, tenancy.Namespace),
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("no_exist", tenancy),
				fmt.Sprintf("test-kv-keys/prefix/not/a/real/key/like/ever?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				[]string{},
			},
		}
	})

	cases = append(cases, tenancyHelper.GenerateDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("exists", tenancy),
				fmt.Sprintf("test-kv-keys/prefix"),
				[]string{
					fmt.Sprintf("foo-%s-%s", tenancy.Partition, tenancy.Namespace),
					fmt.Sprintf("wave/ocean-%s-%s", tenancy.Partition, tenancy.Namespace),
					fmt.Sprintf("zip-%s-%s", tenancy.Partition, tenancy.Namespace),
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("trailing", tenancy),
				fmt.Sprintf("test-kv-keys/prefix/"),
				[]string{
					fmt.Sprintf("foo-%s-%s", tenancy.Partition, tenancy.Namespace),
					fmt.Sprintf("wave/ocean-%s-%s", tenancy.Partition, tenancy.Namespace),
					fmt.Sprintf("zip-%s-%s", tenancy.Partition, tenancy.Namespace),
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("no_exist", tenancy),
				fmt.Sprintf("test-kv-keys/prefix/not/a/real/key/like/ever"),
				[]string{},
			},
		}
	})...)

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVKeysQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(testClients, nil)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.exp, act)
		})
	}

	tenancyHelper.RunWithTenancies(func(tenancy *test.Tenancy) {
		kvQuery := fmt.Sprintf("test-kv-keys/prefix")
		if tenancyHelper.IsConsulEnterprise() {
			kvQuery += fmt.Sprintf("?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace)
		}
		d, err := NewKVKeysQuery(kvQuery)
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
		case <-time.After(100 * time.Millisecond):
			t.Errorf("did not stop")
		}
	}, t, "stops")

	tenancyHelper.RunWithTenancies(func(tenancy *test.Tenancy) {
		kvQuery := fmt.Sprintf("test-kv-keys/prefix")
		if tenancyHelper.IsConsulEnterprise() {
			kvQuery += fmt.Sprintf("?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace)
		}
		d, err := NewKVKeysQuery(kvQuery)
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
			data, _, err := d.Fetch(testClients, &QueryOptions{WaitIndex: qm.LastIndex})
			if err != nil {
				errCh <- err
				return
			}
			dataCh <- data
		}()

		zebraKey := fmt.Sprintf("test-kv-keys/prefix/zebra-%s-%s", tenancy.Partition, tenancy.Namespace)
		if tenancyHelper.IsConsulEnterprise() {
			zebraKey += fmt.Sprintf("?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace)
		}
		testConsul.SetKVString(t, zebraKey, "value")

		select {
		case err := <-errCh:
			t.Fatal(err)
		case act := <-dataCh:
			exp := []string{
				fmt.Sprintf("foo-%s-%s", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("wave/ocean-%s-%s", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("zebra-%s-%s", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("zip-%s-%s", tenancy.Partition, tenancy.Namespace),
			}
			assert.Equal(t, exp, act)
		}
	}, t, "fires_changes")
}

func TestKVKeysQuery_String(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  string
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("prefix", tenancy),
				"prefix",
				"kv.keys(prefix)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc", tenancy),
				"prefix@dc1",
				"kv.keys(prefix@dc1)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_partition", tenancy),
				fmt.Sprintf("prefix?partition=%s@dc1", tenancy.Partition),
				fmt.Sprintf("kv.keys(prefix@dc1@partition=%s)", tenancy.Partition),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_partition_ns", tenancy),
				fmt.Sprintf("prefix?partition=%s&ns=%s@dc1", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("kv.keys(prefix@dc1@partition=%s@ns=%s)", tenancy.Partition, tenancy.Namespace),
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVKeysQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
