package exec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnlyExecutableBinsFound(t *testing.T) {

	expectedScripts := []string{
		"../../testdata/exec-only-test-server/exec.sh",
		"../../testdata/exec-only-test-server/exec.symlink.sh",
		"../../testdata/exec-only-test-server/user/exec.sh",
		"../../testdata/exec-only-test-server/user/repo/exec.sh",
		"../../testdata/exec-only-test-server/user/repo/event/exec.sh",
		"../../testdata/exec-only-test-server/@@/exec.sh",
		"../../testdata/exec-only-test-server/@@/exec.symlink.symlink.sh",
		"../../testdata/exec-only-test-server/@@/repo/exec.sh",
		"../../testdata/exec-only-test-server/@@/repo/event/exec.sh",
		"../../testdata/exec-only-test-server/user/@@/exec.sh",
		"../../testdata/exec-only-test-server/user/@@/event/exec.sh",
		"../../testdata/exec-only-test-server/@@/@@/exec.sh",
		"../../testdata/exec-only-test-server/@@/@@/event/exec.sh",
	}

	expectedErrhandlers := []string{
		"../../testdata/exec-only-test-server/@@error.exec.sh",
		"../../testdata/exec-only-test-server/user/@@error.exec.sh",
		"../../testdata/exec-only-test-server/user/repo/@@error.exec.sh",
	}

	data := strings.NewReader(`{"foo": "bar"}`)

	h := HookExec{
		RootDir: "../../testdata/exec-only-test-server",
		Data:    data,
	}

	scripts, errhandlers, err := h.GetPathExecs("user", "repo", "event", "")
	if err != nil {
		t.Error(err)
		return
	}

	log.Printf("%#v", scripts)

	assert.EqualValues(t, expectedScripts, scripts)

	assert.EqualValues(t, expectedErrhandlers, errhandlers)
}

func TestActionDirectoriesWorkAsExpected(t *testing.T) {

	expectedScripts := []string{
		"../../testdata/exec-only-test-server/exec.sh",
		"../../testdata/exec-only-test-server/exec.symlink.sh",
		"../../testdata/exec-only-test-server/user/exec.sh",
		"../../testdata/exec-only-test-server/user/repo/exec.sh",
		"../../testdata/exec-only-test-server/user/repo/event/exec.sh",
		"../../testdata/exec-only-test-server/user/repo/event/action/exec.sh",
		"../../testdata/exec-only-test-server/@@/exec.sh",
		"../../testdata/exec-only-test-server/@@/exec.symlink.symlink.sh",
		"../../testdata/exec-only-test-server/@@/repo/exec.sh",
		"../../testdata/exec-only-test-server/@@/repo/event/exec.sh",
		"../../testdata/exec-only-test-server/@@/repo/event/action/exec.sh",
		"../../testdata/exec-only-test-server/user/@@/exec.sh",
		"../../testdata/exec-only-test-server/user/@@/event/exec.sh",
		"../../testdata/exec-only-test-server/user/@@/event/action/exec.sh",
		"../../testdata/exec-only-test-server/@@/@@/exec.sh",
		"../../testdata/exec-only-test-server/@@/@@/event/exec.sh",
		"../../testdata/exec-only-test-server/@@/@@/event/action/exec.sh",
	}
	expectedErrhandlers := []string{
		"../../testdata/exec-only-test-server/@@error.exec.sh",
		"../../testdata/exec-only-test-server/user/@@error.exec.sh",
		"../../testdata/exec-only-test-server/user/repo/@@error.exec.sh",
		"../../testdata/exec-only-test-server/user/repo/event/action/@@error.exec.sh",
		"../../testdata/exec-only-test-server/@@/repo/event/action/@@error.exec.sh",
		"../../testdata/exec-only-test-server/user/@@/event/action/@@error.exec.sh",
		"../../testdata/exec-only-test-server/@@/@@/event/action/@@error.exec.sh",
	}

	data := strings.NewReader(`{"foo": "bar"}`)

	h := HookExec{
		RootDir: "../../testdata/exec-only-test-server",
		Data:    data,
	}

	scripts, errhandlers, err := h.GetPathExecs("user", "repo", "event", "action")
	if err != nil {
		t.Error(err)
		return
	}

	log.Printf("%#v", scripts)

	assert.EqualValues(t, expectedScripts, scripts)

	assert.EqualValues(t, expectedErrhandlers, errhandlers)
}

