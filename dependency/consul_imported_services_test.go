// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	capi "github.com/hashicorp/consul/api"
)

func TestListImportedServicesQuery_Fetch(t *testing.T) {
	testCases := map[string]struct {
		partition           string
		skipIfNonEnterprise bool
		exportedServices    []*capi.ExportedServicesConfigEntry
		expected            []ImportedService
	}{
		"no services": {
			partition:           defaultOrEmtpyString(),
			skipIfNonEnterprise: false,
			exportedServices:    nil,
			expected:            []ImportedService{},
		},
		"foo partition - imports from default": {
			partition:           "foo",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: []*capi.ExportedServicesConfigEntry{
				{
					Name:      "default",
					Partition: "default",
					Services: []capi.ExportedService{
						{
							Name:      "web",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "foo",
								},
							},
						},
					},
				},
			},
			expected: []ImportedService{
				{
					Service:         "web",
					Namespace:       "default",
					SourcePartition: "default",
				},
			},
		},
		"foo partition - multiple services from same partition": {
			partition:           "foo",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: []*capi.ExportedServicesConfigEntry{
				{
					Name:      "default",
					Partition: "default",
					Services: []capi.ExportedService{
						{
							Name:      "web",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "foo",
								},
							},
						},
						{
							Name:      "api",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "foo",
								},
							},
						},
					},
				},
			},
			expected: []ImportedService{
				{
					Service:         "api",
					Namespace:       "default",
					SourcePartition: "default",
				},
				{
					Service:         "web",
					Namespace:       "default",
					SourcePartition: "default",
				},
			},
		},
		"default partition - no imports (exports only)": {
			partition:           "default",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: []*capi.ExportedServicesConfigEntry{
				{
					Name:      "default",
					Partition: "default",
					Services: []capi.ExportedService{
						{
							Name:      "web",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "foo",
								},
							},
						},
					},
				},
			},
			expected: []ImportedService{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.skipIfNonEnterprise {
				t.Skipf("skipping test %q as Consul is not enterprise", name)
			}

			// Set up exported services config entries
			if tc.exportedServices != nil {
				for _, entry := range tc.exportedServices {
					opts := &capi.WriteOptions{Partition: entry.Partition}
					_, _, err := testClients.Consul().ConfigEntries().Set(entry, opts)
					require.NoError(t, err)
				}
			}

			// Query imported services
			q, err := NewListImportedServicesQuery(tc.partition)
			require.NoError(t, err)

			actual, _, err := q.Fetch(testClients, nil)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expected, actual)

			// Clean up exported services config entries
			if tc.exportedServices != nil {
				for _, entry := range tc.exportedServices {
					opts := &capi.WriteOptions{Partition: entry.Partition}
					// need to clean up because we use a single shared consul instance
					_, err = testClients.Consul().ConfigEntries().Delete(capi.ExportedServices, entry.Name, opts)
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestListImportedServicesQuery_String(t *testing.T) {
	testCases := []struct {
		partition string
		expected  string
	}{
		{"default", "list.importedServices(default)"},
		{"foo", "list.importedServices(foo)"},
	}

	for _, tc := range testCases {
		t.Run(tc.partition, func(t *testing.T) {
			q, err := NewListImportedServicesQuery(tc.partition)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, q.String())
		})
	}
}
