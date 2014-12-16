package util

import (
	"errors"
	"fmt"
	"log"
	"regexp"

	api "github.com/armon/consul-api"
)

// Node is a node entry in Consul
type Node struct {
	Node    string
	Address string
}

type NodesDependency struct {
	rawKey     string
	DataCenter string
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of Node objects
func (d *NodesDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	if d.DataCenter != "" {
		options.Datacenter = d.DataCenter
	}

	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), options)

	catalog := client.Catalog()
	n, qm, err := catalog.Nodes(options)
	if err != nil {
		return err, qm, nil
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

func (d *NodesDependency) HashCode() string {
	return fmt.Sprintf("NodesDependency|%s", d.Key())
}

func (d *NodesDependency) Key() string {
	return d.rawKey
}

func (d *NodesDependency) Display() string {
	return fmt.Sprintf(`node "%s"`, d.rawKey)
}

// ParseNodesDependency parses a string of the format @dc.
func ParseNodesDependency(s ...string) (*NodesDependency, error) {
	switch len(s) {
	case 0:
		return &NodesDependency{rawKey: ""}, nil
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

		nd := &NodesDependency{
			rawKey:     dc,
			DataCenter: m["datacenter"],
		}

		return nd, nil
	default:
		return nil, fmt.Errorf("expected 0 or 1 arguments, got %d", len(s))
	}
}
