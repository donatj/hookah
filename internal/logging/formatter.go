package logging

import (
	"fmt"
	"io"
	"path/filepath"
	"time"
)

const logDateFmt = "2006/01/02 15:04:05"

// Formatter formats log output with timestamps and prefixes
type Formatter struct {
	rootDir string
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

	return NewPrefixWriter(w, func() string {
		return fmt.Sprintf("| %s %s %s (%s) > ",
			time.Now().Format(logDateFmt),
			deliveryID,
			relPath,
			stream)
	})
}
