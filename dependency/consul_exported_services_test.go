package dependency

import (
	"testing"

	"github.com/stretchr/testify/require"

	capi "github.com/hashicorp/consul/api"
)

func TestListExportedServicesQuery_Fetch(t *testing.T) {
	testCases := map[string]struct {
		partition        string
		exportedServices *capi.ExportedServicesConfigEntry
		expected         []ExportedService
	}{
		//"no services": {},
		"default partition - one exported service - partitions set": {
			partition: "default",
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
						Peers:          []string{},
						Partitions:     []string{"foo"},
						SamenessGroups: []string{},
					},
				},
			},
		},
		"default partition - multiple exported services - partitions set": {
			partition: "default",
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
						Peers:          []string{},
						Partitions:     []string{"foo"},
						SamenessGroups: []string{},
					},
				},
				{
					Service: "service2",
					Consumers: ResolvedConsumers{
						Peers:          []string{},
						Partitions:     []string{"foo"},
						SamenessGroups: []string{},
					},
				},
			},
		},
		"non default partition - multiple exported services - partitions set": {
			partition: "foo",
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
						Peers:          []string{},
						Partitions:     []string{"default"},
						SamenessGroups: []string{},
					},
				},
				{
					Service: "service2",
					Consumers: ResolvedConsumers{
						Peers:          []string{},
						Partitions:     []string{"default"},
						SamenessGroups: []string{},
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			_, _, err := testClients.Consul().ConfigEntries().Set(tc.exportedServices, &capi.WriteOptions{Partition: tc.partition})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			q, err := NewListExportedServicesQuery(tc.partition)
			require.NoError(t, err)

			actual, _, err := q.Fetch(testClients, nil)
			require.NoError(t, err)

			require.ElementsMatch(t, tc.expected, actual)

			// need to clean up because we use a single shared consul instance
			_, err = testClients.Consul().ConfigEntries().Delete(capi.ExportedServices, tc.exportedServices.Name, &capi.WriteOptions{Partition: tc.partition})
			require.NoError(t, err)
		})
	}
}
