package dependency

import (
	"testing"

	"github.com/stretchr/testify/require"

	capi "github.com/hashicorp/consul/api"
)

func TestListExportedServicesQuery_Fetch(t *testing.T) {
	_, _, err := testClients.Consul().ConfigEntries().Set(&capi.ExportedServicesConfigEntry{
		Name:      "default",
		Partition: "default",
		Services: []capi.ExportedService{
			{
				Name: "service1",
				Consumers: []capi.ServiceConsumer{
					{
						Partition: "default.bar",
					},
				},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := &ListExportedServicesQuery{
		stopCh:    make(chan struct{}),
		partition: "default",
	}

	actual, _, err := q.Fetch(testClients, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []ExportedService{
		{
			Service: "service1",
			Consumers: ResolvedConsumers{
				Peers:          []string{},
				Partitions:     []string{"default.bar"},
				SamenessGroups: []string{},
			},
		},
	}

	require.ElementsMatch(t, expected, actual)
}
