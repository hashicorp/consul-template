// gatedio provides wrappers around the io.ReadWriter, io.Writer, and io.Reader
// interfaces to support concurrent usage and access across multiple goroutines.
package gatedio

import (
	"io"
	"sync"
)

// Buffer implements io.ReadWriter which can be passed to multiple
// concurrent goroutines. Buffer buffers all reads and writes using an
// internal mutex so that reading and writing to the buffer is safe.
type Buffer struct {
	sync.Mutex
	rw io.ReadWriter
}

// NewBuffer creates a new buffered io.ReadWriter.
func NewBuffer(rw io.ReadWriter) *Buffer {
	return &Buffer{rw: rw}
}

// Write implements the io.Writer interface.
func (gb *Buffer) Write(p []byte) (int, error) {
	gb.Lock()
	defer gb.Unlock()
	return gb.rw.Write(p)
}

// Read implements the io.Reader interface.
func (gb *Buffer) Read(p []byte) (int, error) {
	gb.Lock()
	defer gb.Unlock()
	return gb.rw.Read(p)
}

// Writer implements io.Writer which can be passed to multiple concurrent
// goroutines. Writer buffers all writes using an internal mutex so that writing
// to the buffer is safe.
type Writer struct {
	sync.Mutex
	w io.Writer
}

// NewWriter creates a new buffered io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Write implements the io.Writer interface.
func (w *Writer) Write(p []byte) (int, error) {
	w.Lock()
	defer w.Unlock()
	return w.w.Write(p)
}

// Reader implements io.Reader which can be passed to multiple concurrent
// goroutines. Reader buffers all reads using an internal mutex so that reading
// from the buffer is safe.
type Reader struct {
	sync.Mutex
	r io.Reader
}

// NewReader creates a new buffered io.Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Read implements the io.Reader interface.
func (r *Reader) Read(p []byte) (int, error) {
	r.Lock()
	defer r.Unlock()
	return r.r.Read(p)
}
