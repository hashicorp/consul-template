//go:build windows
// +build windows

package renderer

import (
	"log"
	"os"
)

func setFileOwnership(path string, uid, gid *int) error {
	if uid != nil || gid != nil {
		log.Printf("[WARN] (runner) cannot set uid/gid for rendered files on Windows")
	}
	return nil
}

func isChownNeeded(path string, wantedUid, wantedGid *int) bool {
	return false
}
