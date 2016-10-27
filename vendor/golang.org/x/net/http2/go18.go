// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.8

package http2

import (
	"crypto/tls"
	"net/http"
)

func cloneTLSConfig(c *tls.Config) *tls.Config { return c.Clone() }

var _ http.Pusher = (*responseWriter)(nil)

// Push implements http.Pusher.
func (w *responseWriter) Push(target string, opts *http.PushOptions) error {
	internalOpts := pushOptions{}
	if opts != nil {
		internalOpts.Method = opts.Method
		internalOpts.Header = opts.Header
	}
	return w.push(target, internalOpts)
}
