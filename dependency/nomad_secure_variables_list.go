package dependency

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/url"
	"regexp"

	"github.com/pkg/errors"
)

var (
	// Ensure implements
	_ Dependency = (*SVListQuery)(nil)

	// SVListQueryRe is the regular expression to use.
	SVListQueryRe = regexp.MustCompile(`\A` + prefixRe + `\z`)
)

func init() {
	gob.Register([]*NomadSVMeta{})
}

// SVListQuery queries the SV store for the metadata for keys matching the given
// prefix.
type SVListQuery struct {
	stopCh chan struct{}

	prefix string
}

// NewSVListQuery parses a string into a dependency.
func NewSVListQuery(s string) (*SVListQuery, error) {
	if s != "" && !SVListQueryRe.MatchString(s) {
		return nil, fmt.Errorf("nomad.secure_variables.list: invalid format: %q", s)
	}

	m := regexpMatch(SVListQueryRe, s)
	return &SVListQuery{
		stopCh: make(chan struct{}, 1),
		prefix: m["prefix"],
	}, nil
}

// Fetch queries the Nomad API defined by the given client.
func (d *SVListQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	opts = opts.Merge(&QueryOptions{})

	log.Printf("[TRACE] %s: GET %s", d, &url.URL{
		Path:     "/v1/vars/",
		RawQuery: opts.String(),
	})

	list, qm, err := clients.Nomad().SecureVariables().PrefixList(d.prefix, opts.ToNomadOpts())
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	log.Printf("[TRACE] %s: returned %d paths", d, len(list))

	vars := make([]*NomadSVMeta, 0, len(list))
	for _, sv := range list {
		vars = append(vars, NewNomadSVMeta(sv))
	}

	rm := &ResponseMetadata{
		LastIndex:   qm.LastIndex,
		LastContact: qm.LastContact,
	}

	return vars, rm, nil
}

// CanShare returns a boolean if this dependency is shareable.
func (d *SVListQuery) CanShare() bool {
	return true
}

// String returns the human-friendly version of this dependency.
func (d *SVListQuery) String() string {
	prefix := d.prefix
	return fmt.Sprintf("nomad.secure_variables.list(%s)", prefix)
}

// Stop halts the dependency's fetch function.
func (d *SVListQuery) Stop() {
	close(d.stopCh)
}

// Type returns the type of this dependency.
func (d *SVListQuery) Type() Type {
	return TypeNomad
}
