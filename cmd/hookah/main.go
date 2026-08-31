package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/donatj/hmacsig"
	"github.com/donatj/hookah/v3"
)

var (
	httpPort   = flag.Uint("http-port", 8080, "HTTP port to listen on")
	serverRoot = flag.String("server-root", ".", "The root directory of the hook script hierarchy")
	secret     = flag.String("secret", "", "Optional GitHub HMAC secret key")
	timeout    = flag.Duration("timeout", 10*time.Minute, "Exec timeout on hook scripts")
	verbose    = flag.Bool("v", false, "Enable verbose logger output")
)

var (
	errlog = flag.String("err-log", "", "Path to write the error log to. Defaults to standard error.")

	logJSON   = flag.Bool("log-json", false, "log in JSON format")
	logLevel  = flag.String("log-level", "info", "log level (options: debug, info, warn, error)")
	logSource = flag.Bool("log-source", false, "log output includes source code location")
)

//go:embed favicon.ico
var favicon []byte

func init() {
	flag.Parse()
}

func main() {
	logger, err := getLogger(*errlog, *logLevel, *logSource, *logJSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	slog.SetDefault(logger)

	options := []hookah.ServerOption{
		hookah.ServerExecTimeout(*timeout),
		hookah.ServerErrorLog(logger),
	}

	if *verbose {
		options = append(options, hookah.ServerInfoLog(logger))
	}

	hServe, err := hookah.NewHookServer(*serverRoot, options...)
	if err != nil {
		slog.Error("failed to create hook server", "error", err)
		os.Exit(1)
	}

	var serve http.Handler = hServe
	if *secret != "" {
		serve = hmacsig.Handler256(hServe, *secret)
	}

	mux := http.NewServeMux()
	mux.Handle("/", serve)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(favicon)
	})

	logger.Info("listening on port", "port", *httpPort)
	err = http.ListenAndServe(":"+strconv.Itoa(int(*httpPort)), mux)
	if err != nil {
		slog.Error("failed to start HTTP server", "error", err)
		os.Exit(1)
	}
}

func getLogger(filename, logLevel string, addSource, json bool) (*slog.Logger, error) {
	var handler slog.Handler

	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		slog.Error("unknown log level", "level", logLevel)
		os.Exit(3)
	}

	opts := &slog.HandlerOptions{
		AddSource: addSource,
		Level:     level,
	}

	var w io.Writer
	if filename != "" {
		f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		w = f
	} else {
		w = os.Stderr
	}

	if json {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler), nil
}
