//go:build windows

package channel

import (
	"context"
	"os/exec"
	"syscall"
)

// ffmpegCommand builds an ffmpeg command for Windows; no process group semantics needed.
func ffmpegCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	// Create a new process group to avoid inheriting console signals if present.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	return cmd
}

// terminateProcess kills the process on Windows.
func terminateProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
