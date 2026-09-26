// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build windows

package template

// openNoFollow is a no-op on Windows, which does not support O_NOFOLLOW. The
// symlink pre-checks in writeToFile still apply; reparse-point/junction
// handling on Windows is a documented limitation.
const openNoFollow = 0
