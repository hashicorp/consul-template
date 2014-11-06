package util

import (
	"fmt"
	"log"
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

	// FinishCh is the chan where the watcher reports it is "done"
	FinishCh chan struct{}

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
func (w *Watcher) Watch(once bool, retry *Retry) {
	log.Printf("[DEBUG] (watcher) starting watch")

	// In once mode, we want to immediately close the stopCh. This tells the
	// underlying WatchData objects to terminate after they get data for the first
	// time.
	if once {
		log.Printf("[DEBUG] (watcher) detected once mode")
		w.Stop()
	}

	views := make([]*WatchData, 0, len(w.dependencies))
	for _, dependency := range w.dependencies {
		view, err := NewWatchData(dependency, *retry)
		if err != nil {
			w.ErrCh <- err
			return
		}

		views = append(views, view)
	}

	for _, view := range views {
		w.waitGroup.Add(1)
		go func(view *WatchData) {
			defer w.waitGroup.Done()
			view.poll(w)
		}(view)
	}

	log.Printf("[DEBUG] (watcher) all pollers have started, waiting for finish")
	w.waitGroup.Wait()

	if once {
		log.Printf("[DEBUG] (watcher) closing finish channel")
		close(w.FinishCh)
	}
}

//
func (w *Watcher) Stop() {
	close(w.stopCh)
}

//
func (w *Watcher) init() error {
	if w.client == nil {
		return fmt.Errorf("watcher: missing Consul API client")
	}

	if len(w.dependencies) == 0 {
		log.Printf("[WARN] (watcher) no dependencies in template(s)")
	}

	// Setup the chans
	w.DataCh = make(chan *WatchData)
	w.ErrCh = make(chan error)
	w.FinishCh = make(chan struct{})
	w.stopCh = make(chan struct{})

	return nil
}

/// ------------------------- ///

type WatchData struct {
	Dependency Dependency
	Data       interface{}

	receivedData bool
	lastIndex    uint64

	retry Retry
}

//
func NewWatchData(dependency Dependency, retry Retry) (*WatchData, error) {
	if dependency == nil {
		return nil, fmt.Errorf("watchdata: missing Dependency")
	}

	return &WatchData{Dependency: dependency}, nil
}

//
func (wd *WatchData) poll(w *Watcher) {
	for {
		log.Printf("[DEBUG] (%s) starting poll", wd.Display())

		options := &api.QueryOptions{
			WaitTime:  defaultWaitTime,
			WaitIndex: wd.lastIndex,
		}
		data, qm, err := wd.Dependency.Fetch(w.client, options)
		if err != nil {
			log.Printf("[ERR] (%s) %s", wd.Display(), err.Error())
			w.ErrCh <- err
			time.Sleep(wd.retry.Tick())
			continue
		}

		// Consul is allowed to return even if there's no new data. Ignore data if
		// the index is the same. For files, the data is fake, index is always 0
		if qm.LastIndex == wd.lastIndex {
			log.Printf("[DEBUG] (%s) no new data (index was the same)", wd.Display())
			continue
		}

		// Update the index in case we got a new version, but the data is the same
		wd.lastIndex = qm.LastIndex

		// Do not trigger a render if we have gotten data and the data is the same
		if wd.receivedData && reflect.DeepEqual(data, wd.Data) {
			log.Printf("[DEBUG] (%s) no new data (contents were the same)", wd.Display())
			continue
		}

		log.Printf("[DEBUG] (%s) writing data to channel", wd.Display())

		// If we got this far, there is new data!
		wd.Data = data
		wd.receivedData = true
		w.DataCh <- wd

		// Break from the function if we are done
		select {
		case <-w.stopCh:
			log.Printf("[DEBUG] (%s) stopping poll (received on stopCh)", wd.Display())
			return
		default:
			continue
		}
	}
}

//
func (wd *WatchData) Display() string {
	return wd.Dependency.Display()
}
