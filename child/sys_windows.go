// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build windows
// +build windows

package child

import "os/exec"

func setSysProcAttr(cmd *exec.Cmd, setpgid, setsid bool) {}

func processNotFoundErr(err error) bool {
	return false
}
