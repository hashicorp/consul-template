package gatedio

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestNewBuffer(t *testing.T) {
	var b bytes.Buffer
	buf := NewBuffer(&b)

	if reflect.DeepEqual(buf.rw, b) {
		t.Errorf("expected %#v to be %#v", buf.rw, b)
	}
}

func TestBuffer_write(t *testing.T) {
	var b bytes.Buffer
	buf := NewBuffer(&b)

	size := 10000
	for i := 0; i < size; i++ {
		go func() { buf.Write([]byte("a")) }()
	}

	bytesLen := func() int {
		buf.Lock()
		defer buf.Unlock()
		return buf.rw.(*bytes.Buffer).Len()
	}

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		for bytesLen() < size {
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-doneCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("bad: %d", bytesLen())
	}
}

func TestNewWriter(t *testing.T) {
	var b bytes.Buffer
	buf := NewWriter(&b)

	if reflect.DeepEqual(buf.w, b) {
		t.Errorf("expected %#v to be %#v", buf.w, b)
	}
}

func TestWriter_write(t *testing.T) {
	var b bytes.Buffer
	buf := NewBuffer(&b)

	size := 10000
	for i := 0; i < size; i++ {
		go func() { buf.Write([]byte("a")) }()
	}

	bytesLen := func() int {
		buf.Lock()
		defer buf.Unlock()
		return buf.rw.(*bytes.Buffer).Len()
	}

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		for bytesLen() < size {
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-doneCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("bad: %d", bytesLen())
	}
}

func TestNewReader(t *testing.T) {
	var b bytes.Buffer
	buf := NewReader(&b)

	if reflect.DeepEqual(buf.r, b) {
		t.Errorf("expected %#v to be %#v", buf.r, b)
	}
}

func TestReader_reader(t *testing.T) {
	var b bytes.Buffer
	buf := NewBuffer(&b)

	size := 10000
	for i := 0; i < size; i++ {
		buf.Write([]byte("a"))
	}

	for i := 0; i < size; i++ {
		go func() { buf.Read([]byte("a")) }()
	}

	bytesLen := func() int {
		buf.Lock()
		defer buf.Unlock()
		return buf.rw.(*bytes.Buffer).Len()
	}

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		for bytesLen() > 0 {
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-doneCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("bad: %d", bytesLen())
	}
}
