// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"fmt"
	"github.com/hashicorp/consul-template/test"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCatalogServicesQuery(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  *CatalogServicesQuery
		err  bool
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("empty", tenancy),
				"",
				&CatalogServicesQuery{},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("invalid query param (unsupported key)", tenancy),
				"?unsupported=foo",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("node", tenancy),
				"node",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc", tenancy),
				"@dc1",
				&CatalogServicesQuery{
					dc: "dc1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace", tenancy),
				fmt.Sprintf("?ns=%s", tenancy.Namespace),
				&CatalogServicesQuery{
					namespace: tenancy.Namespace,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition", tenancy),
				fmt.Sprintf("?partition=%s", tenancy.Partition),
				&CatalogServicesQuery{
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition_and_namespace", tenancy),
				fmt.Sprintf("?ns=%s&partition=%s", tenancy.Namespace, tenancy.Partition),
				&CatalogServicesQuery{
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition_and_namespace_and_dc", tenancy),
				fmt.Sprintf("?ns=%s&partition=%s@dc1", tenancy.Namespace, tenancy.Partition),
				&CatalogServicesQuery{
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
					dc:        "dc1",
				},
				false,
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewCatalogServicesQuery(tc.i)
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

func TestCatalogServicesQuery_Fetch(t *testing.T) {
	type testCase struct {
		name    string
		i       string
		tenancy *test.Tenancy
		opts    *QueryOptions
		exp     []*CatalogSnippet
		err     bool
	}
	cases := tenancyHelper.GenerateDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("all", tenancy),
				"",
				tenancy,
				nil,
				[]*CatalogSnippet{
					{
						Name: "conn-enabled-service-default-default",
						Tags: ServiceTags([]string{}),
					},
					{
						Name: "conn-enabled-service-proxy-default-default",
						Tags: ServiceTags([]string{}),
					},
					{
						Name: "consul",
						Tags: ServiceTags([]string{}),
					},
					{
						Name: "service-meta-default-default",
						Tags: ServiceTags([]string{"tag1"}),
					},
					{
						Name: "service-taggedAddresses-default-default",
						Tags: ServiceTags([]string{}),
					},
				},
				false,
			},
		}
	})

	cases = append(cases, tenancyHelper.GenerateNonDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("all", tenancy),
				"",
				tenancy,
				nil,
				[]*CatalogSnippet{
					{
						Name: "conn-enabled-service-default-default",
						Tags: ServiceTags([]string{}),
					},
					{
						Name: "conn-enabled-service-proxy-default-default",
						Tags: ServiceTags([]string{}),
					},
					{
						Name: "consul",
						Tags: ServiceTags([]string{}),
					},
					{
						Name: "service-meta-default-default",
						Tags: ServiceTags([]string{"tag1"}),
					},
					{
						Name: "service-taggedAddresses-default-default",
						Tags: ServiceTags([]string{}),
					},
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition_and_ns", tenancy),
				fmt.Sprintf("?ns=%s&partition=%s", tenancy.Namespace, tenancy.Partition),
				tenancy,
				nil,
				[]*CatalogSnippet{
					{
						Name: fmt.Sprintf("conn-enabled-service-%s-%s", tenancy.Partition, tenancy.Namespace),
						Tags: ServiceTags([]string{}),
					},
					{
						Name: fmt.Sprintf("conn-enabled-service-proxy-%s-%s", tenancy.Partition, tenancy.Namespace),
						Tags: ServiceTags([]string{}),
					},
					{
						Name: fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
						Tags: ServiceTags([]string{"tag1"}),
					},
					{
						Name: fmt.Sprintf("service-taggedAddresses-%s-%s", tenancy.Partition, tenancy.Namespace),
						Tags: ServiceTags([]string{}),
					},
				},
				false,
			},
		}
	})...)

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {

			d, err := NewCatalogServicesQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(getTestClientsForTenancy(tc.tenancy), tc.opts)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if act == nil && tc.err {
				return
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestCatalogServicesQuery_String(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  string
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("empty", tenancy),
				"",
				"catalog.services",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("datacenter", tenancy),
				"@dc1",
				"catalog.services(@dc1)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("datacenter+namespace", tenancy),
				fmt.Sprintf("?ns=%s@dc1", tenancy.Namespace),
				fmt.Sprintf("catalog.services(@dc1@ns=%s)", tenancy.Namespace),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("datacenter+namespace+partition", tenancy),
				fmt.Sprintf("?partition=%s&ns=%s@dc1", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("catalog.services(@dc1@partition=%s@ns=%s)", tenancy.Partition, tenancy.Namespace),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace+partition", tenancy),
				fmt.Sprintf("?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("catalog.services(@partition=%s@ns=%s)", tenancy.Partition, tenancy.Namespace),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc+partition", tenancy),
				fmt.Sprintf("?partition=%s@dc1", tenancy.Partition),
				fmt.Sprintf("catalog.services(@dc1@partition=%s)", tenancy.Partition),
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewCatalogServicesQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
