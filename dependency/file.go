package dependency

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

type File struct {
	mutex    sync.RWMutex
	rawKey   string
	lastStat os.FileInfo
}

func (d *File) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
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

func (d *File) HashCode() string {
	return fmt.Sprintf("StoreKeyPrefix|%s", d.Key())
}

func (d *File) Key() string {
	return d.rawKey
}

func (d *File) Display() string {
	return fmt.Sprintf(`file "%s"`, d.rawKey)
}

// watch watchers the file for changes
func (d *File) watch() (os.FileInfo, error) {
	for {
		stat, err := os.Stat(d.rawKey)
		if err != nil {
			return nil, err
		}

		changed := func(d *File, stat os.FileInfo) bool {
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

func ParseFile(s string) (*File, error) {
	if len(s) == 0 {
		return nil, errors.New("cannot specify empty file dependency")
	}

	kd := &File{
		rawKey: s,
	}

	return kd, nil
}
