// TODO:
//   - Re-add retry functionality

package util

import (
	"fmt"
	"log"
	"sync"
	"time"

	api "github.com/armon/consul-api"
)

const (
	// The amount of time to do a blocking query for
	defaultWaitTime = 60 * time.Second

	// The amount of time to wait when Consul returns an error
	defaultRetry = 5 * time.Second
)

// RetryFunc is a function that defines the retry for a given watcher. The
// function parameter is the current retry (which might be nil), and the
// return value is the new retry. In this way, you can build complex retry
// functions that are based off the previous values.
type RetryFunc func(time.Duration) time.Duration

type Watcher struct {
	sync.Mutex

	// DataCh is the chan where Views will be published
	DataCh chan *View

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

	// currentRetry is the current value of the retry for the Watcher.
	//
	// retryFunc is a RetryFunc that represents the way retrys and backoffs
	// should occur.
	currentRetry time.Duration
	retryFunc    RetryFunc

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

// SetRetry is used to set the retry to a static value.
func (w *Watcher) SetRetry(duration time.Duration) {
	w.SetRetryFunc(func(current time.Duration) time.Duration {
		return duration
	})
}

// SetRetryFunc is used to set a dynamic retry function.
func (w *Watcher) SetRetryFunc(f RetryFunc) {
	w.Lock()
	defer w.Unlock()
	w.retryFunc = f
}

//
func (w *Watcher) Watch(once bool) {
	log.Printf("[DEBUG] (watcher) starting watch")

	// In once mode, we want to immediately close the stopCh. This tells the
	// underlying View objects to terminate after they get data for the first
	// time.
	if once {
		log.Printf("[DEBUG] (watcher) detected once mode")
		w.Stop()
	}

	views := make([]*View, 0, len(w.dependencies))
	for _, dep := range w.dependencies {
		view, err := NewView(w.client, dep)
		if err != nil {
			w.ErrCh <- err
			return
		}

		views = append(views, view)
	}

	for _, v := range views {
		w.waitGroup.Add(1)
		go func(v *View) {
			defer w.waitGroup.Done()
			v.poll(w.DataCh, w.ErrCh, w.stopCh)
		}(v)
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
	w.DataCh = make(chan *View)
	w.ErrCh = make(chan error)
	w.FinishCh = make(chan struct{})
	w.stopCh = make(chan struct{})

	// Setup the default retry
	w.SetRetry(defaultRetry)

	return nil
}
