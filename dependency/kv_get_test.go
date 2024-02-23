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

func TestNewKVGetQuery(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  *KVGetQuery
		err  bool
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("empty", tenancy),
				"",
				&KVGetQuery{},
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
				"key?unsupported=foo",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("key", tenancy),
				"key",
				&KVGetQuery{
					key: "key",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc", tenancy),
				"key@dc1",
				&KVGetQuery{
					key: "key",
					dc:  "dc1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition", tenancy),
				fmt.Sprintf("key?partition=%s", tenancy.Partition),
				&KVGetQuery{
					key:       "key",
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace", tenancy),
				fmt.Sprintf("key?ns=%s", tenancy.Namespace),
				&KVGetQuery{
					key:       "key",
					namespace: tenancy.Namespace,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace_and_partition", tenancy),
				fmt.Sprintf("key?ns=%s&partition=%s", tenancy.Namespace, tenancy.Partition),
				&KVGetQuery{
					key:       "key",
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace_and_partition_and_dc", tenancy),
				fmt.Sprintf("key?ns=%s&partition=%s@dc1", tenancy.Namespace, tenancy.Partition),
				&KVGetQuery{
					key:       "key",
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
					dc:        "dc1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("empty_query", tenancy),
				"key?ns=&partition=",
				&KVGetQuery{
					key:       "key",
					namespace: "",
					partition: "",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dots", tenancy),
				"key.with.dots",
				&KVGetQuery{
					key: "key.with.dots",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("slashes", tenancy),
				"key/with/slashes",
				&KVGetQuery{
					key: "key/with/slashes",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dashes", tenancy),
				"key-with-dashes",
				&KVGetQuery{
					key: "key-with-dashes",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("leading_slash", tenancy),
				"/leading/slash",
				&KVGetQuery{
					key: "leading/slash",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("trailing_slash", tenancy),
				"trailing/slash/",
				&KVGetQuery{
					key: "trailing/slash/",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("underscores", tenancy),
				"key_with_underscores",
				&KVGetQuery{
					key: "key_with_underscores",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("special_characters", tenancy),
				"config/facet:größe-lf-si",
				&KVGetQuery{
					key: "config/facet:größe-lf-si",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("splat", tenancy),
				"config/*/timeouts/",
				&KVGetQuery{
					key: "config/*/timeouts/",
				},
				false,
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
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

	for _, tenancy := range tenancyHelper.TestTenancies() {
		key := "test-kv-get/key"
		if tenancyHelper.IsConsulEnterprise() {
			key += fmt.Sprintf("?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace)
		}
		testConsul.SetKVString(t, key, fmt.Sprintf("value-%s-%s", tenancy.Partition, tenancy.Namespace))

		emptyKey := "test-kv-get/key_empty"
		if tenancyHelper.IsConsulEnterprise() {
			emptyKey += fmt.Sprintf("?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace)
		}
		testConsul.SetKVString(t, emptyKey, "")
	}

	type testCase struct {
		name string
		i    string
		exp  interface{}
	}

	cases := tenancyHelper.GenerateNonDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("exists", tenancy),
				fmt.Sprintf("test-kv-get/key?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("value-%s-%s", tenancy.Partition, tenancy.Namespace),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("exists_empty_string", tenancy),
				fmt.Sprintf("test-kv-get/key_empty?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				"",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("no_exist", tenancy),
				fmt.Sprintf("test-kv-get/not/a/real/key/like/ever?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				nil,
			},
		}
	})

	cases = append(cases, tenancyHelper.GenerateDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("exists", tenancy),
				"test-kv-get/key",
				"value-default-default",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("exists_empty_string", tenancy),
				"test-kv-get/key_empty",
				"",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("no_exist", tenancy),
				"test-kv-get/not/a/real/key/like/ever",
				nil,
			},
		}
	})...)

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVGetQuery(tc.i)
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
		kvQuery := "test-kv-get/key"
		if tenancyHelper.IsConsulEnterprise() {
			kvQuery += fmt.Sprintf("?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace)
		}
		d, err := NewKVGetQuery(kvQuery)
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
		kvQuery := "test-kv-get/key"
		if tenancyHelper.IsConsulEnterprise() {
			kvQuery += fmt.Sprintf("?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace)
		}
		d, err := NewKVGetQuery(kvQuery)
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

		testConsul.SetKVString(t, kvQuery, "new-value")

		select {
		case err := <-errCh:
			t.Fatal(err)
		case data := <-dataCh:
			assert.Equal(t, data, "new-value")
		}
	}, t, "fires_changes")
}

func TestKVGetQuery_String(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  string
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("key", tenancy),
				"key",
				"kv.get(key)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc", tenancy),
				"key@dc1",
				"kv.get(key@dc1)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_and_partition", tenancy),
				fmt.Sprintf("key?partition=%s@dc1", tenancy.Partition),
				fmt.Sprintf("kv.get(key@dc1@partition=%s)", tenancy.Partition),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_and_partition_and_ns", tenancy),
				fmt.Sprintf("key?partition=%s&ns=%s@dc1", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("kv.get(key@dc1@partition=%s@ns=%s)", tenancy.Partition, tenancy.Namespace),
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVGetQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
