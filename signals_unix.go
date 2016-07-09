// +build linux darwin freebsd openbsd solaris netbsd

package main

import (
	"os"
	"syscall"
)

var Signals = []os.Signal{
	syscall.SIGHUP,  // Hangup detected on controlling terminalor death of controlling process
	syscall.SIGINT,  // Interrupt from keyboard
	syscall.SIGQUIT, // Quit from keyboard
	syscall.SIGTERM, // Termination signal
	syscall.SIGUSR1, // User-defined signal 1
	syscall.SIGUSR2, // User-defined signal 2
}

var SignalLookup = map[string]os.Signal{
	"SIGHUP":  syscall.SIGHUP,
	"SIGINT":  syscall.SIGINT,
	"SIGQUIT": syscall.SIGQUIT,
	"SIGTERM": syscall.SIGTERM,
	"SIGUSR1": syscall.SIGUSR1,
	"SIGUSR2": syscall.SIGUSR2,
}
