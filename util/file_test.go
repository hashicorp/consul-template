package util

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
)

func TestFileFetch(t *testing.T) {
	data := `{"foo":"bar"}`
	inTemplate := test.CreateTempfile([]byte(data), t)
	defer test.DeleteTempfile(inTemplate, t)

	dep := &File{
		rawKey: inTemplate.Name(),
	}

	read, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if read != data {
		t.Fatalf("expected %q to be %q", read, data)
	}
}

func TestFileFetch_waits(t *testing.T) {
	data := `{"foo":"bar"}`
	inTemplate := test.CreateTempfile([]byte(data), t)
	defer test.DeleteTempfile(inTemplate, t)

	dep := &File{
		rawKey: inTemplate.Name(),
	}

	_, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan interface{})
	go func(c chan<- interface{}) {
		data, _, err := dep.Fetch(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		c <- data
	}(dataCh)

	select {
	case <-dataCh:
		t.Fatal("received data, but should not have")
	case <-time.After(1000 * time.Nanosecond):
		return
	}
}

func TestFileFetch_firesChanges(t *testing.T) {
	data := `{"foo":"bar"}`
	inTemplate := test.CreateTempfile([]byte(data), t)
	defer test.DeleteTempfile(inTemplate, t)

	dep := &File{
		rawKey: inTemplate.Name(),
	}

	_, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan interface{})
	go func(c chan<- interface{}) {
		data, _, err := dep.Fetch(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		c <- data
	}(dataCh)

	newData := `{"bar": "baz"}`
	ioutil.WriteFile(inTemplate.Name(), []byte(newData), 0644)

	select {
	case d := <-dataCh:
		if d != newData {
			t.Fatalf("expected %q to be %q", d, newData)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("did not receive data from file changes")
	}
}
