// +build windows

package child

import "os/exec"

func setSysProcAttr(cmd *exec.Cmd, setpgid, setsid bool) {}

func processNotFoundErr(err error) bool {
	return false
}
