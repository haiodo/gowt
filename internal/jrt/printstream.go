package jrt

import (
	"io"
	"os"
)

// PrintStream is java.io.PrintStream where a stream is only written to: System.out and System.err are *os.File,
// a constructed one is a *CapturedStream.
type PrintStream = io.Writer

// CapturedStream is new PrintStream(out, ...). Made System.out/err by SetOut/SetErr, it receives what
// the process writes to os.Stdout/os.Stderr, on Flush: Go cannot swap an *os.File for a buffer, so the
// output goes to a temp file and Flush copies the new bytes into out.
type CapturedStream struct {
	out  OutputStream
	file *os.File
	read int64
}

// NewPrintStream covers (out), (out, autoFlush) and (out, autoFlush, charset); the bytes are UTF-8 either way.
func NewPrintStream(out OutputStream, _ ...any) *CapturedStream { return &CapturedStream{out: out} }

func (c *CapturedStream) Write(p []byte) (int, error) { return AsWriter(c.out).Write(p) }

func (c *CapturedStream) Flush() {
	if c.file != nil {
		buf, err := io.ReadAll(io.NewSectionReader(c.file, c.read, 1<<40))
		if err == nil && len(buf) > 0 {
			c.read += int64(len(buf))
			AsWriter(c.out).Write(buf)
		}
	}
	c.out.Flush()
}

// PrintStreamFlush is PrintStream.flush on a stream of either kind.
func PrintStreamFlush(s PrintStream) {
	switch v := s.(type) {
	case *CapturedStream:
		v.Flush()
	case *os.File:
		v.Sync()
	}
}

var capturedOut, capturedErr *CapturedStream

// SetOut is System.setOut: a CapturedStream takes over os.Stdout, the original *os.File gives it back.
func SetOut(s PrintStream) { setStd(&os.Stdout, &capturedOut, s) }

// SetErr is System.setErr.
func SetErr(s PrintStream) { setStd(&os.Stderr, &capturedErr, s) }

func setStd(std **os.File, active **CapturedStream, s PrintStream) {
	if prev := *active; prev != nil {
		prev.Flush()
		prev.file.Close()
		os.Remove(prev.file.Name())
		prev.file = nil
		*active = nil
	}
	switch v := s.(type) {
	case *os.File:
		*std = v
	case *CapturedStream:
		f, err := os.CreateTemp("", "jrt-capture-")
		if err != nil {
			panic(NewIOException(err.Error()))
		}
		v.file, v.read = f, 0
		*active = v
		*std = f
	}
}
