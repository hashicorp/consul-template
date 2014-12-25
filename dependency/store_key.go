package dependency

import (
	"errors"
	"fmt"
	"log"
	"regexp"

	api "github.com/armon/consul-api"
)

// from inside a template.
type StoreKey struct {
	rawKey     string
	Path       string
	DataCenter string
}

// Fetch queries the Consul API defined by the given client and returns string
// of the value to Path.
func (d *StoreKey) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	if d.DataCenter != "" {
		options.Datacenter = d.DataCenter
	}

	log.Printf("[DEBUG] (%s) querying consul with %+v", d.Display(), options)

	store := client.KV()
	pair, qm, err := store.Get(d.Path, options)
	if err != nil {
		return "", qm, err
	}

	if pair == nil {
		log.Printf("[DEBUG] (%s) Consul returned nothing (does the path exist?)",
			d.Display())
		return "", qm, nil
	}

	log.Printf("[DEBUG] (%s) Consul returned %s", d.Display(), pair.Value)

	return string(pair.Value), qm, nil
}

func (d *StoreKey) HashCode() string {
	return fmt.Sprintf("StoreKey|%s", d.Key())
}

func (d *StoreKey) Key() string {
	return d.rawKey
}

func (d *StoreKey) Display() string {
	return fmt.Sprintf(`key "%s"`, d.rawKey)
}

// ParseStoreKey parses a string of the format a(/b(/c...))
func ParseStoreKey(s string) (*StoreKey, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty key dependency")
	}

	re := regexp.MustCompile(`\A` +
		`(?P<key>[[:word:]\.\:\-\/]+)` +
		`(@(?P<datacenter>[[:word:]\.\-]+))?` +
		`\z`)
	names := re.SubexpNames()
	match := re.FindAllStringSubmatch(s, -1)

	if len(match) == 0 {
		return nil, errors.New("invalid key dependency format")
	}

	r := match[0]

	m := map[string]string{}
	for i, n := range r {
		if names[i] != "" {
			m[names[i]] = n
		}
	}

	key, datacenter := m["key"], m["datacenter"]

	if key == "" {
		return nil, errors.New("key part is required")
	}

	kd := &StoreKey{
		rawKey:     s,
		Path:       key,
		DataCenter: datacenter,
	}

	return kd, nil
}
