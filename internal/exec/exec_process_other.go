//go:build !unix

package exec

import (
	"errors"
	stdexec "os/exec"
)

// configureCommand leaves CommandContext's default cancellation in place on
// non-Unix systems, where process-group termination is not available.
func configureCommand(_ *stdexec.Cmd, _ bool) {}

func exitStatus(err error) (int, bool) {
	exitErr, ok := errors.AsType[*stdexec.ExitError](err)
	if !ok {
		return 0, false
	}
	return exitErr.ExitCode(), true
}
