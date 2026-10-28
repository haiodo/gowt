package gowt

import (
	"fmt"
	"slices"

	"github.com/haiodo/gowt/swt"
)

// Segment is one segment of a Segmented: a short Text, an Image, or both, and the Tooltip shown over
// it. A segment without Text is named to screen readers by its Tooltip.
type Segment struct {
	Text    string
	Image   *Image
	Tooltip string
}

// SegmentedView is a segmented control.
type SegmentedView struct {
	base
	c         *swt.Composite
	seg       *segments
	index     int
	momentary bool
	handlers  []*func(int)
}

// Segmented creates a segmented control: segments side by side of which one is selected, for a choice
// among a few (Fixed, Hug, Fill), or with Momentary a row of buttons. The segments share the width
// equally; GrowX stretches the control. macOS: NSSegmentedControl; GTK: toggle buttons joined like a
// "linked" box; Windows: toggle buttons painted as one WinUI segmented control. No segment is selected
// until Selected, Select or a click.
// Options: Momentary, Small, Selected, OnSelect, Bind (int).
func Segmented(segments []Segment, opts ...Option) Option {
	return leaf("Segmented", 0, opts, func(p *swt.Composite) Widget {
		c := swt.NewCompositeParentStyle(p, swt.NONE)
		s := &SegmentedView{base: base{&c.Control}, c: c, index: -1}
		s.seg = newSegments(s, segments)
		return s
	})
}

// Index returns the selected segment, -1 when none is selected and always for a Momentary control.
func (s *SegmentedView) Index() int { return s.index }

// Select selects segment i without running OnSelect; -1 clears the selection. An index out of range
// and a Momentary control are left as they are.
func (s *SegmentedView) Select(i int) {
	if s.momentary || i < -1 || i >= s.seg.count() {
		return
	}
	s.index = i
	s.seg.selectAt(i)
}

// OnSelect runs f with the segment the user selected; a click on the selected one runs nothing. With
// Momentary it runs with the segment pressed, every time.
func (s *SegmentedView) OnSelect(f func(index int)) { s.handlers = append(s.handlers, &f) }

// SetEnabled enables or dims the control. Disabled() does the same at creation.
func (s *SegmentedView) SetEnabled(v bool) {
	s.base.SetEnabled(v)
	s.seg.setEnabled(v)
}

// Unwrap returns the composite that holds the segments.
func (s *SegmentedView) Unwrap() *swt.Composite { return s.c }

// pressed is a click on segment i.
func (s *SegmentedView) pressed(i int) {
	if i < 0 || i >= s.seg.count() {
		return
	}
	if !s.momentary {
		if i == s.index {
			return
		}
		s.index = i
	}
	for _, f := range slices.Clone(s.handlers) {
		(*f)(i)
	}
}

// listen is OnSelect with a removal, for Bind.
func (s *SegmentedView) listen(f func()) func() {
	g := func(int) { f() }
	p := &g
	s.handlers = append(s.handlers, p)
	return func() { s.handlers = slices.DeleteFunc(s.handlers, func(h *func(int)) bool { return h == p }) }
}

// Momentary makes the segments of a Segmented buttons that run OnSelect and do not stay selected, as
// the "+" and "-" under a list.
func Momentary() Option {
	return hook(func(w Widget) error {
		s, ok := w.(*SegmentedView)
		if !ok {
			return fmt.Errorf("gowt: Momentary applies to Segmented, not to %s", nameOf(w.control()))
		}
		s.Select(-1)
		s.momentary = true
		s.seg.setMomentary()
		return nil
	})
}

// Small gives a Segmented, a Button, a single-line Text or a Number the small control size, as in
// Xcode's inspector: macOS NSControlSizeSmall with its 11-point font; Windows and Linux the font
// smaller by the same ratio, and a field the compact height (24 dips on Windows, 24 px on Linux).
func Small() Option {
	return hook(func(w Widget) error {
		switch v := w.(type) {
		case *SegmentedView:
			v.seg.setSmall()
		case *ButtonView:
			v.SetFont(smallFont)
			setButtonSmall(v.b)
		case *TextView, *NumberView:
			return withAffix(w, "Small", func(a *affix) { a.small = true })
		default:
			return fmt.Errorf("gowt: Small applies to Segmented, Button, a single-line Text and Number, not to %s", nameOf(w.control()))
		}
		return nil
	})
}

// smallFont is the font of Small: 11 to 13, the small and regular control fonts of macOS.
var smallFont = Font{Scale: 11.0 / 13}
