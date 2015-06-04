package main

import (
	"encoding/json"
	"fmt"
	"net"
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

// lookupIPFunc returns or accumulates lookupIP dependencies.
func lookupIPFunc(brain *Brain, used, missing map[string]dep.Dependency,
	parseFunc func(s string) (*dep.LookupIP, error)) func(string) ([]net.IP, error) {
	return func(s string) ([]net.IP, error) {
		result := make([]net.IP, 0)
		if len(s) == 0 {
			return result, nil
		}

		d, err := parseFunc(s)
		if err != nil {
			return result, err
		}

		addDependency(used, d)

		if value, ok := brain.Recall(d); ok {
			return value.([]net.IP), nil
		}

		addDependency(missing, d)

		return result, nil
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

// vaultFunc returns or accumulates secret dependencies.
func vaultFunc(brain *Brain,
	used, missing map[string]dep.Dependency) func(string) (*dep.Secret, error) {
	return func(s string) (*dep.Secret, error) {
		result := &dep.Secret{}

		if len(s) == 0 {
			return result, nil
		}

		d, err := dep.ParseVaultSecret(s)
		if err != nil {
			return result, nil
		}

		addDependency(used, d)

		if value, ok := brain.Recall(d); ok {
			result = value.(*dep.Secret)
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
func byKey(pairs []*dep.KeyPair) (map[string]map[string]*dep.KeyPair, error) {
	m := make(map[string]map[string]*dep.KeyPair)
	for _, pair := range pairs {
		parts := strings.Split(pair.Key, "/")
		top := parts[0]
		key := strings.Join(parts[1:], "/")

		if key == "" {
			// Do not add a key if it has no prefix after stripping.
			continue
		}

		if _, ok := m[top]; !ok {
			m[top] = make(map[string]*dep.KeyPair)
		}

		newPair := *pair
		newPair.Key = key
		m[top][key] = &newPair
	}

	return m, nil
}

// byTag is a template func that takes the provided services and
// produces a map based on Service tags.
//
// The map key is a string representing the service tag. The map value is a
// slice of Services which have the tag assigned.
func byTag(in interface{}) (map[string][]interface{}, error) {
	m := make(map[string][]interface{})

	switch typed := in.(type) {
	case nil:
	case []*dep.CatalogService:
		for _, s := range typed {
			for _, t := range s.Tags {
				m[t] = append(m[t], s)
			}
		}
	case []*dep.HealthService:
		for _, s := range typed {
			for _, t := range s.Tags {
				m[t] = append(m[t], s)
			}
		}
	default:
		return nil, fmt.Errorf("byTag: wrong argument type %T", in)
	}

	return m, nil
}

// env returns the value of the environment variable set
func env(s string) (string, error) {
	return os.Getenv(s), nil
}

// loop accepts varying parameters and differs its behavior. If given one
// parameter, loop will return a goroutine that begins at 0 and loops until the
// given int, increasing the index by 1 each iteration. If given two parameters,
// loop will return a goroutine that begins at the first parameter and loops
// up to but not including the second parameter.
//
//    // Prints 0 1 2 3 4
// 		for _, i := range loop(5) {
// 			print(i)
// 		}
//
//    // Prints 5 6 7
// 		for _, i := range loop(5, 8) {
// 			print(i)
// 		}
//
func loop(ints ...int) (<-chan int, error) {
	var start, stop int
	switch len(ints) {
	case 1:
		start, stop = 0, ints[0]
	case 2:
		start, stop = ints[0], ints[1]
	default:
		return nil, fmt.Errorf("loop: wrong number of arguments, expected 1 or 2"+
			", but got %d", len(ints))
	}

	ch := make(chan int)

	go func() {
		for i := start; i < stop; i++ {
			ch <- i
		}
		close(ch)
	}()

	return ch, nil
}

// join is a version of strings.Join that can be piped
func join(sep string, a []string) (string, error) {
	return strings.Join(a, sep), nil
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

// regexMatch returns true or false if the string matches
// the given regular expression
func regexMatch(re, s string) (bool, error) {
	compiled, err := regexp.Compile(re)
	if err != nil {
		return false, err
	}
	return compiled.MatchString(s), nil
}

// split is a version of strings.Split that can be piped
func split(sep, s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}, nil
	}
	return strings.Split(s, sep), nil
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
		return "", fmt.Errorf("timestamp: wrong number of arguments, expected 0 or 1"+
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
