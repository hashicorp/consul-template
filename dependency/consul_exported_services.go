package dependency

import (
	"fmt"
	"log"
	"net/url"
	"slices"

	capi "github.com/hashicorp/consul/api"
)

const exportedServicesEndpointLabel = "list.exportedServices"

// Ensure implements
var _ Dependency = (*ListExportedServicesQuery)(nil)

// ListExportedServicesQuery is the representation of a requested exported services
// dependency from inside a template.
type ListExportedServicesQuery struct {
	stopCh    chan struct{}
	partition string
}

type ExportedService struct {
	// Name of the service
	Service string

	// Partition of the service
	Partition string

	// Namespace of the service
	Namespace string

	// Consumers is a list of downstream consumers of the service.
	Consumers ResolvedConsumers
}

type ResolvedConsumers struct {
	Peers          []string
	Partitions     []string
	SamenessGroups []string
}

func fromConsulExportedService(svc capi.ExportedService) ExportedService {
	peers := make([]string, 0, len(svc.Consumers))
	partitions := make([]string, 0, len(svc.Consumers))
	samenessGroups := make([]string, 0, len(svc.Consumers))
	for _, consumer := range svc.Consumers {
		if consumer.Peer != "" {
			peers = append(peers, consumer.Peer)
		}
		if consumer.Partition != "" {
			partitions = append(partitions, consumer.Partition)
		}
		if consumer.SamenessGroup != "" {
			samenessGroups = append(samenessGroups, consumer.SamenessGroup)
		}
	}

	return ExportedService{
		Service: svc.Name,
		Consumers: ResolvedConsumers{
			Peers:          peers,
			Partitions:     partitions,
			SamenessGroups: samenessGroups,
		},
	}
}

// NewListExportedServicesQuery parses a string of the format @dc.
func NewListExportedServicesQuery(s string) (*ListExportedServicesQuery, error) {
	return &ListExportedServicesQuery{
		stopCh:    make(chan struct{}),
		partition: s,
	}, nil
}

func (c *ListExportedServicesQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-c.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	opts = opts.Merge(&QueryOptions{
		ConsulPartition: c.partition,
	})

	log.Printf("[TRACE] %s: GET %s", c, &url.URL{
		Path:     "/v1/config/exported-services",
		RawQuery: opts.String(),
	})

	consulExportedServices, qm, err := clients.Consul().ConfigEntries().List(capi.ExportedServices, opts.ToConsulOpts())
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", c.String(), err)
	}

	exportedServices := make([]ExportedService, 0, len(consulExportedServices))
	for _, cfgEntry := range consulExportedServices {
		svc := cfgEntry.(*capi.ExportedServicesConfigEntry)
		for _, svc := range svc.Services {
			exportedServices = append(exportedServices, fromConsulExportedService(svc))
		}
	}

	log.Printf("[TRACE] %s: returned %d results", c, len(exportedServices))

	slices.SortStableFunc(exportedServices, func(i, j ExportedService) int {
		if i.Service < j.Service {
			return -1
		} else if i.Service > j.Service {
			return 1
		}
		return 0
	})

	rm := &ResponseMetadata{
		LastContact: qm.LastContact,
		LastIndex:   qm.LastIndex,
	}

	return exportedServices, rm, nil
}

// CanShare returns if this dependency is shareable when consul-template is running in de-duplication mode.
func (c *ListExportedServicesQuery) CanShare() bool {
	return true
}

func (c *ListExportedServicesQuery) String() string {
	return exportedServicesEndpointLabel
}

func (c *ListExportedServicesQuery) Stop() {
	close(c.stopCh)
}

func (c *ListExportedServicesQuery) Type() Type {
	return TypeConsul
}
