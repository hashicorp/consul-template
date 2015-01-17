package watch

import (
	"fmt"
	"log"
	"sync"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul/api"
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

	// once is used to determine if the views should poll for data exactly once
	once bool

	// DataCh is the chan where Views will be published
	DataCh chan *View

	// ErrCh is the chan where any errors will be published
	ErrCh chan error

	// FinishCh is the chan where the watcher reports it is "done"
	FinishCh chan struct{}

	// client is the mechanism for communicating with the Consul API
	client *api.Client

	// retryFunc is a RetryFunc that represents the way retrys and backoffs
	// should occur.
	retryFunc RetryFunc

	// depViewMap is a map of Templates to Views. Templates are keyed by
	// HashCode().
	depViewMap map[string]*View
}

// NewWatcher creates a new watcher using the given API client.
func NewWatcher(c *api.Client, once bool) (*Watcher, error) {
	watcher := &Watcher{
		client: c,
		once:   once,
	}
	if err := watcher.init(); err != nil {
		return nil, err
	}

	return watcher, nil
}

// Add adds the given dependency to the list of monitored depedencies
// and start the associated view. If the dependency already exists, no action is
// taken.
//
// If the Dependency already existed, it this function will return false. If the
// view was successfully created, it will return true. If an error occurs while
// creating the view, it will be returned here (but future errors returned by
// the view will happen on the channel).
func (w *Watcher) Add(d dep.Dependency) (bool, error) {
	w.Lock()
	defer w.Unlock()

	log.Printf("[INFO] (watcher) adding %s", d.Display())

	if _, ok := w.depViewMap[d.HashCode()]; ok {
		log.Printf("[DEBUG] (watcher) %s already exists, skipping", d.Display())
		return false, nil
	}

	v, err := NewView(w.client, d)
	if err != nil {
		return false, err
	}

	log.Printf("[DEBUG] (watcher) %s starting", d.Display())

	w.depViewMap[d.HashCode()] = v
	go v.poll(w.once, w.DataCh, w.ErrCh, w.retryFunc)

	return true, nil
}

// Watching determines if the given dependency is being watched.
func (w *Watcher) Watching(d dep.Dependency) bool {
	w.Lock()
	defer w.Unlock()

	_, ok := w.depViewMap[d.HashCode()]
	return ok
}

// Remove removes the given dependency from the list and stops the
// associated View. If a View for the given dependency does not exist, this
// function will return false. If the View does exist, this function will return
// true upon successful deletion.
func (w *Watcher) Remove(d dep.Dependency) bool {
	w.Lock()
	defer w.Unlock()

	log.Printf("[INFO] (watcher) removing %s", d.Display())

	if view, ok := w.depViewMap[d.HashCode()]; ok {
		log.Printf("[DEBUG] (watcher) actually removing %s", d.Display())
		view.stop()
		delete(w.depViewMap, d.HashCode())
		return true
	}

	log.Printf("[DEBUG] (watcher) %s did not exist, skipping", d.Display())
	return false
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

// Stop halts this watcher and any currently polling views immediately. If a
// view was in the middle of a poll, no data will be returned.
func (w *Watcher) Stop() {
	w.Lock()
	defer w.Unlock()

	log.Printf("[INFO] (watcher) stopping all views")

	for _, view := range w.depViewMap {
		log.Printf("[DEBUG] (watcher) stopping %+v", view)
		view.stop()
	}

	// Reset the map to have no views
	w.depViewMap = make(map[string]*View)
}

// init sets up the initial values for the watcher.
func (w *Watcher) init() error {
	if w.client == nil {
		return fmt.Errorf("watcher: missing Consul API client")
	}

	// Setup the channels
	w.DataCh = make(chan *View)
	w.ErrCh = make(chan error)
	w.FinishCh = make(chan struct{})

	// Setup the default retry
	w.SetRetryFunc(defaultRetryFunc)

	// Setup our map of dependencies to views
	w.depViewMap = make(map[string]*View)

	return nil
}
