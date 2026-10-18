package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// topOnFirstSize runs top after a table or tree first gets a size, if rows added before that left it scrolled: the
// view's tile leaves the clip view at offset 0, and offset 0 puts the first row under the header.
func topOnFirstSize(c *swt.Control, top func()) {
	c.AddListener(swt.Resize, &firstSize{c, top})
}

type firstSize struct {
	c   *swt.Control
	top func()
}

// HandleEvent waits for the end of the resize: the tile that moves the clip view runs after the Resize event.
func (l *firstSize) HandleEvent(*swt.Event) {
	if l.c.GetSize().Y <= 0 {
		return
	}
	l.c.RemoveListener(swt.Resize, l)
	l.c.GetDisplay().AsyncExec(jrt.NewRunnable(l.check))
}

func (l *firstSize) check() {
	if l.c.IsDisposed() {
		return
	}
	var tv *cocoa.NSTableView
	switch v := l.c.View.Impl().(type) {
	case *cocoa.SWTTableView:
		tv = &v.NSTableView
	case *cocoa.SWTOutlineView:
		tv = &v.NSTableView
	default:
		return
	}
	clip := tv.Superview()
	if tv.HeaderView() == nil || clip == nil || clip.Bounds().Y != 0 {
		return
	}
	l.top()
}
