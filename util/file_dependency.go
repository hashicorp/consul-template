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

func ParseFileDependency(s string) (*FileDependency, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty file dependency")
	}

	kd := &FileDependency{
		rawKey: s,
	}

	return kd, nil
}
