package dependency

import (
	"log"
	"net/url"
	"slices"
	"time"

	capi "github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

const (
	exportedServicesEndpointLabel = "list.exportedServices"

	// ListExportedServicesQuerySleepTime is the amount of time to sleep between
	// queries, since the endpoint does not support blocking queries.
	ListExportedServicesQuerySleepTime = 15 * time.Second
)

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
	Peers      []string
	Partitions []string
}

func fromConsulExportedService(svc capi.ResolvedExportedService) ExportedService {
	return ExportedService{
		Service: svc.Service,
		Consumers: ResolvedConsumers{
			Peers:      slices.Clone(svc.Consumers.Peers),
			Partitions: slices.Clone(svc.Consumers.Partitions),
		},
	}
}

// NewListExportedServicesQuery parses a string of the format @dc.
func NewListExportedServicesQuery(s string) (*ListExportedServicesQuery, error) {
	return &ListExportedServicesQuery{
		stopCh:    make(chan struct{}, 1),
		partition: s,
	}, nil
}

func (c *ListExportedServicesQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	opts = opts.Merge(&QueryOptions{
		ConsulPartition: c.partition,
	})

	log.Printf("[TRACE] %s: GET %s", c, &url.URL{
		Path:     "/v1/exported-services",
		RawQuery: opts.String(),
	})

	// This is certainly not elegant, but the partitions endpoint does not support
	// blocking queries, so we are going to "fake it until we make it". When we
	// first query, the LastIndex will be "0", meaning we should immediately
	// return data, but future calls will include a LastIndex. If we have a
	// LastIndex in the query metadata, sleep for 15 seconds before asking Consul
	// again.
	//
	// This is probably okay given the frequency in which partitions actually
	// change, but is technically not edge-triggering.
	if opts.WaitIndex != 0 {
		log.Printf("[TRACE] %s: long polling for %s", c, ListExportedServicesQuerySleepTime)

		select {
		case <-c.stopCh:
			return nil, nil, ErrStopped
		case <-time.After(ListExportedServicesQuerySleepTime):
		}
	}

	// TODO Consider using a proper context
	consulExportedServices, qm, err := clients.Consul().ExportedServices(opts.ToConsulOpts())
	if err != nil {
		return nil, nil, errors.Wrapf(err, c.String())
	}

	exportedServices := make([]ExportedService, 0, len(consulExportedServices))
	for _, svc := range consulExportedServices {
		exportedServices = append(exportedServices, fromConsulExportedService(svc))
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

// CanShare returns if this dependency is shareable.
// TODO What is this?
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
