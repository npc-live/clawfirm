//go:build !windows

package runtime

import (
	"os/exec"
	"syscall"
)

// setProcGroup configures the command to run in its own process group
// and kills the entire group on cancel.
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
}
