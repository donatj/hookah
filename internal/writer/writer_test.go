package writer

import (
	"bytes"
	"testing"
)

func TestPrefixWriter(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixWriter(&buf, ">> ")

	pw.Write([]byte("Hello\n"))
	pw.Write([]byte("World\n"))
	pw.Write([]byte("Split"))
	pw.Write([]byte(" line\n"))
	pw.Write([]byte("Multiple\nLines\n"))

	expected := ">> Hello\n>> World\n>> Split line\n>> Multiple\n>> Lines\n>> "
	if buf.String() != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, buf.String())
	}
}
