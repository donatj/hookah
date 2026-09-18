package exec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	stdexec "os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ErrPathTraversal is returned when a path component contains characters that could be used for directory traversal.
// Can be checked with errors.Is().
var ErrPathTraversal = errors.New("rejected path traversal attempt")

// Logger handles Printf and Println
type Logger interface {
	Printf(format string, v ...any)
	Println(v ...any)
}

// WriterFactory wraps stdout and stderr writers for a given file path
type WriterFactory func(stdout, stderr io.Writer, filePath string) (wrappedStdout, wrappedStderr io.Writer)

// HookExec represents a call to a hook
type HookExec struct {
	RootDir string
	Data    io.ReadSeeker
	InfoLog Logger

	Stdout io.Writer
	Stderr io.Writer

	// WriterFactory optionally creates custom writers for each executed file
	WriterFactory WriterFactory
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

// WithWriterFactory sets a factory function that creates stdout and stderr writers for each executed file
func WithWriterFactory(factory WriterFactory) HookExecOption {
	return func(h *HookExec) {
		h.WriterFactory = factory
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
//	    WithWriterFactory(myFactory),
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

// validatePathComponent checks if a path component is safe from directory traversal.
// Returns an error if the component is empty or unsafe.
func validatePathComponent(component, name string) error {
	if component == "" {
		return fmt.Errorf("%w: empty %s not allowed", ErrPathTraversal, name)
	}
	// Use filepath.Clean to normalize the path and detect traversal attempts
	cleaned := filepath.Clean(component)
	if cleaned != component || strings.Contains(cleaned, string(filepath.Separator)) || cleaned == "." || cleaned == ".." {
		return fmt.Errorf("%w in %s: %q", ErrPathTraversal, name, component)
	}
	return nil
}

// GetPathExecs fetches the executable filenames for the given path.
// Returns ErrPathTraversal if any component attempts directory traversal.
//
// action is optional - if empty, it will be omitted from the path resolution.
func (h *HookExec) GetPathExecs(owner, repo, event, action string) ([]string, []string, error) {
	// Validate path components to prevent directory traversal attacks
	for name, component := range map[string]string{
		"owner": owner,
		"repo":  repo,
		"event": event,
	} {
		if err := validatePathComponent(component, name); err != nil {
			return nil, nil, err
		}
	}

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
		// Validate action component to prevent directory traversal attacks
		if err := validatePathComponent(action, "action"); err != nil {
			return nil, nil, err
		}

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
		return files, errHandlers, fmt.Errorf("path %q is neither a directory nor an executable file", path)
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
func (h *HookExec) Exec(owner, repo, event, action string, timeout time.Duration, env ...string) error {
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

		err := h.execFile(f, h.Data, timeout, env...)

		if err != nil {
			h.InfoLogf("exec error: %s", err)

			for _, e := range errHandlers {
				h.InfoLogf("beginning error handler execution of %#v", e)

				env2 := append(env, getErrorHandlerEnv(f, err)...)
				err2 := h.execFile(e, h.Data, timeout, env2...)
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

	if status, ok := exitStatus(err); ok {
		env = append(env, fmt.Sprintf("HOOKAH_EXEC_EXIT_STATUS=%d", status))
	}

	return env
}

// execFile executes the hook script at path f with data piped to stdin and the given environment variables.
// If timeout is greater than zero, the process and its children are killed via process group termination after
// the timeout expires. If timeout is zero, the process runs without a timeout. The function always waits for
// the process to exit, preventing zombie processes.
func (h *HookExec) execFile(f string, data io.ReadSeeker, timeout time.Duration, env ...string) (err error) {
	ctx := context.Background()

	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		cancel = func() {}
	}
	defer cancel()

	cmd := stdexec.CommandContext(ctx, f)
	configureCommand(cmd, timeout > 0)

	// Determine base writers
	stdout := h.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	stderr := h.Stderr
	if stderr == nil {
		stderr = os.Stdout // uniformly dump logs to stdout by default
	}

	// Use WriterFactory to wrap them if provided
	if h.WriterFactory != nil {
		cmd.Stdout, cmd.Stderr = h.WriterFactory(stdout, stderr, f)
	} else {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
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
