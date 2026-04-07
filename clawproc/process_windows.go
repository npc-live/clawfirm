//go:build windows

package clawproc

import (
	"os"
	"os/exec"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// Setpgid is not supported on Windows; no-op.
}

func killProcessGroup(pid int) {
	// Windows has no process group kill; kill the process directly.
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill()
	}
}

func sendSIGTERM(p *Process) error {
	// Windows does not support SIGTERM; use Kill instead.
	return p.cmd.Process.Kill()
}
