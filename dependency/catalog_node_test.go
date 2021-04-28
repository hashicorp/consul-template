package dependency

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCatalogNodeQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  *CatalogNodeQuery
		err  bool
	}{
		{
			"empty",
			"",
			&CatalogNodeQuery{},
			false,
		},
		{
			"bad",
			"!4d",
			nil,
			true,
		},
		{
			"dc_only",
			"@dc1",
			nil,
			true,
		},
		{
			"node",
			"node",
			&CatalogNodeQuery{
				name: "node",
			},
			false,
		},
		{
			"dc",
			"node@dc1",
			&CatalogNodeQuery{
				name: "node",
				dc:   "dc1",
			},
			false,
		},
		{
			"periods",
			"node.bar.com@dc1",
			&CatalogNodeQuery{
				name: "node.bar.com",
				dc:   "dc1",
			},
			false,
		},
	}

	for i, tc := range cases {
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
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  *CatalogNode
	}{
		{
			"local",
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
					&CatalogNodeService{
						ID:      "consul",
						Service: "consul",
						Port:    testConsul.Config.Ports.Server,
						Tags:    ServiceTags([]string{}),
						Meta:    map[string]string{},
					},
					&CatalogNodeService{
						ID:      "foo",
						Service: "foo-sidecar-proxy",
						Tags:    ServiceTags([]string{}),
						Meta:    map[string]string{},
						Port:    21999,
					},
					&CatalogNodeService{
						ID:      "service-meta",
						Service: "service-meta",
						Tags:    ServiceTags([]string{"tag1"}),
						Meta: map[string]string{
							"meta1": "value1",
						},
					},
					&CatalogNodeService{
						ID:      "service-taggedAddresses",
						Service: "service-taggedAddresses",
						Tags:    ServiceTags([]string{}),
						Meta:    map[string]string{},
					},
				},
			},
		},
		{
			"unknown",
			"not_a_real_node",
			&CatalogNode{},
		},
	}

	for i, tc := range cases {
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
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  string
	}{
		{
			"empty",
			"",
			"catalog.node",
		},
		{
			"node",
			"node1",
			"catalog.node(node1)",
		},
		{
			"datacenter",
			"node1@dc1",
			"catalog.node(node1@dc1)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewCatalogNodeQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
