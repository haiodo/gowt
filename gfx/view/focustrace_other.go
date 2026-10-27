//go:build linux || windows

package view

// Temporary focus tracing (GOWT_TRACE_FOCUS=1) is macOS only.

func focusTrace(string, ...any) {}

func (v *View) traceFirstResponder(string) {}

func installFocusTrace() {}
