package util

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	api "github.com/armon/consul-api"
)

// KeyPair is a simple Key-Value pair
type KeyPair struct {
	Path  string
	Key   string
	Value string
}

// KeyPrefixDependency is the representation of a requested key dependency
// from inside a template.
type KeyPrefixDependency struct {
	rawKey     string
	Prefix     string
	DataCenter string
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of KeyPair objects
func (d *KeyPrefixDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	if d.DataCenter != "" {
		options.Datacenter = d.DataCenter
	}

	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), options)

	store := client.KV()
	prefixes, qm, err := store.List(d.Prefix, options)
	if err != nil {
		return err, qm, nil
	}

	log.Printf("[DEBUG] (%s) Consul returned %d key pairs", d.Display(), len(prefixes))

	keyPairs := make([]*KeyPair, 0, len(prefixes))
	for _, pair := range prefixes {
		key := strings.TrimPrefix(pair.Key, d.Prefix)
		key = strings.TrimLeft(key, "/")

		keyPairs = append(keyPairs, &KeyPair{
			Path:  pair.Key,
			Key:   key,
			Value: string(pair.Value),
		})
	}

	return keyPairs, qm, nil
}

func (d *KeyPrefixDependency) HashCode() string {
	return fmt.Sprintf("KeyPrefixDependency|%s", d.Key())
}

func (d *KeyPrefixDependency) Key() string {
	return d.rawKey
}

func (d *KeyPrefixDependency) Display() string {
	return fmt.Sprintf(`keyPrefix "%s"`, d.rawKey)
}

// ParseKeyPrefixDependency parses a string of the format a(/b(/c...))
func ParseKeyPrefixDependency(s string) (*KeyPrefixDependency, error) {
	// a(/b(/c))(@datacenter)
	re := regexp.MustCompile(`\A` +
		`(?P<prefix>[[:word:]\.\:\-\/]+)?` +
		`(@(?P<datacenter>[[:word:]\.\-]+))?` +
		`\z`)
	names := re.SubexpNames()
	match := re.FindAllStringSubmatch(s, -1)

	if len(match) == 0 {
		return nil, errors.New("invalid key prefix dependency format")
	}

	r := match[0]

	m := map[string]string{}
	for i, n := range r {
		if names[i] != "" {
			m[names[i]] = n
		}
	}

	prefix, datacenter := m["prefix"], m["datacenter"]

	kpd := &KeyPrefixDependency{
		rawKey:     s,
		Prefix:     prefix,
		DataCenter: datacenter,
	}

	return kpd, nil
}
