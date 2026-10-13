package look

import (
	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

func beginMove(s *swt.Shell, c *swt.Control, e *swt.MouseEvent) {
	p := c.ToDisplay(e.X, e.Y)
	gtk.BeginMoveDrag(s.Handle, e.Button, p.X, p.Y)
}

func doubleClick(s *swt.Shell) { toggleMaximized(s) }
