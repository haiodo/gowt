package swt

import (
	"os"
	"testing"
)

// Run with GOWT_GUI_TEST=1: a resize must lay out a shell whose layout is deferred, since AppKit's
// live-resize loop does not return to ReadAndDispatch, where deferred layouts otherwise run.
func TestResizeRunsDeferredLayout(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	done := make(chan struct{})
	vtMainq <- func() {
		defer close(done)
		d := NewDisplay()
		defer d.Dispose()
		shell := NewShell()
		shell.SetLayout(NewFillLayout())
		child := NewCompositeParentStyle(shell, NONE)
		shell.SetSize(300, 200)
		shell.Layout()
		shell.LayoutOverload4([]*Control{&child.Control}, DEFER)
		shell.SetSize(500, 400)
		ca, b := shell.GetClientArea(), child.GetBounds()
		if b.Width != ca.Width || b.Height != ca.Height {
			t.Errorf("child %dx%d, client area %dx%d", b.Width, b.Height, ca.Width, ca.Height)
		}
	}
	<-done
}
