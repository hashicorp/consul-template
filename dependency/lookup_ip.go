package dependency

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"
)

type LookupIP struct {
	spec   string
	lookup func(string) ([]net.IP, error)
	mutex  sync.RWMutex
	rawKey string
	lastIP []net.IP
}

func (d *LookupIP) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	var err error = nil

	log.Printf("[DEBUG] (%s) querying %s", d.Display(), d.spec)

	newIp, err := d.watch()
	if err != nil {
		return "", nil, fmt.Errorf("%s: error watching: %s", d.spec, err)
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.lastIP = newIp

	ts := time.Now().Unix()
	rm := &ResponseMetadata{
		LastContact: time.Duration(ts),
		LastIndex:   uint64(ts),
	}

	return d.lastIP, rm, nil
}

func (d *LookupIP) HashCode() string {
	return fmt.Sprintf("%s|%s", d.spec, d.rawKey)
}

func (d *LookupIP) Display() string {
	return fmt.Sprintf(`"%s(%s)"`, d.spec, d.rawKey)
}

// watch watchers the LookupIP for changes
func (d *LookupIP) watch() ([]net.IP, error) {
	for {
		ip, err := d.lookup(d.rawKey)
		if err != nil {
			return nil, err
		}

		sort.Sort(byIP(ip))

		changed := func(d *LookupIP, ip []net.IP) bool {
			d.mutex.RLock()
			defer d.mutex.RUnlock()
			if len(d.lastIP) != len(ip) {
				return true
			}
			for i := range d.lastIP {
				if !bytes.Equal(d.lastIP[i], ip[i]) {
					return true
				}
			}
			return false
		}(d, ip)

		if changed {
			return ip, nil
		} else {
			time.Sleep(time.Minute)
		}
	}
}

// byIp for sorting ip addresses
type byIP []net.IP

func (b byIP) Len() int           { return len(b) }
func (b byIP) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byIP) Less(i, j int) bool { return bytes.Compare(b[i], b[j]) == -1 }

func onlyIPv4(in []net.IP, err error) ([]net.IP, error) {
	var out []net.IP
	if err != nil {
		return out, err
	}
	for _, ip := range in {
		if ip.To4() != nil {
			out = append(out, ip)
		}
	}
	return out, err
}

func onlyIPv6(in []net.IP, err error) ([]net.IP, error) {
	var out []net.IP
	if err != nil {
		return out, err
	}
	for _, ip := range in {
		if ip.To4() == nil {
			out = append(out, ip)
		}
	}
	return out, err
}

// ParseLookupIP create a new DNS lookup dependency that returns both IPv4 and
// IPv6 addresses for a hostname.
func ParseLookupIP(s string) (*LookupIP, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty LookupIP dependency")
	}

	kd := &LookupIP{
		spec:   "LookupIP",
		rawKey: s,
		lookup: net.LookupIP,
	}

	return kd, nil
}

// ParseLookupIPv4 create a new DNS lookup dependency that returns IPv4
// addresses for a hostname.
func ParseLookupIPv4(s string) (*LookupIP, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty LookupIPv4 dependency")
	}

	kd := &LookupIP{
		spec:   "LookupIPv4",
		rawKey: s,
		lookup: func(s string) ([]net.IP, error) {
			return onlyIPv4(net.LookupIP(s))
		},
	}

	return kd, nil
}

// ParseLookupIPv6 create a new DNS lookup dependency that returns IPv6
// addresses for a hostname.
func ParseLookupIPv6(s string) (*LookupIP, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty LookupIPv6 dependency")
	}

	kd := &LookupIP{
		spec:   "LookupIPv6",
		rawKey: s,
		lookup: func(s string) ([]net.IP, error) {
			return onlyIPv6(net.LookupIP(s))
		},
	}

	return kd, nil
}
