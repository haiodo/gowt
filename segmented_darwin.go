package gowt

import (
	"math"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// segments is an NSSegmentedControl inside the composite, laid out by segLayout.
type segments struct {
	n int
	c *swt.Composite
	v *cocoa.Segmented
}

func newSegments(s *SegmentedView, segs []Segment) *segments {
	v := cocoa.NewSegmented(s.c.View, len(segs))
	for i, g := range segs {
		var img *cocoa.NSImage
		if g.Image != nil {
			img = g.Image.i.Handle
		}
		v.SetSegment(i, g.Text, img, g.Tooltip)
	}
	v.Clicked = s.pressed
	l := &segLayout{Layout: swt.NewLayout(), v: v}
	l.SetImpl_(l)
	s.c.SetLayout(l)
	s.c.AddDisposeListener(disposeFunc(v.Release))
	return &segments{n: len(segs), c: s.c, v: v}
}

func (s *segments) count() int         { return s.n }
func (s *segments) selectAt(i int)     { s.v.Select(i) }
func (s *segments) setMomentary()      { s.v.SetMomentary(true) }
func (s *segments) setEnabled(on bool) { s.v.SetEnabled(on) }
func (s *segments) setSmall()          { sizing(&s.c.Control, s.v.SetSmall) }

// segLayout asks for the control's fitting size and gives it the composite's width, centred in height.
type segLayout struct {
	*swt.Layout
	v *cocoa.Segmented
}

func (l *segLayout) ComputeSize_(_ *swt.Composite, wHint, hHint int32, _ bool) *swt.Point {
	f := l.v.Fit()
	w, h := int32(math.Ceil(f.Width)), int32(math.Ceil(f.Height))
	if wHint != swt.DEFAULT {
		w = wHint
	}
	if hHint != swt.DEFAULT {
		h = hHint
	}
	return swt.NewPoint(w, h)
}

func (l *segLayout) LayoutFn_(c *swt.Composite, _ bool) {
	r := c.GetClientArea()
	h := min(int32(math.Ceil(l.v.Fit().Height)), r.Height)
	l.v.SetFrame(cocoa.NSRect{Y: float64((r.Height - h) / 2), Width: float64(r.Width), Height: float64(h)})
}
