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

type fakeDependencyFetchError struct {
	name string
}

func (d *fakeDependencyFetchError) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	return nil, nil, fmt.Errorf("failed to contact server")
}

func (d *fakeDependencyFetchError) GoString() string {
	return fmt.Sprintf("%#v", d)
}

func (d *fakeDependencyFetchError) HashCode() string {
	return fmt.Sprintf("fakeDependencyFetchError|%s", d.name)
}

func (d *fakeDependencyFetchError) Key() string {
	return d.name
}

func (d *fakeDependencyFetchError) Display() string {
	return "fakedep"
}

/// ------------------------- ///

type fakeDependency struct {
	name string
}

func (d *fakeDependency) Fetch(client *api.Client, options *api.QueryOptions) (interface{}, *api.QueryMeta, error) {
	data := "this is some data"
	qm := &api.QueryMeta{LastIndex: 1}
	return data, qm, nil
}

func (d *fakeDependency) GoString() string {
	return fmt.Sprintf("%#v", d)
}

func (d *fakeDependency) HashCode() string {
	return fmt.Sprintf("fakeDependency|%s", d.name)
}

func (d *fakeDependency) Key() string {
	return d.name
}

func (d *fakeDependency) Display() string {
	return "fakedep"
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
