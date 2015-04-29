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
