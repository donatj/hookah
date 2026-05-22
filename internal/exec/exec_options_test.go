package exec

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"
)

func TestNewHookExec(t *testing.T) {
	data := strings.NewReader(`{"test": "data"}`)

	t.Run("basic construction", func(t *testing.T) {
		h := NewHookExec("/tmp/hooks", data)
		if h.RootDir != "/tmp/hooks" {
			t.Errorf("expected RootDir=/tmp/hooks, got %s", h.RootDir)
		}
		if h.Data != data {
			t.Error("expected Data to be set")
		}
		if h.InfoLog != nil {
			t.Error("expected InfoLog to be nil by default")
		}
		if h.WriterFactory != nil {
			t.Error("expected WriterFactory to be nil by default")
		}
	})

	t.Run("with options", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		factory := func(filePath string) (io.Writer, io.Writer) {
			return stdout, stderr
		}

		h := NewHookExec(
			"/tmp/hooks",
			data,
			WithInfoLog(logger),
			WithStdout(stdout),
			WithStderr(stderr),
			WithWriterFactory(factory),
		)

		if h.RootDir != "/tmp/hooks" {
			t.Errorf("expected RootDir=/tmp/hooks, got %s", h.RootDir)
		}
		if h.Data != data {
			t.Error("expected Data to be set")
		}
		if h.InfoLog == nil {
			t.Error("expected InfoLog to be set")
		}
		if h.Stdout != stdout {
			t.Error("expected Stdout to be set")
		}
		if h.Stderr != stderr {
			t.Error("expected Stderr to be set")
		}
		if h.WriterFactory == nil {
			t.Error("expected WriterFactory to be set")
		}
	})

	t.Run("selective options", func(t *testing.T) {
		stdout := &bytes.Buffer{}

		h := NewHookExec(
			"/tmp/hooks",
			data,
			WithStdout(stdout),
		)

		if h.Stdout != stdout {
			t.Error("expected Stdout to be set")
		}
		if h.Stderr != nil {
			t.Error("expected Stderr to be nil when not provided")
		}
		if h.InfoLog != nil {
			t.Error("expected InfoLog to be nil when not provided")
		}
	})
}
