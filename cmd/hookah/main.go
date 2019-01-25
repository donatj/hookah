package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/donatj/hmacsig"
	"github.com/donatj/hookah"
)

var (
	httpPort   = flag.Uint("http-port", 8080, "HTTP port to listen on")
	serverRoot = flag.String("server-root", ".", "The root directory of the hook script hierarchy")
	secret     = flag.String("secret", "", "Optional Github HMAC secret key")
	timeout    = flag.Duration("timeout", 10*time.Minute, "Exec timeout on hook scripts")
)

func init() {
	flag.Parse()
}

func main() {
	hServe, err := hookah.NewHookServer(
		*serverRoot,
		hookah.ServerExecTimeout(*timeout),
	)
	if err != nil {
		log.Fatal(err)
	}

	var serve http.Handler = hServe
	if *secret != "" {
		serve = hmacsig.Handler(hServe, *secret)
	}

	err = http.ListenAndServe(":"+strconv.Itoa(int(*httpPort)), serve)
	if err != nil {
		log.Fatal(err)
	}
}
