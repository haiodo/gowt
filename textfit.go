package gowt

import (
	"fmt"

	"github.com/haiodo/gowt/swt"
)

// SetHeight changes the height the layout gives the control, in points, as Size did at creation; 0
// goes back to the natural height. The parents are laid out on the next event-loop turn or at App.Flush,
// as after SetFont. It needs a parent laid out
// by a VBox, HBox, Grid, Flow or Form, like Size.
func (w *base) SetHeight(h int) {
	c := w.ctl
	_, hint, ok := sizeHints(c)
	if !ok {
		reportOn(c, fmt.Errorf("gowt: SetHeight on %s needs a parent laid out by a VBox, HBox, Grid, Flow or Form", nameOf(c)))
		return
	}
	*hint = int32(h)
	if h <= 0 {
		*hint = swt.DEFAULT
	}
	if a := appOf(c); a != nil {
		a.relayout(c)
	}
}

// sizeHints returns the width and height hints in the layout data of c, which a control with none gets
// from its parent's layout; ok is false when that layout reads no such data.
func sizeHints(c *swt.Control) (w, h *int32, ok bool) {
	switch d := c.GetLayoutData().(type) {
	case *swt.GridData:
		return &d.WidthHint, &d.HeightHint, true
	case *swt.RowData:
		return &d.Width, &d.Height, true
	case *swt.FormData:
		return &d.Width, &d.Height, true
	case nil:
		p := c.GetParent()
		if p == nil || p.GetLayout() == nil {
			return nil, nil, false
		}
		switch p.GetLayout().Impl().(type) {
		case swt.GridLayoutLike:
			c.SetLayoutData(swt.NewGridData())
		case *swt.RowLayout:
			c.SetLayoutData(swt.NewRowData())
		default:
			return nil, nil, false
		}
		return sizeHints(c)
	}
	return nil, nil, false
}

// autoGrow is the state of a Text that follows its lines.
type autoGrow struct {
	min, max int
	asked    int // the height asked of the layout last
	extra    int // vertical space added to the inset to centre text shorter than the field
}

// fitTo returns the height of the field for text that needs need points, and the space to add above
// and below the inset to centre the text when the field is higher than it needs.
func (g *autoGrow) fitTo(need int) (h, extra int) {
	h = max(g.min, need)
	if g.max != 0 {
		h = min(h, g.max)
	}
	return h, max(h-need, 0) / 2
}

// SetInset puts x points between the text and the left and right edge of a multi-line Text and y points
// above and below it; the hint moves with the text. macOS has one inset for both sides (textContainerInset),
// so the facade does the same; the default is 0. Inset does it at creation.
func (t *TextView) SetInset(x, y int) {
	if !t.multi() {
		reportOn(t.t, fmt.Errorf("gowt: SetInset applies to a multi-line Text"))
		return
	}
	t.inset = [2]int{max(x, 0), max(y, 0)}
	t.applyInset()
	t.fit()
}

func (t *TextView) multi() bool { return t.t.GetStyle()&swt.MULTI != 0 }

func (t *TextView) applyInset() {
	extra := 0
	if t.grow != nil {
		extra = t.grow.extra
	}
	nativeInset(t.t, t.inset[0], t.inset[1]+extra)
	t.t.Redraw()
}

// ContentHeight returns the height in points the text needs when the field is width points wide:
// the visual lines after wrapping, with the vertical inset, one line at least. It is the number to
// pass to SetHeight for a field that follows its text; AutoGrow does that itself. On a single-line
// Text it returns the natural height.
func (t *TextView) ContentHeight(width int) int {
	if !t.multi() {
		return int(preferred(&t.t.Control).Y)
	}
	return nativeContentHeight(t.t, width, t.inset[0], t.inset[1])
}

// SetAutoGrow makes a multi-line Text ask its layout for the height of its text, as ContentHeight
// reports it, but not less than minHeight and, unless maxHeight is 0, not more than maxHeight
// points; beyond that the text scrolls. Text shorter than minHeight is centred vertically, so a
// one-line field looks as the single line of a native field does. The height follows every edit and
// every change of the width. maxHeight below minHeight is taken as minHeight.
func (t *TextView) SetAutoGrow(minHeight, maxHeight int) {
	if !t.multi() {
		reportOn(t.t, fmt.Errorf("gowt: AutoGrow applies to a multi-line Text"))
		return
	}
	g := &autoGrow{min: max(minHeight, 0), max: max(maxHeight, 0)}
	if g.max != 0 {
		g.max = max(g.max, g.min)
	}
	first := t.grow == nil
	t.grow = g
	if first {
		// Without a width hint the layout asks for the width of the longest line and a long text
		// pushes the field out of its window; with 1 the width is what the layout has left over.
		if w, _, ok := sizeHints(&t.t.Control); ok && *w == swt.DEFAULT {
			*w = 1
		}
		t.t.AddModifyListener(&modifier{t.fit})
		t.t.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { t.fit() }))
	}
	t.fit()
}

// fit sets the height of an auto-grow field for its text and the width it has.
func (t *TextView) fit() {
	g := t.grow
	w := nativeWidth(t.t)
	if g == nil || w <= 0 || t.t.IsDisposed() {
		return
	}
	h, extra := g.fitTo(t.ContentHeight(w))
	if extra != g.extra {
		g.extra = extra
		t.applyInset()
	}
	if h != g.asked {
		g.asked = h
		t.SetHeight(h)
	}
}

// Inset puts x points between the text and the left and right edge of a multi-line Text and y points
// above and below it, as SetInset does.
func Inset(x, y int) Option {
	return hook(func(w Widget) error {
		if t, ok := w.(*TextView); ok && t.multi() {
			t.SetInset(x, y)
			return nil
		}
		return fmt.Errorf("gowt: Inset applies to a multi-line Text, not to %s", nameOf(w.control()))
	})
}

// AutoGrow makes a multi-line Text as high as its text, between minHeight and maxHeight points
// (0 is no limit), as SetAutoGrow does. It needs a parent laid out by a VBox, HBox, Grid, Flow or Form.
func AutoGrow(minHeight, maxHeight int) Option {
	return hook(func(w Widget) error {
		if t, ok := w.(*TextView); ok && t.multi() {
			t.SetAutoGrow(minHeight, maxHeight)
			return nil
		}
		return fmt.Errorf("gowt: AutoGrow applies to a multi-line Text, not to %s", nameOf(w.control()))
	})
}
