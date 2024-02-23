package dependency

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListPeeringsQuery(t *testing.T) {
	cases := []struct {
		name string
		i    string
		exp  *ListPeeringQuery
		err  bool
	}{
		{
			"empty",
			"",
			&ListPeeringQuery{},
			false,
		},
		{
			"invalid query param (unsupported key)",
			"?unsupported=foo",
			nil,
			true,
		},
		{
			"peerings",
			"peerings",
			nil,
			true,
		},
		{
			"partition",
			"?partition=foo",
			&ListPeeringQuery{
				partition: "foo",
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewListPeeringQuery(tc.i)
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

func TestListPeeringsQuery_Fetch(t *testing.T) {
	cases := []struct {
		name string
		i    string
		exp  []string
	}{
		{
			"all",
			"",
			// the peering generated has random IDs,
			// we can't assert on the full response,
			// we can assert on the peering names though.
			[]string{
				"bar",
				"foo",
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			p, err := NewListPeeringQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			res, _, err := p.Fetch(getDefaultTestClient(), nil)
			if err != nil {
				t.Fatal(err)
			}

			if res == nil {
				t.Fatalf("expected non-nil result")
			}

			peerNames := make([]string, 0)
			for _, peering := range res.([]*Peering) {
				peerNames = append(peerNames, peering.Name)
			}

			assert.Equal(t, tc.exp, peerNames)
		})
	}
}

func TestListPeeringsQuery_String(t *testing.T) {
	cases := []struct {
		name string
		i    string
		exp  string
	}{
		{
			"empty",
			"",
			"list.peerings",
		},
		{
			"partition",
			"?partition=foo",
			"list.peerings?partition=foo",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewListPeeringQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			str := d.String()
			assert.Equal(t, tc.exp, str)
		})
	}
}
