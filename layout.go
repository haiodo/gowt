package gowt

import "github.com/haiodo/gowt/swt"

// Layout arranges the children of a Panel or Window.
type Layout interface{ layout() swt.LayoutLike }

// Align is a cell alignment inside a grid cell.
type Align int32

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignFill
)

func (a Align) swt() int32 {
	return [...]int32{swt.BEGINNING, swt.CENTER, swt.END, swt.FILL}[a]
}

// Fill lays children out in one row or column of equal size.
type Fill struct{ Vertical bool }

func (f Fill) layout() swt.LayoutLike {
	if f.Vertical {
		return swt.NewFillLayoutType(swt.VERTICAL)
	}
	return swt.NewFillLayout()
}

// Grid lays children out in Columns columns, left to right, top to bottom.
// Margin and Spacing are in points; zero means none (SWT's own defaults are 5 and 5).
type Grid struct {
	Columns    int
	EqualWidth bool
	Margin     int
	Spacing    int
}

func (g Grid) layout() swt.LayoutLike {
	l := swt.NewGridLayoutNumColumnsMakeColumnsEqualWidth(int32(max(g.Columns, 1)), g.EqualWidth)
	l.MarginWidth, l.MarginHeight = int32(g.Margin), int32(g.Margin)
	l.HorizontalSpacing, l.VerticalSpacing = int32(g.Spacing), int32(g.Spacing)
	return l
}

// GridCell places one child inside a Grid. Zero value: natural size, top left, one cell.
// Align zero is AlignStart.
type GridCell struct {
	Align, VAlign Align
	GrowX, GrowY  bool
	SpanX, SpanY  int
	Width, Height int
}

func (c GridCell) data() *swt.GridData {
	d := swt.NewGridData()
	d.HorizontalAlignment, d.VerticalAlignment = c.Align.swt(), c.VAlign.swt()
	d.GrabExcessHorizontalSpace, d.GrabExcessVerticalSpace = c.GrowX, c.GrowY
	d.HorizontalSpan, d.VerticalSpan = int32(max(c.SpanX, 1)), int32(max(c.SpanY, 1))
	d.WidthHint, d.HeightHint = int32(c.Width), int32(c.Height)
	if c.Width == 0 {
		d.WidthHint = swt.DEFAULT
	}
	if c.Height == 0 {
		d.HeightHint = swt.DEFAULT
	}
	return d
}
