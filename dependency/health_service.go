// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

const (
	HealthAny      = "any"
	HealthPassing  = "passing"
	HealthWarning  = "warning"
	HealthCritical = "critical"
	HealthMaint    = "maintenance"

	QueryNamespace     = "ns"
	QueryPartition     = "partition"
	QueryPeer          = "peer"
	QuerySamenessGroup = "sg"

	NodeMaint    = "_node_maintenance"
	ServiceMaint = "_service_maintenance:"
)

var (
	// Ensure implements
	_ Dependency = (*HealthServiceQuery)(nil)

	// HealthServiceQueryRe is the regular expression to use.
	HealthServiceQueryRe = regexp.MustCompile(`\A` + tagRe + serviceNameRe + queryRe + dcRe + nearRe + filterRe + `\z`)
)

func init() {
	gob.Register([]*HealthService{})
}

// HealthService is a service entry in Consul.
type HealthService struct {
	Node                   string
	NodeID                 string
	NodeAddress            string
	NodeTaggedAddresses    map[string]string
	NodeMeta               map[string]string
	ServiceMeta            map[string]string
	Address                string
	ServiceTaggedAddresses map[string]api.ServiceAddress
	ID                     string
	Name                   string
	Tags                   ServiceTags
	Checks                 api.HealthChecks
	Status                 string
	Port                   int
	Weights                api.AgentWeights
}

// HealthServiceQuery is the representation of all a service query in Consul.
type HealthServiceQuery struct {
	stopCh chan struct{}

	dc            string
	filters       []string
	name          string
	near          string
	tag           string
	connect       bool
	partition     string
	peer          string
	namespace     string
	samenessGroup string
}

// NewHealthServiceQuery processes the strings to build a service dependency.
func NewHealthServiceQuery(s string) (*HealthServiceQuery, error) {
	return healthServiceQuery(s, false)
}

// NewHealthConnect Query processes the strings to build a connect dependency.
func NewHealthConnectQuery(s string) (*HealthServiceQuery, error) {
	return healthServiceQuery(s, true)
}

func healthServiceQuery(s string, connect bool) (*HealthServiceQuery, error) {
	if !HealthServiceQueryRe.MatchString(s) {
		return nil, fmt.Errorf("health.service: invalid format: %q", s)
	}

	m := regexpMatch(HealthServiceQueryRe, s)

	var filters []string
	if filter := m["filter"]; filter != "" {
		split := strings.Split(filter, ",")
		for _, f := range split {
			f = strings.TrimSpace(f)
			switch f {
			case HealthAny,
				HealthPassing,
				HealthWarning,
				HealthCritical,
				HealthMaint:
				filters = append(filters, f)
			case "":
			default:
				return nil, fmt.Errorf(
					"health.service: invalid filter: %q in %q", f, s)
			}
		}
		sort.Strings(filters)
	} else {
		filters = []string{HealthPassing}
	}

	// Parse optional query into key pairs.
	queryParams := url.Values{}
	if queryRaw := m["query"]; queryRaw != "" {
		var err error
		queryParams, err = url.ParseQuery(queryRaw)
		if err != nil {
			return nil, fmt.Errorf(
				"health.service: invalid query: %q: %s", queryRaw, err)
		}
		// Validate keys.
		for key := range queryParams {
			switch key {
			case QueryNamespace,
				QueryPeer,
				QueryPartition,
				QuerySamenessGroup:
			default:
				return nil,
					fmt.Errorf("health.service: invalid query parameter key %q in query %q: supported keys: %s,%s,%s", key, queryRaw, QueryNamespace, QueryPeer, QueryPartition)
			}
		}
	}

	return &HealthServiceQuery{
		stopCh:        make(chan struct{}, 1),
		dc:            m["dc"],
		filters:       filters,
		name:          m["name"],
		near:          m["near"],
		tag:           m["tag"],
		connect:       connect,
		namespace:     queryParams.Get(QueryNamespace),
		peer:          queryParams.Get(QueryPeer),
		partition:     queryParams.Get(QueryPartition),
		samenessGroup: queryParams.Get(QuerySamenessGroup),
	}, nil
}

