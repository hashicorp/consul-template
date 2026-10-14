// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build !windows
// +build !windows

package child

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd, setpgid, setsid bool) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: setpgid,
		Setsid:  setsid,
	}
}

func processNotFoundErr(err error) bool {
	// The process exited before the signal was delivered. Go reports that
	// as ErrProcessDone ("os: process already finished"). A kill that
	// races the exit can also return ESRCH.
	return errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH)
}
