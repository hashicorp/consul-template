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
	SVGetQueryRe = regexp.MustCompile(`\A` + svPathRe + `\z`)
)

// SVGetQuery queries the KV store for a single key.
type SVGetQuery struct {
	stopCh chan struct{}

	path       string
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
		stopCh: make(chan struct{}, 1),
		path:   m["path"],
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

	// NOTE: The Peek method of the Nomad SV API will check a value, return it
	// if it exists, but return a nil value and NO error if it is not found.
	sv, qm, err := clients.Nomad().SecureVariables().Peek(d.path, opts.ToNomadOpts())
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
func (d *SVGetQuery) String() string {
	path := d.path
	if d.blockOnNil {
		return fmt.Sprintf("nomad.secure_variables.block(%s)", path)
	}
	return fmt.Sprintf("nomad.secure_variables.get(%s)", path)
}

// Stop halts the dependency's fetch function.
func (d *SVGetQuery) Stop() {
	close(d.stopCh)
}

// Type returns the type of this dependency.
func (d *SVGetQuery) Type() Type {
	return TypeNomad
}
