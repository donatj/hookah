package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

var (
	validGhEvent = regexp.MustCompile(`^[a-z_]{1,30}$`)
)

var (
	httpPort   = flag.Uint("http-port", 8080, "HTTP port to listen on")
	serverRoot = flag.String("server-root", ".", "The root directory of the deploy script hierarchy")
)

func init() {
	flag.Parse()
}

func main() {
	httpMux := http.NewServeMux()

	httpMux.HandleFunc("/", httpHandler)

	err := http.ListenAndServe(":"+strconv.Itoa(int(*httpPort)), httpMux)
	if err != nil {
		log.Fatal(err)
	}
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
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
