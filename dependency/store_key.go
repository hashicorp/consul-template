package dependency

import (
	"errors"
	"fmt"
	"log"
	"regexp"
)

// from inside a template.
type StoreKey struct {
	rawKey     string
	Path       string
	DataCenter string
}

// Fetch queries the Consul API defined by the given client and returns string
// of the value to Path.
func (d *StoreKey) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}

	consulOpts := opts.consulQueryOptions()
	if d.DataCenter != "" {
		consulOpts.Datacenter = d.DataCenter
	}

	log.Printf("[DEBUG] (%s) querying consul with %+v", d.Display(), consulOpts)

	consul, err := clients.Consul()
	if err != nil {
		return nil, nil, fmt.Errorf("store key: error getting client: %s", err)
	}

	store := consul.KV()
	pair, qm, err := store.Get(d.Path, consulOpts)
	if err != nil {
		return "", nil, fmt.Errorf("store key: error fetching: %s", err)
	}

	rm := &ResponseMetadata{
		LastIndex:   qm.LastIndex,
		LastContact: qm.LastContact,
	}

	if pair == nil {
		log.Printf("[WARN] (%s) Consul returned no data (does the path exist?)",
			d.Display())
		return "", rm, nil
	}

	log.Printf("[DEBUG] (%s) Consul returned %s", d.Display(), pair.Value)

	return string(pair.Value), rm, nil
}

func (d *StoreKey) HashCode() string {
	return fmt.Sprintf("StoreKey|%s", d.rawKey)
}

func (d *StoreKey) Display() string {
	return fmt.Sprintf(`"key(%s)"`, d.rawKey)
}

// ParseStoreKey parses a string of the format a(/b(/c...))
func ParseStoreKey(s string) (*StoreKey, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty key dependency")
	}

	re := regexp.MustCompile(`\A` +
		`(?P<key>[^@]+)` +
		`(@(?P<datacenter>.+))?` +
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
