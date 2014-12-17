package util

import (
	"fmt"
	"log"
	"reflect"
	"time"

	api "github.com/armon/consul-api"
)

const (
	// The amount of time to do a blocking query for
	defaultWaitTime = 60 * time.Second

	// The amount of time to wait when Consul returns an error
	defaultRetry = 1 * time.Second
)

// View is a representation of a Dependency and the most recent data it has
// received from Consul.
type View struct {
	Dependency Dependency

	Data         interface{}
	receivedData bool
	lastIndex    uint64

	client *api.Client
}

// NewView creates a new view object from the given Consul API client and
// Dependency. If an error occurs, it will be returned.
func NewView(client *api.Client, dep Dependency) (*View, error) {
	if client == nil {
		return nil, fmt.Errorf("view: missing Consul API client")
	}

	if dep == nil {
		return nil, fmt.Errorf("view: missing Dependency")
	}

	return &View{
		client:     client,
		Dependency: dep,
	}, nil
}

// poll queries the Consul instance for data using the fetch function, but also
// accounts for interrupts on the interrupt channel. This allows the poll
// function to be fired in a goroutine, but then halted even if the fetch
// function is in the middle of a blocking query.
func (v *View) poll(once bool, viewCh chan<- *View,
	errCh chan<- error, stopCh <-chan struct{}, retryFunc RetryFunc) {
	currentRetry := defaultRetry
	doneCh, fetchErrCh := make(chan struct{}, 1), make(chan error, 1)

	for {
		go v.fetch(doneCh, fetchErrCh)

		select {
		case <-doneCh:
			// Reset the retry to avoid exponentially incrementing retries when we
			// have some successful requests
			currentRetry = defaultRetry

			log.Printf("[INFO] (%s) received data from consul", v.display())
			viewCh <- v

			// If we are operating in once mode, do not loop - we received data at
			// least once which is the API promise here.
			if once {
				return
			}
		case err := <-fetchErrCh:
			log.Printf("[ERR] (%s) %s", v.display(), err)
			errCh <- err

			// Sleep and retry
			currentRetry = retryFunc(currentRetry)
			time.Sleep(currentRetry)
			continue
		case <-stopCh:
			log.Printf("[DEBUG] (%s) stopping poll (received on stopCh)", v.display())
			return
		}
	}
}

// fetch queries the Consul instance for the attached dependency. This API
// promises that either data will be written to doneCh or an error will be
// written to errCh. It is designed to be run in a goroutine that selects the
// result of doneCh and errCh. It is assumed that only one instance of fetch
// is running per View and therefore no locking or mutexes are used.
func (v *View) fetch(doneCh chan struct{}, errCh chan<- error) {
	log.Printf("[DEBUG] (%s) starting fetch", v.display())

	for {
		options := &api.QueryOptions{
			WaitTime:  defaultWaitTime,
			WaitIndex: v.lastIndex,
		}
		data, qm, err := v.Dependency.Fetch(v.client, options)
		if err != nil {
			errCh <- err
			return
		}

		if qm == nil {
			errCh <- fmt.Errorf("consul returned nil qm; this should never happen" +
				"and is probably a bug in consul-template or consulapi")
			return
		}

		if qm.LastIndex == v.lastIndex {
			log.Printf("[DEBUG] (%s) no new data (index was the same)", v.display())
			continue
		}

		v.lastIndex = qm.LastIndex

		if v.receivedData && reflect.DeepEqual(data, v.Data) {
			log.Printf("[DEBUG] (%s) no new data (contents were the same)", v.display())
			continue
		}

		v.Data = data
		v.receivedData = true
		close(doneCh)
		return
	}
}

// display returns a string that represents this view.
func (v *View) display() string {
	return v.Dependency.Display()
}
