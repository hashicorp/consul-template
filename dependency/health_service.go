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
)

// Ripped from https://github.com/hashicorp/consul/blob/master/consul/structs/structs.go#L31
const (
	HealthAny      = "any"
	HealthUnknown  = "unknown"
	HealthPassing  = "passing"
	HealthWarning  = "warning"
	HealthCritical = "critical"
)

// HealthService is a service entry in Consul
type HealthService struct {
	Node        string
	NodeAddress string
	Address     string
	ID          string
	Name        string
	Tags        ServiceTags
	Port        uint64
}

// from inside a template.
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
	entries, qm, err := health.Service(d.Name, d.Tag, d.Status.onlyAllowPassing(), options)
	if err != nil {
		return nil, qm, err
	}

	log.Printf("[DEBUG] (%s) Consul returned %d services", d.Display(), len(entries))

	services := make([]*HealthService, 0, len(entries))

	for _, entry := range entries {
		if !d.Status.accept(entry.Checks) {
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
	var (
		query  string
		status ServiceStatusFilter
	)

	switch len(s) {
	case 1:
		query = s[0]
		status = ServiceStatusFilter{HealthPassing}
	case 2:
		query = s[0]
		rawStatuses := strings.Split(s[1], ",")
		status = make(ServiceStatusFilter, len(rawStatuses))

		for i, rawStatus := range rawStatuses {
			rawStatus = strings.TrimSpace(rawStatus)
			if rawStatus == HealthAny {
				status = ServiceStatusFilter{}
				break
			} else if rawStatus == HealthPassing || rawStatus == HealthWarning || rawStatus == HealthCritical {
				status[i] = rawStatus
			} else {
				return nil, fmt.Errorf("expected some of %q as health status",
					strings.Join([]string{HealthAny, HealthPassing, HealthWarning, HealthCritical}, ", "))
			}
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

func (f ServiceStatusFilter) String() string {
	if len(f) < 1 {
		return fmt.Sprintf("[%s]", HealthAny)
	}
	return fmt.Sprintf("[%s]", strings.Join(f, ","))
}

// onlyAllowPassing allows us to use the passingOnly argumeny in the health service
func (f ServiceStatusFilter) onlyAllowPassing() bool {
	if len(f) < 1 {
		return false
	}
	for _, status := range f {
		if status != HealthPassing {
			return false
		}
	}
	return true
}

// accept allows us to check if a slice of health checks pass this filter.
func (f ServiceStatusFilter) accept(checks []*api.HealthCheck) bool {
	if len(f) < 1 {
		return true
	}
	for _, check := range checks {
		accept := false
		for _, status := range f {
			if status == check.Status {
				accept = true
				break
			}
		}
		if !accept {
			return false
		}
	}
	return true
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
