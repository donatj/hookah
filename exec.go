package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
)

type HookExec struct {
	Owner string
	Repo  string
	Event string
}

func (h *HookExec) GetPathExecs() ([]string, error) {
	path := filepath.Join(".", h.Owner, h.Repo, h.Event)

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
	// for _, f := range files {
	// 	fmt.Println(f.Name())
	// }

}

func (h *HookExec) Exec() error {
	files, err := h.GetPathExecs()
	if err != nil {
		return err
	}

	var result error

	for _, f := range files {
		cmd := exec.Command(f)
		// io.Copy( cmd.StdinPipe
		err := cmd.Run()
		if err != nil {
			multierror.Append(result, err)
		}
	}

	return result
}

// todo: base this on OS
func isExecFile(fi os.FileInfo) bool {
	return fi.Mode().IsRegular() && fi.Mode()|0111 == fi.Mode()
}
