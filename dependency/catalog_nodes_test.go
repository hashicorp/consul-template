// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCatalogNodesQuery(t *testing.T) {
	cases := []struct {
		name string
		i    string
		exp  *CatalogNodesQuery
		err  bool
	}{
		{
			"empty",
			"",
			&CatalogNodesQuery{},
			false,
		},
		{
			"node",
			"node",
			nil,
			true,
		},
		{
			"dc",
			"@dc1",
			&CatalogNodesQuery{
				dc: "dc1",
			},
			false,
		},
		{
			"near",
			"~node1",
			&CatalogNodesQuery{
				near: "node1",
			},
			false,
		},
		{
			"dc_near",
			"@dc1~node1",
			&CatalogNodesQuery{
				dc:   "dc1",
				near: "node1",
			},
			false,
		},
	}

	for i, tc := range cases {
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
	cases := []struct {
		name string
		i    string
		exp  []*Node
	}{
		{
			"all",
			"",
			[]*Node{
				{
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
			},
		},
	}

	for i, tc := range cases {
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
				}
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestCatalogNodesQuery_String(t *testing.T) {
	cases := []struct {
		name string
		i    string
		exp  string
	}{
		{
			"empty",
			"",
			"catalog.nodes",
		},
		{
			"datacenter",
			"@dc1",
			"catalog.nodes(@dc1)",
		},
		{
			"near",
			"~node1",
			"catalog.nodes(~node1)",
		},
		{
			"datacenter_near",
			"@dc1~node1",
			"catalog.nodes(@dc1~node1)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewCatalogNodesQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
