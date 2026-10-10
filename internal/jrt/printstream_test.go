package jrt

import (
	"fmt"
	"os"
	"testing"
)

func TestSetOutCapturesStdoutAndRestores(t *testing.T) {
	orig := os.Stdout
	buf := NewByteArrayOutputStream()
	ps := NewPrintStream(buf, true)
	SetOut(ps)
	fmt.Fprint(os.Stdout, "hello")
	PrintStreamFlush(ps)
	PrintStreamFlush(ps)
	if got := buf.ToString(StandardCharsetsUTF_8); got != "hello" {
		t.Fatalf("captured %q, want hello", got)
	}
	SetOut(orig)
	if os.Stdout != orig {
		t.Fatal("os.Stdout not restored")
	}
}
