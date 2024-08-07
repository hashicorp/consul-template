// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/hashicorp/consul-template/test"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func TestNewHealthServiceQuery(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  *HealthServiceQuery
		err  error
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("empty", tenancy),
				"",
				nil,
				fmt.Errorf(`health.service: invalid format: ""`),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_only", tenancy),
				"@dc1",
				nil,
				fmt.Errorf(`health.service: invalid format: "@dc1"`),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("near_only", tenancy),
				"~near",
				nil,
				fmt.Errorf(`health.service: invalid format: "~near"`),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_only", tenancy),
				"tag.",
				nil,
				fmt.Errorf(`health.service: invalid format: "tag."`),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("query_only", tenancy),
				fmt.Sprintf("?ns=%s", tenancy.Namespace),
				nil,
				fmt.Errorf(`health.service: invalid format: "?ns=%s"`, tenancy.Namespace),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("invalid query param (unsupported key)", tenancy),
				"name?unsupported=test",
				nil,
				fmt.Errorf(`health.service: invalid query parameter key "unsupported" in query "unsupported=test": supported keys: ns,peer,partition,sameness-group`),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name", tenancy),
				"name",
				&HealthServiceQuery{
					filters: []string{"passing"},
					name:    "name",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_dc", tenancy),
				"name@dc1",
				&HealthServiceQuery{
					dc:      "dc1",
					filters: []string{"passing"},
					name:    "name",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_dc_near", tenancy),
				"name@dc1~near",
				&HealthServiceQuery{
					dc:      "dc1",
					filters: []string{"passing"},
					name:    "name",
					near:    "near",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_near", tenancy),
				"name~near",
				&HealthServiceQuery{
					filters: []string{"passing"},
					name:    "name",
					near:    "near",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name", tenancy),
				"tag.name",
				&HealthServiceQuery{
					filters: []string{"passing"},
					name:    "name",
					tag:     "tag",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc", tenancy),
				"tag.name@dc",
				&HealthServiceQuery{
					dc:      "dc",
					filters: []string{"passing"},
					name:    "name",
					tag:     "tag",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_near", tenancy),
				"tag.name~near",
				&HealthServiceQuery{
					filters: []string{"passing"},
					name:    "name",
					near:    "near",
					tag:     "tag",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc_near", tenancy),
				"tag.name@dc~near",
				&HealthServiceQuery{
					dc:      "dc",
					filters: []string{"passing"},
					name:    "name",
					near:    "near",
					tag:     "tag",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_partition", tenancy),
				fmt.Sprintf("name?partition=%s", tenancy.Partition),
				&HealthServiceQuery{
					filters:   []string{"passing"},
					name:      "name",
					partition: tenancy.Partition,
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_peer", tenancy),
				"name?peer=foo",
				&HealthServiceQuery{
					filters: []string{"passing"},
					name:    "name",
					peer:    "foo",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_ns", tenancy),
				fmt.Sprintf("name?ns=%s", tenancy.Namespace),
				&HealthServiceQuery{
					filters:   []string{"passing"},
					name:      "name",
					namespace: tenancy.Namespace,
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_ns_peer_partition", tenancy),
				fmt.Sprintf("name?ns=%s&peer=bar&partition=%s", tenancy.Namespace, tenancy.Partition),
				&HealthServiceQuery{
					filters:   []string{"passing"},
					name:      "name",
					namespace: tenancy.Namespace,
					peer:      "bar",
					partition: tenancy.Partition,
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_samenessgroup", tenancy),
				"name?sameness-group=sg1",
				&HealthServiceQuery{
					filters:       []string{"passing"},
					name:          "name",
					samenessGroup: "sg1",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("samenessgroup and peer should return error if both set", tenancy),
				"name?sameness-group=sg1&peer=peer1",
				nil,
				fmt.Errorf("health.service: cannot specify both peer and sameness-group"),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace set twice should use first", tenancy),
				fmt.Sprintf("name?ns=%s&ns=random", tenancy.Namespace),
				&HealthServiceQuery{
					filters:   []string{"passing"},
					name:      "name",
					namespace: tenancy.Namespace,
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("empty value in query param", tenancy),
				"name?ns=&peer=&partition=",
				&HealthServiceQuery{
					filters:   []string{"passing"},
					name:      "name",
					namespace: "",
					peer:      "",
					partition: "",
				},
				nil,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("query with other parameters", tenancy),
				fmt.Sprintf("tag.name?peer=foo&ns=%s&partition=%s@dc2~near", tenancy.Namespace, tenancy.Partition),
				&HealthServiceQuery{
					filters:   []string{"passing"},
					tag:       "tag",
					name:      "name",
					dc:        "dc2",
					near:      "near",
					peer:      "foo",
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
				},
				nil,
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewHealthServiceQuery(tc.i)
			assert.Equal(t, tc.err, err)

			if act != nil {
				act.stopCh = nil
			}

			assert.Equal(t, tc.exp, act)
		})
	}
	// Connect
	// all tests above also test connect, just need to check enabling it
	t.Run("connect_query", func(t *testing.T) {
		act, err := NewHealthConnectQuery("name")
		if err != nil {
			t.Fatal(err)
		}
		if act != nil {
			act.stopCh = nil
		}
		exp := &HealthServiceQuery{
			filters: []string{"passing"},
			name:    "name",
			connect: true,
		}

		assert.Equal(t, exp, act)
	})
}

func TestHealthConnectServiceQuery_Fetch(t *testing.T) {
	type testCase struct {
		name string
		in   string
		exp  []*HealthService
	}
	cases := tenancyHelper.GenerateNonDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("connect-service", tenancy),
				"conn-enabled-service-default-default",
				[]*HealthService{
					{
						Name:        "conn-enabled-service-proxy-default-default",
						ID:          "conn-enabled-service-proxy-default-default",
						Port:        21999,
						Status:      "passing",
						Address:     "127.0.0.1",
						NodeAddress: "127.0.0.1",
						Tags:        ServiceTags([]string{}),
						NodeMeta:    map[string]string{
							//"consul-network-segment": "",
						},
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("connect-service", tenancy),
				fmt.Sprintf("conn-enabled-service-%s-%s?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace, tenancy.Partition, tenancy.Namespace),
				[]*HealthService{
					{
						Name:        fmt.Sprintf("conn-enabled-service-proxy-%s-%s", tenancy.Partition, tenancy.Namespace),
						ID:          fmt.Sprintf("conn-enabled-service-proxy-%s-%s", tenancy.Partition, tenancy.Namespace),
						Port:        21999,
						Status:      "passing",
						Address:     "127.0.0.1",
						NodeAddress: "127.0.0.1",
						Tags:        ServiceTags([]string{}),
						NodeMeta:    map[string]string{
							//"consul-network-segment": "",
						},
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
		}
	})

	cases = append(cases, tenancyHelper.GenerateDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("connect-service", tenancy),
				"conn-enabled-service-default-default",
				[]*HealthService{
					{
						Name:        "conn-enabled-service-proxy-default-default",
						ID:          "conn-enabled-service-proxy-default-default",
						Port:        21999,
						Status:      "passing",
						Address:     "127.0.0.1",
						NodeAddress: "127.0.0.1",
						Tags:        ServiceTags([]string{}),
						NodeMeta:    map[string]string{
							//"consul-network-segment": "",
						},
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
		}
	})...)

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewHealthConnectQuery(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				d.Stop()
			}()
			res, _, err := d.Fetch(testClients, nil)
			if err != nil {
				t.Fatal(err)
			}
			var act []*HealthService
			if act = res.([]*HealthService); len(act) != 1 {
				t.Fatal("Expected 1 result, got ", len(act))
			}
			// blank out fields we don't want to test
			inst := act[0]
			inst.Node, inst.NodeID = "", ""
			inst.Checks = nil
			inst.NodeTaggedAddresses = nil
			inst.ServiceTaggedAddresses = nil
			inst.NodeMeta = filterVersionMeta(inst.NodeMeta)

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestHealthServiceQuery_Fetch(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  []*HealthService
	}
	cases := tenancyHelper.GenerateDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("consul", tenancy),
				"consul",
				[]*HealthService{
					{
						Node:                testConsul.Config.NodeName,
						NodeAddress:         testConsul.Config.Bind,
						NodeTaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceMeta: map[string]string{},
						Address:     testConsul.Config.Bind,
						ID:          "consul",
						Name:        "consul",
						Tags:        []string{},
						Status:      "passing",
						Port:        testConsul.Config.Ports.Server,
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("filters", tenancy),
				"consul|warning",
				[]*HealthService{},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("multifilter", tenancy),
				"consul|warning,passing",
				[]*HealthService{
					{
						Node:                testConsul.Config.NodeName,
						NodeAddress:         testConsul.Config.Bind,
						NodeTaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceMeta: map[string]string{},
						Address:     testConsul.Config.Bind,
						ID:          "consul",
						Name:        "consul",
						Tags:        []string{},
						Status:      "passing",
						Port:        testConsul.Config.Ports.Server,
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("service-meta", tenancy),
				"service-meta-default-default",
				[]*HealthService{
					{
						Node:                "node" + tenancy.Partition,
						NodeAddress:         testConsul.Config.Bind,
						NodeTaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceMeta: map[string]string{
							"meta1": "value1",
						},
						Address: testConsul.Config.Bind,
						ID:      "service-meta-default-default",
						Name:    "service-meta-default-default",
						Tags:    []string{"tag1"},
						Status:  "passing",
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("service-taggedAddresses", tenancy),
				"service-taggedAddresses-default-default",
				[]*HealthService{
					{
						Node:                "node" + tenancy.Partition,
						NodeAddress:         testConsul.Config.Bind,
						NodeTaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceMeta: map[string]string{},
						Address:     testConsul.Config.Bind,
						ServiceTaggedAddresses: map[string]api.ServiceAddress{
							"lan": {
								Address: "192.0.2.1",
								Port:    80,
							},
							"wan": {
								Address: "192.0.2.2",
								Port:    443,
							},
						},
						ID:     "service-taggedAddresses-default-default",
						Name:   "service-taggedAddresses-default-default",
						Tags:   []string{},
						Status: "passing",
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
		}
	})

	cases = append(cases, tenancyHelper.GenerateNonDefaultTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("consul", tenancy),
				"consul",
				[]*HealthService{
					{
						Node:                testConsul.Config.NodeName,
						NodeAddress:         testConsul.Config.Bind,
						NodeTaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceMeta: map[string]string{},
						Address:     testConsul.Config.Bind,
						ID:          "consul",
						Name:        "consul",
						Tags:        []string{},
						Status:      "passing",
						Port:        testConsul.Config.Ports.Server,
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("filters", tenancy),
				"consul|warning",
				[]*HealthService{},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("multifilter", tenancy),
				"consul|warning,passing",
				[]*HealthService{
					{
						Node:                testConsul.Config.NodeName,
						NodeAddress:         testConsul.Config.Bind,
						NodeTaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceMeta: map[string]string{},
						Address:     testConsul.Config.Bind,
						ID:          "consul",
						Name:        "consul",
						Tags:        []string{},
						Status:      "passing",
						Port:        testConsul.Config.Ports.Server,
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("service-meta", tenancy),
				fmt.Sprintf("service-meta-%s-%s?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace, tenancy.Partition, tenancy.Namespace),
				[]*HealthService{
					{
						Node:                "node" + tenancy.Partition,
						NodeAddress:         testConsul.Config.Bind,
						NodeTaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceMeta: map[string]string{
							"meta1": "value1",
						},
						Address: testConsul.Config.Bind,
						ID:      fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
						Name:    fmt.Sprintf("service-meta-%s-%s", tenancy.Partition, tenancy.Namespace),
						Tags:    []string{"tag1"},
						Status:  "passing",
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("service-taggedAddresses", tenancy),
				fmt.Sprintf("service-taggedAddresses-%s-%s?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace, tenancy.Partition, tenancy.Namespace),
				[]*HealthService{
					{
						Node:                "node" + tenancy.Partition,
						NodeAddress:         testConsul.Config.Bind,
						NodeTaggedAddresses: map[string]string{
							//"lan": "127.0.0.1",
							//"wan": "127.0.0.1",
						},
						NodeMeta: map[string]string{
							//"consul-network-segment": "",
						},
						ServiceMeta: map[string]string{},
						Address:     testConsul.Config.Bind,
						ServiceTaggedAddresses: map[string]api.ServiceAddress{
							"lan": {
								Address: "192.0.2.1",
								Port:    80,
							},
							"wan": {
								Address: "192.0.2.2",
								Port:    443,
							},
						},
						ID:     fmt.Sprintf("service-taggedAddresses-%s-%s", tenancy.Partition, tenancy.Namespace),
						Name:   fmt.Sprintf("service-taggedAddresses-%s-%s", tenancy.Partition, tenancy.Namespace),
						Tags:   []string{},
						Status: "passing",
						Weights: api.AgentWeights{
							Passing: 1,
							Warning: 1,
						},
					},
				},
			},
		}
	})...)

	for i, test := range cases {
		tc := test.(testCase)
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
					// delete any version data from ServiceMeta
					v.ServiceMeta = filterVersionMeta(v.ServiceMeta)
					v.NodeTaggedAddresses = filterAddresses(
						v.NodeTaggedAddresses)
					v.NodeMeta = filterVersionMeta(v.NodeMeta)

				}
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

// TestHealthServiceQuery_Fetch_SamenessGroup ensures that consul-template re-runs when the blocking query updates.
// The different behaviors of the blocking query, e.g. "when a service becomes unhealthy it should failover,
// when it becomes healthy again it should fail back", are tested in the Consul codebase.
func TestHealthServiceQuery_Fetch_SamenessGroup(t *testing.T) {
	if !tenancyHelper.IsConsulEnterprise() {
		t.Skip("Enterprise only test")
	}
	// Arrange - set up test data
	catalog := testClients.Consul().Catalog()

	partitionOne := "sg-partition-1"
	partitionTwo := "sg-partition-2"
	nodeOne := "node" + partitionOne
	nodeTwo := "node" + partitionTwo
	samenessGroup := "test-sameness-group"

	require.NoError(t, testClients.createConsulPartition(partitionOne))
	require.NoError(t, testClients.createConsulPartition(partitionTwo))
	require.NoError(t, testClients.createConsulSamenessGroups(samenessGroup, partitionOne, partitionTwo))

	// Register services with the same name in partionOne and partitionTwo and them to a sameness group so that we can test failover.
	svcName := "sameness-group-service"
	registerSvc := func(service, node, partition, status string) {
		checkName := fmt.Sprintf("%s:%s", service, node)
		svcRegistration := &api.CatalogRegistration{
			Service: &api.AgentService{
				ID:        service,
				Service:   service,
				Port:      12345,
				Partition: partition,
				Namespace: "default",
				Connect:   &api.AgentServiceConnect{},
			},
			Partition: partition,
			Node:      node,
			Address:   "127.0.0.1",
			Checks: api.HealthChecks{
				&api.HealthCheck{
					Node:        node,
					CheckID:     checkName,
					Name:        checkName,
					Status:      status,
					ServiceID:   svcName,
					ServiceName: svcName,
				},
			},
		}

		_, err := catalog.Register(svcRegistration, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	// set up service in each partition
	registerSvc(svcName, nodeOne, partitionOne, "passing")
	registerSvc(svcName, nodeTwo, partitionTwo, "passing")

	// Act - fetch the service
	query := fmt.Sprintf("%s?sameness-group=%s&partition=%s", svcName, samenessGroup, partitionOne)
	d, err := NewHealthServiceQuery(query)
	if err != nil {
		t.Fatal(err)
	}

	svcs, meta, err := d.Fetch(testClients, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Assert that the service instance returned is from the first listed partition.
	// This is expected because the service is healthy so it shouldn't failover
	// to other partitions.
	healthServices := svcs.([]*HealthService)
	require.True(t, len(healthServices) == 1)
	require.Equal(t, "node"+partitionOne, healthServices[0].Node)

	// set up blocking query with last index
	dataCh := make(chan interface{}, 1)
	errCh := make(chan error, 1)
	go func() {
		data, _, err := d.Fetch(testClients, &QueryOptions{WaitIndex: meta.LastIndex})
		if err != nil {
			errCh <- err
			return
		}
		dataCh <- data
	}()

	// update partition one to initiate failover
	registerSvc(svcName, nodeOne, partitionOne, "critical")

	select {
	case err := <-errCh:
		if err != ErrStopped {
			t.Fatal(err)
		}
	case <-time.After(1 * time.Minute):
		t.Errorf("did not stop")
	case val := <-dataCh:
		if val != nil {
			for _, v := range val.([]*HealthService) {
				v.NodeID = ""
				v.Checks = nil
				// delete any version data from ServiceMeta
				v.ServiceMeta = filterVersionMeta(v.ServiceMeta)
				v.NodeTaggedAddresses = filterAddresses(
					v.NodeTaggedAddresses)
				v.NodeMeta = filterVersionMeta(v.NodeMeta)
			}
		}

		// Assert - verify the results
		expectedResult := []*HealthService{
			{
				// The instance should now be from the failover partition.
				Node:                nodeTwo,
				NodeAddress:         testConsul.Config.Bind,
				ServiceMeta:         map[string]string{},
				Address:             testConsul.Config.Bind,
				NodeTaggedAddresses: map[string]string{},
				NodeMeta:            map[string]string{},
				Port:                12345,
				ID:                  svcName,
				Name:                svcName,
				Tags:                []string{},
				Status:              HealthPassing,
				Weights: api.AgentWeights{
					Passing: 1,
					Warning: 1,
				},
			},
		}
		assert.Equal(t, expectedResult, val)
	}
}

func TestHealthServiceQuery_String(t *testing.T) {
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
				"health.service(name|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_dc", tenancy),
				"name@dc",
				"health.service(name@dc|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_filter", tenancy),
				"name|any",
				"health.service(name|any)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_multifilter", tenancy),
				"name|warning,passing",
				"health.service(name|passing,warning)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_near", tenancy),
				"name~near",
				"health.service(name~near|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_near_filter", tenancy),
				"name~near|any",
				"health.service(name~near|any)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_dc_near", tenancy),
				"name@dc~near",
				"health.service(name@dc~near|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_dc_near_filter", tenancy),
				"name@dc~near|any",
				"health.service(name@dc~near|any)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name", tenancy),
				"tag.name",
				"health.service(tag.name|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc", tenancy),
				"tag.name@dc",
				"health.service(tag.name@dc|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_near", tenancy),
				"tag.name~near",
				"health.service(tag.name~near|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc_near", tenancy),
				"tag.name@dc~near",
				"health.service(tag.name@dc~near|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc_near_partition", tenancy),
				fmt.Sprintf("tag.name?partition=%s@dc~near", tenancy.Partition),
				fmt.Sprintf("health.service(tag.name@dc@partition=%s~near|passing)", tenancy.Partition),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("tag_name_dc_near_partition_ns", tenancy),
				fmt.Sprintf("tag.name?partition=%s&ns=%s@dc~near", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("health.service(tag.name@dc@partition=%s@ns=%s~near|passing)", tenancy.Partition, tenancy.Namespace),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition_ns", tenancy),
				fmt.Sprintf("tag.name?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("health.service(tag.name@partition=%s@ns=%s|passing)", tenancy.Partition, tenancy.Namespace),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("peer", tenancy),
				"tag.name?peer=peer-name",
				"health.service(tag.name@peer=peer-name|passing)",
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewHealthServiceQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}

func TestHealthServiceQueryConnect_String(t *testing.T) {
	type testCase struct {
		name string
		fact func(string) (*HealthServiceQuery, error)
		in   string
		exp  string
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *test.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("name", tenancy),
				NewHealthServiceQuery,
				"name",
				"health.service(name|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name", tenancy),
				NewHealthConnectQuery,
				"name",
				"health.connect(name|passing)",
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_ns_partition", tenancy),
				NewHealthServiceQuery,
				fmt.Sprintf("name?ns=%s&partition=%s", tenancy.Namespace, tenancy.Partition),
				fmt.Sprintf("health.service(name@partition=%s@ns=%s|passing)", tenancy.Partition, tenancy.Namespace),
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("name_ns_partition", tenancy),
				NewHealthConnectQuery,
				fmt.Sprintf("name?ns=%s&partition=%s", tenancy.Namespace, tenancy.Partition),
				fmt.Sprintf("health.connect(name@partition=%s@ns=%s|passing)", tenancy.Partition, tenancy.Namespace),
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := tc.fact(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}

// Test that if new fields are added to the struct, the String() method
// is also updated. This test uses reflection to iterate over the fields
// in order to catch the case where someone adds a new field but doesn't
// know they need to also update String().
func TestHealthServiceQuery_String_Reflection(t *testing.T) {
	query := HealthServiceQuery{}
	val := reflect.ValueOf(&query).Elem()
	// prev is set to the previous output of String()
	prev := query.String()
	// Iterate over each field using reflection.
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldName := val.Type().Field(i).Name
		// Need to use this to be able to set private fields.
		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
		setField := false
		if field.CanSet() {
			// Set the fields to something, so we can see if String() changes.
			// If a string or []string, set to field name, if a bool, set to true.
			switch {
			case field.Type().Kind() == reflect.String:
				field.SetString(fieldName)
				setField = true
			case field.Type().Kind() == reflect.Bool:
				field.SetBool(true)
				setField = true
			case field.Type() == reflect.TypeOf([]string{}):
				field.Set(reflect.ValueOf([]string{fieldName}))
				setField = true
			}
		}

		// As new fields are set, the value of String() should change.
		if setField && prev == query.String() {
			t.Fatalf("Expected output of String() to change after setting field %q, but got same value as before: %q."+
				" This likely means you've added a field but haven't updated String(). If the field should change the query, "+
				"e.g. you add namespace to query a specific Consul namespace, but you don't update String() then other queries"+
				" using the same function will return the same data because String() is used as the cache key. To fix, update"+
				" String() to change based on your new field.", fieldName, prev)
		}
		prev = query.String()
	}
}
