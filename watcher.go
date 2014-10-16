package main

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	api "github.com/armon/consul-api"
)

const (
	// The amount of time to do a blocking query for
	defaultWaitTime = 60 * time.Second

	// pollErrorSleep the amount of time to sleep when an error occurs
	// TODO: make this an exponential backoff.
	pollErrorSleep = 5 * time.Second
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

	// waitGroup is the WaitGroup to ensure all Go routines return when we stop
	waitGroup sync.WaitGroup
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
func (w *Watcher) Watch() error {
	views := make([]*WatchData, 0, len(w.dependencies))
	for _, dependency := range w.dependencies {
		view, err := NewWatchData(dependency)
		if err != nil {
			return err
		}

		views = append(views, view)
	}

	for _, view := range views {
		go view.poll(w)
		w.waitGroup.Add(1)
	}

	return nil
}

//
func (w *Watcher) Stop() {
	close(w.stopCh)
	w.waitGroup.Wait()
}

//
func (w *Watcher) init() error {
	if w.client == nil {
		return fmt.Errorf("watcher: missing Consul API client")
	}

	if len(w.dependencies) == 0 {
		return fmt.Errorf("watcher: must supply at least one Dependency")
	}

	// Setup the chans
	w.DataCh = make(chan *WatchData)
	w.ErrCh = make(chan error)
	w.stopCh = make(chan struct{})

	return nil
}

/// ------------------------- ///

type WatchData struct {
	dependency   Dependency
	data         interface{}
	receivedData bool
	lastIndex    uint64
}

//
func NewWatchData(dependency Dependency) (*WatchData, error) {
	if dependency == nil {
		return nil, fmt.Errorf("watchdata: missing Dependency")
	}

	return &WatchData{dependency: dependency}, nil
}

//
func (wd *WatchData) poll(w *Watcher) {
	for {
		options := &api.QueryOptions{
			WaitTime:  defaultWaitTime,
			WaitIndex: wd.lastIndex,
		}
		data, qm, err := wd.dependency.Fetch(w.client, options)
		if err != nil {
			w.ErrCh <- err
			time.Sleep(pollErrorSleep)
			continue
		}

		// Consul is allowed to return even if there's no new data. Ignore data if
		// the index is the same.
		if qm.LastIndex == wd.lastIndex {
			continue
		}

		// Update the index in case we got a new version, but the data is the same
		wd.lastIndex = qm.LastIndex

		// Do not trigger a render if we have gotten data and the data is the same
		if wd.receivedData && reflect.DeepEqual(data, wd.data) {
			continue
		}

		// If we got this far, there is new data!
		wd.data = data
		wd.receivedData = true
		w.DataCh <- wd

		// Break from the function if we are done
		select {
		case <-w.stopCh:
			w.waitGroup.Done()
			return
		default:
			continue
		}
	}
}
