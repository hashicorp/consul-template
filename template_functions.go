package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
)

// now is function that represents the current time in UTC. This is here
// primarily for the tests to override times.
var now = func() time.Time { return time.Now().UTC() }

// datacentersFunc returns or accumulates datacenter dependencies.
func datacentersFunc(brain *Brain,
	used, missing map[string]dep.Dependency) func(...string) ([]string, error) {
	return func(s ...string) ([]string, error) {
		result := make([]string, 0)

		d, err := dep.ParseDatacenters(s...)
		if err != nil {
			return result, err
		}

		addDependency(used, d)

		if value, ok := brain.Recall(d); ok {
			return value.([]string), nil
		}

		addDependency(missing, d)

		return result, nil
	}
}

// fileFunc returns or accumulates file dependencies.
func fileFunc(brain *Brain,
	used, missing map[string]dep.Dependency) func(string) (string, error) {
	return func(s string) (string, error) {
		if len(s) == 0 {
			return "", nil
		}

		d, err := dep.ParseFile(s)
		if err != nil {
			return "", err
		}

		addDependency(used, d)

		if value, ok := brain.Recall(d); ok {
			if value == nil {
				return "", nil
			} else {
				return value.(string), nil
			}
		}

		addDependency(missing, d)

		return "", nil
	}
}

// keyFunc returns or accumulates key dependencies.
func keyFunc(brain *Brain,
	used, missing map[string]dep.Dependency) func(string) (string, error) {
	return func(s string) (string, error) {
		if len(s) == 0 {
			return "", nil
		}

		d, err := dep.ParseStoreKey(s)
		if err != nil {
			return "", err
		}

		addDependency(used, d)

		if value, ok := brain.Recall(d); ok {
			if value == nil {
				return "", nil
			} else {
				return value.(string), nil
			}
		}

		addDependency(missing, d)

		return "", nil
	}
}

// lsFunc returns or accumulates keyPrefix dependencies.
func lsFunc(brain *Brain,
	used, missing map[string]dep.Dependency) func(string) ([]*dep.KeyPair, error) {
	return func(s string) ([]*dep.KeyPair, error) {
		result := make([]*dep.KeyPair, 0)

		if len(s) == 0 {
			return result, nil
		}

		d, err := dep.ParseStoreKeyPrefix(s)
		if err != nil {
			return result, err
		}

		addDependency(used, d)

		// Only return non-empty top-level keys
		if value, ok := brain.Recall(d); ok {
			for _, pair := range value.([]*dep.KeyPair) {
				if pair.Key != "" && !strings.Contains(pair.Key, "/") {
					result = append(result, pair)
				}
			}
			return result, nil
		}

		addDependency(missing, d)

		return result, nil
	}
}

// nodesFunc returns or accumulates catalog node dependencies.
func nodesFunc(brain *Brain,
	used, missing map[string]dep.Dependency) func(...string) ([]*dep.Node, error) {
	return func(s ...string) ([]*dep.Node, error) {
		result := make([]*dep.Node, 0)

		d, err := dep.ParseCatalogNodes(s...)
		if err != nil {
			return nil, err
		}

		addDependency(used, d)

		if value, ok := brain.Recall(d); ok {
			return value.([]*dep.Node), nil
		}

		addDependency(missing, d)

		return result, nil
	}
}

// serviceFunc returns or accumulates health service dependencies.
func serviceFunc(brain *Brain,
	used, missing map[string]dep.Dependency) func(...string) ([]*dep.HealthService, error) {
	return func(s ...string) ([]*dep.HealthService, error) {
		result := make([]*dep.HealthService, 0)

		if len(s) == 0 || s[0] == "" {
			return result, nil
		}

		d, err := dep.ParseHealthServices(s...)
		if err != nil {
			return nil, err
		}

		addDependency(used, d)

		if value, ok := brain.Recall(d); ok {
			return value.([]*dep.HealthService), nil
		}

		addDependency(missing, d)

		return result, nil
	}
}

// servicesFunc returns or accumulates catalog services dependencies.
func servicesFunc(brain *Brain,
	used, missing map[string]dep.Dependency) func(...string) ([]*dep.CatalogService, error) {
	return func(s ...string) ([]*dep.CatalogService, error) {
		result := make([]*dep.CatalogService, 0)

		d, err := dep.ParseCatalogServices(s...)
		if err != nil {
			return nil, err
		}

		addDependency(used, d)

		if value, ok := brain.Recall(d); ok {
			return value.([]*dep.CatalogService), nil
		}

		addDependency(missing, d)

		return result, nil
	}
}

