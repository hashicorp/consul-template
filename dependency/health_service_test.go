package dependency

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHealthServiceQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  *HealthServiceQuery
		err  bool
	}{
		{
			"empty",
			"",
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
			"near_only",
			"~near",
			nil,
			true,
		},
		{
			"tag_only",
			"tag.",
			nil,
			true,
		},
		{
			"name",
			"name",
			&HealthServiceQuery{
				filters: []string{"passing"},
				name:    "name",
			},
			false,
		},
		{
			"name_dc",
			"name@dc1",
			&HealthServiceQuery{
				dc:      "dc1",
				filters: []string{"passing"},
				name:    "name",
			},
			false,
		},
		{
			"name_dc_near",
			"name@dc1~near",
			&HealthServiceQuery{
				dc:      "dc1",
				filters: []string{"passing"},
				name:    "name",
				near:    "near",
			},
			false,
		},
		{
			"name_near",
			"name~near",
			&HealthServiceQuery{
				filters: []string{"passing"},
				name:    "name",
				near:    "near",
			},
			false,
		},
		{
			"tag_name",
			"tag.name",
			&HealthServiceQuery{
				filters: []string{"passing"},
				name:    "name",
				tag:     "tag",
			},
			false,
		},
		{
			"tag_name_dc",
			"tag.name@dc",
			&HealthServiceQuery{
				dc:      "dc",
				filters: []string{"passing"},
				name:    "name",
				tag:     "tag",
			},
			false,
		},
		{
			"tag_name_near",
			"tag.name~near",
			&HealthServiceQuery{
				filters: []string{"passing"},
				name:    "name",
				near:    "near",
				tag:     "tag",
			},
			false,
		},
		{
			"tag_name_dc_near",
			"tag.name@dc~near",
			&HealthServiceQuery{
				dc:      "dc",
				filters: []string{"passing"},
				name:    "name",
				near:    "near",
				tag:     "tag",
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewHealthServiceQuery(tc.i)
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

func TestHealthServiceQuery_Fetch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  []*HealthService
	}{
		{
			"consul",
			"consul",
			[]*HealthService{
				&HealthService{
					Node:        testConsul.Config.NodeName,
					NodeAddress: testConsul.Config.Bind,
					NodeTaggedAddresses: map[string]string{
						"lan": "127.0.0.1",
						"wan": "127.0.0.1",
					},
					NodeMeta: map[string]string{},
					Address:  testConsul.Config.Bind,
					ID:       "consul",
					Name:     "consul",
					Tags:     []string{},
					Status:   "passing",
					Port:     testConsul.Config.Ports.Server,
				},
			},
		},
		{
			"filters",
			"consul|warning",
			[]*HealthService{},
		},
		{
			"multifilter",
			"consul|warning,passing",
			[]*HealthService{
				&HealthService{
					Node:        testConsul.Config.NodeName,
					NodeAddress: testConsul.Config.Bind,
					NodeTaggedAddresses: map[string]string{
						"lan": "127.0.0.1",
						"wan": "127.0.0.1",
					},
					NodeMeta: map[string]string{},
					Address:  testConsul.Config.Bind,
					ID:       "consul",
					Name:     "consul",
					Tags:     []string{},
					Status:   "passing",
					Port:     testConsul.Config.Ports.Server,
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewHealthServiceQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(testClients, nil)
			if err != nil {
				t.Fatal(err)
			}

			if act != nil {
				for _, v := range act.([]*HealthService) {
					v.NodeID = ""
					v.Checks = nil
				}
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestHealthServiceQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		exp  string
	}{
		{
			"name",
			"name",
			"health.service(name|passing)",
		},
		{
			"name_dc",
			"name@dc",
			"health.service(name@dc|passing)",
		},
		{
			"name_filter",
			"name|any",
			"health.service(name|any)",
		},
		{
			"name_multifilter",
			"name|warning,passing",
			"health.service(name|passing,warning)",
		},
		{
			"name_near",
			"name~near",
			"health.service(name~near|passing)",
		},
		{
			"name_near_filter",
			"name~near|any",
			"health.service(name~near|any)",
		},
		{
			"name_dc_near",
			"name@dc~near",
			"health.service(name@dc~near|passing)",
		},
		{
			"name_dc_near_filter",
			"name@dc~near|any",
			"health.service(name@dc~near|any)",
		},
		{
			"tag_name",
			"tag.name",
			"health.service(tag.name|passing)",
		},
		{
			"tag_name_dc",
			"tag.name@dc",
			"health.service(tag.name@dc|passing)",
		},
		{
			"tag_name_near",
			"tag.name~near",
			"health.service(tag.name~near|passing)",
		},
		{
			"tag_name_dc_near",
			"tag.name@dc~near",
			"health.service(tag.name@dc~near|passing)",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewHealthServiceQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
