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
// Options: common only, children.
func Section(opts ...Option) Option {
	return &node{"Section", func(h host) Widget {
		c := swt.NewCompositeParentStyle(h.into(), resolve(swt.NONE, opts))
		l := swt.NewGridLayout()
		l.MarginWidth, l.MarginHeight = 12, 4
		l.HorizontalSpacing, l.VerticalSpacing = 0, 4
		c.SetLayout(l)
		installCard(c)
		s := &SectionView{base: base{&c.Control}, c: c}
		finish(s, 0, opts)
		return s
	}}
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

// installCard gives c the card colour, inherited by rows and labels, paints the corners outside the
// rounded rectangle in the parent's colour, and takes the colour again when the system theme changes.
func installCard(c *swt.Composite) {
	d := c.GetDisplay()
	c.SetBackgroundWithColor(cardColor(d))
	c.SetBackgroundMode(swt.INHERIT_DEFAULT)
	c.AddPaintListener(painter(func(e *swt.PaintEvent) {
		r := c.GetClientArea()
		g := e.Gc
		g.SetAntialias(swt.ON)
		// GC fills ignore alpha: a transparent parent (look's glass panes) gave black corners. The corners
		// then stay in the card colour, square; rounding them there would need a clip of the view itself.
		if p := c.GetParent().GetBackground(); p.GetAlpha() != 0 {
			g.SetBackground(p)
			g.FillRectangle(0, 0, r.Width, r.Height)
		}
		g.SetBackground(c.GetBackground())
		g.FillRoundRectangle(0, 0, r.Width, r.Height, 2*sectionRadius, 2*sectionRadius)
	}))
	t := &cardTheme{c}
	d.AddListener(swt.Settings, t)
	c.AddListener(swt.Dispose, t)
}

type cardTheme struct{ c *swt.Composite }

func (t *cardTheme) HandleEvent(e *swt.Event) {
	if e.Type == swt.Dispose {
		t.c.GetDisplay().RemoveListener(swt.Settings, t)
		return
	}
	t.c.SetBackgroundWithColor(cardColor(t.c.GetDisplay()))
}
