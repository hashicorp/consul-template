package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	api "github.com/armon/consul-api"
)

/// ------------------------- ///

type FakeDependencyFetchError struct {
	name string
}

func (d *FakeDependencyFetchError) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	return nil, nil, fmt.Errorf("failed to contact server")
}

func (d *FakeDependencyFetchError) GoString() string {
	return fmt.Sprintf("%#v", d)
}

func (d *FakeDependencyFetchError) HashCode() string {
	return fmt.Sprintf("FakeDependencyFetchError|%s", d.name)
}

func (d *FakeDependencyFetchError) Key() string {
	return d.name
}

/// ------------------------- ///

type FakeDependency struct {
	name string
}

func (d *FakeDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	data := "this is some data"
	qm := &api.QueryMeta{LastIndex: 1}
	return data, qm, nil
}

func (d *FakeDependency) GoString() string {
	return fmt.Sprintf("%#v", d)
}

func (d *FakeDependency) HashCode() string {
	return fmt.Sprintf("FakeDependency|%s", d.name)
}

func (d *FakeDependency) Key() string {
	return d.name
}

/// ------------------------- ///
///          Helpers          ///
/// ------------------------- ///

func createTempfile(b []byte, t *testing.T) *os.File {
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

func deleteTempfile(f *os.File, t *testing.T) {
	if err := os.Remove(f.Name()); err != nil {
		t.Fatal(err)
	}
}

func demoConsulClient(t *testing.T) (*api.Client, *api.QueryOptions) {
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
