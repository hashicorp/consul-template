package dependency

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/hashicorp/consul/api"
)

var sleepTime = 15 * time.Second

// Datacenters is the dependency to query all datacenters
type Datacenters struct {
	rawKey string
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of strings representing the datacenters
func (d *Datacenters) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), options)

	// This is pretty ghetto, but the datacenters endpoint does not support
	// blocking queries, so we are going to "fake it until we make it". When we
	// first query, the LastIndex will be "0", meaning we should immediately
	// return data, but future calls will include a LastIndex. If we have a
	// LastIndex in the query metadata, sleep for 15 seconds before asking Consul
	// again.
	//
	// This is probably okay given the frequency in which datacenters actually
	// change, but is technically not edge-triggering.
	if options.WaitIndex != 0 {
		log.Printf("[DEBUG] (%s) pretending to long-poll", d.Display())
		time.Sleep(sleepTime)
	}

	catalog := client.Catalog()
	result, err := catalog.Datacenters()
	if err != nil {
		return nil, nil, err
	}

	log.Printf("[DEBUG] (%s) Consul returned %d datacenters", d.Display(), len(result))
	sort.Strings(result)

	qm := &api.QueryMeta{LastIndex: uint64(time.Now().Unix())}

	return result, qm, nil
}

// HashCode returns the hash code for this dependency.
func (d *Datacenters) HashCode() string {
	return fmt.Sprintf("Datacenters|%s", d.rawKey)
}

// Display returns a string that should be displayed to the user in output (for
// example).
func (d *Datacenters) Display() string {
	if d.rawKey == "" {
		return fmt.Sprintf(`"datacenters"`)
	}

	return fmt.Sprintf(`"datacenters(%s)"`, d.rawKey)
}

// ParseDatacenters creates a new datacenter dependency.
func ParseDatacenters(s ...string) (*Datacenters, error) {
	switch len(s) {
	case 0:
		return &Datacenters{rawKey: ""}, nil
	default:
		return nil, fmt.Errorf("expected 0 arguments, got %d", len(s))
	}
}
