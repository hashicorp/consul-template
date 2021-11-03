//go:build !windows
// +build !windows

package renderer

import (
	"os"
	"syscall"
)

func getFileOwnership(path string) (*int, *int) {
	file_info, err := os.Stat(path)

	if err != nil {
		return nil, nil
	}

	file_sys := file_info.Sys()
	st := file_sys.(*syscall.Stat_t)
	return intPtr(int(st.Uid)), intPtr(int(st.Gid))
}

func setFileOwnership(path string, uid, gid *int) error {
	wantedUid := sanitizeUidGid(uid)
	wantedGid := sanitizeUidGid(gid)
	if wantedUid == -1 && wantedGid == -1 {
		return nil //noop
	}
	return os.Chown(path, wantedUid, wantedGid)
}

func isChownNeeded(path string, uid, gid *int) bool {
	wantedUid := sanitizeUidGid(uid)
	wantedGid := sanitizeUidGid(gid)
	if wantedUid == -1 && wantedGid == -1 {
		return false
	}

	currUid, currGid := getFileOwnership(path)
	return wantedUid != *currUid || wantedGid != *currGid
}

// sanitizeUidGid sanitizes the uid/gid so that can be an input for os.Chown
func sanitizeUidGid(id *int) int {
	if id == nil {
		return -1
	}
	return *id
}
