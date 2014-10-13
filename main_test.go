package main

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	api "github.com/armon/consul-api"
)

/*
 * Helpers
 */
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
