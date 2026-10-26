package gowt

import (
	"fmt"

	"github.com/haiodo/gowt/swt"
)

// canvasHost lets a widget description build its widget in a canvas.
type canvasHost struct{ *CanvasView }

func (h canvasHost) into() *swt.Composite { return &h.c.Composite }
func (canvasHost) added(Widget)           {}

// AddChild creates the widget opt describes as a child of the canvas, over what the canvas paints;
// nil if opt is not a widget. The canvas has no layout: place the child with SetChildBounds.
func (c *CanvasView) AddChild(opt Option) Widget {
	n, ok := opt.(*node)
	if !ok {
		reportOn(c.ctl, fmt.Errorf("gowt: AddChild on Canvas takes a widget; give options where the widget is created"))
		return nil
	}
	return n.build(canvasHost{c})
}

// SetChildBounds puts child, made by AddChild, at (x, y) with size w by h, in points of the canvas.
func (c *CanvasView) SetChildBounds(child Widget, x, y, w, h int) {
	child.control().SetBounds(int32(x), int32(y), int32(w), int32(h))
}
