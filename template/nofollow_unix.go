// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build !windows

package template

import "syscall"

// openNoFollow adds O_NOFOLLOW to the open flags so that, if the final path
// component is a symlink, the open fails rather than following it. This closes
// the TOCTOU window between the symlink pre-check and the open on Unix-like
// systems. It is not available on Windows (see nofollow_windows.go).
const openNoFollow = syscall.O_NOFOLLOW
