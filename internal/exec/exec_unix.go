//go:build unix

package exec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	fi, err := os.Lstat(fs)
	if err != nil {
		return false, err
	}

	mode := fi.Mode()
	if mode&os.ModeSymlink != 0 {
		link, err := os.Readlink(fs)
		if err != nil {
			return false, err
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(fs), link)
		}

		fss = append(fss, link)
		return isExecFile(fss...)
	}

	return mode.IsRegular() && mode&0o111 != 0, nil
}
