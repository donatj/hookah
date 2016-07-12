package main

import (
	"flag"
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

	hServe := HookServer{}

	httpMux.Handle("/", &hServe)

	err := http.ListenAndServe(":"+strconv.Itoa(int(*httpPort)), httpMux)
	if err != nil {
		log.Fatal(err)
	}
}
