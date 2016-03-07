package dependency

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

	dep, err := ParseFile(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	read, _, err := dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if read != data {
		t.Fatalf("expected %q to be %q", read, data)
	}
}

func TestFileFetch_stopped(t *testing.T) {
	data := `{"foo":"bar"}`
	inTemplate := test.CreateTempfile([]byte(data), t)
	defer test.DeleteTempfile(inTemplate, t)

	dep, err := ParseFile(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error)
	go func() {
		results, _, err := dep.Fetch(nil, &QueryOptions{WaitIndex: 100})
		if results != nil {
			t.Fatalf("should not get results: %#v", results)
		}
		errCh <- err
	}()

	dep.Stop()

	select {
	case err := <-errCh:
		if err != ErrStopped {
			t.Errorf("expected %q to be %q", err, ErrStopped)
		}
	case <-time.After(50 * time.Millisecond):
		t.Errorf("did not return in 50ms")
	}
}

func TestFileFetch_waits(t *testing.T) {
	data := `{"foo":"bar"}`
	inTemplate := test.CreateTempfile([]byte(data), t)
	defer test.DeleteTempfile(inTemplate, t)

	dep, err := ParseFile(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	doneCh := make(chan struct{})
	errCh := make(chan error)
	go func() {
		if _, _, err := dep.Fetch(nil, nil); err != nil {
			errCh <- err
			return
		}
		close(doneCh)
	}()

	select {
	case err := <-errCh:
		t.Fatal(err)
	case <-doneCh:
		t.Fatal("received data, but should not have")
	case <-time.After(1000 * time.Nanosecond):
		return
	}
}

func TestFileFetch_firesChanges(t *testing.T) {
	data := `{"foo":"bar"}`
	inTemplate := test.CreateTempfile([]byte(data), t)
	defer test.DeleteTempfile(inTemplate, t)

	dep, err := ParseFile(inTemplate.Name())
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = dep.Fetch(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	dataCh := make(chan interface{})
	errCh := make(chan error)
	go func() {
		data, _, err := dep.Fetch(nil, nil)
		if err != nil {
			errCh <- err
			return
		}
		dataCh <- data
	}()

	newData := `{"bar": "baz"}`
	ioutil.WriteFile(inTemplate.Name(), []byte(newData), 0644)

	select {
	case d := <-dataCh:
		if d != newData {
			t.Fatalf("expected %q to be %q", d, newData)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive data from file changes")
	}
}
