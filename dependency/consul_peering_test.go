// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

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
	// the peering generated has random IDs,
	// we can't assert on the full response,
	// we can assert on the peering names though.
	expectedPeerNames := []string{
		"bar",
		"foo",
	}

	p, err := NewListPeeringQuery("")
	if err != nil {
		t.Fatal(err)
	}

	res, meta, err := p.Fetch(testClients, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	peerNames := make([]string, 0)
	for _, peering := range res.([]*Peering) {
		peerNames = append(peerNames, peering.Name)
	}
	assert.Equal(t, expectedPeerNames, peerNames)

	client := testClients.Consul()
	th, err := test.NewTenancyHelper(client)
	require.NoError(t, err)
	if th.IsConsulEnterprise() {
		// set up blocking query with last index
		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			data, _, err := p.Fetch(testClients, &QueryOptions{WaitIndex: meta.LastIndex})
			if err != nil {
				errCh <- err
				return
			}
			dataCh <- data
		}()

		tenancy := th.Tenancy("default.baz")
		ap := &api.Partition{Name: tenancy.Partition}
		partition, _, err := client.Partitions().Create(context.Background(), ap, nil)
		defer func() {
			_, _ = client.Partitions().Delete(context.Background(), partition.Name, nil)
		}()
		require.NoError(t, err)
		generateReq := api.PeeringGenerateTokenRequest{PeerName: "baz", Partition: tenancy.Partition}
		_, _, err = client.Peerings().GenerateToken(context.Background(), generateReq, &api.WriteOptions{})
		require.NoError(t, err)
		defer func() {
			_, _ = client.Peerings().Delete(context.Background(), generateReq.PeerName, nil)
		}()

		// create another peer
		err = testClients.createConsulPeerings(tenancy)
		require.NoError(t, err)

		select {
		case err := <-errCh:
			if err != ErrStopped {
				t.Fatal(err)
			}
		case <-time.After(1 * time.Minute):
			t.Errorf("did not stop")
		case val := <-dataCh:
			if val != nil {
				require.Equal(t, 3, len(val.([]*Peering)))
			}
		}
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
