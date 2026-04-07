//go:build !windows

package clawproc

import (
	"os/exec"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

func sendSIGTERM(p *Process) error {
	return p.cmd.Process.Signal(syscall.SIGTERM)
}
