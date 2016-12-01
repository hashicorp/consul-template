package watch

import (
	"fmt"
	"sync"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
)

// TestDep is a special dependency that does not actually speaks to a server.
type TestDep struct {
	name string
}

func (d *TestDep) Fetch(clients *dep.ClientSet, opts *dep.QueryOptions) (interface{}, *dep.ResponseMetadata, error) {
	time.Sleep(10 * time.Millisecond)
	data := "this is some data"
	rm := &dep.ResponseMetadata{LastIndex: 1}
	return data, rm, nil
}

func (d *TestDep) CanShare() bool {
	return true
}

func (d *TestDep) String() string {
	return fmt.Sprintf("test_dep(%s)", d.name)
}

func (d *TestDep) Stop() {}

// TestDepStale is a special dependency that can be used to test what happens when
// stale data is permitted.
type TestDepStale struct {
	name string
}

// Fetch is used to implement the dependency interface.
func (d *TestDepStale) Fetch(clients *dep.ClientSet, opts *dep.QueryOptions) (interface{}, *dep.ResponseMetadata, error) {
	time.Sleep(10 * time.Millisecond)

	if opts == nil {
		opts = &dep.QueryOptions{}
	}

	if opts.AllowStale {
		data := "this is some stale data"
		rm := &dep.ResponseMetadata{LastIndex: 1, LastContact: 50 * time.Millisecond}
		return data, rm, nil
	} else {
		data := "this is some fresh data"
		rm := &dep.ResponseMetadata{LastIndex: 1}
		return data, rm, nil
	}
}

func (d *TestDepStale) CanShare() bool {
	return true
}

func (d *TestDepStale) String() string {
	return fmt.Sprintf("test_dep_stale(%s)", d.name)
}

func (d *TestDepStale) Stop() {}

// TestDepFetchError is a special dependency that returns an error while fetching.
type TestDepFetchError struct {
	name string
}

func (d *TestDepFetchError) Fetch(clients *dep.ClientSet, opts *dep.QueryOptions) (interface{}, *dep.ResponseMetadata, error) {
	time.Sleep(10 * time.Millisecond)
	return nil, nil, fmt.Errorf("failed to contact server")
}

func (d *TestDepFetchError) CanShare() bool {
	return true
}

func (d *TestDepFetchError) String() string {
	return fmt.Sprintf("test_dep_fetch_error(%s)", d.name)
}

func (d *TestDepFetchError) Stop() {}

// TestDepRetry is a special dependency that errors on the first fetch and
// succeeds on subsequent fetches.
type TestDepRetry struct {
	sync.Mutex
	name    string
	retried bool
}

func (d *TestDepRetry) Fetch(clients *dep.ClientSet, opts *dep.QueryOptions) (interface{}, *dep.ResponseMetadata, error) {
	time.Sleep(10 * time.Millisecond)

	d.Lock()
	defer d.Unlock()

	if d.retried {
		data := "this is some data"
		rm := &dep.ResponseMetadata{LastIndex: 1}
		return data, rm, nil
	} else {
		d.retried = true
		return nil, nil, fmt.Errorf("failed to contact server (try again)")
	}
}

func (d *TestDepRetry) CanShare() bool {
	return true
}

func (d *TestDepRetry) String() string {
	return fmt.Sprintf("test_dep_retry(%s)", d.name)
}

func (d *TestDepRetry) Stop() {}
