// +build windows plan9

package main

import (
	"os"
	"syscall"
)

var Signals = []os.Signal{
	syscall.SIGHUP,
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGQUIT,
}

var SignalLookup = map[string]os.Signal{
	"SIGHUP":  syscall.SIGHUP,
	"SIGINT":  syscall.SIGINT,
	"SIGTERM": syscall.SIGTERM,
	"SIGQUIT": syscall.SIGQUIT,
}
