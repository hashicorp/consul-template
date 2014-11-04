package util

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	api "github.com/armon/consul-api"
)

// Dependency is an interface
type Dependency interface {
	Fetch(*api.Client, *api.QueryOptions) (interface{}, *api.QueryMeta, error)
	HashCode() string
	Key() string
	Display() string
}

/// ------------------------- ///

// ServiceDependency is the representation of a requested service dependency
// from inside a template.
type ServiceDependency struct {
	rawKey     string
	Name       string
	Tag        string
	DataCenter string
	Port       uint64
}

// Fetch queries the Consul API defined by the given client and returns a slice
// of Service objects.
func (d *ServiceDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	if d.DataCenter != "" {
		options.Datacenter = d.DataCenter
	}

	log.Printf("[DEBUG] (%s) querying Consul with %+v", d.Display(), options)

	health := client.Health()
	entries, qm, err := health.Service(d.Name, d.Tag, true, options)
	if err != nil {
		return nil, qm, err
	}

	log.Printf("[DEBUG] (%s) Consul returned %d services", d.Display(), len(entries))

	services := make([]*Service, 0, len(entries))

	for _, entry := range entries {
		services = append(services, &Service{
			Node:    entry.Node.Node,
			Address: entry.Node.Address,
			ID:      entry.Service.ID,
			Name:    entry.Service.Service,
			Tags:    entry.Service.Tags,
			Port:    uint64(entry.Service.Port),
		})
	}

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

// ParseServiceDependency parses a string of the format
// service(.tag(@datacenter(:port)))
func ParseServiceDependency(s string) (*ServiceDependency, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty service dependency")
	}

	// (tag.)service(@datacenter(:port))
	re := regexp.MustCompile(`\A` +
		`((?P<tag>[[:word:]\-.]+)\.)?` +
		`((?P<name>[[:word:]\-/_]+))` +
		`(@(?P<datacenter>[[:word:]\.\-]+))?(:(?P<port>[0-9]+))?` +
		`\z`)
	names := re.SubexpNames()
	match := re.FindAllStringSubmatch(s, -1)

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
		rawKey:     s,
		Name:       name,
		Tag:        tag,
		DataCenter: datacenter,
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

/// ------------------------- ///

// KeyDependency is the representation of a requested key dependency
// from inside a template.
type KeyDependency struct {
	rawKey     string
	Path       string
	DataCenter string
}

// Fetch queries the Consul API defined by the given client and returns string
// of the value to Path.
func (d *KeyDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
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

func (d *KeyDependency) HashCode() string {
	return fmt.Sprintf("KeyDependency|%s", d.Key())
}

func (d *KeyDependency) Key() string {
	return d.rawKey
}

func (d *KeyDependency) Display() string {
	return fmt.Sprintf(`key "%s"`, d.rawKey)
}

// ParseKeyDependency parses a string of the format a(/b(/c...))
func ParseKeyDependency(s string) (*KeyDependency, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty key dependency")
	}

	// a(/b(/c))(@datacenter)
	re := regexp.MustCompile(`\A` +
		`(?P<key>[[:word:]\.\-\/]+)` +
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

	kd := &KeyDependency{
		rawKey:     s,
		Path:       key,
		DataCenter: datacenter,
	}

	return kd, nil
}

func ParseFileDependency(s string) (*FileDependency, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty file dependency")
	}

	kd := &FileDependency{
		rawKey: s,
	}

	return kd, nil
}

/// ------------------------- ///

type FileDependency struct {
	mutex    sync.RWMutex
	rawKey   string
	lastStat os.FileInfo
}

func (d *FileDependency) HashCode() string {
	return fmt.Sprintf("KeyPrefixDependency|%s", d.Key())
}

func (d *FileDependency) Key() string {
	return d.rawKey
}

func (d *FileDependency) Display() string {
	return fmt.Sprintf(`file "%s"`, d.rawKey)
}

func (d *FileDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	var err error = nil
	var data []byte

	log.Printf("[DEBUG] (%s) querying file", d.Display())

	// Block until we get a change
	newStat, err := d.watch()
	if err != nil {
		return "", nil, err
	}

	// Lock updating the stat in case another process is also fetching
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.lastStat = newStat

	// fake metadata for calling function
	fakeMeta := &api.QueryMeta{LastIndex: uint64(newStat.ModTime().Unix())}

	if data, err = ioutil.ReadFile(d.rawKey); err == nil {
		return string(data), fakeMeta, err
	}
	return "", nil, err
}

// watch watchers the file for changes
func (d *FileDependency) watch() (os.FileInfo, error) {
	for {
		stat, err := os.Stat(d.rawKey)
		if err != nil {
			return nil, err
		}

		changed := func(d *FileDependency, stat os.FileInfo) bool {
			d.mutex.RLock()
			defer d.mutex.RUnlock()

			if d.lastStat == nil {
				return true
			}
			if d.lastStat.Size() != stat.Size() {
				return true
			}

			if d.lastStat.ModTime() != stat.ModTime() {
				return true
			}

			return false
		}(d, stat)

		if changed {
			return stat, nil
		} else {
			time.Sleep(3 * time.Second)
		}
	}
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

// ParseKeyDependency parses a string of the format a(/b(/c...))
func ParseKeyPrefixDependency(s string) (*KeyPrefixDependency, error) {
	// a(/b(/c))(@datacenter)
	re := regexp.MustCompile(`\A` +
		`(?P<prefix>[[:word:]\.\-\/]+)?` +
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

// Service is a service entry in Consul
type Service struct {
	Node    string
	Address string
	ID      string
	Name    string
	Tags    []string
	Port    uint64
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

// KeyPair is a simple Key-Value pair
type KeyPair struct {
	Path  string
	Key   string
	Value string
}
