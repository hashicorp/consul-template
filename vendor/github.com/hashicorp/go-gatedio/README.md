Golang GatedIO
==============
[![Build Status](http://img.shields.io/travis/hashicorp/go-gatedio.svg?style=flat-square)][travis]
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]

[travis]: http://travis-ci.org/hashicorp/go-gatedio
[godocs]: http://godoc.org/github.com/hashicorp/go-gatedio

The `gatedio` package provides tiny wrappers around the `io.ReadWriter`,
`io.Writer`, and `io.Reader` interfaces to support concurrent usage and access
across multiple goroutines.

This library is especially useful in tests where a `bytes.Buffer` may be used.
Go's native `bytes.Buffer` is not safe across multiple goroutes and therefore
must be wrapped in some kind of mutex.


Usage & Examples
----------------
API Package documentation can be found on
[GoDoc](https://godoc.org/github.com/hashicorp/go-gatedio).

The `gatedio.*` functions can replace any `io.Reader`, `io.Writer`, or
`io.ReadWriter`. This is especially useful in tests:

```go
func TestSomething(t *testing.T) {
  buf := gatedio.NewBuffer(make(bytes.Buffer))

  go func() { buf.Write([]byte("a")) }()
  go func() { buf.Write([]byte("b")) }()
}
```

Please note, accessing the underlying data structure is still not safe across
multiple goroutines without locking:

```go
// This is not safe!
var b bytes.Buffer
buf := gatedio.NewBuffer(&b)

go func() { buf.Write([]byte("a")) }()

// This is still a race condition:
b.Len() // or b.Anything()
```

For these cases, it is better to use the `GatedBytesBuffer`:

```go
buf := gatedio.BytesBuffer()
```

This implements all functions of a `bytes.Buffer`, but wraps all calls in a
mutex for safe concurrent access.


Developing
----------
To install, clone from GitHub:

    $ git clone https://github.com/hashicorp/go-gatedio

Then install dependencies:

    $ make updatedeps

Then test;

    $ make test testrace
