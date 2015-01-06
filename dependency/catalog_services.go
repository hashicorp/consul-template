package dependency

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"

	"github.com/hashicorp/consul/api"
)

// CatalogService is a catalog entry in Consul.
type CatalogService struct {
	Name string
	Tags []string
}

// CatalogServices is the representation of a requested catalog service
// dependency from inside a template.
type CatalogServices struct {
	rawKey     string
	Name       string
	Tags       []string
	DataCenter string
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of CatalogService objects.
func (d *CatalogServices) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	if d.DataCenter != "" {
		options.Datacenter = d.DataCenter
	}

	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), options)

	catalog := client.Catalog()
	entries, qm, err := catalog.Services(options)
	if err != nil {
		return nil, qm, err
	}

	log.Printf("[DEBUG] (%s) Consul returned %d catalog services", d.Display(), len(entries))

	var catalogServices []*CatalogService
	for name, tags := range entries {
		tags = deepCopyAndSortTags(tags)
		catalogServices = append(catalogServices, &CatalogService{
			Name: name,
			Tags: tags,
		})
	}

	sort.Stable(CatalogServicesList(catalogServices))

	return catalogServices, qm, nil
}

// HashCode returns the hash code for this dependency.
func (d *CatalogServices) HashCode() string {
	return fmt.Sprintf("CatalogServices|%s", d.Key())
}

// Key returns the key given by the user in the template.
func (d *CatalogServices) Key() string {
	return d.rawKey
}

// Display returns a string that should be displayed to the user in output (for
// example).
func (d *CatalogServices) Display() string {
	if d.rawKey == "" {
		return fmt.Sprintf("catalog services")
	}

	return fmt.Sprintf(`catalog services "%s"`, d.rawKey)
}

// ParseCatalogServices parses a string of the format @dc.
func ParseCatalogServices(s ...string) (*CatalogServices, error) {
	switch len(s) {
	case 0:
		return &CatalogServices{rawKey: ""}, nil
	case 1:
		dc := s[0]

		re := regexp.MustCompile(`\A` +
			`(@(?P<datacenter>[[:word:]\.\-]+))?` +
			`\z`)
		names := re.SubexpNames()
		match := re.FindAllStringSubmatch(dc, -1)

		if len(match) == 0 {
			return nil, errors.New("invalid catalog service dependency format")
		}

		r := match[0]

		m := map[string]string{}
		for i, n := range r {
			if names[i] != "" {
				m[names[i]] = n
			}
		}

		nd := &CatalogServices{
			rawKey:     dc,
			DataCenter: m["datacenter"],
		}

		return nd, nil
	default:
		return nil, fmt.Errorf("expected 0 or 1 arguments, got %d", len(s))
	}
}

/// --- Sorting

// CatalogServicesList is a sortable slice of CatalogService structs.
type CatalogServicesList []*CatalogService

func (s CatalogServicesList) Len() int {
	return len(s)
}

func (s CatalogServicesList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s CatalogServicesList) Less(i, j int) bool {
	if s[i].Name <= s[j].Name {
		return true
	}
	return false
}
