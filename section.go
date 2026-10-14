package gowt

import "github.com/haiodo/gowt/swt"

// sectionRadius is the corner radius of a Section card in points.
const sectionRadius = 12

// Section is a grouped "inset" section as in macOS System Settings: rows on a rounded card in the
// platform's grouped background colour, separated by lines. A heading is a Label above it.
type Section struct {
	c    *swt.Composite
	rows int
}

// Section adds a grouped section with a one-column Grid of its own; pass Cell(GridCell{GrowX: true,
// Align: AlignFill}) to stretch it in a Grid parent. docs/theme-look.md tells how each OS fills the card.
func (p *panel) Section(opts ...Option) *Section {
	c := swt.NewCompositeParentStyle(p.c, resolve(swt.NONE, opts))
	l := swt.NewGridLayout()
	l.MarginWidth, l.MarginHeight = 12, 4
	l.HorizontalSpacing, l.VerticalSpacing = 0, 4
	c.SetLayout(l)
	applyOpts(&c.Control, opts)
	installCard(c)
	return &Section{c: c}
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
	return &Panel{&panel{c: c}}
}

func (s *Section) control() *swt.Control { return &s.c.Control }

// Unwrap returns the underlying composite.
func (s *Section) Unwrap() *swt.Composite { return s.c }

// AsComposite returns the underlying composite, as Panel does, for the look package.
func (s *Section) AsComposite() *swt.Composite { return s.c }

type eventFunc func(*swt.Event)

func (f eventFunc) HandleEvent(e *swt.Event) { f(e) }
