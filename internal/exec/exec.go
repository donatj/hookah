package exec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/donatj/hookah/v4/internal/writer"
)

// Logger handles Printf and Println
type Logger interface {
	Printf(format string, v ...any)
	Println(v ...any)
}

// HookExec represents a call to a hook
type HookExec struct {
	RootDir string
	Data    io.ReadSeeker
	InfoLog Logger

	Stdout io.Writer
	Stderr io.Writer

	// DisableLogPrefixes disables timestamp and file path prefixes on stdout/stderr
	DisableLogPrefixes bool

	longestPrefix   int
	longestFileName int
}

// HookExecOption is a functional option for configuring HookExec
type HookExecOption func(*HookExec)

// WithInfoLog sets the info logger
func WithInfoLog(logger Logger) HookExecOption {
	return func(h *HookExec) {
		h.InfoLog = logger
	}
}

// WithStdout sets the stdout writer
func WithStdout(w io.Writer) HookExecOption {
	return func(h *HookExec) {
		h.Stdout = w
	}
}

// WithStderr sets the stderr writer
func WithStderr(w io.Writer) HookExecOption {
	return func(h *HookExec) {
		h.Stderr = w
	}
}

// WithDisableLogPrefixes disables timestamp and file path prefixes on stdout/stderr
func WithDisableLogPrefixes(disable bool) HookExecOption {
	return func(h *HookExec) {
		h.DisableLogPrefixes = disable
	}
}

// NewHookExec creates a new HookExec with the given required parameters and optional configuration.
//
// Example:
//
//	data := strings.NewReader(`{"event": "push"}`)
//	hook := NewHookExec(
//	    "/path/to/hooks",
//	    data,
//	    WithInfoLog(logger),
//	    WithStdout(os.Stdout),
//	    WithDisableLogPrefixes(false),
//	)
func NewHookExec(rootDir string, data io.ReadSeeker, opts ...HookExecOption) *HookExec {
	h := &HookExec{
		RootDir: rootDir,
		Data:    data,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// GetPathExecs fetches the executable filenames for the given path
func (h *HookExec) GetPathExecs(owner, repo, event, action string) ([]string, []string, error) {
	outfiles := []string{}
	outErrHandlers := []string{}

	var pathSets [][]string
	if action == "" {
		pathSets = [][]string{
			{h.RootDir, owner, repo, event},
			{filepath.Join(h.RootDir, "@@"), repo, event},
			{filepath.Join(h.RootDir, owner, "@@"), event},
			{filepath.Join(h.RootDir, "@@", "@@"), event},
		}
	} else {
		pathSets = [][]string{
			{h.RootDir, owner, repo, event, action},
			{filepath.Join(h.RootDir, "@@"), repo, event, action},
			{filepath.Join(h.RootDir, owner, "@@"), event, action},
			{filepath.Join(h.RootDir, "@@", "@@"), event, action},
		}
	}

	for _, paths := range pathSets {
		workpath := ""
		for _, path := range paths {
			workpath = filepath.Join(workpath, path)

			files, errHandlers, err := pathScan(workpath)
			if err != nil {
				return []string{}, []string{}, err
			}
			outfiles = append(outfiles, files...)
			outErrHandlers = append(outErrHandlers, errHandlers...)
		}
	}

	return outfiles, outErrHandlers, nil
}

// pathScan scans the given path for executable files
// returns a list of files and a list of error handlers
// error handlers are files that start with @@error.
func pathScan(path string) ([]string, []string, error) {
	files := []string{}
	errHandlers := []string{}

	fs, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return files, errHandlers, nil
		}

		return files, errHandlers, err
	}

	if fs.IsDir() {
		d, err := os.Open(path)
		if err != nil {
			return files, errHandlers, err
		}
		defer d.Close()

		fi, err := d.Readdir(-1)
		if err != nil {
			return files, errHandlers, err
		}
		// I don't think this is necessary but it makes the tests deterministic
		sort.Slice(fi, func(i, j int) bool { return fi[i].Name() < fi[j].Name() })

		for _, fi := range fi {
			fpath := filepath.Join(path, fi.Name())
			is, err := isExecFile(fpath)
			if err != nil {
				return files, errHandlers, err
			}

			if is {
				if strings.HasPrefix(fi.Name(), "@@error.") {
					errHandlers = append(errHandlers, filepath.Join(path, fi.Name()))
				} else {
					files = append(files, filepath.Join(path, fi.Name()))
				}
			}
		}

	} else if is, _ := isExecFile(path); is {
		// fmt.Println(fs.Name(), fs.Size(), "bytes")
		// files = append(files, filepath.Join(path, fs.Name()))
		// this should be picked up on a different sweep
	} else {
		return files, errHandlers, errors.New("bad file mumbo jumbo")
	}

	return files, errHandlers, nil
}

