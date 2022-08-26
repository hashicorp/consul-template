package dependency

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	// Ensure implements
	_ Dependency = (*SVGetQuery)(nil)

	// SVGetQueryRe is the regular expression to use.
	SVGetQueryRe = regexp.MustCompile(`\A` + svPathRe + svNamespaceRe + svRegionRe + `\z`)
)

// SVGetQuery queries the KV store for a single key.
type SVGetQuery struct {
	stopCh chan struct{}

	path      string
	namespace string
	region    string

	blockOnNil bool
}

// NewSVGetQuery parses a string into a dependency.
func NewSVGetQuery(s string) (*SVGetQuery, error) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")

	if s != "" && !SVGetQueryRe.MatchString(s) {
		return nil, fmt.Errorf("nomad.secure_variables.get: invalid format: %q", s)
	}

	m := regexpMatch(SVGetQueryRe, s)
	return &SVGetQuery{
		stopCh:    make(chan struct{}, 1),
		path:      m["path"],
		namespace: m["namespace"],
		region:    m["region"],
	}, nil
}

// Fetch queries the Nomad API defined by the given client.
func (d *SVGetQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	opts = opts.Merge(&QueryOptions{})

	log.Printf("[TRACE] %s: GET %s", d, &url.URL{
		Path:     "/v1/var/" + d.path,
		RawQuery: opts.String(),
	})

	nOpts := opts.ToNomadOpts()
	nOpts.Namespace = d.namespace
	nOpts.Region = d.region
	// NOTE: The Peek method of the Nomad SV API will check a value, return it
	// if it exists, but return a nil value and NO error if it is not found.
	sv, qm, err := clients.Nomad().SecureVariables().Peek(d.path, nOpts)
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	rm := &ResponseMetadata{
		LastIndex:   qm.LastIndex,
		LastContact: qm.LastContact,
		BlockOnNil:  d.blockOnNil,
	}

	if sv == nil {
		log.Printf("[TRACE] %s: returned nil", d)
		return nil, rm, nil
	}

	items := &NewNomadSecureVariable(sv).Items
	log.Printf("[TRACE] %s: returned %q", d, sv.Path)
	return items, rm, nil
}

// EnableBlocking turns this into a blocking KV query.
func (d *SVGetQuery) EnableBlocking() {
	d.blockOnNil = true
}

// CanShare returns a boolean if this dependency is shareable.
func (d *SVGetQuery) CanShare() bool {
	return true
}

// String returns the human-friendly version of this dependency.
// This value is also used to disambiguate multiple instances in the Brain
func (d *SVGetQuery) String() string {
	ns := d.namespace
	if ns == "" {
		ns = "default"
	}
	region := d.region
	if region == "" {
		region = "global"
	}
	path := d.path
	key := fmt.Sprintf("%s@%s.%s", path, ns, region)
	if d.blockOnNil {
		return fmt.Sprintf("nomad.var.block(%s)", key)
	}
	return fmt.Sprintf("nomad.var.get(%s)", key)
}

// Stop halts the dependency's fetch function.
func (d *SVGetQuery) Stop() {
	close(d.stopCh)
}

// Type returns the type of this dependency.
func (d *SVGetQuery) Type() Type {
	return TypeNomad
}
