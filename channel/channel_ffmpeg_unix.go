//go:build !windows

package channel

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// ffmpegCommand builds an ffmpeg command that is killable as a process group on Unix.
func ffmpegCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// terminateProcess attempts graceful then hard kill of the process group.
func terminateProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
		time.Sleep(2 * time.Second)
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		return nil
	}

	// Fallback to direct process kill.
	return cmd.Process.Kill()
}
