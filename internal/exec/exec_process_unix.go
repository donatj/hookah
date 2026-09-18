//go:build unix

package exec

import (
	"errors"
	stdexec "os/exec"
	"syscall"
)

func configureCommand(cmd *stdexec.Cmd, timed bool) {
	if !timed {
		return
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Kill the entire process group instead of just the parent.
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			// Preserve os/exec's successful-exit behavior when cancellation
			// races with the process exiting.
			return cmd.Process.Kill()
		}
		return err
	}
}

func exitStatus(err error) (int, bool) {
	exitErr, ok := errors.AsType[*stdexec.ExitError](err)
	if !ok {
		return 0, false
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	return status.ExitStatus(), ok
}
