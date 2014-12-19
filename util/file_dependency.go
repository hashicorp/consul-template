package util

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	api "github.com/armon/consul-api"
)

type FileDependency struct {
	mutex    sync.RWMutex
	rawKey   string
	lastStat os.FileInfo
}

func (d *FileDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	var err error = nil
	var data []byte

	log.Printf("[DEBUG] (%s) querying file", d.Display())

	newStat, err := d.watch()
	if err != nil {
		return "", nil, err
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.lastStat = newStat

	fakeMeta := &api.QueryMeta{LastIndex: uint64(newStat.ModTime().Unix())}

	if data, err = ioutil.ReadFile(d.rawKey); err == nil {
		return string(data), fakeMeta, err
	}
	return "", nil, err
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

// AddToContext accepts a TemplateContext and data. It coerces the interface{}
// data into the correct format via type assertions, returning an errors that
// occur. The data is then set on the TemplateContext.
func (d *FileDependency) AddToContext(context *TemplateContext, data interface{}) error {
	coerced, ok := data.(string)
	if !ok {
		return fmt.Errorf("file dependency: could not convert to string")
	}

	context.Files[d.rawKey] = coerced
	return nil
}

// InContext checks if the dependency is contained in the given TemplateContext.
func (d *FileDependency) InContext(c *TemplateContext) bool {
	_, ok := c.Files[d.rawKey]
	return ok
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

func FileDependencyFunc(deps map[string]Dependency) func(...string) (interface{}, error) {
	return func(s ...string) (interface{}, error) {
		if len(s) != 1 {
			return nil, fmt.Errorf("file: expected 1 argument, got %d", len(s))
		}

		d, err := ParseFileDependency(s[0])
		if err != nil {
			return nil, err
		}

		if _, ok := deps[d.HashCode()]; !ok {
			deps[d.HashCode()] = d
		}

		return "", nil
	}
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
