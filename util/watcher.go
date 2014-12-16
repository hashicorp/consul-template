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

	// The amount of time to wait when Consul returns an error
	defaultTimeout = 5 * time.Second
)

// TimeoutFunc is a function that defines the timeout for a given watcher. The
// function parameter is the current timeout (which might be nil), and the
// return value is the new timeout. In this way, you can build complex timeout
// functions that are based off the previous values.
type TimeoutFunc func(time.Duration) time.Duration

type Watcher struct {
	sync.Mutex

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

	// currentTimeout is the current value of the timeout for the Watcher.
	//
	// timeoutFunc is a TimeoutFunc that represents the way timeouts and backoffs
	// should occur.
	currentTimeout time.Duration
	timeoutFunc    TimeoutFunc

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

// SetTimeout is used to set the timeout to a static value.
func (w *Watcher) SetTimeout(duration time.Duration) {
	w.SetTimeoutFunc(func(current time.Duration) time.Duration {
		return duration
	})
}

// SetTimeoutFunc is used to set a dynamic timeout function.
func (w *Watcher) SetTimeoutFunc(f TimeoutFunc) {
	w.Lock()
	defer w.Unlock()
	w.timeoutFunc = f
}

//
func (w *Watcher) Watch(once bool) {
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
		view, err := NewWatchData(dependency)
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

	// Setup the default timeout
	w.SetTimeout(defaultTimeout)

	return nil
}

/// ------------------------- ///

type WatchData struct {
	Dependency Dependency
	Data       interface{}

	receivedData bool
	lastIndex    uint64
}

//
func NewWatchData(dependency Dependency) (*WatchData, error) {
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

			w.Lock()
			w.currentTimeout = w.timeoutFunc(w.currentTimeout)
			w.Unlock()

			time.Sleep(w.currentTimeout)
			continue
		}

		// If the query metadata is nil, return an error instead of panicing. See
		// (GH-72) for more information. This does not actually "fix" the issue,
		// which appears to be a bug in armon/consul-api, but will at least give a
		// nicer error message to the user and help us better trace this issue.
		if qm == nil {
			err := fmt.Errorf("consul returned nil qm; this should never happen" +
				"and is probably a bug in consul-template or consulapi")
			log.Printf("[ERR] (%s) %s", wd.Display(), err)
			w.ErrCh <- err
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
