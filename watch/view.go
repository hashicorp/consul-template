package watch

import (
	"fmt"
	"log"
	"reflect"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul/api"
)

const (
	// The amount of time to do a blocking query for
	defaultWaitTime = 60 * time.Second
)

// View is a representation of a Dependency and the most recent data it has
// received from Consul.
type View struct {
	// Dependency is the dependency that is associated with this View
	Dependency dep.Dependency

	// config is the configuration for the watcher that created this view and
	// contains important information about how this view should behave when
	// polling including retry functions and handling stale queries.
	config *WatcherConfig

	// Data is the most-recently-received data from Consul for this View
	Data         interface{}
	ReceivedData bool
	LastIndex    uint64

	// stopCh is used to stop polling on this View
	stopCh chan struct{}
}

// NewView creates a new view object from the given Consul API client and
// Dependency. If an error occurs, it will be returned.
func NewView(config *WatcherConfig, d dep.Dependency) (*View, error) {
	if config == nil {
		return nil, fmt.Errorf("view: missing config")
	}

	if d == nil {
		return nil, fmt.Errorf("view: missing dependency")
	}

	return &View{
		Dependency: d,
		config:     config,
		stopCh:     make(chan struct{}),
	}, nil
}

// poll queries the Consul instance for data using the fetch function, but also
// accounts for interrupts on the interrupt channel. This allows the poll
// function to be fired in a goroutine, but then halted even if the fetch
// function is in the middle of a blocking query.
func (v *View) poll(viewCh chan<- *View, errCh chan<- error) {
	defaultRetry := v.config.RetryFunc(1 * time.Second)
	currentRetry := defaultRetry

	for {
		doneCh, fetchErrCh := make(chan struct{}, 1), make(chan error, 1)
		go v.fetch(doneCh, fetchErrCh)

		select {
		case <-doneCh:
			// Reset the retry to avoid exponentially incrementing retries when we
			// have some successful requests
			currentRetry = defaultRetry

			log.Printf("[INFO] (view) %s received data from consul", v.display())
			select {
			case <-v.stopCh:
			case viewCh <- v:
			}

			// If we are operating in once mode, do not loop - we received data at
			// least once which is the API promise here.
			if v.config.Once {
				return
			}
		case err := <-fetchErrCh:
			log.Printf("[ERR] (view) %s %s", v.display(), err)

			// Push the error back up to the watcher
			select {
			case <-v.stopCh:
			case errCh <- err:
			}

			// Sleep and retry
			if v.config.RetryFunc != nil {
				currentRetry = v.config.RetryFunc(currentRetry)
			}
			log.Printf("[INFO] (view) %s errored, retrying in %s", v.display(), currentRetry)
			time.Sleep(currentRetry)
			continue
		case <-v.stopCh:
			log.Printf("[DEBUG] (view) %s stopping poll (received on view stopCh)", v.display())
			return
		}
	}
}

// fetch queries the Consul instance for the attached dependency. This API
// promises that either data will be written to doneCh or an error will be
// written to errCh. It is designed to be run in a goroutine that selects the
// result of doneCh and errCh. It is assumed that only one instance of fetch
// is running per View and therefore no locking or mutexes are used.
func (v *View) fetch(doneCh chan<- struct{}, errCh chan<- error) {
	log.Printf("[DEBUG] (view) %s starting fetch", v.display())

	var allowStale bool
	if v.config.MaxStale != 0 {
		allowStale = true
	}

	for {
		options := &api.QueryOptions{
			AllowStale: allowStale,
			WaitTime:   defaultWaitTime,
			WaitIndex:  v.LastIndex,
		}
		data, qm, err := v.Dependency.Fetch(v.config.Client, options)
		if err != nil {
			errCh <- err
			return
		}

		if qm == nil {
			errCh <- fmt.Errorf("consul returned nil qm; this should never happen" +
				"and is probably a bug in consul-template or consul/api")
			return
		}

		if allowStale && qm.LastContact > v.config.MaxStale {
			allowStale = false
			log.Printf("[DEBUG] (view) %s stale data (last contact exceeded max_stale)", v.display())
			continue
		}

		if v.config.MaxStale != 0 {
			allowStale = true
		}

		if qm.LastIndex == v.LastIndex {
			log.Printf("[DEBUG] (view) %s no new data (index was the same)", v.display())
			continue
		}

		if qm.LastIndex < v.LastIndex {
			log.Printf("[DEBUG] (view) %s had a lower index, resetting", v.display())
			v.LastIndex = 0
			continue
		}

		v.LastIndex = qm.LastIndex

		if v.ReceivedData && reflect.DeepEqual(data, v.Data) {
			log.Printf("[DEBUG] (view) %s no new data (contents were the same)", v.display())
			continue
		}

		v.Data = data
		v.ReceivedData = true
		close(doneCh)
		return
	}
}

// display returns a string that represents this view.
func (v *View) display() string {
	return v.Dependency.Display()
}

// stop halts polling of this view.
func (v *View) stop() {
	close(v.stopCh)
}
