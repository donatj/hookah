//go:build !unix

package exec

import "os"

// isExecFile reports whether f is a regular file. On non-Unix systems,
// os/exec determines whether the file can be started.
func isExecFile(f string) (bool, error) {
	fi, err := os.Stat(f)
	if err != nil {
		return false, err
	}
	return fi.Mode().IsRegular(), nil
}
