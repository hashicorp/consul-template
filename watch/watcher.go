package watch

import (
	"fmt"
	"log"
	"sync"
	"time"

	api "github.com/armon/consul-api"
	"github.com/hashicorp/consul-template/util"
)

// defaultRetryFunc is the default return function, which just echos whatever
// duration it was given.
var defaultRetryFunc RetryFunc = func(t time.Duration) time.Duration {
	return t
}

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
	dependencies []util.Dependency

	// retryFunc is a RetryFunc that represents the way retrys and backoffs
	// should occur.
	retryFunc RetryFunc

	// waitGroup is the WaitGroup to ensure all Go routines return when we stop
	waitGroup sync.WaitGroup
}

//
func NewWatcher(client *api.Client, dependencies []util.Dependency) (*Watcher, error) {
	watcher := &Watcher{
		client:       client,
		dependencies: dependencies,
	}
	if err := watcher.init(); err != nil {
		return nil, err
	}

	return watcher, nil
}

// SetRetry is used to set the retry to a static value. See SetRetryFunc for
// more informatoin.
func (w *Watcher) SetRetry(duration time.Duration) {
	w.SetRetryFunc(func(current time.Duration) time.Duration {
		return duration
	})
}

// SetRetryFunc is used to set a dynamic retry function. Only new views created
// after this function has been set will inherit the new retry functionality.
// Existing views will use the retry functionality with which they were created.
func (w *Watcher) SetRetryFunc(f RetryFunc) {
	w.Lock()
	defer w.Unlock()
	w.retryFunc = f
}

// Watch creates a series of Consul views which poll for data in parallel. If
// the `once` flag is true, each view will return data (or an error) exactly
// once and terminate. If the `once` flag is false, views will continue to poll
// indefinitely unless they encounter an irrecoverable error.
func (w *Watcher) Watch(once bool) {
	log.Printf("[DEBUG] (watcher) starting watch")

	// Create the views
	views := make([]*View, 0, len(w.dependencies))
	for _, dep := range w.dependencies {
		view, err := NewView(w.client, dep)
		if err != nil {
			w.ErrCh <- err
			return
		}

		views = append(views, view)
	}

	// Poll on all the views
	for _, v := range views {
		w.waitGroup.Add(1)
		go func(once bool, v *View) {
			defer w.waitGroup.Done()
			v.poll(once, w.DataCh, w.ErrCh, w.stopCh, w.retryFunc)
		}(once, v)
	}

	// Wait for them to stop
	log.Printf("[DEBUG] (watcher) all pollers have started, waiting for finish")
	w.waitGroup.Wait()

	// Close everything up
	if once {
		log.Printf("[DEBUG] (watcher) closing finish channel")
		close(w.FinishCh)
	}
}

// Stop halts this watcher and any currently polling views immediately. If a
// view was in the middle of a poll, no data will be returned.
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

	// Setup the default retry
	w.SetRetryFunc(defaultRetryFunc)

	// Setup the channels
	w.DataCh = make(chan *View)
	w.ErrCh = make(chan error)
	w.FinishCh = make(chan struct{})
	w.stopCh = make(chan struct{})

	return nil
}
