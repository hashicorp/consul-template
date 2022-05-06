package template

import (
	"errors"
	"strings"

	dep "github.com/hashicorp/consul-template/dependency"
)

// nomadServicesFunc returns or accumulates a list of service registration
// stubs from Nomad.
func nomadServicesFunc(b *Brain, used, missing *dep.Set) func(...string) ([]*dep.NomadServicesSnippet, error) {
	return func(s ...string) ([]*dep.NomadServicesSnippet, error) {
		var result []*dep.NomadServicesSnippet

		d, err := dep.NewNomadServicesQuery(strings.Join(s, ""))
		if err != nil {
			return nil, err
		}

		used.Add(d)

		if value, ok := b.Recall(d); ok {
			return value.([]*dep.NomadServicesSnippet), nil
		}

		missing.Add(d)

		return result, nil
	}
}

// nomadServiceFunc returns or accumulates a list of service registrations from
// Nomad matching an individual name.
func nomadServiceFunc(b *Brain, used, missing *dep.Set) func(...interface{}) ([]*dep.NomadService, error) {
	return func(params ...interface{}) ([]*dep.NomadService, error) {
		var query *dep.NomadServiceQuery
		var err error

		switch len(params) {
		case 1:
			service, ok := params[0].(string)
			if !ok {
				return nil, errors.New("expected string for <service> argument")
			}
			if query, err = dep.NewNomadServiceQuery(service); err != nil {
				return nil, err
			}
		case 3:
			count, ok := params[0].(int)
			if !ok {
				return nil, errors.New("expected integer for <count> argument")
			}
			hash, ok := params[1].(string)
			if !ok {
				return nil, errors.New("expected string for <key> argument")
			}
			service, ok := params[2].(string)
			if !ok {
				return nil, errors.New("expected string for <service> argument")
			}
			if query, err = dep.NewNomadServiceChooseQuery(count, hash, service); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("expected arguments <service> or <count> <key> <service>")
		}

		used.Add(query)
		if value, ok := b.Recall(query); ok {
			return value.([]*dep.NomadService), nil
		}
		missing.Add(query)

		return nil, nil
	}
}
