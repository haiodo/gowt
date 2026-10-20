package swt

import (
	"os"
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

// Run with GOWT_GUI_TEST=1: while the window is its own first responder, Tab focuses the first control
// and Shift+Tab the last; with a control focused, or with Cmd held, Tab is left to SWT and AppKit.
func TestTabFromWindow(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	done := make(chan struct{})
	skip := ""
	vtMainq <- func() {
		defer close(done)
		d := NewDisplay()
		defer d.Dispose()
		shell := NewShell()
		shell.SetLayout(NewFillLayout())
		first := NewText(shell, BORDER)
		last := NewText(shell, BORDER)
		shell.SetSize(300, 100)
		shell.Open()
		for d.ReadAndDispatch() {
		}
		// SWT gives the focus only to a control of the active window (Control.forceFocus).
		if d.GetActiveShell() != shell {
			skip = "the test window is not the active window: run from a frontmost app"
			return
		}
		clear := func() { shell.View.Window().MakeFirstResponder(nil) }
		clear()
		if shell.tabKeyFromWindow(keyCodeTab, int64(cocoa.OSNSEventModifierFlagCommand)) {
			t.Error("Cmd+Tab was taken")
		}
		if !shell.tabKeyFromWindow(keyCodeTab, 0) || d.GetFocusControl() != &first.Control {
			t.Errorf("Tab from the window focused %v, want the first text", d.GetFocusControl())
		}
		if shell.tabKeyFromWindow(keyCodeTab, 0) {
			t.Error("Tab with a focused control was taken")
		}
		clear()
		if !shell.tabKeyFromWindow(keyCodeTab, int64(cocoa.OSNSEventModifierFlagShift)) || d.GetFocusControl() != &last.Control {
			t.Errorf("Shift+Tab from the window focused %v, want the last text", d.GetFocusControl())
		}
	}
	<-done
	if skip != "" {
		t.Skip(skip)
	}
}
