package gowt

import "github.com/haiodo/gowt/swt"

// sectionRadius is the corner radius of a Section card in points.
const sectionRadius = 12

// Section is a grouped "inset" section as in macOS System Settings: rows on a rounded card in the
// platform's grouped background colour, separated by lines. A heading is a Label above it.
type Section struct {
	base
	c    *swt.Composite
	rows int
}

// Section adds a grouped section with a one-column Grid of its own; pass GrowX() to stretch it in a
// Grid parent. docs/theme-look.md tells how each OS fills the card.
// Options: common only.
func (p *panel) Section(opts ...Option) *Section {
	c := swt.NewCompositeParentStyle(p.c, resolve(swt.NONE, opts))
	l := swt.NewGridLayout()
	l.MarginWidth, l.MarginHeight = 12, 4
	l.HorizontalSpacing, l.VerticalSpacing = 0, 4
	c.SetLayout(l)
	applyOpts(&c.Control, opts)
	installCard(c)
	return &Section{base: base{&c.Control}, c: c}
}

// Row adds a row, after a separator unless it is the first. The row is a Panel without a layout:
// give it one with SetLayout.
func (s *Section) Row() *Panel {
	if s.rows > 0 {
		sep := swt.NewLabel(s.c, swt.SEPARATOR|swt.HORIZONTAL)
		sep.SetLayoutData(swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.FILL, swt.CENTER, true, false))
	}
	s.rows++
	c := swt.NewCompositeParentStyle(s.c, swt.NONE)
	c.SetLayoutData(swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.FILL, swt.CENTER, true, false))
	return &Panel{newPanel(c)}
}

// Unwrap returns the underlying composite.
func (s *Section) Unwrap() *swt.Composite { return s.c }

// AsComposite returns the underlying composite, as Panel does, for the look package.
func (s *Section) AsComposite() *swt.Composite { return s.c }

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
		g.SetBackground(c.GetParent().GetBackground())
		g.FillRectangle(0, 0, r.Width, r.Height)
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
