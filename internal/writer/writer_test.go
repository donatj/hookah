package writer

import (
	"bytes"
	"fmt"
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
