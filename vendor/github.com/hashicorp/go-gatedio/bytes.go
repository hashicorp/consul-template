package gatedio

import (
	"bytes"
	"io"
	"sync"
)

// ByteBuffer is a wrapper around a bytes.Buffer.
type ByteBuffer struct {
	sync.Mutex
	b *bytes.Buffer
}

// NewByteBuffer returns a wrapper around a bytes.Buffer that is safe to
// use across concurrent goroutines.
func NewByteBuffer() *ByteBuffer {
	return &ByteBuffer{b: new(bytes.Buffer)}
}

// Bytes wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) Bytes() []byte {
	b.Lock()
	defer b.Unlock()
	return b.b.Bytes()
}

// Cap wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) Cap() int {
	b.Lock()
	defer b.Unlock()
	return b.b.Cap()
}

// Grow wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) Grow(n int) {
	b.Lock()
	defer b.Unlock()
	b.b.Grow(n)
}

// Len wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) Len() int {
	b.Lock()
	defer b.Unlock()
	return b.b.Len()
}

// Next wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) Next(n int) []byte {
	b.Lock()
	defer b.Unlock()
	return b.b.Next(n)
}

// Read wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) Read(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.Read(p)
}

// ReadByte wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) ReadByte() (byte, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.ReadByte()
}

// ReadBytes wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) ReadBytes(delim byte) ([]byte, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.ReadBytes(delim)
}

// ReadFrom wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) ReadFrom(r io.Reader) (int64, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.ReadFrom(r)
}

// ReadRune wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) ReadRune() (rune, int, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.ReadRune()
}

// ReadString wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) ReadString(delim byte) (string, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.ReadString(delim)
}

// Reset wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) Reset() {
	b.Lock()
	defer b.Unlock()
	b.b.Reset()
}

// String wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) String() string {
	b.Lock()
	defer b.Unlock()
	return b.b.String()
}

// Truncate wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) Truncate(n int) {
	b.Lock()
	defer b.Unlock()
	b.b.Truncate(n)
}

// UnreadByte wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) UnreadByte() error {
	b.Lock()
	defer b.Unlock()
	return b.b.UnreadByte()
}

// UnreadRune wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) UnreadRune() error {
	b.Lock()
	defer b.Unlock()
	return b.b.UnreadRune()
}

// Write wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) Write(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.Write(p)
}

// WriteByte wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) WriteByte(c byte) error {
	b.Lock()
	defer b.Unlock()
	return b.b.WriteByte(c)
}

// WriteRune wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) WriteRune(r rune) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.WriteRune(r)
}

// WriteString wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) WriteString(s string) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.WriteString(s)
}

// WriteTo wraps a mutex around the underlying bytes.Buffer function call.
func (b *ByteBuffer) WriteTo(w io.Writer) (int64, error) {
	b.Lock()
	defer b.Unlock()
	return b.b.WriteTo(w)
}
