package hookah

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnlyExecutableBinsFound(t *testing.T) {

	expectedScripts := []string{
		"testdata/exec-only-test-server/exec.sh",
		"testdata/exec-only-test-server/exec.symlink.sh",
		"testdata/exec-only-test-server/user/exec.sh",
		"testdata/exec-only-test-server/user/repo/exec.sh",
		"testdata/exec-only-test-server/user/repo/event/exec.sh",
		"testdata/exec-only-test-server/@@/exec.sh",
		"testdata/exec-only-test-server/@@/exec.symlink.symlink.sh",
		"testdata/exec-only-test-server/@@/repo/exec.sh",
		"testdata/exec-only-test-server/@@/repo/event/exec.sh",
		"testdata/exec-only-test-server/user/@@/exec.sh",
		"testdata/exec-only-test-server/user/@@/event/exec.sh",
		"testdata/exec-only-test-server/@@/@@/exec.sh",
		"testdata/exec-only-test-server/@@/@@/event/exec.sh",
	}

	expectedErrhandlers := []string{
		"testdata/exec-only-test-server/@@error.exec.sh",
		"testdata/exec-only-test-server/user/@@error.exec.sh",
		"testdata/exec-only-test-server/user/repo/@@error.exec.sh",
	}

	data := strings.NewReader(`{"foo": "bar"}`)

	h := HookExec{
		RootDir: "./testdata/exec-only-test-server",
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
		"testdata/exec-only-test-server/exec.sh",
		"testdata/exec-only-test-server/exec.symlink.sh",
		"testdata/exec-only-test-server/user/exec.sh",
		"testdata/exec-only-test-server/user/repo/exec.sh",
		"testdata/exec-only-test-server/user/repo/event/exec.sh",
		"testdata/exec-only-test-server/user/repo/event/action/exec.sh",
		"testdata/exec-only-test-server/@@/exec.sh",
		"testdata/exec-only-test-server/@@/exec.symlink.symlink.sh",
		"testdata/exec-only-test-server/@@/repo/exec.sh",
		"testdata/exec-only-test-server/@@/repo/event/exec.sh",
		"testdata/exec-only-test-server/@@/repo/event/action/exec.sh",
		"testdata/exec-only-test-server/user/@@/exec.sh",
		"testdata/exec-only-test-server/user/@@/event/exec.sh",
		"testdata/exec-only-test-server/user/@@/event/action/exec.sh",
		"testdata/exec-only-test-server/@@/@@/exec.sh",
		"testdata/exec-only-test-server/@@/@@/event/exec.sh",
		"testdata/exec-only-test-server/@@/@@/event/action/exec.sh",
	}
	expectedErrhandlers := []string{
		"testdata/exec-only-test-server/@@error.exec.sh",
		"testdata/exec-only-test-server/user/@@error.exec.sh",
		"testdata/exec-only-test-server/user/repo/@@error.exec.sh",
		"testdata/exec-only-test-server/user/repo/event/action/@@error.exec.sh",
		"testdata/exec-only-test-server/@@/repo/event/action/@@error.exec.sh",
		"testdata/exec-only-test-server/user/@@/event/action/@@error.exec.sh",
		"testdata/exec-only-test-server/@@/@@/event/action/@@error.exec.sh",
	}

	data := strings.NewReader(`{"foo": "bar"}`)

	h := HookExec{
		RootDir: "./testdata/exec-only-test-server",
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
		RootDir: "./testdata/env-test-server",
		Data:    data,

		Stdout: out,
	}

	err := h.Exec("user", "repo", "event", "action", 1*time.Minute, "FOO=BAR", "BAZ=QUX")
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

	_, _ = io.WriteString(f, "#!/bin/sh\nsleep 30\n")
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0700))

	h := HookExec{
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	data := strings.NewReader(`{}`)

	start := time.Now()
	err = h.execFile(f.Name(), data, 200*time.Millisecond)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Less(t, elapsed, 5*time.Second, "execFile should not hang after timeout")
	assert.Contains(t, err.Error(), "timed out")
}

// TestExecFileCopyError verifies that a Read error during stdin copy still allows
// the child process to be reaped without the call hanging (no zombie processes).
func TestExecFileCopyError(t *testing.T) {
	f, err := os.CreateTemp("", "hookah-test-*.sh")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	_, _ = io.WriteString(f, "#!/bin/sh\ncat\n")
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0700))

	h := HookExec{
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	readErr := errors.New("simulated read error")
	data := &readErrSeeker{readErr: readErr}

	done := make(chan error, 1)
	go func() {
		done <- h.execFile(f.Name(), data, 5*time.Second)
	}()

	select {
	case err := <-done:
		assert.ErrorContains(t, err, readErr.Error())
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
