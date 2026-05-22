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
}
