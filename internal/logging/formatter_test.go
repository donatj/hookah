package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatter(t *testing.T) {
	t.Run("basic formatting", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewFormatter("/tmp/hooks")

		w := formatter.Wrap(&buf, "delivery-123", "/tmp/hooks/user/repo/script.sh", "stdout")
		w.Write([]byte("test output\n"))

		output := buf.String()
		if !strings.Contains(output, "delivery-123") {
			t.Error("expected output to contain delivery ID")
		}
		if !strings.Contains(output, "user/repo/script.sh") {
			t.Error("expected output to contain relative path")
		}
		if !strings.Contains(output, "(stdout)") {
			t.Error("expected output to contain stream type")
		}
		if !strings.Contains(output, "test output") {
			t.Error("expected output to contain actual content")
		}
	})

	t.Run("alignment grows with longer values", func(t *testing.T) {
		var buf1, buf2 bytes.Buffer
		formatter := NewFormatter("/tmp/hooks")

		// First write with short values
		w1 := formatter.Wrap(&buf1, "short", "/tmp/hooks/a.sh", "stdout")
		w1.Write([]byte("line1\n"))
		line1 := buf1.String()

		// Second write with longer values
		w2 := formatter.Wrap(&buf2, "very-long-delivery-id", "/tmp/hooks/very/long/path/to/script.sh", "stdout")
		w2.Write([]byte("line2\n"))
		line2 := buf2.String()

		// The second line should be longer due to increased padding
		if len(line2) <= len(line1) {
			t.Error("expected alignment to grow with longer values")
		}
	})
}
