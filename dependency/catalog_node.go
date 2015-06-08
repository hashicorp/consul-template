package dependency

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
)

type NodeDetail struct {
	Node     *Node
	Services NodeServiceList
}

type NodeService struct {
	Service string
	Tags    ServiceTags
	Port    int
}

type CatalogNode struct {
	rawKey     string
	dataCenter string
}

// Fetch queries the Consul API defined by the given client and returns a
// of NodeDetail object
func (d *CatalogNode) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}

	consulOpts := opts.consulQueryOptions()
	if d.dataCenter != "" {
		consulOpts.Datacenter = d.dataCenter
	}

	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), consulOpts)

	consul, err := clients.Consul()
	if err != nil {
		return nil, nil, fmt.Errorf("catalog node: error getting client: %s", err)
	}

	nodeName := d.rawKey
	if nodeName == "" {
		nodeName, err = consul.Agent().NodeName()
		if err != nil {
			return nil, nil, fmt.Errorf("catalog node: error getting the node name of the current agent: %s", err)
		}
	}

	catalog := consul.Catalog()
	n, qm, err := catalog.Node(nodeName, consulOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("catalog node: error fetching: %s", err)
	}

	var node *NodeDetail
	if n != nil {
		services := make(NodeServiceList, len(n.Services))
		i := 0
		for _, v := range n.Services {
			services[i] = &NodeService{
				Service: v.Service,
				Tags:    ServiceTags(deepCopyAndSortTags(v.Tags)),
				Port:    v.Port,
			}
			i++
		}
		sort.Stable(services)
		node = &NodeDetail{
			Node: &Node{
				Node:    n.Node.Node,
				Address: n.Node.Address,
			},
			Services: services,
		}
	}

	rm := &ResponseMetadata{
		LastIndex:   qm.LastIndex,
		LastContact: qm.LastContact,
	}

	return node, rm, nil
}

func (d *CatalogNode) HashCode() string {
	if d.dataCenter != "" {
		return fmt.Sprintf("NodeDetail|%s@%s", d.rawKey, d.dataCenter)
	}
	return fmt.Sprintf("NodeDetail|%s", d.rawKey)
}

func (d *CatalogNode) Display() string {
	if d.dataCenter != "" {
		return fmt.Sprintf("node(%s@%s)", d.rawKey, d.dataCenter)
	}
	return fmt.Sprintf(`"node(%s)"`, d.rawKey)
}

// ParseCatalogNode parses a name name and optional datacenter value.
// If the name is empty or not provided then the current agent is used.
func ParseCatalogNode(s ...string) (*CatalogNode, error) {
	switch len(s) {
	case 0:
		return &CatalogNode{}, nil
	case 1:
		return &CatalogNode{rawKey: s[0]}, nil
	case 2:
		dc := s[1]

		re := regexp.MustCompile(`\A` +
			`(@(?P<datacenter>[[:word:]\.\-]+))?` +
			`\z`)
		names := re.SubexpNames()
		match := re.FindAllStringSubmatch(dc, -1)

		if len(match) == 0 {
			return nil, errors.New("invalid node dependency format")
		}

		r := match[0]

		m := map[string]string{}
		for i, n := range r {
			if names[i] != "" {
				m[names[i]] = n
			}
		}

		nd := &CatalogNode{
			rawKey:     s[0],
			dataCenter: m["datacenter"],
		}

		return nd, nil
	default:
		return nil, fmt.Errorf("expected 0, 1, or 2 arguments, got %d", len(s))
	}
}

// Sorting

type NodeServiceList []*NodeService

func (s NodeServiceList) Len() int      { return len(s) }
func (s NodeServiceList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s NodeServiceList) Less(i, j int) bool {
	return s[i].Service <= s[j].Service
}
