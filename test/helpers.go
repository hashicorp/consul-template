package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
)

type FakeDependencyStale struct {
	Name string
}

func (d *FakeDependencyStale) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	time.Sleep(10 * time.Millisecond)

	if options.AllowStale {
		data := "this is some stale data"
		qm := &api.QueryMeta{LastIndex: 1, LastContact: 50 * time.Millisecond}
		return data, qm, nil
	} else {
		data := "this is some fresh data"
		qm := &api.QueryMeta{LastIndex: 1}
		return data, qm, nil
	}
}

func (d *FakeDependencyStale) HashCode() string {
	return fmt.Sprintf("FakeDependencyStale|%s", d.Name)
}

func (d *FakeDependencyStale) Display() string {
	return "fakedep"
}

type FakeDependencyFetchError struct {
	Name string
}

func (d *FakeDependencyFetchError) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	time.Sleep(10 * time.Millisecond)
	return nil, nil, fmt.Errorf("failed to contact server")
}

func (d *FakeDependencyFetchError) HashCode() string {
	return fmt.Sprintf("FakeDependencyFetchError|%s", d.Name)
}

func (d *FakeDependencyFetchError) Display() string {
	return "fakedep"
}

type FakeDependencyFetchRetry struct {
	sync.Mutex
	Name    string
	retried bool
}

func (d *FakeDependencyFetchRetry) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	d.Lock()
	defer d.Unlock()

	time.Sleep(10 * time.Millisecond)

	if d.retried {
		data := "this is some data"
		qm := &api.QueryMeta{LastIndex: 1}
		return data, qm, nil
	} else {
		d.retried = true
		return nil, nil, fmt.Errorf("failed to contact server (try again)")
	}
}

func (d *FakeDependencyFetchRetry) HashCode() string {
	return fmt.Sprintf("FakeDependencyFetchRetry|%s", d.Name)
}

func (d *FakeDependencyFetchRetry) Display() string {
	return "fakedep"
}

type FakeDependency struct {
	Name string
}

func (d *FakeDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	time.Sleep(10 * time.Millisecond)
	data := "this is some data"
	qm := &api.QueryMeta{LastIndex: 1}
	return data, qm, nil
}

func (d *FakeDependency) HashCode() string {
	return fmt.Sprintf("FakeDependency|%s", d.Name)
}

func (d *FakeDependency) Display() string {
	return "fakedep"
}

func DemoConsulClient(t *testing.T) (*api.Client, *api.QueryOptions) {
	config := api.DefaultConfig()
	config.Address = "demo.consul.io"

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Agent().NodeName(); err != nil {
		t.Fatal(err)
	}

	options := &api.QueryOptions{WaitTime: 10 * time.Second}

	return client, options
}

func CreateTempfile(b []byte, t *testing.T) *os.File {
	f, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	if len(b) > 0 {
		_, err = f.Write(b)
		if err != nil {
			t.Fatal(err)
		}
	}

	return f
}

func DeleteTempfile(f *os.File, t *testing.T) {
	if err := os.Remove(f.Name()); err != nil {
		t.Fatal(err)
	}
}

func WaitForFileContents(path string, contents []byte, t *testing.T) {
	readCh := make(chan struct{})

	go func(ch chan struct{}, path string, contents []byte) {
		for {
			data, err := ioutil.ReadFile(path)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
				return
			}

			if bytes.Equal(data, contents) {
				close(readCh)
				return
			}

			time.Sleep(50 * time.Millisecond)
		}
	}(readCh, path, contents)

	select {
	case <-readCh:
	case <-time.After(2 * time.Second):
		t.Fatal("file contents not present after 2 seconds")
	}
}
