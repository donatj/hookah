package hookah

import (
	"bytes"
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

	multierror "github.com/hashicorp/go-multierror"
)

var errsNotDir = errors.New("Given path is not a dir")
var validGhEvent = regexp.MustCompile(`^[a-z\d_]{1,30}$`)

// Logger handles Printf
type Logger interface {
	Printf(format string, v ...interface{})
}

// HookServer implements net/http.Handler
type HookServer struct {
	RootDir string

	Timeout  time.Duration
	ErrorLog Logger
	InfoLog  Logger

	sync.Mutex
}

// ServerOption sets an option of the HookServer
type ServerOption func(*HookServer) error

// NewHookServer instantiates a new HookServer with some basic validation
// on the root directory
func NewHookServer(rootdir string, options ...ServerOption) (*HookServer, error) {
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

	server := &HookServer{
		RootDir: rootdir,
	}

	var result *multierror.Error

	for _, option := range options {
		err := option(server)
		result = multierror.Append(result, err)
	}

	return server, result.ErrorOrNil()
}

// ServerExecTimeout configures the HookServer per-script execution timeout
func ServerExecTimeout(timeout time.Duration) ServerOption {
	return func(h *HookServer) error {
		h.Timeout = timeout
		return nil
	}
}

// ServerErrorLog configures the HookServer error logger
func ServerErrorLog(log Logger) ServerOption {
	return func(h *HookServer) error {
		h.ErrorLog = log
		return nil
	}
}

// ServerInfoLog configures the HookServer info logger
func ServerInfoLog(log Logger) ServerOption {
	return func(h *HookServer) error {
		h.InfoLog = log
		return nil
	}
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

	ghDelivery := r.Header.Get("X-GitHub-Delivery")
	if ghDelivery == "" {
		http.Error(w, "Request requires valid X-GitHub-Delivery", http.StatusBadRequest)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(ghDelivery, err)
		return
	}
	buff := bytes.NewReader(b)

	basicHook := &HookJSON{}

	decoder := json.NewDecoder(buff)
	err = decoder.Decode(basicHook)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(ghDelivery, err)
		return
	}

	login := basicHook.Repository.Owner.GetLogin()
	repo := basicHook.Repository.Name
	if repo == "" || login == "" {
		msg := "Failed parsing JSON HTTP Body"
		http.Error(w, msg, http.StatusBadRequest)
		log.Println(ghDelivery, msg)
		return
	}

	fmt.Fprintf(w, "%s/%s", login, repo)

	hook := HookExec{
		RootDir: h.RootDir,
		Data:    buff,
		InfoLog: h.InfoLog,
	}

	go func() {
		h.Lock()
		defer h.Unlock()

		err := hook.Exec(login, repo, ghEvent, h.Timeout, "GITHUB_DELIVERY="+ghDelivery, "GITHUB_EVENT="+ghEvent)
		if err != nil && h.ErrorLog != nil {
			h.ErrorLog.Printf("%s - %s/%s:%s - '%s'", ghDelivery, login, repo, ghEvent, err)
		}
	}()
}

// HookUserJSON exists because some hooks use Login, some use Name
// - it's horribly inconsistent and a bad flaw on GitHubs part
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
