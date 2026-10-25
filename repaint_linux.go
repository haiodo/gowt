package gowt

import (
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// repaintShell makes the window of c draw again, children included, at the next turn of the event loop.
// On GTK, SWT's redraw of one control invalidates the whole GdkWindow it shares with its siblings
// without their children, so the card and window background is painted over siblings that are not
// invalidated themselves: a label beside a Canvas, or beside a label whose text changed, goes blank.
// One repaint per window and turn, however many controls changed.
func (a *App) repaintShell(c *swt.Control) {
	sh := c.GetShell()
	for _, s := range a.repaints {
		if s == sh {
			return
		}
	}
	a.repaints = append(a.repaints, sh)
	if len(a.repaints) == 1 {
		a.display.AsyncExec(jrt.NewRunnable(a.live(a.runRepaints)))
	}
}

func (a *App) runRepaints() {
	list := a.repaints
	a.repaints = nil
	for _, sh := range list {
		if !sh.IsDisposed() {
			sh.RedrawChildren(true)
		}
	}
}
