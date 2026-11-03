package gowt

import "github.com/haiodo/gowt/swt"

type overlayScrollable interface {
	GetVerticalBar() *swt.ScrollBar
	GetHorizontalBar() *swt.ScrollBar
}

func overlayScroll(c *swt.Control) {
	s, ok := c.Impl().(overlayScrollable)
	if !ok {
		return
	}
	hide := func() {
		for _, bar := range []*swt.ScrollBar{s.GetVerticalBar(), s.GetHorizontalBar()} {
			if bar != nil {
				bar.SetVisible(false)
			}
		}
	}
	hide()
	c.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { hide() }))
	c.AddListener(swt.MouseWheel, eventFunc(func(e *swt.Event) {
		if e.Count == 0 {
			return
		}
		bar := s.GetVerticalBar()
		if bar == nil {
			return
		}
		delta := int32(3)
		if e.Count > 0 {
			delta = -delta
		}
		bar.SetSelection(bar.GetSelection() + delta)
		e.Doit = false
		c.Redraw()
	}))
	c.AddPaintListener(painter(func(e *swt.PaintEvent) {
		r := c.GetBounds()
		if r.Width < 8 || r.Height < 8 {
			return
		}
		drawOverlayBar(c, e, r, s.GetVerticalBar(), true)
		drawOverlayBar(c, e, r, s.GetHorizontalBar(), false)
	}))
	c.Redraw()
}

func drawOverlayBar(c *swt.Control, e *swt.PaintEvent, r *swt.Rectangle, bar *swt.ScrollBar, vertical bool) {
	if bar == nil {
		return
	}
	minimum, maximum, thumb := bar.GetMinimum(), bar.GetMaximum(), bar.GetThumb()
	if maximum-minimum <= thumb {
		return
	}
	selection := bar.GetSelection()
	x, y, w, h := overlayThumb(r, vertical, minimum, maximum, thumb, selection)
	e.Gc.SetBackground(c.GetDisplay().GetSystemColor(swt.COLOR_WIDGET_NORMAL_SHADOW))
	if vertical {
		e.Gc.FillRectangle(r.Width-4, 2, 2, r.Height-4)
	} else {
		e.Gc.FillRectangle(2, r.Height-4, r.Width-4, 2)
	}
	e.Gc.SetBackground(c.GetDisplay().GetSystemColor(swt.COLOR_WIDGET_FOREGROUND))
	e.Gc.FillRectangle(x, y, w, h)
}

func overlayThumb(r *swt.Rectangle, vertical bool, minimum, maximum, thumb, selection int32) (x, y, w, h int32) {
	span := maximum - minimum
	selection = min(max(selection, minimum), maximum-thumb)
	length := r.Height - 4
	if !vertical {
		length = r.Width - 4
	}
	thumbLength := max(int32(8), thumb*length/span)
	position := (selection - minimum) * (length - thumbLength) / max(span-thumb, 1)
	if vertical {
		return r.Width - 4, 2 + position, 2, thumbLength
	}
	return 2 + position, r.Height - 4, thumbLength, 2
}
