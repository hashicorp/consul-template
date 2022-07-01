//go:build windows
// +build windows

package child

import (
	"os/exec"
	"strings"
)

func CommandPrep(command []string) ([]string, error) {
	switch {
	case len(command) == 1 && len(strings.Fields(command[0])) == 1:
		// command is []string{"foo"}
		return []string{command[0]}, nil
	case len(command) > 1:
		// command is []string{"foo", "bar"}
		return command, nil
	default:
		// command is []string{}, []string{""}, []string{"foo bar"}
		return []string{}, exec.ErrNotFound
	}
}
