// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"fmt"
	"github.com/hashicorp/consul-template/test"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCatalogServiceQuery(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  *CatalogServiceQuery
		err  bool
	}

	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("empty", tenancy),
				"",
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
				tenancyHelper.AppendTenancyInfo("invalid query param (unsupported key)", tenancy),
				"name?unsupported=foo",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("near_only", tenancy),
				"~near",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_only", tenancy),
				"tag.",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name", tenancy),
				"name",
				&CatalogServiceQuery{
					name: "name",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_dc", tenancy),
				"name@dc1",
				&CatalogServiceQuery{
					dc:   "dc1",
					name: "name",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_query", tenancy),
				fmt.Sprintf("name?ns=%s", tenancy.Namespace),
				&CatalogServiceQuery{
					name:      "name",
					namespace: tenancy.Namespace,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_dc_near", tenancy),
				"name@dc1~near",
				&CatalogServiceQuery{
					dc:   "dc1",
					name: "name",
					near: "near",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_query_near", tenancy),
				fmt.Sprintf("name?ns=%s~near", tenancy.Namespace),
				&CatalogServiceQuery{
					name:      "name",
					near:      "near",
					namespace: tenancy.Namespace,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_near", tenancy),
				"name~near",
				&CatalogServiceQuery{
					name: "name",
					near: "near",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name", tenancy),
				"tag.name",
				&CatalogServiceQuery{
					name: "name",
					tag:  "tag",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc", tenancy),
				"tag.name@dc",
				&CatalogServiceQuery{
					dc:   "dc",
					name: "name",
					tag:  "tag",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_near", tenancy),
				"tag.name~near",
				&CatalogServiceQuery{
					name: "name",
					near: "near",
					tag:  "tag",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("every_option", tenancy),
				fmt.Sprintf("tag.name?ns=%s&partition=%s@dc~near", tenancy.Namespace, tenancy.Partition),
				&CatalogServiceQuery{
					dc:        "dc",
					name:      "name",
					near:      "near",
					tag:       "tag",
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_with_colon", tenancy),
				"tag:value.name",
				&CatalogServiceQuery{
					name: "name",
					tag:  "tag:value",
				},
				false,
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewCatalogServiceQuery(tc.i)
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

func TestCatalogServiceQuery_Fetch(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  []*CatalogService
	}
	cases := tenancyHelper.GenerateDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("consul", tenancy),
				"consul",
				[]*CatalogService{
					{
						Node:            testConsul.Config.NodeName,
						Address:         testConsul.Config.Bind,
						Datacenter:      "dc1",
						TaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceID:      "consul",
						ServiceName:    "consul",
						ServiceAddress: "",
						ServiceTags:    ServiceTags([]string{}),
						ServiceMeta:    map[string]string{},
						ServicePort:    testConsul.Config.Ports.Server,
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("service-meta", tenancy),
				"service-meta-default-default",
				[]*CatalogService{
					{
						Node:            testConsul.Config.NodeName,
						Address:         testConsul.Config.Bind,
						Datacenter:      "dc1",
						TaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceID:      "service-meta-default-default",
						ServiceName:    "service-meta-default-default",
						ServiceAddress: "",
						ServiceTags:    ServiceTags([]string{"tag1"}),
						ServiceMeta:    map[string]string{"meta1": "value1"},
					},
				},
			},
		}
	})

	cases = append(cases, tenancyHelper.GenerateNonDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("consul", tenancy),
				fmt.Sprintf("consul?ns=%s&partition=%s", tenancy.Namespace, tenancy.Partition),
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("service-meta", tenancy),
				fmt.Sprintf("service-meta-%s-%s?ns=%s&partition=%s", tenancy.Partition, tenancy.Namespace, tenancy.Namespace, tenancy.Partition),
				[]*CatalogService{
					{
						Node:            testConsul.Config.NodeName,
						Address:         testConsul.Config.Bind,
						Datacenter:      "dc1",
						TaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceID:      fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
						ServiceName:    fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
						ServiceAddress: "",
						ServiceTags:    ServiceTags([]string{"tag1"}),
						ServiceMeta:    map[string]string{"meta1": "value1"},
					},
				},
			},
		}
	})...)

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewCatalogServiceQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(testClients, nil)
			if err != nil {
				t.Fatal(err)
			}

			if act != nil {
				for _, s := range act.([]*CatalogService) {
					s.ID = ""
					s.TaggedAddresses = filterAddresses(s.TaggedAddresses)
				}
			}

			// delete any version data from ServiceMeta
			act_list := act.([]*CatalogService)
			for i := range act_list {
				act_list[i].ServiceMeta = filterVersionMeta(
					act_list[i].ServiceMeta)
				act_list[i].NodeMeta = filterVersionMeta(
					act_list[i].NodeMeta)
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestCatalogServiceQuery_String(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  string
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("name", tenancy),
				"name",
				"catalog.service(name)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_dc", tenancy),
				"name@dc",
				"catalog.service(name@dc)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_near", tenancy),
				"name~near",
				"catalog.service(name~near)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_dc_near", tenancy),
				"name@dc~near",
				"catalog.service(name@dc~near)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name", tenancy),
				"tag.name",
				"catalog.service(tag.name)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc", tenancy),
				"tag.name@dc",
				"catalog.service(tag.name@dc)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_near", tenancy),
				"tag.name~near",
				"catalog.service(tag.name~near)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc_near", tenancy),
				"tag.name@dc~near",
				"catalog.service(tag.name@dc~near)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc_near_ns", tenancy),
				fmt.Sprintf("tag.name?ns=%s@dc~near", tenancy.Namespace),
				fmt.Sprintf("catalog.service(tag.name@dc@ns=%s~near)", tenancy.Namespace),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc_near_ns_partiton", tenancy),
				fmt.Sprintf("tag.name?ns=%s&partition=%s@dc~near", tenancy.Namespace, tenancy.Partition),
				fmt.Sprintf("catalog.service(tag.name@dc@partition=%s@ns=%s~near)", tenancy.Partition, tenancy.Namespace),
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewCatalogServiceQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
