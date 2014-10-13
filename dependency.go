package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// Dependency is an interface which implements the Key() method.
type Dependency interface {
	Key() string
}

/// ------------------------- ///

// ServiceDependency is the representation of a requested service dependency
// from inside a template.
type ServiceDependency struct {
	RawKey     string
	Name       string
	Tag        string
	DataCenter string
	Port       uint64
}

func (s *ServiceDependency) Key() string {
	return s.RawKey
}

// GoString returns the detailed format of this object
func (s *ServiceDependency) GoString() string {
	return fmt.Sprintf("*%#v", *s)
}

// ParseServiceDependency parses a string of the format
// service(.tag(@datacenter(:port)))
func ParseServiceDependency(s string) (*ServiceDependency, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty service dependency")
	}

	// (tag.)service(@datacenter(:port))
	re := regexp.MustCompile(`\A` +
		`((?P<tag>[[:word:]\-]+)\.)?` +
		`((?P<name>[[:word:]\-]+))` +
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
		RawKey:     s,
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
	Path       string
	DataCenter string
}

func (kd *KeyDependency) Key() string {
	return kd.Path
}

// GoString returns the detailed format of this object
func (kd *KeyDependency) GoString() string {
	return fmt.Sprintf("*%#v", *kd)
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
		Path:       key,
		DataCenter: datacenter,
	}

	return kd, nil
}

/// ------------------------- ///

// KeyPrefixDependency is the representation of a requested key dependency
// from inside a template.
type KeyPrefixDependency struct {
	Prefix     string
	DataCenter string
}

func (kpd *KeyPrefixDependency) Key() string {
	return kpd.Prefix
}

// GoString returns the detailed format of this object
func (kpd *KeyPrefixDependency) GoString() string {
	return fmt.Sprintf("*%#v", *kpd)
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
		Prefix:     prefix,
		DataCenter: datacenter,
	}

	return kpd, nil
}
