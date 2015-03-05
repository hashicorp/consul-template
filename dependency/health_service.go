package dependency

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-multierror"
)

// Ripped from https://github.com/hashicorp/consul/blob/master/consul/structs/structs.go#L31
const (
	HealthAny      = "any"
	HealthUnknown  = "unknown"
	HealthPassing  = "passing"
	HealthWarning  = "warning"
	HealthCritical = "critical"
)

// HealthService is a service entry in Consul.
type HealthService struct {
	Node        string
	NodeAddress string
	Address     string
	ID          string
	Name        string
	Tags        ServiceTags
	Port        uint64
}

// HealthServices is the struct that is formed from the dependency inside a
// template.
type HealthServices struct {
	rawKey     string
	Name       string
	Tag        string
	DataCenter string
	Port       uint64
	Status     ServiceStatusFilter
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of HealthService objects.
func (d *HealthServices) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	if d.DataCenter != "" {
		options.Datacenter = d.DataCenter
	}

	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), options)

	health := client.Health()
	entries, qm, err := health.Service(d.Name, d.Tag, d.Status.passingOnly(), options)
	if err != nil {
		return nil, qm, err
	}

	log.Printf("[DEBUG] (%s) Consul returned %d services", d.Display(), len(entries))

	services := make([]*HealthService, 0, len(entries))

	for _, entry := range entries {
		if !d.Status.Accept(entry.Checks) {
			continue
		}

		tags := deepCopyAndSortTags(entry.Service.Tags)

		// Get the address of the service, falling back to the address of the node.
		var address string
		if entry.Service.Address != "" {
			address = entry.Service.Address
		} else {
			address = entry.Node.Address
		}

		services = append(services, &HealthService{
			Node:        entry.Node.Node,
			NodeAddress: entry.Node.Address,
			Address:     address,
			ID:          entry.Service.ID,
			Name:        entry.Service.Service,
			Tags:        tags,
			Port:        uint64(entry.Service.Port),
		})
	}

	log.Printf("[DEBUG] (%s) %d services after health check status filtering", d.Display(), len(services))

	sort.Stable(HealthServiceList(services))

	return services, qm, nil
}

func (d *HealthServices) HashCode() string {
	return fmt.Sprintf("HealthServices|%s", d.rawKey)
}

func (d *HealthServices) Display() string {
	return fmt.Sprintf(`"service(%s)"`, d.rawKey)
}

// ParseHealthServices processes the incoming strings to build a service dependency.
//
// Supported arguments
//   ParseHealthServices("service_id")
//   ParseHealthServices("service_id", "health_check")
//
// Where service_id is in the format of service(.tag(@datacenter(:port)))
// and health_check is either "any" or "passing".
//
// If no health_check is provided then its the same as "passing".
func ParseHealthServices(s ...string) (*HealthServices, error) {
	var query string
	var status ServiceStatusFilter
	var err error

	switch len(s) {
	case 1:
		query = s[0]
		status, err = NewServiceStatusFilter("")
		if err != nil {
			return nil, err
		}
	case 2:
		query = s[0]
		status, err = NewServiceStatusFilter(s[1])
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("expected 1 or 2 arguments, got %d", len(s))
	}

	if len(query) == 0 {
		return nil, errors.New("cannot specify empty health service dependency")
	}

	re := regexp.MustCompile(`\A` +
		`((?P<tag>[[:word:]\-.]+)\.)?` +
		`((?P<name>[[:word:]\-/_]+))` +
		`(@(?P<datacenter>[[:word:]\.\-]+))?(:(?P<port>[0-9]+))?` +
		`\z`)
	names := re.SubexpNames()
	match := re.FindAllStringSubmatch(query, -1)

	if len(match) == 0 {
		return nil, errors.New("invalid health service dependency format")
	}

	r := match[0]

	m := map[string]string{}
	for i, n := range r {
		if names[i] != "" {
			m[names[i]] = n
		}
	}

	tag, name, datacenter, port := m["tag"], m["name"], m["datacenter"], m["port"]

	if name == "" {
		return nil, errors.New("name part is required")
	}

	sd := &HealthServices{
		rawKey:     fmt.Sprintf("%s %s", query, status),
		Name:       name,
		Tag:        tag,
		DataCenter: datacenter,
		Status:     status,
	}

	if port != "" {
		port, err := strconv.ParseUint(port, 0, 64)
		if err != nil {
			return nil, err
		} else {
			sd.Port = port
		}
	}

	return sd, nil
}

