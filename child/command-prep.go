//go:build !windows
// +build !windows

package child

import (
	"os/exec"
	"strings"
)

func CommandPrep(command []string) ([]string, error) {
	switch {
	case len(command) == 1 && len(strings.Fields(command[0])) > 1:
		// command is []string{"command using arguments or shell features"}
		shell := "sh"
		// default to 'sh' on path, else try a couple common absolute paths
		if _, err := exec.LookPath(shell); err != nil {
			shell = ""
			for _, sh := range []string{"/bin/sh", "/usr/bin/sh"} {
				if sh, err := exec.LookPath(sh); err == nil {
					shell = sh
					break
				}
			}
		}
		if shell == "" {
			return []string{}, exec.ErrNotFound
		}
		cmd := []string{shell, "-c", command[0]}
		return cmd, nil
	case len(command) >= 1 && len(strings.TrimSpace(command[0])) > 0:
		// command is already good ([]string{"foo"}, []string{"foo", "bar"}, ..)
		return command, nil
	default:
		// command is []string{} or []string{""}
		return []string{}, exec.ErrNotFound
	}
}
