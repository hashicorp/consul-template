// +build go1.12

package cmdflag

import (
	"runtime/debug"
)

func buildinfo() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		return bi.Main.Version
	}
	return "no version available (not built with module support)"
}
