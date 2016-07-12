package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
)

var (
	httpPort   = flag.Uint("http-port", 8080, "HTTP port to listen on")
	serverRoot = flag.String("server-root", ".", "The root directory of the deploy script hierarchy")
)

func init() {
	flag.Parse()
}

func main() {
	hServe, err := NewHookServer(*serverRoot)
	if err != nil {
		log.Fatal(err)
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/", hServe)

	err = http.ListenAndServe(":"+strconv.Itoa(int(*httpPort)), httpMux)
	if err != nil {
		log.Fatal(err)
	}
}
