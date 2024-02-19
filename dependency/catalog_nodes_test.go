// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"fmt"
	"github.com/hashicorp/consul-template/test"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCatalogNodesQuery(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  *CatalogNodesQuery
		err  bool
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("empty", tenancy),
				"",
				&CatalogNodesQuery{},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("invalid query param (unsupported key)", tenancy),
				"key?unsupported=foo",
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
				&CatalogNodesQuery{
					dc: "dc1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace", tenancy),
				fmt.Sprintf("?ns=%s", tenancy.Namespace),
				&CatalogNodesQuery{
					namespace: tenancy.Namespace,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition", tenancy),
				fmt.Sprintf("?partition=%s", tenancy.Partition),
				&CatalogNodesQuery{
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace_and_partition", tenancy),
				fmt.Sprintf("?ns=%s&partition=%s", tenancy.Namespace, tenancy.Partition),
				&CatalogNodesQuery{
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace_and_partition_and_near", tenancy),
				fmt.Sprintf("?ns=%s&partition=%s~node1", tenancy.Namespace, tenancy.Partition),
				&CatalogNodesQuery{
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
					near:      "node1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("near", tenancy),
				"~node1",
				&CatalogNodesQuery{
					near: "node1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_near", tenancy),
				"@dc1~node1",
				&CatalogNodesQuery{
					dc:   "dc1",
					near: "node1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("query_near", tenancy),
				fmt.Sprintf("?ns=%s~node1", tenancy.Namespace),
				&CatalogNodesQuery{
					namespace: tenancy.Namespace,
					near:      "node1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("every_option", tenancy),
				fmt.Sprintf("?ns=%s&partition=%s@dc1~node1", tenancy.Namespace, tenancy.Partition),
				&CatalogNodesQuery{
					dc:        "dc1",
					near:      "node1",
					partition: tenancy.Partition,
					namespace: tenancy.Namespace,
				},
				false,
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewCatalogNodesQuery(tc.i)
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

func TestCatalogNodesQuery_Fetch(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  []*Node
	}
	cases := tenancyHelper.GenerateNonDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("all", tenancy),
				"",
				[]*Node{
					{
						Node:            testConsul.Config.NodeName,
						Address:         testConsul.Config.Bind,
						Datacenter:      "dc1",
						TaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						Meta: map[string]string{
							//"consul-network-segment": "",
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition and namespace", tenancy),
				fmt.Sprintf("?partition=%s&ns=%s@dc1", tenancy.Partition, tenancy.Namespace),
				[]*Node{
					{
						Node:            testConsul.Config.NodeName,
						Address:         testConsul.Config.Bind,
						Datacenter:      "dc1",
						TaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						Meta: map[string]string{
							//"consul-network-segment": "",
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace", tenancy),
				fmt.Sprintf("?ns=%s@dc1", tenancy.Namespace),
				[]*Node{
					{
						Node:            testConsul.Config.NodeName,
						Address:         testConsul.Config.Bind,
						Datacenter:      "dc1",
						TaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						Meta: map[string]string{
							//"consul-network-segment": "",
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition", tenancy),
				fmt.Sprintf("?partition=%s@dc1", tenancy.Partition),
				[]*Node{
					{
						Node:            testConsul.Config.NodeName,
						Address:         testConsul.Config.Bind,
						Datacenter:      "dc1",
						TaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						Meta: map[string]string{
							//"consul-network-segment": "",
						},
					},
				},
			},
		}
	})

	cases = append(cases, tenancyHelper.GenerateDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("all", tenancy),
				"",
				[]*Node{
					{
						Node:            testConsul.Config.NodeName,
						Address:         testConsul.Config.Bind,
						Datacenter:      "dc1",
						TaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						Meta: map[string]string{
							//"consul-network-segment": "",
						},
					},
				},
			},
		}
	})...)

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewCatalogNodesQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(testClients, nil)
			if err != nil {
				t.Fatal(err)
			}

			if act != nil {
				for _, n := range act.([]*Node) {
					n.ID = ""
					n.TaggedAddresses = filterAddresses(n.TaggedAddresses)
					n.Meta = filterVersionMeta(n.Meta)
				}
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestCatalogNodesQuery_String(t *testing.T) {
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
				"catalog.nodes",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("datacenter", tenancy),
				"@dc1",
				"catalog.nodes(@dc1)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("near", tenancy),
				"~node1",
				"catalog.nodes(~node1)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("datacenter_near", tenancy),
				"@dc1~node1",
				"catalog.nodes(@dc1~node1)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition", tenancy),
				fmt.Sprintf("?partition=%s@dc1~node1", tenancy.Partition),
				fmt.Sprintf("catalog.nodes(@dc1@partition=%s~node1)", tenancy.Partition),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace and partition", tenancy),
				fmt.Sprintf("?partition=%s&ns=%s@dc1~node1", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("catalog.nodes(@dc1@partition=%s@ns=%s~node1)", tenancy.Partition, tenancy.Namespace),
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewCatalogNodesQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
