//go:build !windows
// +build !windows

package renderer

import (
	"os"
	"os/user"
	"strconv"
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

func setFileOwnership(path string, uid, gid int) error {
	if uid == -1 && gid == -1 {
		return nil //noop
	}
	return os.Chown(path, uid, gid)
}

func isChownNeeded(path string, uid, gid int) bool {
	if uid == -1 && gid == -1 {
		return false
	}

	currUid, currGid := getFileOwnership(path)
	return uid != *currUid || gid != *currGid
}

// parseUidGid parses the uid/gid so that it can be input to os.Chown
func parseUidGid(s string) (int, error) {
	if s == "" {
		return -1, nil
	}
	return strconv.Atoi(s)
}

func lookupUser(s string) (int, error) {
	if id, err := parseUidGid(s); err == nil {
		return id, nil
	}
	u, err := user.Lookup(s)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(u.Uid)
}

func lookupGroup(s string) (int, error) {
	if id, err := parseUidGid(s); err == nil {
		return id, nil
	}
	u, err := user.LookupGroup(s)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(u.Gid)
}
