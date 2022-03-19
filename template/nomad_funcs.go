package template

import (
	"strings"

	dep "github.com/hashicorp/consul-template/dependency"
)

// nomadServicesFunc returns or accumulates a list of service registration
// stubs from Nomad.
func nomadServicesFunc(b *Brain, used, missing *dep.Set) func(...string) ([]*dep.NomadServicesSnippet, error) {
	return func(s ...string) ([]*dep.NomadServicesSnippet, error) {
		result := []*dep.NomadServicesSnippet{}

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
func nomadServiceFunc(b *Brain, used, missing *dep.Set) func(...string) ([]*dep.NomadService, error) {
	return func(s ...string) ([]*dep.NomadService, error) {
		result := []*dep.NomadService{}

		d, err := dep.NewNomadServiceQuery(strings.Join(s, ""))
		if err != nil {
			return nil, err
		}

		used.Add(d)

		if value, ok := b.Recall(d); ok {
			return value.([]*dep.NomadService), nil
		}

		missing.Add(d)

		return result, nil
	}
}
