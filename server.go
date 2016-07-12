package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

// HookServer implements net/http.Handler
type HookServer struct {
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

	x := HookExec{
		Root: *serverRoot,

		Owner: basicHook.Repository.Owner.Login,
		Repo:  basicHook.Repository.Name,
		Event: ghEvent,
		Data:  buff,
	}

	err = x.Exec()
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
