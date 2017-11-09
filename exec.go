package hookah

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-multierror"
)

// HookExec represents a call to a hook
type HookExec struct {
	RootDir string

	Owner string
	Repo  string

	Event string
	Data  io.ReadSeeker

	HookServer *HookServer
}

// GetPathExecs fetches the executable filenames for the given path
func (h *HookExec) GetPathExecs() ([]string, error) {
	path := filepath.Join(h.RootDir, h.Owner, h.Repo, h.Event)

	files := []string{}

	fs, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}

		return files, err
	}

	if fs.IsDir() {
		d, err := os.Open(path)
		defer d.Close()
		if err != nil {
			return files, err
		}

		fi, err := d.Readdir(-1)
		if err != nil {
			return files, err
		}

		for _, fi := range fi {
			if isExecFile(fi) {
				// fmt.Println(fi.Name(), fi.Size(), "bytes")
				files = append(files, filepath.Join(path, fi.Name()))
			}
		}

	} else if isExecFile(fs) {
		// fmt.Println(fs.Name(), fs.Size(), "bytes")
		files = append(files, filepath.Join(path, fs.Name()))
	} else {
		return files, errors.New("bad file mumbo jumbo")
	}

	return files, nil
}

// Exec triggers the execution of all scripts associated with the given Hook
func (h *HookExec) Exec(timeout time.Duration) error {
	files, err := h.GetPathExecs()
	if err != nil {
		return err
	}

	var result error

	h.HookServer.Lock()
	defer h.HookServer.Unlock()

	for _, f := range files {
		err := execFile(f, h, timeout)
		multierror.Append(result, err)
	}

	return result
}

func execFile(f string, h *HookExec, timeout time.Duration) error {
	cmd := exec.Command(f)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	h.Data.Seek(0, 0)
	io.Copy(stdin, h.Data)
	stdin.Close()

	timer := time.AfterFunc(timeout, func() {
		cmd.Process.Kill()
	})

	err = cmd.Wait()
	timer.Stop()

	if err != nil {
		return err
	}

	return nil
}

// todo: base this on OS
func isExecFile(fi os.FileInfo) bool {
	return fi.Mode().IsRegular() && fi.Mode()|0111 == fi.Mode()
}
