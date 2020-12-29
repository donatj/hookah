package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/donatj/hmacsig"
	"github.com/donatj/hookah/v2"
)

var (
	httpPort   = flag.Uint("http-port", 8080, "HTTP port to listen on")
	serverRoot = flag.String("server-root", ".", "The root directory of the hook script hierarchy")
	secret     = flag.String("secret", "", "Optional Github HMAC secret key")
	timeout    = flag.Duration("timeout", 10*time.Minute, "Exec timeout on hook scripts")
	verbose    = flag.Bool("v", false, "Enable verbose logger output")

	errlog = flag.String("err-log", "", "Path to write the error log to. Defaults to standard error.")
)

func init() {
	flag.Parse()
}

func main() {
	logger := getLogger(*errlog)
	options := []hookah.ServerOption{
		hookah.ServerExecTimeout(*timeout),
		hookah.ServerErrorLog(logger),
	}

	if *verbose {
		options = append(options, hookah.ServerInfoLog(logger))
	}

	hServe, err := hookah.NewHookServer(*serverRoot, options...)
	if err != nil {
		log.Fatal(err)
	}

	var serve http.Handler = hServe
	if *secret != "" {
		serve = hmacsig.Handler256(hServe, *secret)
	}

	err = http.ListenAndServe(":"+strconv.Itoa(int(*httpPort)), serve)
	if err != nil {
		log.Fatal(err)
	}
}

func getLogger(filename string) hookah.Logger {
	if filename != "" {
		f, err := os.OpenFile(*errlog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		return log.New(f, "", log.LstdFlags)
	}

	return log.New(os.Stderr, "", log.LstdFlags)
}
