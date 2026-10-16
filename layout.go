package gowt

import (
	"fmt"
	"strings"

	"github.com/haiodo/gowt/swt"
)

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
// Align zero is AlignStart. Grow, GrowX, GrowY, Span, Size and Place set the common fields one at a time.
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

// InRow sets the size of a child laid out by Row; Size does the same and also works in Grid and Form.
func InRow(c RowCell) Option {
	return Option{place: func(pl *placement) { pl.row, pl.rowc = true, c }}
}

func (c RowCell) data() *swt.RowData {
	d := swt.NewRowData()
	d.Width, d.Height = int32(c.Width), int32(c.Height)
	if c.Width == 0 {
		d.Width = swt.DEFAULT
	}
	if c.Height == 0 {
		d.Height = swt.DEFAULT
	}
	return d
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
	return Option{place: func(pl *placement) { pl.form, pl.formc = true, c }}
}

func (c FormCell) data() *swt.FormData {
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
	return d
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

// Placement options. They describe where a child sits and how it grows without naming the parent's
// layout, and add up: GrowX(), Span(2) give one cell that spans two columns and fills its width.
// When several set the same field, the later one wins. An option the parent's layout cannot use is
// reported to App.OnError naming the widget and the layout, and the widget keeps its natural size.

// Grow fills the grid cell and takes the free width and height of the Grid.
func Grow() Option {
	return gridOpt("Grow", func(c *GridCell) { c.Align, c.VAlign, c.GrowX, c.GrowY = AlignFill, AlignFill, true, true })
}

// GrowX fills the width of the grid cell and takes the free width of the Grid.
func GrowX() Option {
	return gridOpt("GrowX", func(c *GridCell) { c.Align, c.GrowX = AlignFill, true })
}

// GrowY fills the height of the grid cell and takes the free height of the Grid.
func GrowY() Option {
	return gridOpt("GrowY", func(c *GridCell) { c.VAlign, c.GrowY = AlignFill, true })
}

// Span makes the child cover cols columns of the Grid.
func Span(cols int) Option { return gridOpt("Span", func(c *GridCell) { c.SpanX = cols }) }

// Place aligns the child inside its grid cell: Place(AlignEnd, AlignCenter) puts it at the right, centred vertically.
func Place(h, v Align) Option {
	return gridOpt("Place", func(c *GridCell) { c.Align, c.VAlign = h, v })
}

// Cell sets every field of the child's grid cell at once; placement options after it change single fields.
func Cell(c GridCell) Option { return gridOpt("Cell", func(g *GridCell) { *g = c }) }

// Size asks for a width and height in points, zero keeps the natural one. It works in Grid, Row and Form.
func Size(w, h int) Option {
	return Option{place: func(pl *placement) {
		pl.cell.Width, pl.cell.Height = w, h
		pl.rowc.Width, pl.rowc.Height = w, h
		pl.formc.Width, pl.formc.Height = w, h
	}}
}

func gridOpt(name string, f func(*GridCell)) Option {
	return Option{place: func(pl *placement) {
		if pl.grid == "" {
			pl.grid = name
		}
		f(&pl.cell)
	}}
}

// placement collects the placement options of one widget; applyOpts turns it into layout data.
type placement struct {
	grid      string // the first option only a Grid can use, "" if none
	row, form bool   // InRow, Anchor given
	cell      GridCell
	rowc      RowCell
	formc     FormCell
}

// data returns the layout data for widget in a parent laid out by layout, a layoutName. With no
// layout yet the options choose the kind, and SetLayout checks it later (checkChildren).
func (pl *placement) data(widget, layout string) (any, error) {
	kind := layout
	if kind == "" {
		kind = pl.kind()
	}
	switch {
	case pl.grid != "" && kind != "Grid":
		return nil, misplaced(pl.grid, "a Grid", widget, layout)
	case pl.row && kind != "Row":
		return nil, misplaced("InRow", "a Row", widget, layout)
	case pl.form && kind != "Form":
		return nil, misplaced("Anchor", "a Form", widget, layout)
	}
	switch kind {
	case "Grid":
		return pl.cell.data(), nil
	case "Row":
		return pl.rowc.data(), nil
	case "Form":
		return pl.formc.data(), nil
	}
	return nil, misplaced("Size", "a Grid, Row or Form", widget, layout)
}

func (pl *placement) kind() string {
	switch {
	case pl.grid != "":
		return "Grid"
	case pl.row:
		return "Row"
	case pl.form:
		return "Form"
	}
	return "Grid"
}

func misplaced(opt, need, widget, layout string) error {
	if layout == "" {
		layout = "no layout"
	}
	return fmt.Errorf("gowt: %s on %s needs %s layout, its parent has %s", opt, widget, need, layout)
}

// layoutName names the layout of c as the facade does ("Grid", "Row", ...), "" for none.
func layoutName(c *swt.Composite) string {
	if c == nil || c.GetLayout() == nil {
		return ""
	}
	switch l := c.GetLayout().Impl().(type) {
	case *swt.GridLayout:
		return "Grid"
	case *swt.RowLayout:
		return "Row"
	case *swt.FormLayout:
		return "Form"
	case *swt.FillLayout:
		return "Fill"
	case *swt.StackLayout:
		return "Stack"
	default:
		return strings.TrimPrefix(fmt.Sprintf("%T", l), "*swt.")
	}
}

// checkChildren runs after SetLayout: a child placed for another layout would fail the next layout
// pass, so its layout data is dropped and the mismatch reported. Stack reads no layout data.
func checkChildren(c *swt.Composite) {
	have := layoutName(c)
	if have == "Stack" {
		return
	}
	for _, ch := range c.GetChildren() {
		var need string
		switch ch.GetLayoutData().(type) {
		case *swt.GridData:
			need = "Grid"
		case *swt.RowData:
			need = "Row"
		case *swt.FormData:
			need = "Form"
		case *swt.FillData:
			need = "Fill"
		}
		if need == "" || need == have {
			continue
		}
		reportOn(ch, fmt.Errorf("gowt: %s is placed for a %s layout, its parent's new layout is %s", ch.GetName(), need, have))
		ch.SetLayoutData(nil)
	}
}
