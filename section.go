package gowt

import "github.com/haiodo/gowt/swt"

// sectionRadius is the corner radius of a Section card in points.
const sectionRadius = 12

// SectionView is a grouped "inset" section as in macOS System Settings: rows on a rounded card in the
// platform's grouped background colour, separated by lines. A heading is a Label above it.
type SectionView struct {
	base
	c    *swt.Composite
	rows int
}

// Section creates a grouped section whose children are its rows, separated by lines; a row fills
// the width unless it is placed otherwise. Pass GrowX() to stretch the section in its parent.
// docs/theme-look.md tells how each OS fills the card.
// Options: CardBorder, SectionFill, common only, children.
func Section(opts ...Option) Option {
	return &node{"Section", func(h host) Widget {
		c := swt.NewCompositeParentStyle(h.into(), resolve(swt.NONE, opts))
		l := swt.NewGridLayout()
		l.MarginWidth, l.MarginHeight = 12, 4
		l.HorizontalSpacing, l.VerticalSpacing = 0, 4
		c.SetLayout(l)
		fill, custom := cardFillColor(opts)
		border, _ := cardStrokeColor(opts)
		installCard(c, fill, custom, border)
		s := &SectionView{base: base{&c.Control}, c: c}
		finish(s, 0, opts)
		return s
	}}
}

// cardBorder is the CardBorder option: a one-point rounded outline inside the card edge.
type cardBorder struct{ c Paint }

func (cardBorder) option() {}

// CardBorder draws a one-point rounded outline on the same rounded edge as the Section card's fill.
// An Adaptive colour follows the appearance.
func CardBorder(c Paint) Option { return cardBorder{c} }

// sectionFill is the SectionFill option: the card colour instead of the grouped background.
type sectionFill struct{ c RGB }

func (sectionFill) option() {}

// SectionFill gives a Section card this colour instead of the grouped background; a theme switch
// keeps it.
func SectionFill(c RGB) Option { return sectionFill{c} }

// cardStrokeColor returns the CardBorder colour of opts, false without it.
func cardStrokeColor(opts []Option) (Paint, bool) {
	for _, o := range opts {
		if b, ok := o.(cardBorder); ok {
			return b.c, true
		}
	}
	return nil, false
}

// cardFillColor returns the SectionFill colour of opts, false without it.
func cardFillColor(opts []Option) (RGB, bool) {
	for _, o := range opts {
		if f, ok := o.(sectionFill); ok {
			return f.c, true
		}
	}
	return RGB{}, false
}

// Add appends rows.
func (s *SectionView) Add(rows ...Option) {
	add(s, rows)
	if !s.c.IsDisposed() {
		s.c.Layout()
	}
}

func (s *SectionView) into() *swt.Composite { return s.c }

// added puts a separator above every row but the first.
func (s *SectionView) added(w Widget) {
	c := w.control()
	if c.GetLayoutData() == nil {
		c.SetLayoutData(swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.FILL, swt.CENTER, true, false))
	}
	if s.rows > 0 {
		sep := swt.NewLabel(s.c, swt.SEPARATOR|swt.HORIZONTAL)
		sep.SetLayoutData(swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.FILL, swt.CENTER, true, false))
		sep.MoveAbove(c)
	}
	s.rows++
}

// Unwrap returns the underlying composite.
func (s *SectionView) Unwrap() *swt.Composite { return s.c }

// AsComposite returns the underlying composite, as Panel does, for the look package.
func (s *SectionView) AsComposite() *swt.Composite { return s.c }

// installCard gives c the card colour, inherited by rows and labels, rounds its corners, and takes the
// colour again when the system theme changes; a custom fill keeps its colour. Where the view cannot be
// clipped (roundCard), the corners outside the rounded rectangle are painted in the parent's colour.
func installCard(c *swt.Composite, fill RGB, custom bool, border Paint) {
	d := c.GetDisplay()
	var col *swt.Color
	if custom {
		col = swt.NewColorRedGreenBlue(int32(fill.R), int32(fill.G), int32(fill.B))
	} else {
		col = cardColor(d)
	}
	c.SetBackgroundWithColor(col)
	c.SetBackgroundMode(swt.INHERIT_DEFAULT)
	rounded := roundCard(c)
	nativeBorder := border != nil && rounded && strokeCard(c, border.in(cardDark(c)))
	if (border != nil && !nativeBorder) || !rounded {
		c.AddPaintListener(painter(func(e *swt.PaintEvent) {
			drawCard(e, c, border)
		}))
	}
	if !custom || border != nil {
		t := &cardTheme{c: c, col: col, border: border, custom: custom, nativeBorder: nativeBorder}
		d.AddListener(swt.Settings, t)
		c.AddListener(swt.Dispose, t)
	}
}

type cardTheme struct {
	c            *swt.Composite
	col          *swt.Color // the background colour this listener set last
	border       Paint
	custom       bool
	nativeBorder bool
}

// HandleEvent follows the system theme with the colour installCard set, unless the app has given the
// card a Background of its own since: the common option stays across a theme switch.
func (t *cardTheme) HandleEvent(e *swt.Event) {
	if e.Type == swt.Dispose {
		t.c.GetDisplay().RemoveListener(swt.Settings, t)
		return
	}
	if !t.custom && sameColour(t.c.GetBackground(), t.col) {
		col := cardColor(t.c.GetDisplay())
		t.col = col
		t.c.SetBackgroundWithColor(col)
	}
	if t.border == nil {
		return
	}
	col := t.border.in(cardDark(t.c))
	if t.nativeBorder {
		strokeCard(t.c, col)
	}
	t.c.Redraw()
}

// sameColour reports whether a and b have the same RGB.
func sameColour(a, b *swt.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ra, rb := a.GetRGB(), b.GetRGB()
	return ra.Red == rb.Red && ra.Green == rb.Green && ra.Blue == rb.Blue
}

// cardDark is the appearance the Paint of a card resolves under, as setPaint resolves it.
func cardDark(c *swt.Composite) bool {
	if a := appOf(c); a != nil {
		return a.dark
	}
	return systemDark()
}

// drawCard fills the rounded card and, with a CardBorder, strokes it on the same path: the outline
// hugs the fill on every port, whatever the port does with the edge of the view.
func drawCard(e *swt.PaintEvent, c *swt.Composite, border Paint) {
	r := c.GetClientArea()
	g := e.Gc
	g.SetAntialias(swt.ON)
	if p := c.GetParent().GetBackground(); p.GetAlpha() != 0 { // GC fills ignore alpha: black corners
		g.SetBackground(p)
		g.FillRectangle(0, 0, r.Width, r.Height)
	}
	g.SetBackground(c.GetBackground())
	g.FillRoundRectangle(0, 0, r.Width, r.Height, 2*sectionRadius, 2*sectionRadius)
	if border == nil {
		return
	}
	stroke := border.in(cardDark(c)).color()
	g.SetForeground(stroke)
	g.SetLineWidth(2) // the port clips the outer point of a stroke: two leave one inside the edge
	g.DrawRoundRectangle(1, 1, r.Width-2, r.Height-2, 2*sectionRadius, 2*sectionRadius)
	stroke.Dispose()
}
