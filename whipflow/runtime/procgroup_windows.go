//go:build windows

package runtime

import "os/exec"

// setProcGroup is a no-op on Windows; exec.CommandContext already
// terminates the process when the context is cancelled.
func setProcGroup(cmd *exec.Cmd) {
	// Windows has no Setpgid/process-group kill equivalent in syscall.
	// CommandContext will call Process.Kill on cancel, which is sufficient.
}
