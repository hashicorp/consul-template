package test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"
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

func WaitForFileContents(path string, expected []byte, t *testing.T) {
	readCh := make(chan struct{})
	var last []byte

	go func(ch chan struct{}, path string, expected []byte) {
		for {
			actual, err := ioutil.ReadFile(path)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
				return
			}

			last = actual
			if bytes.Equal(actual, expected) {
				close(readCh)
				return
			}

			time.Sleep(50 * time.Millisecond)
		}
	}(readCh, path, expected)

	select {
	case <-readCh:
	case <-time.After(2 * time.Second):
		t.Errorf("contents not present after 2 seconds, expected: %q, actual: %q",
			expected, last)
	}
}