// ServiceStatusFilter is used to specify a list of service statuses that you want filter by.
type ServiceStatusFilter []string

// String returns the string representation of this status filter
func (f ServiceStatusFilter) String() string {
	return fmt.Sprintf("[%s]", strings.Join(f, ","))
}

// NewServiceStatusFilter creates a status filter from the given string in the
// format `[key[,key[,key...]]]`. Each status is split on the comma character
// and must match one of the valid status names.
//
// If the empty string is given, it is assumed only "passing" statuses are to
// be returned.
//
// If the user specifies "any" with other keys, an error will be returned.
func NewServiceStatusFilter(s string) (ServiceStatusFilter, error) {
	// If no statuses were given, use the default status of "passing".
	if len(s) == 0 {
		return ServiceStatusFilter{HealthPassing}, nil
	}

	var errs *multierror.Error
	var hasAny bool

	raw := strings.Split(s, ",")
	trimmed := make(ServiceStatusFilter, 0, len(raw))
	for _, r := range raw {
		trim := strings.TrimSpace(r)

		// Ignore the empty string.
		if len(trim) == 0 {
			continue
		}

		// Record the case where we have the "any" status - it will be used later.
		if trim == HealthAny {
			hasAny = true
		}

		// Validate that the service is actually a valid name.
		if trim != HealthAny &&
			trim != HealthUnknown &&
			trim != HealthPassing &&
			trim != HealthWarning &&
			trim != HealthCritical {
			errs = multierror.Append(errs, fmt.Errorf("service filter: invalid filter %q", trim))
		}
		trimmed = append(trimmed, trim)
	}

	// If the user specified "any" with additional keys, that is invalid.
	if hasAny && len(trimmed) != 1 {
		errs = multierror.Append(errs, fmt.Errorf("service filter: cannot specify extra keys when using %q", "any"))
	}

	return trimmed, errs.ErrorOrNil()
}

// Accept allows us to check if a slice of health checks pass this filter.
func (f ServiceStatusFilter) Accept(checks []*api.HealthCheck) bool {
	// If the any filter is activated, pass everything.
	if f.any() {
		return true
	}

	// If ONLY the passing filter is activated, ensure all checks are passing and
	// return. If more than one filter is given (like "passing, unknown"), then
	// this conditional will not fire.
	if f.passingOnly() {
		for _, check := range checks {
			if check.Status != HealthPassing {
				return false
			}
		}
		return true
	}

	// Iterate over each status and see if ANY of the checks have that status.
	for _, status := range f {
		for _, check := range checks {
			if status == check.Status {
				return true
			}
		}
	}

	return false
}

// any is a helper method to determine if this is an "any" service status
// filter. If "any" was given, it must be the only item in the list.
func (f ServiceStatusFilter) any() bool {
	return len(f) == 1 && f[0] == HealthAny
}

// onlyPassing returns true if the filter contains only a "passing" service
// status filter.
//
// NOTE: If there is more than one filter, this will return false, even if one
// of the filters is "passing". This is because "passingOnly" is used to alter
// the type of query made to Consul.
func (f ServiceStatusFilter) passingOnly() bool {
	return len(f) == 1 && f[0] == HealthPassing
}

// ServiceTags is a slice of tags assigned to a Service
type ServiceTags []string

// Contains returns true if the tags exists in the ServiceTags slice.
func (t ServiceTags) Contains(s string) bool {
	for _, v := range t {
		if v == s {
			return true
		}
	}
	return false
}

// HealthServiceList is a sortable slice of Service
type HealthServiceList []*HealthService

// Len, Swap, and Less are used to implement the sort.Sort interface.
func (s HealthServiceList) Len() int      { return len(s) }
func (s HealthServiceList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s HealthServiceList) Less(i, j int) bool {
	if s[i].Node < s[j].Node {
		return true
	} else if s[i].Node == s[j].Node {
		return s[i].ID <= s[j].ID
	}
	return false
}
