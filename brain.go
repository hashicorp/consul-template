package main

import (
	"log"
	"sync"

	dep "github.com/hashicorp/consul-template/dependency"
)

// Brain is what Template uses to determine the values that are
// available for template parsing.
type Brain struct {
	sync.Mutex

	// data is the map of individual dependencies (by HashCode()) and the most
	// recent data for that dependency.
	data map[string]interface{}

	// receivedData is an internal tracker of which dependencies have stored data
	// in the brain.
	receivedData map[string]struct{}
}

// NewBrain creates a new Brain with empty values for each
// of the key structs.
func NewBrain() *Brain {
	return &Brain{
		data:         make(map[string]interface{}),
		receivedData: make(map[string]struct{}),
	}
}

// Remember accepts a dependency and the data to store associated with that
// dep. This function converts the given data to a proper type and stores
// it interally.
func (b *Brain) Remember(d dep.Dependency, data interface{}) {
	b.Lock()
	defer b.Unlock()

	log.Printf("[INFO] (brain) remembering %s", d.Display())

	b.data[d.HashCode()] = data
	b.receivedData[d.HashCode()] = struct{}{}
}

// Recall gets the current value for the given dependency in the Brain.
func (b *Brain) Recall(d dep.Dependency) (interface{}, bool) {
	b.Lock()
	defer b.Unlock()

	log.Printf("[INFO] (brain) getting data for %s", d.Display())

	// If we have not received data for this dependency, return now.
	if _, ok := b.receivedData[d.HashCode()]; !ok {
		log.Printf("[DEBUG] (brain) %s did not have data", d.Display())
		return nil, false
	}

	log.Printf("[DEBUG] (brain) %s had data", d.Display())
	return b.data[d.HashCode()], true
}

// Forget accepts a dependency and removes all associated data with this
// dependency. It also resets the "receivedData" internal map.
func (b *Brain) Forget(d dep.Dependency) {
	b.Lock()
	defer b.Unlock()

	log.Printf("[INFO] (brain) forgetting %s", d.Display())

	delete(b.data, d.HashCode())
	delete(b.receivedData, d.HashCode())
}
