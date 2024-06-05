package dependency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	capi "github.com/hashicorp/consul/api"
)

func TestListExportedServicesQuery_Fetch(t *testing.T) {
	testCases := map[string]struct {
		partition           string
		skipIfNonEnterprise bool
		exportedServices    *capi.ExportedServicesConfigEntry
		expected            []ExportedService
	}{
		"no services": {
			partition:        defaultOrEmtpyString(),
			exportedServices: nil,
			expected:         []ExportedService{},
		},
		"default partition - one exported service - partitions set": {
			partition:           "default",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: &capi.ExportedServicesConfigEntry{
				Name:      "default",
				Partition: "default",
				Services: []capi.ExportedService{
					{
						Name: "service1",
						Consumers: []capi.ServiceConsumer{
							{
								Partition: "foo",
							},
						},
					},
				},
			},
			expected: []ExportedService{
				{
					Service: "service1",
					Consumers: ResolvedConsumers{
						Peers:      []string{},
						Partitions: []string{"foo"},
					},
				},
			},
		},
		"default partition - multiple exported services - partitions set": {
			partition:           "default",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: &capi.ExportedServicesConfigEntry{
				Name:      "default",
				Partition: "default",
				Services: []capi.ExportedService{
					{
						Name: "service1",
						Consumers: []capi.ServiceConsumer{
							{
								Partition: "foo",
							},
						},
					},
					{
						Name: "service2",
						Consumers: []capi.ServiceConsumer{
							{
								Partition: "foo",
							},
						},
					},
				},
			},
			expected: []ExportedService{
				{
					Service: "service1",
					Consumers: ResolvedConsumers{
						Peers:      []string{},
						Partitions: []string{"foo"},
					},
				},
				{
					Service: "service2",
					Consumers: ResolvedConsumers{
						Peers:      []string{},
						Partitions: []string{"foo"},
					},
				},
			},
		},
		"non default partition - multiple exported services - partitions set": {
			partition:           "foo",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: &capi.ExportedServicesConfigEntry{
				Name:      "foo",
				Partition: "foo",
				Services: []capi.ExportedService{
					{
						Name: "service1",
						Consumers: []capi.ServiceConsumer{
							{
								Partition: "default",
							},
						},
					},
					{
						Name: "service2",
						Consumers: []capi.ServiceConsumer{
							{
								Partition: "default",
							},
						},
					},
				},
			},
			expected: []ExportedService{
				{
					Service: "service1",
					Consumers: ResolvedConsumers{
						Peers:      []string{},
						Partitions: []string{"default"},
					},
				},
				{
					Service: "service2",
					Consumers: ResolvedConsumers{
						Peers:      []string{},
						Partitions: []string{"default"},
					},
				},
			},
		},
		"default partition - one exported service - peers set": {
			partition:           defaultOrEmtpyString(),
			skipIfNonEnterprise: false,
			exportedServices: &capi.ExportedServicesConfigEntry{
				Name:      "default",
				Partition: defaultOrEmtpyString(),
				Services: []capi.ExportedService{
					{
						Name: "service1",
						Consumers: []capi.ServiceConsumer{
							{
								Peer: "another",
							},
						},
					},
				},
			},
			expected: []ExportedService{
				{
					Service: "service1",
					Consumers: ResolvedConsumers{
						Peers:      []string{"another"},
						Partitions: []string{},
					},
				},
			},
		},
		"default partition - multiple exported services - peers set": {
			partition:           defaultOrEmtpyString(),
			skipIfNonEnterprise: false,
			exportedServices: &capi.ExportedServicesConfigEntry{
				Name:      "default",
				Partition: defaultOrEmtpyString(),
				Services: []capi.ExportedService{
					{
						Name: "service1",
						Consumers: []capi.ServiceConsumer{
							{
								Peer: "another",
							},
						},
					},
					{
						Name: "service2",
						Consumers: []capi.ServiceConsumer{
							{
								Peer: "another",
							},
						},
					},
				},
			},
			expected: []ExportedService{
				{
					Service: "service1",
					Consumers: ResolvedConsumers{
						Peers:      []string{"another"},
						Partitions: []string{},
					},
				},
				{
					Service: "service2",
					Consumers: ResolvedConsumers{
						Peers:      []string{"another"},
						Partitions: []string{},
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.skipIfNonEnterprise {
				t.Skipf("skipping test %q as Consul is not enterprise", name)
			}

			opts := &capi.WriteOptions{Partition: tc.partition}

			if tc.exportedServices != nil {
				_, _, err := testClients.Consul().ConfigEntries().Set(tc.exportedServices, opts)
				require.NoError(t, err)
			}

			q, err := NewListExportedServicesQuery(tc.partition)
			require.NoError(t, err)

			actual, _, err := q.Fetch(testClients, nil)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expected, actual)

			if tc.exportedServices != nil {
				// need to clean up because we use a single shared consul instance
				_, err = testClients.Consul().ConfigEntries().Delete(capi.ExportedServices, tc.exportedServices.Name, opts)
				require.NoError(t, err)
			}
		})
	}
}

func defaultOrEmtpyString() string {
	if tenancyHelper.IsConsulEnterprise() {
		return "default"
	}

	return ""
}
