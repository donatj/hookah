package hookah

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

var errsNotDir = errors.New("Given path is not a dir")
var validGhEvent = regexp.MustCompile(`^[a-z_]{1,30}$`)

// HookServer implements net/http.Handler
type HookServer struct {
	RootDir string
	Timeout time.Duration
	secret  string
	sync.Mutex
}

// NewHookServer instantiates a new HookServer with some basic validation
// on the root directory
func NewHookServer(rootdir, secret string, timeout time.Duration) (*HookServer, error) {
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
		Timeout: timeout,
		secret:  secret,
	}, nil
}

func (h *HookServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ghEvent := r.Header.Get("X-Github-Event")

	if !validGhEvent.MatchString(ghEvent) {
		http.Error(w, "Request requires valid X-Github-Event", http.StatusBadRequest)
		return
	}

	if ghEvent == "ping" {
		fmt.Fprintln(w, "pong")
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	buff := bytes.NewReader(b)

	if h.secret != "" {
		xSig := r.Header.Get("X-Hub-Signature")

		if xSig == "" {
			http.Error(w, "Missing required X-Hub-Signature for HMAC verification", http.StatusForbidden)
			log.Println("missing X-Hub-Signature")
			return
		}

		hash := hmac.New(sha1.New, []byte(h.secret))
		hash.Write(b)

		ehash := hash.Sum(nil)
		esig := "sha1=" + hex.EncodeToString(ehash)

		if !hmac.Equal([]byte(esig), []byte(xSig)) {
			http.Error(w, "HMAC verification failed", http.StatusForbidden)
			log.Println("HMAC verification failed")
			return
		}
	}

	basicHook := &HookJSON{}

	decoder := json.NewDecoder(buff)
	err = decoder.Decode(basicHook)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	login := basicHook.Repository.Owner.GetLogin()
	repo := basicHook.Repository.Name

	fmt.Fprintf(w, "%s/%s", login, repo)

	if repo == "" || login == "" {
		http.Error(w, "Failed parsing JSON HTTP Body", http.StatusBadRequest)
		log.Println(err)
		return
	}

	hook := HookExec{
		RootDir: h.RootDir,

		Owner: login,
		Repo:  repo,

		Event: ghEvent,
		Data:  buff,

		HookServer: h,
	}

	go hook.Exec(h.Timeout)
}

// HookUserJSON exists because some hooks use Login, some use Name
// - it's horribly inconsistent
type HookUserJSON struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

// GetLogin is used to get the login from the data github decided to pass today
func (h *HookUserJSON) GetLogin() string {
	if h.Login != "" {
		return h.Login
	}

	return h.Name
}

// HookJSON represents the minimum body we need to parse
type HookJSON struct {
	Repository struct {
		Name  string       `json:"name"`
		Owner HookUserJSON `json:"owner"`
	} `json:"repository"`
	Sender HookUserJSON `json:"sender"`
}
