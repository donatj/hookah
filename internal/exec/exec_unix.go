//go:build unix

package exec

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// isExecFile checks if a file is executable by checking the Unix permission bits.
// It follows symlinks up to a depth of 10.
func isExecFile(fss ...string) (bool, error) {
	if len(fss) > 10 {
		return false, fmt.Errorf("maximum symlink depth exceeded: %s", strings.Join(fss, " -> "))
	}

	if len(fss) == 0 {
		return false, errors.New("no file info provided")
	}

	fs := fss[len(fss)-1]
	fi, err := os.Stat(fs)
	if err != nil {
		return false, err
	}

	mode := fi.Mode()
	if mode.IsRegular() && mode&0o111 != 0 {
		return true, nil
	}

	if mode&os.ModeSymlink != 0 {
		link, err := os.Readlink(fi.Name())
		if err != nil {
			return false, err
		}

		fss = append(fss, link)
		return isExecFile(fss...)
	}

	return false, nil
}
