// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"fmt"
	"log"
	"net/url"
	"slices"
	"strings"

	capi "github.com/hashicorp/consul/api"
)

const importedServicesEndpointLabel = "list.importedServices"

// Ensure implements
var _ Dependency = (*ListImportedServicesQuery)(nil)

// ListImportedServicesQuery is the representation of a requested imported services
// dependency from inside a template.
type ListImportedServicesQuery struct {
	stopCh    chan struct{}
	partition string
}

// ImportedService represents a service imported into a partition from another partition.
type ImportedService struct {
	// Service is the name of the service
	Service string

	// Namespace is the namespace of the service
	Namespace string

	// SourcePartition is the partition exporting the service
	SourcePartition string
}

func fromConsulImportedService(svc capi.ImportedService) ImportedService {
	return ImportedService{
		Service:         svc.Service,
		Namespace:       svc.Namespace,
		SourcePartition: svc.SourcePartition,
	}
}

// NewListImportedServicesQuery parses a string representing the partition name.
func NewListImportedServicesQuery(s string) (*ListImportedServicesQuery, error) {
	return &ListImportedServicesQuery{
		stopCh:    make(chan struct{}),
		partition: s,
	}, nil
}

func (c *ListImportedServicesQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-c.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	opts = opts.Merge(&QueryOptions{
		ConsulPartition: c.partition,
	})

	log.Printf("[TRACE] %s: GET %s", c, &url.URL{
		Path:     "/v1/imported-services",
		RawQuery: opts.String(),
	})

	consulImportedServicesResp, qm, err := clients.Consul().ImportedServices(opts.ToConsulOpts())
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", c.String(), err)
	}

	var importedServices []ImportedService
	if consulImportedServicesResp != nil && consulImportedServicesResp.ImportedServices != nil {
		importedServices = make([]ImportedService, 0, len(consulImportedServicesResp.ImportedServices))
		for _, importedService := range consulImportedServicesResp.ImportedServices {
			importedServices = append(importedServices, fromConsulImportedService(importedService))
		}
	} else {
		importedServices = make([]ImportedService, 0)
	}

	log.Printf("[TRACE] %s: returned %d results", c, len(importedServices))

	slices.SortStableFunc(importedServices, func(i, j ImportedService) int {
		return strings.Compare(i.Service, j.Service)
	})

	rm := &ResponseMetadata{
		LastContact: qm.LastContact,
		LastIndex:   qm.LastIndex,
	}

	return importedServices, rm, nil
}

// CanShare returns if this dependency is shareable when consul-template is running in de-duplication mode.
func (c *ListImportedServicesQuery) CanShare() bool {
	return true
}

func (c *ListImportedServicesQuery) String() string {
	return fmt.Sprintf("%s(%s)", importedServicesEndpointLabel, c.partition)
}

func (c *ListImportedServicesQuery) Stop() {
	close(c.stopCh)
}

func (c *ListImportedServicesQuery) Type() Type {
	return TypeConsul
}
