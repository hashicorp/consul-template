package main

import (
	"io/ioutil"
	"os"
	"testing"
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
