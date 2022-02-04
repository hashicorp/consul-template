//go:build windows
// +build windows

package renderer

import (
	"log"
)

func setFileOwnership(path string, uid, gid int) error {
	return nil
}

func isChownNeeded(path string, wantedUid, wantedGid int) (bool, error) {
	return false, nil
}

func lookupUser(user string) (int, error) {
	if user != "" {
		log.Printf("[WARN] (runner) cannot set user for rendered files on Windows")
	}
	return -1, nil
}

func lookupGroup(group string) (int, error) {
	if group != "" {
		log.Printf("[WARN] (runner) cannot set group for rendered files on Windows")
	}
	return -1, nil
}
