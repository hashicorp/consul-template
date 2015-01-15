package dependency

import (
	"github.com/hashicorp/consul/api"
	"log"
	"sort"
	"time"
)

type Datacenters struct {
}

func (d *Datacenters) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), options)

	catalog := client.Catalog()
	dcs, err := catalog.Datacenters()
	if err != nil {
		return nil, nil, err
	}

	log.Printf("[DEBUG] (%s) Consul returned %d datacenters", d.Display(), len(dcs))
	sort.Strings(dcs)
	return dcs, &api.QueryMeta{LastIndex: uint64(time.Now().Unix())}, nil
}

func (d *Datacenters) HashCode() string {
	return "Datacenters"
}

func (d *Datacenters) Key() string {
	return "datacenters"
}

func (d *Datacenters) Display() string {
	return "datacenters"
}

// ParseDataCenters generates a service dependency from incoming strings
//
// Except that the data centers function doesn't take any arguments,
// so it really just instantiates the dependency
func ParseDatacenters() (*Datacenters, error) {
	return &Datacenters{}, nil
}
