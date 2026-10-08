package gowt

import "github.com/haiodo/gowt/swt"

// Layout arranges the children of a Panel or Window.
type Layout interface{ layout() swt.LayoutLike }

// Align is a cell alignment inside a grid cell.
type Align int32

// Cell alignments.
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

// Row lays children out in a row (or column with Vertical) at their natural size, wrapping if Wrap is set.
// Fill stretches every child to the row's full height (width for Vertical), Center centers them.
type Row struct {
	Vertical, Wrap, Fill, Center bool
	Margin, Spacing              int
}

func (r Row) layout() swt.LayoutLike {
	l := swt.NewRowLayout()
	if r.Vertical {
		l.Type = swt.VERTICAL
	}
	l.Wrap, l.Fill, l.Center = r.Wrap, r.Fill, r.Center
	l.MarginWidth, l.MarginHeight = 0, 0
	l.MarginLeft, l.MarginTop, l.MarginRight, l.MarginBottom = int32(r.Margin), int32(r.Margin), int32(r.Margin), int32(r.Margin)
	l.Spacing = int32(r.Spacing)
	return l
}

// RowCell sets the size of one child of a Row; zero means natural size.
type RowCell struct{ Width, Height int }

// InRow sets the size of a child laid out by Row.
func InRow(c RowCell) Option {
	return Option{apply: func(w *swt.Control) {
		d := swt.NewRowData()
		d.Width, d.Height = int32(c.Width), int32(c.Height)
		if c.Width == 0 {
			d.Width = swt.DEFAULT
		}
		if c.Height == 0 {
			d.Height = swt.DEFAULT
		}
		w.SetLayoutData(d)
	}}
}

// Form positions each child by attaching its edges (see Anchor), which lets children stretch or follow siblings.
type Form struct{ Margin, Spacing int }

func (f Form) layout() swt.LayoutLike {
	l := swt.NewFormLayout()
	l.MarginWidth, l.MarginHeight = int32(f.Margin), int32(f.Margin)
	l.Spacing = int32(f.Spacing)
	return l
}

// Edge says where one side of a Form child sits. The zero Edge leaves the side free.
type Edge struct {
	kind   uint8
	pct    int
	offset int
	to     Widget
}

const (
	edgeFree uint8 = iota
	edgePercent
	edgeBeside
	edgeSame
)

// Percent attaches the side to pct percent of the Form's width or height, plus offset points.
// Percent(0, 5) is 5 from the left/top, Percent(100, -5) is 5 from the right/bottom.
func Percent(pct, offset int) Edge { return Edge{kind: edgePercent, pct: pct, offset: offset} }

// Beside attaches the side to the opposite side of w, gap points away: a left edge sits right of w.
func Beside(w Widget, gap int) Edge { return Edge{kind: edgeBeside, offset: gap, to: w} }

// Same attaches the side to the same side of w (left to left, top to top, ...), offset points away.
func Same(w Widget, offset int) Edge { return Edge{kind: edgeSame, offset: offset, to: w} }

func (e Edge) attachment(side int32) *swt.FormAttachment {
	switch e.kind {
	case edgePercent:
		return swt.NewFormAttachmentNumeratorOffset(int32(e.pct), int32(e.offset))
	case edgeBeside:
		return swt.NewFormAttachmentControlOffset(e.to.control(), int32(e.offset))
	case edgeSame:
		return swt.NewFormAttachmentControlOffsetAlignment(e.to.control(), int32(e.offset), side)
	}
	return nil
}

// FormCell places one child of a Form. Width and Height are hints in points, zero is natural size.
// A child with no edge set sticks to the top left.
type FormCell struct {
	Left, Right, Top, Bottom Edge
	Width, Height            int
}

// Anchor sets the position of a child laid out by Form. Edges that refer to a sibling need that sibling to exist already.
func Anchor(c FormCell) Option {
	return Option{apply: func(w *swt.Control) {
		d := swt.NewFormData()
		d.Left, d.Right = c.Left.attachment(swt.LEFT), c.Right.attachment(swt.RIGHT)
		d.Top, d.Bottom = c.Top.attachment(swt.TOP), c.Bottom.attachment(swt.BOTTOM)
		d.Width, d.Height = int32(c.Width), int32(c.Height)
		if c.Width == 0 {
			d.Width = swt.DEFAULT
		}
		if c.Height == 0 {
			d.Height = swt.DEFAULT
		}
		w.SetLayoutData(d)
	}}
}

// Stack shows one child at a time, all at the full size of the container; choose it with Panel.ShowTop.
type Stack struct{ Margin int }

func (s Stack) layout() swt.LayoutLike {
	l := swt.NewStackLayout()
	l.MarginWidth, l.MarginHeight = int32(s.Margin), int32(s.Margin)
	return l
}

// ShowTop makes w, a child of p, the visible one. p must have a Stack layout.
func (p *panel) ShowTop(w Widget) {
	if p.stack == nil {
		panic("gowt: ShowTop on a Panel without a Stack layout")
	}
	p.stack.TopControl = w.control()
	p.c.Layout()
}
