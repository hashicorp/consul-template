package dependency

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	nomadapi "github.com/hashicorp/nomad/api"
)

const (
	dcRe           = `(@(?P<dc>[[:word:]\.\-\_]+))?`
	keyRe          = `/?(?P<key>[^@]+)`
	filterRe       = `(\|(?P<filter>[[:word:]\,]+))?`
	serviceNameRe  = `(?P<name>[[:word:]\-\_]+)`
	nodeNameRe     = `(?P<name>[[:word:]\.\-\_]+)`
	nearRe         = `(~(?P<near>[[:word:]\.\-\_]+))?`
	prefixRe       = `/?(?P<prefix>[^@]+)`
	tagRe          = `((?P<tag>[[:word:]=:\.\-\_]+)\.)?`
	regionRe       = `(@(?P<region>[[:word:]\.\-\_]+))?`
	nvPathRe       = `/?(?P<path>[^@]+)`
	nvNamespaceRe  = `(@(?P<namespace>[[:word:]\-\_]+))?`
	nvListPrefixRe = `/?(?P<prefix>[^@]*)`
	nvListNSRe     = `(@(?P<namespace>([[:word:]\-\_]+|\*)))?`
	nvRegionRe     = `(\.(?P<region>[[:word:]\-\_]+))?`
)

type Type int

const (
	TypeConsul Type = iota
	TypeVault
	TypeLocal
	TypeNomad
)

// Dependency is an interface for a dependency that Consul Template is capable
// of watching.
type Dependency interface {
	Fetch(*ClientSet, *QueryOptions) (interface{}, *ResponseMetadata, error)
	CanShare() bool
	String() string
	Stop()
	Type() Type
}

// ServiceTags is a slice of tags assigned to a Service
type ServiceTags []string

// QueryOptions is a list of options to send with the query. These options are
// client-agnostic, and the dependency determines which, if any, of the options
// to use.
type QueryOptions struct {
	AllowStale        bool
	Datacenter        string
	Region            string
	Near              string
	Choose            string
	RequireConsistent bool
	VaultGrace        time.Duration
	WaitIndex         uint64
	WaitTime          time.Duration
}

func (q *QueryOptions) Merge(o *QueryOptions) *QueryOptions {
	var r QueryOptions

	if q == nil {
		if o == nil {
			return &QueryOptions{}
		}
		r = *o
		return &r
	}

	r = *q

	if o == nil {
		return &r
	}

	if o.AllowStale != false {
		r.AllowStale = o.AllowStale
	}

	if o.Datacenter != "" {
		r.Datacenter = o.Datacenter
	}

	if o.Region != "" {
		r.Region = o.Region
	}

	if o.Near != "" {
		r.Near = o.Near
	}

	if o.Choose != "" {
		r.Choose = o.Choose
	}

	if o.RequireConsistent != false {
		r.RequireConsistent = o.RequireConsistent
	}

	if o.WaitIndex != 0 {
		r.WaitIndex = o.WaitIndex
	}

	if o.WaitTime != 0 {
		r.WaitTime = o.WaitTime
	}

	return &r
}

func (q *QueryOptions) ToConsulOpts() *consulapi.QueryOptions {
	return &consulapi.QueryOptions{
		AllowStale:        q.AllowStale,
		Datacenter:        q.Datacenter,
		Near:              q.Near,
		RequireConsistent: q.RequireConsistent,
		WaitIndex:         q.WaitIndex,
		WaitTime:          q.WaitTime,
	}
}

func (q *QueryOptions) ToNomadOpts() *nomadapi.QueryOptions {
	var params map[string]string
	if q.Choose != "" {
		params = map[string]string{
			"choose": q.Choose,
		}
	}
	return &nomadapi.QueryOptions{
		AllowStale: q.AllowStale,
		Region:     q.Region,
		Params:     params,
		WaitIndex:  q.WaitIndex,
		WaitTime:   q.WaitTime,
	}
}

func (q *QueryOptions) String() string {
	u := &url.Values{}

	if q.AllowStale {
		u.Add("stale", strconv.FormatBool(q.AllowStale))
	}

	if q.Datacenter != "" {
		u.Add("dc", q.Datacenter)
	}

	if q.Region != "" {
		u.Add("region", q.Region)
	}

	if q.Near != "" {
		u.Add("near", q.Near)
	}

	if q.Choose != "" {
		u.Add("choose", q.Choose)
	}

	if q.RequireConsistent {
		u.Add("consistent", strconv.FormatBool(q.RequireConsistent))
	}

	if q.WaitIndex != 0 {
		u.Add("index", strconv.FormatUint(q.WaitIndex, 10))
	}

	if q.WaitTime != 0 {
		u.Add("wait", q.WaitTime.String())
	}

	return u.Encode()
}

// ResponseMetadata is a struct that contains metadata about the response. This
// is returned from a Fetch function call.
type ResponseMetadata struct {
	LastIndex   uint64
	LastContact time.Duration
	BlockOnNil  bool // keep blocking on `nil` data returns
}

// deepCopyAndSortTags deep copies the tags in the given string slice and then
// sorts and returns the copied result.
func deepCopyAndSortTags(tags []string) []string {
	newTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		newTags = append(newTags, tag)
	}
	sort.Strings(newTags)
	return newTags
}

// respWithMetadata is a short wrapper to return the given interface with fake
// response metadata for non-Consul dependencies.
func respWithMetadata(i interface{}) (interface{}, *ResponseMetadata, error) {
	return i, &ResponseMetadata{
		LastContact: 0,
		LastIndex:   uint64(time.Now().Unix()),
	}, nil
}

// regexpMatch matches the given regexp and extracts the match groups into a
// named map.
func regexpMatch(re *regexp.Regexp, q string) map[string]string {
	names := re.SubexpNames()
	match := re.FindAllStringSubmatch(q, -1)

	if len(match) == 0 {
		return map[string]string{}
	}

	m := map[string]string{}
	for i, n := range match[0] {
		if names[i] != "" {
			m[names[i]] = n
		}
	}

	return m
}