type queryLocality struct {
	// datacenter is the datacenter parsed from a label that has an explicit datacenter part.
	// Example query: <service>.virtual.<namespace>.ns.<partition>.ap.<datacenter>.dc.consul
	partition string

	// peer is the peer name parsed from a label that has explicit parts.
	// Example query: <service>.virtual.<namespace>.ns.<peer>.peer.<partition>.ap.consul
	peer string

	namespace string
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of HealthService objects.
// When sameness group is specified, fetch all members in sameness group
func (d *HealthServiceQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	list := make([]*HealthService, 0)
	localities := make([]queryLocality, 0)
	rm := &ResponseMetadata{}
	if d.samenessGroup != "" {
		log.Printf("[TRACE] %s: GET %s", d, &url.URL{
			Path:     "V1/config/sameness-group/" + d.samenessGroup,
			RawQuery: opts.String(),
		})

		// fetch all members in sameness group
		configEntry, _, err := clients.Consul().ConfigEntries().Get("sameness-group", d.samenessGroup, opts.ToConsulOpts())
		if err != nil {
			return nil, nil, errors.Wrap(err, d.String())
		}

		sgConfigEntry, ok := configEntry.(*api.SamenessGroupConfigEntry)
		if !ok {
			return nil, nil, fmt.Errorf("could not convert config ") //todo fix it
		}

		for _, sgm := range sgConfigEntry.Members {
			localities = append(localities, samenessGroupMemberToLocality(sgm, d.namespace, d.partition))
		}

		// Only one of dc or peer can be used.
		if d.peer != "" {
			d.dc = ""
		}
	}

	for _, locality := range localities {
		opts = opts.Merge(&QueryOptions{
			Datacenter:      d.dc,
			Near:            d.near,
			ConsulNamespace: locality.namespace,
			ConsulPartition: locality.partition,
			ConsulPeer:      locality.peer,
		})

		u := &url.URL{
			Path:     "/v1/health/service/" + d.name,
			RawQuery: opts.String(),
		}
		if d.tag != "" {
			q := u.Query()
			q.Set("tag", d.tag)
			u.RawQuery = q.Encode()
		}
		log.Printf("[TRACE] %s: GET %s", d, u)

		// Check if a user-supplied filter was given. If so, we may be querying for
		// more than healthy services, so we need to implement client-side
		// filtering.
		passingOnly := len(d.filters) == 1 && d.filters[0] == HealthPassing

		nodes := clients.Consul().Health().Service
		if d.connect {
			nodes = clients.Consul().Health().Connect
		}
		entries, qm, err := nodes(d.name, d.tag, passingOnly, opts.ToConsulOpts())
		if err != nil {
			return nil, nil, errors.Wrap(err, d.String())
		}

		log.Printf("[TRACE] %s: returned %d results", d, len(entries))

		if len(entries) == 0 {
			continue
		}
		for _, entry := range entries {
			// Get the status of this service from its checks.
			status := entry.Checks.AggregatedStatus()

			// If we are not checking only healthy services, filter out services
			// that do not match the given filter.
			if !acceptStatus(d.filters, status) {
				continue
			}

			// Get the address of the service, falling back to the address of the
			// node.
			address := entry.Service.Address
			if address == "" {
				address = entry.Node.Address
			}

			list = append(list, &HealthService{
				Node:                   entry.Node.Node,
				NodeID:                 entry.Node.ID,
				NodeAddress:            entry.Node.Address,
				NodeTaggedAddresses:    entry.Node.TaggedAddresses,
				NodeMeta:               entry.Node.Meta,
				ServiceMeta:            entry.Service.Meta,
				Address:                address,
				ServiceTaggedAddresses: entry.Service.TaggedAddresses,
				ID:                     entry.Service.ID,
				Name:                   entry.Service.Service,
				Tags: ServiceTags(
					deepCopyAndSortTags(entry.Service.Tags)),
				Status:  status,
				Checks:  entry.Checks,
				Port:    entry.Service.Port,
				Weights: entry.Service.Weights,
			})
		}

		log.Printf("[TRACE] %s: returned %d results after filtering", d, len(list))

		// Sort unless the user explicitly asked for nearness
		if d.near == "" {
			sort.Stable(ByNodeThenID(list))
		}

		rm = &ResponseMetadata{
			LastIndex:   qm.LastIndex,
			LastContact: qm.LastContact,
		}

		return list, rm, nil
	}

	return list, rm, nil
}

// CanShare returns a boolean if this dependency is shareable.
func (d *HealthServiceQuery) CanShare() bool {
	return true
}

// Stop halts the dependency's fetch function.
func (d *HealthServiceQuery) Stop() {
	close(d.stopCh)
}

func samenessGroupMemberToLocality(sgm api.SamenessGroupMember, ns string, ap string) queryLocality {

	var locality queryLocality
	labels := make([]string, 0)

	if sgm.Peer != "" {
		labels = append(labels, []string{sgm.Peer, "peer"}...)
		locality.peer = sgm.Peer
	}

	// If we are looking for a partition member, add that as the ap
	// otherwise use the provided ap
	if sgm.Partition != "" {
		labels = append(labels, []string{sgm.Partition, "ap"}...)
		locality.partition = sgm.Partition
	} else if ap != "" {
		labels = append(labels, []string{ap, "ap"}...)
		locality.partition = ap
	}

	if ns != "" {
		labels = append(labels, []string{ns, "ns"}...)
		locality.namespace = ns
	}

	return locality
}

// String returns the human-friendly version of this dependency.
func (d *HealthServiceQuery) String() string {
	name := d.name
	if d.tag != "" {
		name = d.tag + "." + name
	}
	if d.dc != "" {
		name = name + "@" + d.dc
	}
	if d.near != "" {
		name = name + "~" + d.near
	}
	if len(d.filters) > 0 {
		name = name + "|" + strings.Join(d.filters, ",")
	}
	if d.connect {
		return fmt.Sprintf("health.connect(%s)", name)
	}
	return fmt.Sprintf("health.service(%s)", name)
}

// Type returns the type of this dependency.
func (d *HealthServiceQuery) Type() Type {
	return TypeConsul
}

// acceptStatus allows us to check if a slice of health checks pass this filter.
func acceptStatus(list []string, s string) bool {
	for _, status := range list {
		if status == s || status == HealthAny {
			return true
		}
	}
	return false
}

// ByNodeThenID is a sortable slice of Service
type ByNodeThenID []*HealthService

// Len, Swap, and Less are used to implement the sort.Sort interface.
func (s ByNodeThenID) Len() int      { return len(s) }
func (s ByNodeThenID) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ByNodeThenID) Less(i, j int) bool {
	if s[i].Node < s[j].Node {
		return true
	} else if s[i].Node == s[j].Node {
		return s[i].ID < s[j].ID
	}
	return false
}
