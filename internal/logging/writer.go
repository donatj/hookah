package logging

import (
	"errors"
	"io"
	"sync"
)

var _ io.Writer = (*PrefixWriter)(nil)

// PrefixWriter wraps an io.Writer and prefixes each line with a dynamically generated string.
// Safe for concurrent use.
type PrefixWriter struct {
	prefixFunc  func() string
	w           io.Writer
	needsPrefix bool

	mu *sync.Mutex
}

// NewPrefixWriter creates a new PrefixWriter that prefixes each line written to w.
// The prefixFunc is called each time a prefix needs to be written, allowing for dynamic prefixes.
func NewPrefixWriter(w io.Writer, prefixFunc func() string) *PrefixWriter {
	return newPrefixWriter(w, prefixFunc, &sync.Mutex{})
}

// NewPrefixWriterPair creates two prefix writers which serialize writes through
// a shared lock. This prevents lines from separate streams from interleaving
// when they ultimately write to the same destination.
func NewPrefixWriterPair(stdout, stderr io.Writer, stdoutPrefix, stderrPrefix func() string) (*PrefixWriter, *PrefixWriter) {
	mu := &sync.Mutex{}
	return newPrefixWriter(stdout, stdoutPrefix, mu), newPrefixWriter(stderr, stderrPrefix, mu)
}

func newPrefixWriter(w io.Writer, prefixFunc func() string, mu *sync.Mutex) *PrefixWriter {
	return &PrefixWriter{
		prefixFunc:  prefixFunc,
		w:           w,
		needsPrefix: true,
		mu:          mu,
	}
}

func (pw *PrefixWriter) Write(p []byte) (n int, err error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if len(p) == 0 {
		return 0, nil
	}

	start := 0

	for i := 0; i <= len(p); i++ {
		if i != len(p) && p[i] != '\n' {
			continue
		}

		chunk := p[start:i]
		if i < len(p) {
			chunk = p[start : i+1]
		}
		if len(chunk) == 0 {
			continue
		}

		if pw.needsPrefix {
			if err = pw.writePrefix(); err != nil {
				return n, err
			}
			pw.needsPrefix = false
		}

		written, writeErr := pw.w.Write(chunk)
		if written < 0 || written > len(chunk) {
			return n, errors.New("logging: invalid write count")
		}
		n += written
		if writeErr == nil && written != len(chunk) {
			writeErr = io.ErrShortWrite
		}
		if writeErr != nil {
			return n, writeErr
		}

		if chunk[len(chunk)-1] == '\n' {
			pw.needsPrefix = true
		}
		start = i + 1
	}

	return n, nil
}

func (pw *PrefixWriter) writePrefix() error {
	prefix := []byte(pw.prefixFunc())
	written, err := pw.w.Write(prefix)
	if written < 0 || written > len(prefix) {
		return errors.New("logging: invalid write count")
	}
	if err == nil && written != len(prefix) {
		return io.ErrShortWrite
	}
	return err
}
