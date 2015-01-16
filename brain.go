package main

import (
	"fmt"
	"log"
	"sync"

	dep "github.com/hashicorp/consul-template/dependency"
)

// Brain is what Template uses to determine the values that are
// available for template parsing.
type Brain struct {
	sync.Mutex

	catalogNodes     map[string][]*dep.Node
	catalogServices  map[string][]*dep.CatalogService
	datacenters      map[string][]string
	files            map[string]string
	healthServices   map[string][]*dep.HealthService
	storeKeys        map[string]string
	storeKeyPrefixes map[string][]*dep.KeyPair

	// receivedData is an internal tracker of which dependencies have stored data
	// in the brain
	receivedData map[string]struct{}
}

// NewBrain creates a new Brain with empty values for each
// of the key structs.
func NewBrain() *Brain {
	return &Brain{
		catalogNodes:     make(map[string][]*dep.Node),
		catalogServices:  make(map[string][]*dep.CatalogService),
		datacenters:      make(map[string][]string),
		files:            make(map[string]string),
		healthServices:   make(map[string][]*dep.HealthService),
		storeKeys:        make(map[string]string),
		storeKeyPrefixes: make(map[string][]*dep.KeyPair),
		receivedData:     make(map[string]struct{}),
	}
}

// Remember accepts a dependency and the data to store associated with that
// dep. This function converts the given data to a proper type and stores
// it interally.
func (b *Brain) Remember(d dep.Dependency, data interface{}) {
	log.Printf("[INFO] (brain) remembering %s", d.Display())

	b.Lock()
	defer b.Unlock()

	switch t := d.(type) {
	case *dep.CatalogNodes:
		b.catalogNodes[d.HashCode()] = data.([]*dep.Node)
	case *dep.CatalogServices:
		b.catalogServices[d.HashCode()] = data.([]*dep.CatalogService)
	case *dep.Datacenters:
		b.datacenters[d.HashCode()] = data.([]string)
	case *dep.File:
		b.files[d.HashCode()] = data.(string)
	case *dep.HealthServices:
		b.healthServices[d.HashCode()] = data.([]*dep.HealthService)
	case *dep.StoreKey:
		b.storeKeys[d.HashCode()] = data.(string)
	case *dep.StoreKeyPrefix:
		b.storeKeyPrefixes[d.HashCode()] = data.([]*dep.KeyPair)
	default:
		panic(fmt.Sprintf("brain: unknown dependency type %T", t))
	}

	b.receivedData[d.HashCode()] = struct{}{}
}

// Remembered returns true if the given dependency has received data at least once.
func (b *Brain) Remembered(d dep.Dependency) bool {
	log.Printf("[INFO] (brain) checking if %s has data", d.Display())

	b.Lock()
	defer b.Unlock()

	if _, ok := b.receivedData[d.HashCode()]; ok {
		log.Printf("[DEBUG] (brain) %s had data", d.Display())
		return true
	}

	log.Printf("[DEBUG] (brain) %s did not have data", d.Display())
	return false
}

// Forget accepts a dependency and removes all associated data with this
// dependency. It also resets the "receivedData" internal map.
func (b *Brain) Forget(d dep.Dependency) {
	log.Printf("[INFO] (brain) forgetting %s", d.Display())

	b.Lock()
	defer b.Unlock()

	switch t := d.(type) {
	case *dep.CatalogNodes:
		delete(b.catalogNodes, d.HashCode())
	case *dep.CatalogServices:
		delete(b.catalogServices, d.HashCode())
	case *dep.Datacenters:
		delete(b.datacenters, d.HashCode())
	case *dep.File:
		delete(b.files, d.HashCode())
	case *dep.HealthServices:
		delete(b.healthServices, d.HashCode())
	case *dep.StoreKey:
		delete(b.storeKeys, d.HashCode())
	case *dep.StoreKeyPrefix:
		delete(b.storeKeyPrefixes, d.HashCode())
	default:
		panic(fmt.Sprintf("brain: unknown dependency type %T", t))
	}

	delete(b.receivedData, d.HashCode())
}
