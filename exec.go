package hookah

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/go-multierror"
)

// HookExec represents a call to a hook
type HookExec struct {
	RootDir string
	Data    io.ReadSeeker
	InfoLog Logger

	Stdout io.Writer
	Stderr io.Writer
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
		defer d.Close()
		if err != nil {
			return files, errHandlers, err
		}

		fi, err := d.Readdir(-1)
		if err != nil {
			return files, errHandlers, err
		}

		for _, fi := range fi {
			if isExecFile(fi) {
				if strings.HasPrefix(fi.Name(), "@@error.") {
					errHandlers = append(errHandlers, filepath.Join(path, fi.Name()))
				} else {
					files = append(files, filepath.Join(path, fi.Name()))
				}
			}
		}

	} else if isExecFile(fs) {
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

// Exec triggers the execution of all scripts associated with the given Hook
func (h *HookExec) Exec(owner, repo, event, action string, timeout time.Duration, env ...string) error {
	files, errHandlers, err := h.GetPathExecs(owner, repo, event, action)

	if err != nil {
		return err
	}

	var result *multierror.Error

	for _, f := range files {
		h.InfoLogf("beginning execution of %#v", f)

		err := h.execFile(f, h.Data, timeout, env...)

		if err != nil {
			h.InfoLogf("exec error: %s", err)

			for _, e := range errHandlers {
				h.InfoLogf("beginning error handler execution of %#v", e)

				env2 := append(env, getErrorHandlerEnv(f, err)...)
				err2 := h.execFile(e, h.Data, timeout, env2...)
				result = multierror.Append(result, err2)
			}
		}
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func getErrorHandlerEnv(f string, err error) []string {
	env := []string{
		"HOOKAH_EXEC_ERROR_FILE=" + f,
		"HOOKAH_EXEC_ERROR=" + err.Error(),
	}

	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			env = append(env, fmt.Sprintf("HOOKAH_EXEC_EXIT_STATUS=%d", status.ExitStatus()))
		}
	}

	return env
}

func (h *HookExec) execFile(f string, data io.ReadSeeker, timeout time.Duration, env ...string) error {
	cmd := exec.Command(f)

	if h.Stdout != nil {
		cmd.Stdout = h.Stdout
	} else {
		cmd.Stdout = os.Stdout
	}

	if h.Stderr != nil {
		cmd.Stderr = h.Stderr
	} else {
		cmd.Stderr = os.Stderr
	}

	cmd.Env = append(os.Environ(), env...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	_, err = data.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = io.Copy(stdin, data)
	if err != nil {
		return err
	}
	stdin.Close()

	timer := time.AfterFunc(timeout, func() {
		cmd.Process.Kill()
	})

	err = cmd.Wait()
	timer.Stop()

	return err
}

// todo: base this on OS
func isExecFile(fi os.FileInfo) bool {
	return fi.Mode().IsRegular() && fi.Mode()|0111 == fi.Mode()
}
