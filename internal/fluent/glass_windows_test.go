package fluent

import (
	"os"
	"runtime"
	"testing"

	"github.com/haiodo/gowt/swt"
)

// Disposing a window that used Glass removes the focus filters; the event table compares listeners, so
// a func-typed listener panicked there.
func TestGlassDisposeRemovesFilters(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	d := swt.NewDisplay()
	defer d.Dispose()
	sh := swt.NewShellDisplay(d)
	c := swt.NewCompositeParentStyle(sh, 0)
	swt.NewButton(c, swt.PUSH)
	swt.NewLabel(c, 0)
	Glass(c, true)
	sh.Dispose()
	if !c.IsDisposed() {
		t.Fatal("composite not disposed")
	}
}
