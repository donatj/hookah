package logging

import (
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"
)

const logDateFmt = "2006/01/02 15:04:05"

// Formatter formats log output with timestamps and aligned prefixes
type Formatter struct {
	rootDir string

	mu              sync.Mutex
	longestPrefix   int
	longestFileName int
}

// NewFormatter creates a new Formatter for the given root directory
func NewFormatter(rootDir string) *Formatter {
	return &Formatter{
		rootDir: rootDir,
	}
}

// Wrap wraps a writer with prefix formatting for the given file path, delivery ID, and stream type
func (f *Formatter) Wrap(w io.Writer, deliveryID, filePath, stream string) io.Writer {
	relPath, err := filepath.Rel(f.rootDir, filePath)
	if err != nil {
		relPath = filePath
	}

	f.mu.Lock()
	if len(deliveryID) > f.longestPrefix {
		f.longestPrefix = len(deliveryID)
	}
	if len(relPath) > f.longestFileName {
		f.longestFileName = len(relPath)
	}
	longestPrefix := f.longestPrefix
	longestFileName := f.longestFileName
	f.mu.Unlock()

	return NewPrefixWriter(w, func() string {
		return fmt.Sprintf(": %s %*s %*s (%s) > ",
			time.Now().Format(logDateFmt),
			longestPrefix, deliveryID,
			longestFileName, relPath,
			stream)
	})
}
