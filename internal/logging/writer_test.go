package logging

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestPrefixWriter(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixWriter(&buf, func() string { return ">> " })

	pw.Write([]byte("Hello\n"))
	pw.Write([]byte("World\n"))
	pw.Write([]byte("Split"))
	pw.Write([]byte(" line\n"))
	pw.Write([]byte("Multiple\nLines\n"))

	expected := ">> Hello\n>> World\n>> Split line\n>> Multiple\n>> Lines\n"
	if buf.String() != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, buf.String())
	}
}

func TestPrefixWriterDynamic(t *testing.T) {
	var buf bytes.Buffer
	counter := 0
	pw := NewPrefixWriter(&buf, func() string {
		counter++
		return fmt.Sprintf("[%d] ", counter)
	})

	pw.Write([]byte("First\n"))
	pw.Write([]byte("Second\n"))
	pw.Write([]byte("Third\n"))

	expected := "[1] First\n[2] Second\n[3] Third\n"
	if buf.String() != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, buf.String())
	}
}

func TestPrefixWriterReportsPartialPayloadWrite(t *testing.T) {
	writeErr := errors.New("write failed")
	w := &scriptedWriter{results: []writeResult{
		{n: 3}, // prefix
		{n: 2, err: writeErr},
	}}
	pw := NewPrefixWriter(w, func() string { return ">> " })

	n, err := pw.Write([]byte("hello"))
	if !errors.Is(err, writeErr) {
		t.Fatalf("expected write error, got %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 payload bytes written, got %d", n)
	}
}

func TestPrefixWriterConvertsShortWriteToError(t *testing.T) {
	w := &scriptedWriter{results: []writeResult{
		{n: 3}, // prefix
		{n: 2},
	}}
	pw := NewPrefixWriter(w, func() string { return ">> " })

	n, err := pw.Write([]byte("hello"))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("expected io.ErrShortWrite, got %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 payload bytes written, got %d", n)
	}
}

func TestPrefixWriterPairSharesLock(t *testing.T) {
	stdout, stderr := NewPrefixWriterPair(
		io.Discard,
		io.Discard,
		func() string { return "out: " },
		func() string { return "err: " },
	)
	if stdout.mu != stderr.mu {
		t.Fatal("paired writers must share a lock to keep output lines intact")
	}
}

type writeResult struct {
	n   int
	err error
}

type scriptedWriter struct {
	results []writeResult
}

func (w *scriptedWriter) Write(p []byte) (int, error) {
	if len(w.results) == 0 {
		return len(p), nil
	}

	result := w.results[0]
	w.results = w.results[1:]
	return result.n, result.err
}
