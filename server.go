package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
)

var errsNotDir = errors.New("Given path is not a dir")
var validGhEvent = regexp.MustCompile(`^[a-z_]{1,30}$`)

// HookServer implements net/http.Handler
type HookServer struct {
	RootDir string
}

// NewHookServer instantiates a new HookServer with some basic validation
// on the root directory
func NewHookServer(rootdir string) (*HookServer, error) {
	f, err := os.Open(rootdir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, errsNotDir
	}

	return &HookServer{
		RootDir: rootdir,
	}, nil
}

func (h *HookServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ghEvent := r.Header.Get("X-Github-Event")

	if !validGhEvent.MatchString(ghEvent) {
		http.Error(w, "Request requires valid X-Github-Event", http.StatusBadRequest)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	buff := bytes.NewReader(b)

	basicHook := &HookJSON{}

	decoder := json.NewDecoder(buff)
	err = decoder.Decode(basicHook)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	if basicHook.Repository.Name == "" || basicHook.Repository.Owner.Login == "" {
		http.Error(w, "Failed parsing JSON HTTP Body", http.StatusBadRequest)
		return
	}

	hook := HookExec{
		RootDir: h.RootDir,

		Owner: basicHook.Repository.Owner.Login,
		Repo:  basicHook.Repository.Name,

		Event: ghEvent,
		Data:  buff,
	}

	err = hook.Exec()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
	}
}

// HookJSON represents the minimum body we need to parse
type HookJSON struct {
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}
