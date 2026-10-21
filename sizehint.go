package gowt

import (
	"fmt"

	"github.com/haiodo/gowt/swt"
)

// SetSize changes the width and the height the layout gives the control, in points, as Size did at
// creation; 0 goes back to the natural value. It replaces a MaxHeight. The parents are laid out on the
// next event-loop turn or at App.Flush, as after SetHeight.
func (w *base) SetSize(width, height int) {
	c := w.ctl
	wh, hh, ok := sizeHints(c)
	if !ok {
		reportOn(c, fmt.Errorf("gowt: SetSize on %s needs a parent laid out by a VBox, HBox, Grid, Flow or Form", nameOf(c)))
		return
	}
	delete(capped, c)
	*wh, *hh = hintOf(width), hintOf(height)
	setMinHeight(c, height)
	if a := appOf(c); a != nil {
		a.relayout(c)
	}
}

// SetMaxHeight keeps the natural height of the control, as it follows its content (the rows of a Tree
// or Table, the lines of a Label), at most h points, border included; at h it stops growing and
// scrolls when it can. 0 removes the limit. A Tree or Table follows its rows, nodes expanded and
// collapsed included; another widget is measured after SetText, SetFont and the other setters that
// lay out. SetHeight and SetSize replace the limit. It needs a parent laid out by a VBox, HBox, Grid,
// Flow or Form.
func (w *base) SetMaxHeight(h int) {
	c := w.ctl
	if _, _, ok := sizeHints(c); !ok {
		reportOn(c, fmt.Errorf("gowt: SetMaxHeight on %s needs a parent laid out by a VBox, HBox, Grid, Flow or Form", nameOf(c)))
		return
	}
	if h <= 0 {
		if _, had := capped[c]; had {
			delete(capped, c)
			w.SetHeight(0)
		}
		return
	}
	if _, had := capped[c]; !had {
		c.AddDisposeListener(&capRelease{c})
	}
	capped[c] = h
	refitMax(c)
	if a := appOf(c); a != nil {
		a.relayout(c)
	}
}

// MaxHeight keeps the natural height of a widget at most h points, as SetMaxHeight does.
func MaxHeight(h int) Option {
	return hook(func(w Widget) error {
		w.(interface{ SetMaxHeight(int) }).SetMaxHeight(h)
		return nil
	})
}

func hintOf(v int) int32 {
	if v <= 0 {
		return swt.DEFAULT
	}
	return int32(v)
}

// capped holds the MaxHeight of controls; UI thread only.
var capped = map[*swt.Control]int{}

type capRelease struct{ c *swt.Control }

func (r *capRelease) WidgetDisposed(*swt.DisposeEvent) { delete(capped, r.c) }

// setMinHeight makes h the least height of a control that grows in a grid, as Size does at creation.
func setMinHeight(c *swt.Control, h int) {
	if d, ok := c.GetLayoutData().(*swt.GridData); ok && d.GrabExcessVerticalSpace {
		d.MinimumHeight = int32(max(h, 0))
	}
}

// refitMax sets the height hint of a control with a MaxHeight: none (natural) while the natural height
// fits under the limit, the limit less the border beyond it, since SWT adds the trim of a Scrollable to
// a height hint. In a VBox the limit is also what the other rows leave of the parent.
func refitMax(c *swt.Control) {
	limit, ok := capped[c]
	if !ok || c.IsDisposed() {
		return
	}
	_, hint, ok := sizeHints(c)
	if !ok {
		return
	}
	if room := roomFor(c); room >= 0 {
		limit = min(limit, max(room, 1))
	}
	w := int32(swt.DEFAULT)
	if x := c.GetSize().X; c.GetStyle()&swt.WRAP != 0 && x > 0 {
		w = x
	}
	want := int32(swt.DEFAULT)
	if int(c.ComputeSizeWHintHHintChanged(w, swt.DEFAULT, true).Y) > limit {
		const probe = 100
		trim := c.ComputeSizeWHintHHintChanged(w, probe, true).Y - probe
		want = max(int32(limit)-trim, 1)
	}
	if *hint != want {
		*hint = want
		// The grid keeps the size it measured for the old hint until the cache of the control is flushed.
		if l := c.GetParent().GetLayout(); l != nil {
			l.FlushCache(c)
		}
	}
}

// roomFor is the height of c's VBox less its margins, the spacing, and what the other rows need: a
// growing row its minimum, any other its own height. It is -1 for any other parent, or before the VBox has a size.
func roomFor(c *swt.Control) int {
	p := c.GetParent()
	if p == nil || p.GetLayout() == nil {
		return -1
	}
	box, ok := p.GetLayout().Impl().(*boxLayout)
	if !ok || box.row || p.GetClientArea().Height <= 0 {
		return -1
	}
	room := int(p.GetClientArea().Height - 2*box.MarginHeight - box.MarginTop - box.MarginBottom)
	rows := 0
	for _, s := range p.GetChildren() {
		d, _ := s.GetLayoutData().(*swt.GridData)
		if d != nil && d.Exclude {
			continue
		}
		if rows++; s == c || d == nil {
			continue
		}
		if d.GrabExcessVerticalSpace {
			room -= int(d.MinimumHeight)
		} else {
			room -= int(s.ComputeSizeWHintHHintChanged(swt.DEFAULT, d.HeightHint, false).Y)
		}
	}
	return room - int(box.VerticalSpacing)*max(rows-1, 0)
}

// fitCapped applies the MaxHeight of the children of a VBox that is about to lay out.
func fitCapped(p *swt.Composite) {
	if len(capped) == 0 {
		return
	}
	for _, c := range p.GetChildren() {
		refitMax(c)
	}
}

// refitUnder applies the MaxHeight of the controls inside p, for Relayout.
func refitUnder(p *swt.Composite) {
	for c := range capped {
		for q := c.GetParent(); q != nil; q = q.GetParent() {
			if q == p {
				refitMax(c)
				break
			}
		}
	}
}

// contentChanged lays the parents of c out on the next event-loop turn, once the native control has
// caught up with a change of its rows.
func contentChanged(c *swt.Control) {
	if a := appOf(c); a != nil {
		a.relayout(c)
	}
}
