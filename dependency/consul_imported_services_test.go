// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"context"
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
			partition:           "downstream",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices:    nil,
			expected:            []ImportedService{},
		},
		"downstream partition - imports from upstream": {
			partition:           "downstream",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: []*capi.ExportedServicesConfigEntry{
				{
					Name:      "upstream",
					Partition: "upstream",
					Services: []capi.ExportedService{
						{
							Name:      "web",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "downstream",
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
					SourcePartition: "upstream",
				},
			},
		},
		"downstream partition - imports from multiple partitions": {
			partition:           "downstream",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: []*capi.ExportedServicesConfigEntry{
				{
					Name:      "upstream",
					Partition: "upstream",
					Services: []capi.ExportedService{
						{
							Name:      "web",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "downstream",
								},
							},
						},
					},
				},
				{
					Name:      "default",
					Partition: "default",
					Services: []capi.ExportedService{
						{
							Name:      "api",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "downstream",
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
					SourcePartition: "upstream",
				},
			},
		},
		"downstream partition - multiple services from same partition": {
			partition:           "downstream",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: []*capi.ExportedServicesConfigEntry{
				{
					Name:      "upstream",
					Partition: "upstream",
					Services: []capi.ExportedService{
						{
							Name:      "web",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "downstream",
								},
							},
						},
						{
							Name:      "api",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "downstream",
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
					SourcePartition: "upstream",
				},
				{
					Service:         "web",
					Namespace:       "default",
					SourcePartition: "upstream",
				},
			},
		},
		"upstream partition - no imports (exports only)": {
			partition:           "upstream",
			skipIfNonEnterprise: !tenancyHelper.IsConsulEnterprise(),
			exportedServices: []*capi.ExportedServicesConfigEntry{
				{
					Name:      "upstream",
					Partition: "upstream",
					Services: []capi.ExportedService{
						{
							Name:      "web",
							Namespace: "default",
							Consumers: []capi.ServiceConsumer{
								{
									Partition: "downstream",
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

			// Create partitions if they don't exist
			if tenancyHelper.IsConsulEnterprise() {
				partitionsToCreate := []string{"upstream", "downstream"}
				for _, partName := range partitionsToCreate {
					partition := capi.Partition{Name: partName}
					_, _, err := testClients.Consul().Partitions().Create(context.TODO(), &partition, nil)
					// Ignore error if partition already exists
					if err != nil && !isAlreadyExistsError(err) {
						require.NoError(t, err)
					}
				}
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
					_, err = testClients.Consul().ConfigEntries().Delete(capi.ExportedServices, entry.Name, opts)
					require.NoError(t, err)
				}
			}

			// Clean up partitions created during test
			if tenancyHelper.IsConsulEnterprise() {
				partitionsToDelete := []string{"upstream", "downstream"}
				for _, partName := range partitionsToDelete {
					_, err := testClients.Consul().Partitions().Delete(context.TODO(), partName, nil)
					// Ignore error if partition doesn't exist or can't be deleted
					if err != nil {
						// Log but don't fail the test for cleanup issues
						t.Logf("Warning: failed to delete partition %q: %v", partName, err)
					}
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
		{"upstream", "list.importedServices(upstream)"},
		{"downstream", "list.importedServices(downstream)"},
	}

	for _, tc := range testCases {
		t.Run(tc.partition, func(t *testing.T) {
			q, err := NewListImportedServicesQuery(tc.partition)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, q.String())
		})
	}
}

// Helper function to check if error is "already exists"
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains "already exists" or similar
	errMsg := err.Error()
	return contains(errMsg, "already exists") || contains(errMsg, "Partition already exists")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexString(s, substr) >= 0))
}

func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
