// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"fmt"
	"github.com/hashicorp/consul-template/test"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCatalogNodeQuery(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  *CatalogNodeQuery
		err  bool
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("empty", tenancy),
				"",
				&CatalogNodeQuery{},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("invalid query param (unsupported key)", tenancy),
				"node?unsupported=foo",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("bad", tenancy),
				"!4d",
				nil,
				true,
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
				tenancyHelper.AppendTenancyInfo("node", tenancy),
				"node",
				&CatalogNodeQuery{
					name: "node",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc", tenancy),
				"node@dc1",
				&CatalogNodeQuery{
					name: "node",
					dc:   "dc1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("every_option", tenancy),
				fmt.Sprintf("node?ns=%s&partition=%s@dc1", tenancy.Namespace, tenancy.Partition),
				&CatalogNodeQuery{
					name:      "node",
					dc:        "dc1",
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition", tenancy),
				fmt.Sprintf("node?&partition=%s@dc1", tenancy.Partition),
				&CatalogNodeQuery{
					name:      "node",
					dc:        "dc1",
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace", tenancy),
				fmt.Sprintf("node?ns=%s@dc1", tenancy.Namespace),
				&CatalogNodeQuery{
					name:      "node",
					dc:        "dc1",
					namespace: tenancy.Namespace,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("periods", tenancy),
				"node.bar.com@dc1",
				&CatalogNodeQuery{
					name: "node.bar.com",
					dc:   "dc1",
				},
				false,
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewCatalogNodeQuery(tc.i)
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

func TestCatalogNodeQuery_Fetch(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  *CatalogNode
	}
	cases := tenancyHelper.GenerateNonDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("local", tenancy),
				"",
				&CatalogNode{
					Node: &Node{
						Node:       testConsul.Config.NodeName,
						Address:    testConsul.Config.Bind,
						Datacenter: "dc1",
						TaggedAddresses: map[string]string{
							"lan": "127.0.0.1",
							"wan": "127.0.0.1",
						},
						Meta: map[string]string{
							"consul-network-segment": "",
						},
					},
					Services: []*CatalogNodeService{
						{
							ID:      "conn-enabled-service-default-default",
							Service: "conn-enabled-service-default-default",
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
							Port:    12345,
						},
						{
							ID:      "conn-enabled-service-proxy-default-default",
							Service: "conn-enabled-service-proxy-default-default",
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
							Port:    21999,
						},
						{
							ID:      "consul",
							Service: "consul",
							Port:    testConsul.Config.Ports.Server,
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
						},
						{
							ID:      "service-meta-default-default",
							Service: "service-meta-default-default",
							Tags:    ServiceTags([]string{"tag1"}),
							Meta: map[string]string{
								"meta1": "value1",
							},
						},
						{
							ID:      "service-taggedAddresses-default-default",
							Service: "service-taggedAddresses-default-default",
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition and ns", tenancy),
				fmt.Sprintf("%s?partition=%s&ns=%s", testConsul.Config.NodeName, tenancy.Partition, tenancy.Namespace),
				&CatalogNode{
					Node: &Node{
						Node:       testConsul.Config.NodeName,
						Address:    testConsul.Config.Bind,
						Datacenter: "dc1",
						TaggedAddresses: map[string]string{
							"lan": "127.0.0.1",
							"wan": "127.0.0.1",
						},
						Meta: map[string]string{
							"consul-network-segment": "",
						},
					},
					Services: []*CatalogNodeService{
						{
							ID:      fmt.Sprintf("conn-enabled-service-%s-%s", tenancy.Partition, tenancy.Namespace),
							Service: fmt.Sprintf("conn-enabled-service-%s-%s", tenancy.Partition, tenancy.Namespace),
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
							Port:    12345,
						},
						{
							ID:      fmt.Sprintf("conn-enabled-service-proxy-%s-%s", tenancy.Partition, tenancy.Namespace),
							Service: fmt.Sprintf("conn-enabled-service-proxy-%s-%s", tenancy.Partition, tenancy.Namespace),
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
							Port:    21999,
						},
						{

							ID:      fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
							Service: fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
							Tags:    ServiceTags([]string{"tag1"}),
							Meta: map[string]string{
								"meta1": "value1",
							},
						},
						{
							ID:      fmt.Sprintf("service-taggedAddresses-%s-%s", tenancy.Partition, tenancy.Namespace),
							Service: fmt.Sprintf("service-taggedAddresses-%s-%s", tenancy.Partition, tenancy.Namespace),
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("unknown", tenancy),
				"not_a_real_node",
				&CatalogNode{},
			},
		}
	})

	cases = append(cases, tenancyHelper.GenerateDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("local", tenancy),
				"",
				&CatalogNode{
					Node: &Node{
						Node:       testConsul.Config.NodeName,
						Address:    testConsul.Config.Bind,
						Datacenter: "dc1",
						TaggedAddresses: map[string]string{
							"lan": "127.0.0.1",
							"wan": "127.0.0.1",
						},
						Meta: map[string]string{
							"consul-network-segment": "",
						},
					},
					Services: []*CatalogNodeService{
						{
							ID:      "conn-enabled-service-default-default",
							Service: "conn-enabled-service-default-default",
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
							Port:    12345,
						},
						{
							ID:      "conn-enabled-service-proxy-default-default",
							Service: "conn-enabled-service-proxy-default-default",
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
							Port:    21999,
						},
						{
							ID:      "consul",
							Service: "consul",
							Port:    testConsul.Config.Ports.Server,
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
						},
						{
							ID:      "service-meta-default-default",
							Service: "service-meta-default-default",
							Tags:    ServiceTags([]string{"tag1"}),
							Meta: map[string]string{
								"meta1": "value1",
							},
						},
						{
							ID:      "service-taggedAddresses-default-default",
							Service: "service-taggedAddresses-default-default",
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition and ns", tenancy),
				fmt.Sprintf("%s?partition=%s&ns=%s", testConsul.Config.NodeName, tenancy.Partition, tenancy.Namespace),
				&CatalogNode{
					Node: &Node{
						Node:       testConsul.Config.NodeName,
						Address:    testConsul.Config.Bind,
						Datacenter: "dc1",
						TaggedAddresses: map[string]string{
							"lan": "127.0.0.1",
							"wan": "127.0.0.1",
						},
						Meta: map[string]string{
							"consul-network-segment": "",
						},
					},
					Services: []*CatalogNodeService{
						{
							ID:      fmt.Sprintf("conn-enabled-service-%s-%s", tenancy.Partition, tenancy.Namespace),
							Service: fmt.Sprintf("conn-enabled-service-%s-%s", tenancy.Partition, tenancy.Namespace),
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
							Port:    12345,
						},
						{
							ID:      fmt.Sprintf("conn-enabled-service-proxy-%s-%s", tenancy.Partition, tenancy.Namespace),
							Service: fmt.Sprintf("conn-enabled-service-proxy-%s-%s", tenancy.Partition, tenancy.Namespace),
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
							Port:    21999,
						},
						{
							ID:      "consul",
							Service: "consul",
							Port:    testConsul.Config.Ports.Server,
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
						},
						{

							ID:      fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
							Service: fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
							Tags:    ServiceTags([]string{"tag1"}),
							Meta: map[string]string{
								"meta1": "value1",
							},
						},
						{
							ID:      fmt.Sprintf("service-taggedAddresses-%s-%s", tenancy.Partition, tenancy.Namespace),
							Service: fmt.Sprintf("service-taggedAddresses-%s-%s", tenancy.Partition, tenancy.Namespace),
							Tags:    ServiceTags([]string{}),
							Meta:    map[string]string{},
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("unknown", tenancy),
				"not_a_real_node",
				&CatalogNode{},
			},
		}
	})...)

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewCatalogNodeQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(testClients, nil)
			if err != nil {
				t.Fatal(err)
			}

			if act != nil {
				if n := act.(*CatalogNode).Node; n != nil {
					n.ID = ""
					n.TaggedAddresses = filterAddresses(n.TaggedAddresses)
					n.Meta = filterVersionMeta(n.Meta)
				}
				// delete any version data from ServiceMeta
				services := act.(*CatalogNode).Services
				for i := range services {
					services[i].Meta = filterVersionMeta(services[i].Meta)
				}
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestCatalogNodeQuery_String(t *testing.T) {
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
				"catalog.node",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("node", tenancy),
				"node1",
				"catalog.node(node1)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("datacenter", tenancy),
				"node1@dc1",
				"catalog.node(node1@dc1)",
			},
			testCase{
				name: tenancyHelper.AppendTenancyInfo("partition", tenancy),
				i:    fmt.Sprintf("node1?&partition=%s&ns=%s@dc1", tenancy.Partition, tenancy.Namespace),
				exp:  fmt.Sprintf("catalog.node(node1@dc1@partition=%s@ns=%s)", tenancy.Partition, tenancy.Namespace),
			},
			testCase{
				name: tenancyHelper.AppendTenancyInfo("namespace", tenancy),
				i:    fmt.Sprintf("node1?&partition=%s@dc1", tenancy.Partition),
				exp:  fmt.Sprintf("catalog.node(node1@dc1@partition=%s)", tenancy.Partition),
			},
		}
	})
	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewCatalogNodeQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