// treeFunc returns or accumulates keyPrefix dependencies.
func treeFunc(brain *Brain,
	used, missing map[string]dep.Dependency) func(string) ([]*dep.KeyPair, error) {
	return func(s string) ([]*dep.KeyPair, error) {
		result := make([]*dep.KeyPair, 0)

		if len(s) == 0 {
			return result, nil
		}

		d, err := dep.ParseStoreKeyPrefix(s)
		if err != nil {
			return result, err
		}

		addDependency(used, d)

		// Only return non-empty top-level keys
		if value, ok := brain.Recall(d); ok {
			for _, pair := range value.([]*dep.KeyPair) {
				parts := strings.Split(pair.Key, "/")
				if parts[len(parts)-1] != "" {
					result = append(result, pair)
				}
			}
			return result, nil
		}

		addDependency(missing, d)

		return result, nil
	}

}

// byKey accepts a slice of KV pairs and returns a map of the top-level
// key to all its subkeys. For example:
//
//		elasticsearch/a //=> "1"
//		elasticsearch/b //=> "2"
//		redis/a/b //=> "3"
//
// Passing the result from Consul through byTag would yield:
//
// 		map[string]map[string]string{
//	  	"elasticsearch": &dep.KeyPair{"a": "1"}, &dep.KeyPair{"b": "2"},
//			"redis": &dep.KeyPair{"a/b": "3"}
//		}
//
// Note that the top-most key is stripped from the Key value. Keys that have no
// prefix after stripping are removed from the list.
func byKey(pairs []*dep.KeyPair) (map[string][]*dep.KeyPair, error) {
	m := make(map[string][]*dep.KeyPair)
	for _, pair := range pairs {
		parts := strings.Split(pair.Key, "/")
		top := parts[0]
		key := strings.Join(parts[1:], "/")

		if key == "" {
			// Do not add a key if it has no prefix after stripping.
			continue
		}

		if _, ok := m[top]; !ok {
			m[top] = make([]*dep.KeyPair, 0, 1)
		}
		pair.Key = key
		m[top] = append(m[top], pair)
	}
	return m, nil
}

// byTag is a template func that takes the provided services and
// produces a map based on Service tags.
//
// The map key is a string representing the service tag. The map value is a
// slice of Services which have the tag assigned.
func byTag(in []*dep.HealthService) (map[string][]*dep.HealthService, error) {
	m := make(map[string][]*dep.HealthService)
	for _, s := range in {
		for _, t := range s.Tags {
			m[t] = append(m[t], s)
		}
	}
	return m, nil
}

// returns the value of the environment variable set
func env(s string) (string, error) {
	return os.Getenv(s), nil
}

// parseJSON returns a structure for valid JSON
func parseJSON(s string) (interface{}, error) {
	if s == "" {
		return make([]interface{}, 0), nil
	}

	var data interface{}
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// replaceAll replaces all occurrences of a value in a string with the given
// replacement value.
func replaceAll(f, t, s string) (string, error) {
	return strings.Replace(s, f, t, -1), nil
}

// regexReplaceAll replaces all occurrences of a regular expression with
// the given replacement value.
func regexReplaceAll(re, pl, s string) (string, error) {
	compiled, err := regexp.Compile(re)
	if err != nil {
		return "", err
	}
	return compiled.ReplaceAllString(s, pl), nil
}

// timestamp returns the current UNIX timestamp in UTC. If an argument is
// specified, it will be used to format the timestamp.
func timestamp(s ...string) (string, error) {
	switch len(s) {
	case 0:
		return now().Format(time.RFC3339), nil
	case 1:
		return now().Format(s[0]), nil
	default:
		return "", fmt.Errorf("timestamp: too many arguments, expected 0 or 1"+
			", but got %d", len(s))
	}
}

// toLower converts the given string (usually by a pipe) to lowercase.
func toLower(s string) (string, error) {
	return strings.ToLower(s), nil
}

// toTitle converts the given string (usually by a pipe) to titlecase.
func toTitle(s string) (string, error) {
	return strings.Title(s), nil
}

// toUpper converts the given string (usually by a pipe) to uppercase.
func toUpper(s string) (string, error) {
	return strings.ToUpper(s), nil
}

// addDependency adds the given Dependency to the map.
func addDependency(m map[string]dep.Dependency, d dep.Dependency) {
	if _, ok := m[d.HashCode()]; !ok {
		m[d.HashCode()] = d
	}
}