func TestEnvPopulatedCorrectly(t *testing.T) {

	out := &bytes.Buffer{}

	data := strings.NewReader(`{"foo": "bar"}`)

	h := HookExec{
		RootDir: "../../testdata/env-test-server",
		Data:    data,
		Stdout:  out,
	}

	err := h.Exec("user", "repo", "event", "action", "test-delivery", 1*time.Minute, "FOO=BAR", "BAZ=QUX")
	if err != nil {
		t.Error(err)
	}

	env := out.String()
	lines := strings.Split(strings.TrimSpace(env), "\n")
	envMap := make(map[string]string, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		envMap[parts[0]] = parts[1]
	}

	expectedEnv := map[string]string{
		"FOO": "BAR",
		"BAZ": "QUX",
	}

	for k, expectedV := range expectedEnv {
		if actualV, ok := envMap[k]; !ok || actualV != expectedV {
			t.Error("expected", k, "to be", expectedV, "got", actualV)
		}
	}

}

// TestExecFileTimeout verifies that execFile respects the timeout and returns
// a timeout error without hanging for long-running scripts.
func TestExecFileTimeout(t *testing.T) {
	f, err := os.CreateTemp("", "hookah-test-*.sh")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	_, err = io.WriteString(f, "#!/bin/sh\nsleep 30\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0755))

	h := HookExec{
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	// Exceed the pipe buffer so the timeout must also unblock stdin writes.
	data := strings.NewReader(strings.Repeat("x", 1024*1024))

	done := make(chan error, 1)
	go func() { done <- h.execFile(f.Name(), data, 200*time.Millisecond) }()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "timed out")
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Contains(t, getErrorHandlerEnv(f.Name(), err), fmt.Sprintf("HOOKAH_EXEC_EXIT_STATUS=%d", exitErr.ExitCode()))
	case <-time.After(5 * time.Second):
		t.Fatal("execFile hung after timeout")
	}
}

// TestExecFileCopyError verifies that a Read error during stdin copy still allows
// the child process to be reaped without the call hanging (no zombie processes).
func TestExecFileCopyError(t *testing.T) {
	root := t.TempDir()
	completed := filepath.Join(root, "completed")
	f := filepath.Join(root, "hook")
	require.NoError(t, os.WriteFile(f, []byte("#!/bin/sh\ncat >/dev/null\nsleep 0.1\nprintf completed > \"$HOOK_COMPLETED_FILE\"\n"), 0755))

	h := HookExec{
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	readErr := errors.New("simulated read error")
	data := &readErrSeeker{readErr: readErr}

	done := make(chan error, 1)
	go func() {
		done <- h.execFile(f, data, 5*time.Second, "HOOK_COMPLETED_FILE="+completed)
	}()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, readErr)
		contents, readCompletedErr := os.ReadFile(completed)
		require.NoError(t, readCompletedErr, "execFile returned before the child completed")
		assert.Equal(t, "completed", string(contents))
	case <-time.After(3 * time.Second):
		t.Fatal("execFile hung waiting for process to be reaped")
	}
}

// readErrSeeker is a ReadSeeker whose Seek always succeeds but whose Read always
// returns the configured error, simulating an io.Copy failure mid-transfer.
type readErrSeeker struct {
	readErr error
}

func (r *readErrSeeker) Seek(_ int64, _ int) (int64, error) { return 0, nil }
func (r *readErrSeeker) Read(_ []byte) (int, error)         { return 0, r.readErr }

// Exercise the real error-handler environment, including a joined copy/exit error.
func TestExecErrorHandlerExitStatus(t *testing.T) {
	for _, copyFailure := range []bool{false, true} {
		name := "exit"
		if copyFailure {
			name = "copy-and-exit"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(root, "hook"), []byte("#!/bin/sh\ncat >/dev/null\nexit 42\n"), 0755))
			require.NoError(t, os.WriteFile(filepath.Join(root, "@@error.handler"), []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s' \"$HOOKAH_EXEC_EXIT_STATUS\"\n"), 0755))
			var data io.ReadSeeker = strings.NewReader("payload")
			readErr := errors.New("copy failed")
			if copyFailure {
				data = &readErrSeeker{readErr: readErr}
			}
			var out bytes.Buffer
			h := HookExec{RootDir: root, Data: data, Stdout: &out, Stderr: io.Discard}
			err := h.Exec("owner", "repo", "event", "", time.Second)
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, 42, exitErr.ExitCode())
			assert.Equal(t, "42", out.String())
			if copyFailure {
				assert.ErrorIs(t, err, readErr)
			}
		})
	}
}

func TestExecFileWithoutTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hook")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\ncat\n"), 0755))
	var out bytes.Buffer
	h := HookExec{Stdout: &out, Stderr: io.Discard}
	require.NoError(t, h.execFile(path, strings.NewReader("payload"), 0))
	assert.Equal(t, "payload", out.String())
}
