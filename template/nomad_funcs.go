package template

import (
	"errors"
	"fmt"
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

// nomadSecureVariableItemsFunc returns a given secure variable rooted at the
// items map.
func nomadSecureVariableItemsFunc(b *Brain, used, missing *dep.Set) func(string) (*dep.NomadSVItems, error) {
	return func(s string) (*dep.NomadSVItems, error) {
		if len(s) == 0 {
			return nil, nil
		}

		d, err := dep.NewSVGetQuery(s)
		if err != nil {
			return nil, err
		}
		d.EnableBlocking()

		used.Add(d)

		if value, ok := b.Recall(d); ok {
			if value == nil {
				return nil, nil
			}
			return value.(*dep.NomadSVItems), nil
		}

		missing.Add(d)

		return nil, nil
	}
}

// nomadSecureVariableExistsFunc returns true if a secure variable exists, false
// otherwise.
func nomadSecureVariableExistsFunc(b *Brain, used, missing *dep.Set) func(string) (bool, error) {
	return func(s string) (bool, error) {
		if len(s) == 0 {
			return false, nil
		}

		d, err := dep.NewSVGetQuery(s)
		if err != nil {
			return false, err
		}

		used.Add(d)

		if value, ok := b.Recall(d); ok {
			return value != nil, nil
		}

		missing.Add(d)

		return false, nil
	}
}

func nomadSafeSecureVariablesFunc(b *Brain, used, missing *dep.Set) func(...string) ([]*dep.NomadSVMeta, error) {
	// call nomadSecureVariablesFunc but explicitly mark that empty data set
	// returned on monitored secure variable prefix is NOT safe
	return nomadSecureVariablesFunc(b, used, missing, false)
}

// nomadSecureVariablesFunc returns or accumulates nomad secure variable prefix
// list dependencies.
func nomadSecureVariablesFunc(b *Brain, used, missing *dep.Set, emptyIsSafe bool) func(...string) ([]*dep.NomadSVMeta, error) {
	return func(args ...string) ([]*dep.NomadSVMeta, error) {
		if len(args) > 1 {
			return nil, fmt.Errorf("nomadVarList takes either a single \"prefix\" parameter or none for all available variables; got: %v", args)
		}

		result := []*dep.NomadSVMeta{}
		s := ""

		if len(args) == 1 {
			s = args[0]
		}

		d, err := dep.NewSVListQuery(s)
		if err != nil {
			return result, err
		}

		used.Add(d)

		// Only return non-empty top-level keys
		value, ok := b.Recall(d)
		if ok {
			result = append(result, value.([]*dep.NomadSVMeta)...)

			if len(result) == 0 {
				if emptyIsSafe {
					// Operator used potentially unsafe nomadSecureVariables
					// function in the template instead of the safe version.
					return result, nil
				}
			} else {
				// non empty result is good so we just return the data
				return result, nil
			}

			// If we reach this part of the code result is completely empty as
			// value returned no secure variables. Operator selected to use
			// safeSecureVariables on the specific secure variable prefix so we
			// will refuse to render template by marking d as missing
		}

		// b.Recall either returned an error or safeSecureVariables entered unsafe case
		missing.Add(d)

		return result, nil
	}
}
