package writer

import (
	"io"
	"sync"
)

var _ io.Writer = (*PrefixWriter)(nil)

// PrefixWriter wraps an io.Writer and prefixes each line with a given string.
// Safe for concurrent use.
type PrefixWriter struct {
	prefix  []byte
	w       io.Writer
	started bool

	sync.Mutex
}

// NewPrefixWriter creates a new PrefixWriter that prefixes each line written to w.
func NewPrefixWriter(w io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{
		prefix: []byte(prefix),
		w:      w,
	}
}

func (pw *PrefixWriter) Write(p []byte) (n int, err error) {
	pw.Lock()
	defer pw.Unlock()

	if !pw.started {
		if _, err = pw.w.Write(pw.prefix); err != nil {
			return 0, err
		}
		pw.started = true
	}

	n = len(p)
	start := 0

	for i, b := range p {
		if b == '\n' {
			if _, err = pw.w.Write(p[start : i+1]); err != nil {
				return 0, err
			}

			if _, err = pw.w.Write(pw.prefix); err != nil {
				return 0, err
			}

			start = i + 1
		}
	}

	if start < len(p) {
		if _, err = pw.w.Write(p[start:]); err != nil {
			return 0, err
		}
	}

	return n, nil
}