// InfoLogf logs to the info logger if not nil
func (h *HookExec) InfoLogf(format string, v ...any) {
	if h.InfoLog != nil {
		h.InfoLog.Printf(format, v...)
	}
}

func (h *HookExec) InfoLogln(msg string) {
	if h.InfoLog != nil {
		h.InfoLog.Println(msg)
	}
}

// Exec triggers the execution of all scripts associated with the given Hook
func (h *HookExec) Exec(owner, repo, event, action, delivery string, timeout time.Duration, env ...string) error {
	files, errHandlers, err := h.GetPathExecs(owner, repo, event, action)

	if err != nil {
		return err
	}

	if len(files) > 0 {
		msg := fmt.Sprintf("executing hook scripts (%d) for %s/%s %s.%s", len(files), owner, repo, event, action)
		msg = strings.TrimRight(msg, ".")
		h.InfoLogln(msg)
	}

	var errs []error
	for _, f := range files {
		h.InfoLogf("beginning execution of %#v", f)

		err := h.execFile(f, delivery, h.Data, timeout, env...)

		if err != nil {
			h.InfoLogf("exec error: %s", err)

			for _, e := range errHandlers {
				h.InfoLogf("beginning error handler execution of %#v", e)

				env2 := append(env, getErrorHandlerEnv(f, err)...)
				err2 := h.execFile(e, "[err] "+delivery, h.Data, timeout, env2...)
				errs = append(errs, err2)
			}
		}
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func getErrorHandlerEnv(f string, err error) []string {
	env := []string{
		"HOOKAH_EXEC_ERROR_FILE=" + f,
		"HOOKAH_EXEC_ERROR=" + err.Error(),
	}

	if exiterr, ok := errors.AsType[*exec.ExitError](err); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			env = append(env, fmt.Sprintf("HOOKAH_EXEC_EXIT_STATUS=%d", status.ExitStatus()))
		}
	}

	return env
}

const logDateFmt = "2006/01/02 15:04:05"

// execFile executes the hook script at path f with data piped to stdin and the given environment variables.
// If timeout is greater than zero, the process and its children are killed via process group termination after
// the timeout expires. If timeout is zero, the process runs without a timeout. The function always waits for
// the process to exit, preventing zombie processes.
func (h *HookExec) execFile(f, prefix string, data io.ReadSeeker, timeout time.Duration, env ...string) (err error) {
	ctx := context.Background()

	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		cancel = func() {}
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, f)
	if timeout > 0 {
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

	if h.Stdout != nil {
		cmd.Stdout = h.Stdout
	} else {
		cmd.Stdout = os.Stdout
	}

	if h.Stderr != nil {
		cmd.Stderr = h.Stderr
	} else {
		cmd.Stderr = os.Stdout // uniformly dump logs to stdout by default
	}

	if !h.DisableLogPrefixes {
		relPath, err := filepath.Rel(h.RootDir, f)
		if err != nil {
			relPath = f
		}

		if len(prefix) > h.longestPrefix {
			h.longestPrefix = len(prefix)
		}

		if len(relPath) > h.longestFileName {
			h.longestFileName = len(relPath)
		}

		cmd.Stdout = writer.NewPrefixWriter(cmd.Stdout, func() string {
			return fmt.Sprintf(": %s %*s %*s (stdout) > ", time.Now().Format(logDateFmt), h.longestPrefix, prefix, h.longestFileName, relPath)
		})
		cmd.Stderr = writer.NewPrefixWriter(cmd.Stderr, func() string {
			return fmt.Sprintf(": %s %*s %*s (stderr) > ", time.Now().Format(logDateFmt), h.longestPrefix, prefix, h.longestFileName, relPath)
		})
	}

	cmd.Env = append(os.Environ(), env...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if _, err := data.Seek(0, 0); err != nil {
		_ = stdin.Close()
		return err
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}

	defer func() {
		waitErr := cmd.Wait()

		if waitErr != nil && ctx.Err() == context.DeadlineExceeded {
			waitErr = fmt.Errorf("hook timed out after %s: %w", timeout, waitErr)
		}

		switch {
		case err == nil:
			err = waitErr
		case waitErr != nil:
			err = errors.Join(err, waitErr)
		}
	}()

	if _, err := io.Copy(stdin, data); err != nil {
		_ = stdin.Close()
		return err
	}

	// Ignore close error - child may exit early without reading all stdin
	_ = stdin.Close()

	return nil
}
