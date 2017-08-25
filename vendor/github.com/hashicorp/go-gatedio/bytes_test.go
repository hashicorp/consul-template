package gatedio

import (
	"testing"
	"time"
)

func TestNewByteBuffer(t *testing.T) {
	buf := NewByteBuffer()
	if buf.b == nil {
		t.Error("missing buffer")
	}
}

func TestByteBuffer_bytes(t *testing.T) {
	buf := NewByteBuffer()
	size := 100

	for i := 0; i < size; i++ {
		go func() { buf.Write([]byte("a")) }()
	}

	doneCh := make(chan struct{}, 1)
	go func() {
		for buf.Len() < size {
			time.Sleep(100 * time.Millisecond)
		}
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-time.After(500 * time.Millisecond):
		t.Errorf("bad len: %d", buf.Len())
	}
}

func TestByteBuffer_cap(t *testing.T) {
	buf := NewByteBuffer()
	size := 100

	for i := 0; i < size; i++ {
		go func() { buf.Write([]byte("a")) }()
	}

	doneCh := make(chan struct{}, 1)
	go func() {
		for buf.Cap() < size {
			time.Sleep(100 * time.Millisecond)
		}
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-time.After(500 * time.Millisecond):
		t.Errorf("bad cap: %d", buf.Cap())
	}
}
