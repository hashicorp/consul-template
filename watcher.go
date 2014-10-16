package main

import (
	"errors"
	"reflect"
	"time"

	api "github.com/armon/consul-api"
)

const (
	// The amount of time to do a blocking query for
	defaultWaitTime = 60 * time.Second
)

type Watcher struct {
	// DataCh is the chan where new WatchData will be published
	DataCh chan *WatchData

	// ErrCh is the chan where any errors will be published
	ErrCh chan error

	// stopCh is a chan that is only published when polling should stop
	stopCh chan struct{}

	// client is the mechanism for communicating with the Consul API
	client *api.Client

	// dependencies is the slice of Dependencies this Watcher will poll
	dependencies []Dependency
}

//
func NewWatcher(client *api.Client, dependencies []Dependency) (*Watcher, error) {
	watcher := &Watcher{
		client:       client,
		dependencies: dependencies,
	}
	if err := watcher.init(); err != nil {
		return nil, err
	}

	return watcher, nil
}

//
func (w *Watcher) Watch() {
	for _, dependency := range w.dependencies {
		go func() {
			view, err := NewWatchData(dependency)
			if err != nil {
				w.ErrCh <- err
				return
			}

			view.poll(w.client, w.DataCh, w.ErrCh, w.stopCh)
		}()
	}
}

//
func (w *Watcher) Stop() {
	close(w.stopCh)
	// TODO: wait for routines to finish?
}

//
func (w *Watcher) init() error {
	if w.client == nil {
		return errors.New("watcher: missing Consul API client")
	}

	if len(w.dependencies) == 0 {
		return errors.New("watcher: must supply at least one Dependency")
	}

	// Setup the chans
	w.DataCh = make(chan *WatchData)
	w.ErrCh = make(chan error)
	w.stopCh = make(chan struct{})

	return nil
}

/// ------------------------- ///

type WatchData struct {
	dependency Dependency
	data       interface{}
	lastIndex  uint64
}

//
func NewWatchData(dependency Dependency) (*WatchData, error) {
	if dependency == nil {
		return nil, errors.New("watchdata: missing Dependency")
	}

	return &WatchData{dependency: dependency}, nil
}

//
func (wd *WatchData) poll(client *api.Client, dataCh chan *WatchData, errCh chan error, stopCh chan struct{}) {
	for {
		options := &api.QueryOptions{
			WaitTime:  defaultWaitTime,
			WaitIndex: wd.lastIndex,
		}
		data, qm, err := wd.dependency.Fetch(client, options)
		if err != nil {
			errCh <- err
			continue // TODO: should we continue or return?
		}

		// Consul is allowed to return even if there's no new data. Ignore data if
		// the index is the same.
		if qm.LastIndex == wd.lastIndex {
			continue
		}

		// Update the index in case we got a new version, but the data is the same
		wd.lastIndex = qm.LastIndex

		// Do not trigger a render if the data is the same
		if reflect.DeepEqual(data, wd.data) {
			continue
		}

		// If we got this far, there is new data!
		wd.data = data
		dataCh <- wd

		// Break from the function if we are done - this happens at the end of the
		// function to ensure it runs at least once
		select {
		case <-stopCh:
			return
		default:
			continue
		}
	}
}

// loaded determines if the data view has already received data from Consul at
// least once
func (v *WatchData) loaded() bool {
	return v.lastIndex != 0
}
