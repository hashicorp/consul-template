package dependency

import (
	"fmt"
	"log"
	"sort"
	"time"
)

var sleepTime = 15 * time.Second

// Datacenters is the dependency to query all datacenters
type Datacenters struct {
	rawKey string
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of strings representing the datacenters
func (d *Datacenters) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}

	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), opts)

	// This is pretty ghetto, but the datacenters endpoint does not support
	// blocking queries, so we are going to "fake it until we make it". When we
	// first query, the LastIndex will be "0", meaning we should immediately
	// return data, but future calls will include a LastIndex. If we have a
	// LastIndex in the query metadata, sleep for 15 seconds before asking Consul
	// again.
	//
	// This is probably okay given the frequency in which datacenters actually
	// change, but is technically not edge-triggering.
	if opts.WaitIndex != 0 {
		log.Printf("[DEBUG] (%s) pretending to long-poll", d.Display())
		time.Sleep(sleepTime)
	}

	consul, err := clients.Consul()
	if err != nil {
		return nil, nil, fmt.Errorf("datacenters: error getting client: %s", err)
	}

	catalog := consul.Catalog()
	result, err := catalog.Datacenters()
	if err != nil {
		return nil, nil, fmt.Errorf("datacenters: error fetching: %s", err)
	}

	log.Printf("[DEBUG] (%s) Consul returned %d datacenters", d.Display(), len(result))
	sort.Strings(result)

	ts := time.Now().Unix()
	rm := &ResponseMetadata{
		LastContact: 0,
		LastIndex:   uint64(ts),
	}

	return result, rm, nil
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
