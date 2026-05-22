package writer

import (
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

	sync.Mutex
}

// NewPrefixWriter creates a new PrefixWriter that prefixes each line written to w.
// The prefixFunc is called each time a prefix needs to be written, allowing for dynamic prefixes.
func NewPrefixWriter(w io.Writer, prefixFunc func() string) *PrefixWriter {
	return &PrefixWriter{
		prefixFunc:  prefixFunc,
		w:           w,
		needsPrefix: true,
	}
}

func (pw *PrefixWriter) Write(p []byte) (n int, err error) {
	pw.Lock()
	defer pw.Unlock()

	if len(p) == 0 {
		return 0, nil
	}

	if pw.needsPrefix {
		prefix := []byte(pw.prefixFunc())
		if _, err = pw.w.Write(prefix); err != nil {
			return 0, err
		}
		pw.needsPrefix = false
	}

	n = len(p)
	start := 0

	for i, b := range p {
		if b == '\n' {
			if _, err = pw.w.Write(p[start : i+1]); err != nil {
				return 0, err
			}

			start = i + 1

			if start < len(p) {
				prefix := []byte(pw.prefixFunc())
				if _, err = pw.w.Write(prefix); err != nil {
					return 0, err
				}
			} else {
				pw.needsPrefix = true
			}
		}
	}

	if start < len(p) {
		if _, err = pw.w.Write(p[start:]); err != nil {
			return 0, err
		}
	}

	return n, nil
}
