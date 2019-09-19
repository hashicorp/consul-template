package dependency

import (
	"encoding/gob"
	"fmt"
	"regexp"

	"github.com/pkg/errors"
)

var (
	// Ensure implements
	_ Dependency = (*CatalogPreparedQuery)(nil)

	// CatalogPreparedQueryRe is the regular expression to use.
	CatalogPreparedQueryRe = regexp.MustCompile(`\A` + serviceNameRe + `\z`)
)

func init() {
	gob.Register([]*CatalogQuery{})
}

// CatalogQuery is a catalog entry in Consul.
type CatalogQuery struct {
	ID              string
	Node            string
	Address         string
	Datacenter      string
	TaggedAddresses map[string]string
	NodeMeta        map[string]string
	ServiceID       string
	ServiceName     string
	ServiceAddress  string
	ServiceTags     ServiceTags
	ServiceMeta     map[string]string
	ServicePort     int
}

// CatalogPreparedQuery is the representation of a requested prepared query executions
// dependency from inside a template.
type CatalogPreparedQuery struct {
	stopCh chan struct{}

	name string
}

// NewCatalogPreparedQuery parses a string into a CatalogPreparedQuery.
func NewCatalogPreparedQuery(s string) (*CatalogPreparedQuery, error) {
	if !CatalogPreparedQueryRe.MatchString(s) {
		return nil, fmt.Errorf("catalog.query: invalid format: %q", s)
	}

	m := regexpMatch(CatalogPreparedQueryRe, s)
	return &CatalogPreparedQuery{
		stopCh: make(chan struct{}, 1),
		name:   m["name"],
	}, nil
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of CatalogService objects.
func (d *CatalogPreparedQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	entries, qm, err := clients.Consul().PreparedQuery().Execute(d.name, opts.ToConsulOpts())
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	var list []*CatalogQuery
	for _, s := range entries.Nodes {

		list = append(list, &CatalogQuery{
			ID:              s.Service.ID,
			Node:            s.Node.Node,
			Address:         s.Service.Address,
			Datacenter:      s.Node.Datacenter,
			TaggedAddresses: s.Node.TaggedAddresses,
			NodeMeta:        s.Node.Meta,
			ServiceID:       s.Service.ID,
			ServiceName:     s.Service.Service,
			ServiceAddress:  s.Service.Address,
			ServiceTags:     ServiceTags(deepCopyAndSortTags(s.Service.Tags)),
			ServiceMeta:     s.Service.Meta,
			ServicePort:     s.Service.Port,
		})
	}

	rm := &ResponseMetadata{
		LastIndex:   qm.LastIndex,
		LastContact: qm.LastContact,
	}

	return list, rm, nil
}

// CanShare returns a boolean if this dependency is shareable.
func (d *CatalogPreparedQuery) CanShare() bool {
	return true
}

// String returns the human-friendly version of this dependency.
func (d *CatalogPreparedQuery) String() string {
	name := d.name
	return fmt.Sprintf("query(%s)", name)
}

// Stop halts the dependency's fetch function.
func (d *CatalogPreparedQuery) Stop() {
	close(d.stopCh)
}

// Type returns the type of this dependency.
func (d *CatalogPreparedQuery) Type() Type {
	return TypeConsul
}
