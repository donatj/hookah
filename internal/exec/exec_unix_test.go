//go:build unix

package exec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsExecFileFollowsRelativeSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	requireNoError(t, os.WriteFile(target, []byte("#!/bin/sh\n"), 0700))

	link := filepath.Join(dir, "link")
	requireNoError(t, os.Symlink("target", link))

	isExec, err := isExecFile(link)
	requireNoError(t, err)
	if !isExec {
		t.Fatal("expected an owner-executable symlink target to be executable")
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
