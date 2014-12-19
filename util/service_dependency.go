package util

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	api "github.com/armon/consul-api"
)

// Ripped from https://github.com/hashicorp/consul/blob/master/consul/structs/structs.go#L31
const (
	HealthAny      = "any"
	HealthUnknown  = "unknown"
	HealthPassing  = "passing"
	HealthWarning  = "warning"
	HealthCritical = "critical"
)

// Service is a service entry in Consul
type Service struct {
	Node    string
	Address string
	ID      string
	Name    string
	Tags    ServiceTags
	Port    uint64
}

// from inside a template.
type ServiceDependency struct {
	rawKey     string
	Name       string
	Tag        string
	DataCenter string
	Port       uint64
	Status     ServiceStatusFilter
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of Service objects.
func (d *ServiceDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
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

	services := make([]*Service, 0, len(entries))

	for _, entry := range entries {
		if !d.Status.accept(entry.Checks) {
			continue
		}

		tags := deepCopyAndSortTags(entry.Service.Tags)

		services = append(services, &Service{
			Node:    entry.Node.Node,
			Address: entry.Node.Address,
			ID:      entry.Service.ID,
			Name:    entry.Service.Service,
			Tags:    tags,
			Port:    uint64(entry.Service.Port),
		})
	}

	log.Printf("[DEBUG] (%s) %d services after health check status filtering", d.Display(), len(services))

	sort.Stable(ServiceList(services))

	return services, qm, nil
}

func (d *ServiceDependency) HashCode() string {
	return fmt.Sprintf("ServiceDependency|%s", d.Key())
}

func (d *ServiceDependency) Key() string {
	return d.rawKey
}

func (d *ServiceDependency) Display() string {
	return fmt.Sprintf(`service "%s"`, d.rawKey)
}

// AddToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *ServiceDependency) AddToContext(context *TemplateContext, data interface{}) error {
	coerced, ok := data.([]*Service)
	if !ok {
		return fmt.Errorf("service dependency: could not convert to Service")
	}

	context.Services[d.rawKey] = coerced
	return nil
}

// InContext checks if the dependency is contained in the given TemplateContext.
func (d *ServiceDependency) InContext(c *TemplateContext) bool {
	_, ok := c.Services[d.rawKey]
	return ok
}

func ServiceFunc(deps map[string]Dependency) func(...string) (interface{}, error) {
	return func(s ...string) (interface{}, error) {
		d, err := ParseServiceDependency(s...)
		if err != nil {
			return nil, err
		}

		if _, ok := deps[d.HashCode()]; !ok {
			deps[d.HashCode()] = d
		}

		return []*Service{}, nil
	}
}

// ParseServiceDependency processes the incoming strings to build a service dependency.
//
// Supported arguments
//   ParseServiceDependency("service_id")
//   ParseServiceDependency("service_id", "health_check")
//
// Where service_id is in the format of service(.tag(@datacenter(:port)))
// and health_check is either "any" or "passing".
//
// If no health_check is provided then its the same as "passing".
func ParseServiceDependency(s ...string) (*ServiceDependency, error) {
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
		return nil, errors.New("cannot specify empty service dependency")
	}

	re := regexp.MustCompile(`\A` +
		`((?P<tag>[[:word:]\-.]+)\.)?` +
		`((?P<name>[[:word:]\-/_]+))` +
		`(@(?P<datacenter>[[:word:]\.\-]+))?(:(?P<port>[0-9]+))?` +
		`\z`)
	names := re.SubexpNames()
	match := re.FindAllStringSubmatch(query, -1)

	if len(match) == 0 {
		return nil, errors.New("invalid service dependency format")
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

	sd := &ServiceDependency{
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

// ServiceList is a sortable slice of Service
type ServiceList []*Service

func (s ServiceList) Len() int {
	return len(s)
}

func (s ServiceList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ServiceList) Less(i, j int) bool {
	if s[i].Node < s[j].Node {
		return true
	} else if s[i].Node == s[j].Node {
		return s[i].ID <= s[j].ID
	}
	return false
}
