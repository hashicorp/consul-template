package test

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/consul/sdk/testutil"
)

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

func WaitForContents(t *testing.T, d time.Duration, p, c string) {
	errCh := make(chan error, 1)
	matchCh := make(chan struct{}, 1)
	stopCh := make(chan struct{}, 1)
	var last string

	go func() {
		for {
			select {
			case <-stopCh:
				return
			default:
			}

			actual, err := ioutil.ReadFile(p)
			if err != nil && !os.IsNotExist(err) {
				errCh <- err
				return
			}

			last = string(actual)
			if strings.EqualFold(last, c) {
				close(matchCh)
				return
			}

			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-matchCh:
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(d):
		close(stopCh)
		t.Errorf("contents not present after %s, expected: %q, actual: %q",
			d, c, last)
	}
}

// Meets consul/sdk/testutil/TestingTB interface
var _ testutil.TestingTB = (*TestingTB)(nil)

type TestingTB struct {
	cleanup func()
	sync.Mutex
}

func (t *TestingTB) DoCleanup() {
	t.Lock()
	defer t.Unlock()
	t.cleanup()
}

func (*TestingTB) Failed() bool                { return false }
func (*TestingTB) Logf(string, ...interface{}) {}
func (*TestingTB) Name() string                { return "TestingTB" }
func (t *TestingTB) Cleanup(f func()) {
	t.Lock()
	defer t.Unlock()
	prev := t.cleanup
	t.cleanup = func() {
		f()
		if prev != nil {
			prev()
		}
	}
}
