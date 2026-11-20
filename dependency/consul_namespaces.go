package dependency

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"log"
	"net/url"
	"slices"
	"strings"
	"time"
)

// Ensure implements
var (
	_ Dependency = (*ListNamespacesQuery)(nil)

	// ListNamespacesQuerySleepTime is the amount of time to sleep between
	// queries, since the endpoint does not support blocking queries.
	ListNamespacesQuerySleepTime = DefaultNonBlockingQuerySleepTime
)

type Namespace struct {
	Name        string
	Description string
}

// ListNamespacesQuery is the representation of a requested namespaces
// dependency from inside a template.
type ListNamespacesQuery struct {
	stopCh chan struct{}
}

func (c *ListNamespacesQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	opts = opts.Merge(&QueryOptions{})

	log.Printf("[TRACE] %s: GET %s", c, &url.URL{
		Path:     "/v1/namespaces",
		RawQuery: opts.String(),
	})

	// This is certainly not elegant, but the namespaces endpoint does not support
	// blocking queries, so we are going to "fake it until we make it". When we
	// first query, the LastIndex will be "0", meaning we should immediately
	// return data, but future calls will include a LastIndex. If we have a
	// LastIndex in the query metadata, sleep for 15 seconds before asking Consul
	// again.
	//
	// This is probably okay given the frequency in which namespaces actually
	// change, but is technically not edge-triggering.
	if opts.WaitIndex != 0 {
		log.Printf("[TRACE] %s: long polling for %s", c, ListNamespacesQuerySleepTime)

		select {
		case <-c.stopCh:
			return nil, nil, ErrStopped
		case <-time.After(ListNamespacesQuerySleepTime):
		}
	}

	namespaces, _, err := clients.Consul().Namespaces().List(opts.ToConsulOpts())
	if err != nil {
		if strings.Contains(err.Error(), "Invalid URL path") {
			return nil, nil, fmt.Errorf("%s: Namespaces are an enterprise feature: %w", c.String(), err)
		}

		return nil, nil, fmt.Errorf("%s: %w", c.String(), err)
	}

	log.Printf("[TRACE] %s: returned %d results", c, len(namespaces))

	slices.SortFunc(namespaces, func(i, j *api.Namespace) int {
		return strings.Compare(i.Name, j.Name)
	})

	resp := []*Namespace{}
	for _, namespace := range namespaces {
		if namespace != nil {
			resp = append(resp, &Namespace{
				Name:        namespace.Name,
				Description: namespace.Description,
			})
		}
	}

	// Use respWithMetadata which always increments LastIndex and results
	// in fetching new data for endpoints that don't support blocking queries
	return respWithMetadata(resp)
}

// CanShare returns if this dependency is shareable when consul-template is running in de-duplication mode.
func (c *ListNamespacesQuery) CanShare() bool {
	return true
}

func (c *ListNamespacesQuery) String() string {
	return "list.namespaces"
}

func (c *ListNamespacesQuery) Stop() {
	close(c.stopCh)
}

func (c *ListNamespacesQuery) Type() Type {
	return TypeConsul
}
