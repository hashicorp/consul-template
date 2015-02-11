package dependency

import (
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/consul/api"
)

// Node is a node entry in Consul
type Node struct {
	Node    string
	Address string
}

type CatalogNodes struct {
	rawKey     string
	DataCenter string
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of Node objects
func (d *CatalogNodes) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	if d.DataCenter != "" {
		options.Datacenter = d.DataCenter
	}

	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), options)

	catalog := client.Catalog()
	n, qm, err := catalog.Nodes(options)
	if err != nil {
		return nil, qm, err
	}

	log.Printf("[DEBUG] (%s) Consul returned %d nodes", d.Display(), len(n))

	nodes := make([]*Node, 0, len(n))
	for _, node := range n {
		nodes = append(nodes, &Node{
			Node:    node.Node,
			Address: node.Address,
		})
	}

	return nodes, qm, nil
}

func (d *CatalogNodes) HashCode() string {
	return fmt.Sprintf("CatalogNodes|%s", d.rawKey)
}

func (d *CatalogNodes) Display() string {
	if d.rawKey == "" {
		return fmt.Sprintf("nodes")
	}

	return fmt.Sprintf(`nodes "%s"`, d.rawKey)
}

// ParseCatalogNodes parses a string of the format @dc.
func ParseCatalogNodes(s ...string) (*CatalogNodes, error) {
	switch len(s) {
	case 0:
		return &CatalogNodes{rawKey: ""}, nil
	case 1:
		dc := s[0]

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

		nd := &CatalogNodes{
			rawKey:     dc,
			DataCenter: m["datacenter"],
		}

		return nd, nil
	default:
		return nil, fmt.Errorf("expected 0 or 1 arguments, got %d", len(s))
	}
}
