package gowt

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/haiodo/gowt/swt"
)

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

// VBox creates a container that stacks its children top to bottom, Gap apart. A child takes its
// natural size unless GrowX, GrowY or Grow stretch it; a Spacer takes the free height.
// Options: Gap, Padding, Border, children.
func VBox(opts ...Option) Option {
	return box("VBox", opts, func(gap, pad int, _ int32) swt.LayoutLike { return newBox(false, gap, pad) })
}

// HBox creates a container that lines its children up left to right, Gap apart, one column per
// shown child. GrowX gives a child the free width; a Spacer takes it and pushes the rest to the end.
// Options: Gap, Padding, Border, children.
func HBox(opts ...Option) Option {
	return box("HBox", opts, func(gap, pad int, _ int32) swt.LayoutLike { return newBox(true, gap, pad) })
}

// Grid creates a container that lays its children out in columns columns, left to right, top to
// bottom, Gap apart in both directions: Grid(2, Label("Name"), Text(GrowX())) is a form row.
// Options: Gap, Padding, Border, children.
func Grid(columns int, opts ...Option) Option {
	return box("Grid", opts, func(gap, pad int, _ int32) swt.LayoutLike {
		return gridLayout(swt.NewGridLayoutNumColumnsMakeColumnsEqualWidth(int32(max(columns, 1)), false), gap, pad)
	})
}

// Flow creates a container that puts its children in a row at their natural size and wraps them to
// the next row when the width runs out; Vertical() makes it a column that wraps to the next column.
// Size sets the size of a child.
// Options: Vertical, Gap, Padding, Border, children.
func Flow(opts ...Option) Option {
	return box("Flow", opts, func(gap, pad int, style int32) swt.LayoutLike {
		l := swt.NewRowLayout()
		if style&swt.VERTICAL != 0 {
			l.Type = swt.VERTICAL
		}
		l.Wrap = true
		l.MarginWidth, l.MarginHeight = 0, 0
		l.MarginLeft, l.MarginTop, l.MarginRight, l.MarginBottom = int32(pad), int32(pad), int32(pad), int32(pad)
		l.Spacing = int32(gap)
		return l
	})
}

// Fill creates a container whose children share its whole width in equal parts, or its height
// with Vertical(); with one child, the child fills the container.
// Options: Vertical, Gap, Padding, Border, children.
func Fill(opts ...Option) Option {
	return box("Fill", opts, func(gap, pad int, style int32) swt.LayoutLike {
		l := swt.NewFillLayout()
		if style&swt.VERTICAL != 0 {
			l.Type = swt.VERTICAL
		}
		l.MarginWidth, l.MarginHeight = int32(pad), int32(pad)
		l.Spacing = int32(gap)
		return l
	})
}

// Form creates a container that positions each child by attaching its edges (see Anchor), which
// lets children stretch or follow siblings. Gap is the space Form leaves between attached siblings.
// Options: Gap, Padding, Border, children.
func Form(opts ...Option) Option {
	return box("Form", opts, func(gap, pad int, _ int32) swt.LayoutLike {
		l := swt.NewFormLayout()
		l.MarginWidth, l.MarginHeight = int32(pad), int32(pad)
		l.Spacing = int32(gap)
		return l
	})
}

// box returns the node of a Panel laid out by lay(gap, padding, style bits).
func box(name string, opts []Option, lay func(gap, pad int, style int32) swt.LayoutLike) Option {
	return &node{name, func(h host) Widget {
		style := resolve(swt.NONE, opts)
		// VERTICAL is V_SCROLL to a composite: it only chooses the layout's direction.
		c := swt.NewCompositeParentStyle(h.into(), style&^swt.VERTICAL)
		gap, pad := spacingOf(opts, 0)
		c.SetLayout(lay(gap, pad, style))
		p := &Panel{newPanel(c)}
		allow := sBorder
		if name == "Flow" || name == "Fill" {
			allow |= sVertical
		}
		finish(p, allow, opts)
		return p
	}}
}

func gridLayout(l *swt.GridLayout, gap, pad int) *swt.GridLayout {
	l.MarginWidth, l.MarginHeight = int32(pad), int32(pad)
	l.HorizontalSpacing, l.VerticalSpacing = int32(gap), int32(gap)
	return l
}

// boxLayout is the GridLayout of a VBox, or of an HBox, which gets one column per shown child at
// each layout pass, so that Add, Dispose and SetVisible need not count columns.
type boxLayout struct {
	*swt.GridLayout
	row bool
}

func newBox(row bool, gap, pad int) *boxLayout {
	l := &boxLayout{gridLayout(swt.NewGridLayout(), gap, pad), row}
	l.SetImpl_(l)
	return l
}

func (l *boxLayout) ComputeSize_(c *swt.Composite, wHint, hHint int32, flush bool) *swt.Point {
	l.columns(c)
	return l.GridLayout.ComputeSize(c, wHint, hHint, flush)
}

func (l *boxLayout) LayoutFn_(c *swt.Composite, flush bool) {
	l.columns(c)
	l.GridLayout.LayoutFn(c, flush)
}

func (l *boxLayout) columns(c *swt.Composite) {
	if !l.row {
		return
	}
	n := 0
	for _, ch := range c.GetChildren() {
		if d, ok := ch.GetLayoutData().(*swt.GridData); ok && d.Exclude {
			continue
		}
		n++
	}
	l.NumColumns = int32(max(n, 1))
}

// GridCell places one child inside a Grid. Zero value: natural size, top left, one cell.
// Align zero is AlignStart. Grow, GrowX, GrowY, Span, Size, Place and Indent set the common fields one at a time.
type GridCell struct {
	Align, VAlign    Align
	GrowX, GrowY     bool
	SpanX, SpanY     int
	Width, Height    int
	IndentX, IndentY int // points the child sits right of and below the cell's top left
}

func (c GridCell) data() *swt.GridData {
	d := swt.NewGridData()
	d.HorizontalAlignment, d.VerticalAlignment = c.Align.swt(), c.VAlign.swt()
	d.GrabExcessHorizontalSpace, d.GrabExcessVerticalSpace = c.GrowX, c.GrowY
	d.HorizontalSpan, d.VerticalSpan = int32(max(c.SpanX, 1)), int32(max(c.SpanY, 1))
	d.WidthHint, d.HeightHint = int32(c.Width), int32(c.Height)
	d.HorizontalIndent, d.VerticalIndent = int32(c.IndentX), int32(c.IndentY)
	if c.Width == 0 {
		d.WidthHint = swt.DEFAULT
	}
	if c.Height == 0 {
		d.HeightHint = swt.DEFAULT
	}
	return d
}

func rowData(w, h int) *swt.RowData {
	d := swt.NewRowData()
	d.Width, d.Height = int32(w), int32(h)
	if w == 0 {
		d.Width = swt.DEFAULT
	}
	if h == 0 {
		d.Height = swt.DEFAULT
	}
	return d
}

// Edge says where one side of a Form child sits. The zero Edge leaves the side free.
type Edge struct {
	kind   uint8
	pct    int
	offset int
	to     func() Widget
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

// Beside attaches the side to the opposite side of the sibling *w, gap points away: a left edge sits
// right of it. *w is read when the child is placed, so a Ref of an earlier sibling fills it in time.
func Beside[T Widget](w *T, gap int) Edge {
	return Edge{kind: edgeBeside, offset: gap, to: func() Widget { return *w }}
}

// Same attaches the side to the same side of the sibling *w (left to left, top to top, ...), offset points away.
func Same[T Widget](w *T, offset int) Edge {
	return Edge{kind: edgeSame, offset: offset, to: func() Widget { return *w }}
}

var errNoSibling = errors.New("a Beside or Same edge refers to a sibling that is not created yet")

func (e Edge) attachment(side int32) (*swt.FormAttachment, error) {
	switch e.kind {
	case edgeFree:
		return nil, nil
	case edgePercent:
		return swt.NewFormAttachmentNumeratorOffset(int32(e.pct), int32(e.offset)), nil
	}
	to := e.to()
	if isNil(to) {
		return nil, errNoSibling
	}
	if e.kind == edgeBeside {
		return swt.NewFormAttachmentControlOffset(to.control(), int32(e.offset)), nil
	}
	return swt.NewFormAttachmentControlOffsetAlignment(to.control(), int32(e.offset), side), nil
}

// isNil reports a nil Widget, also a typed nil pointer read from a Ref variable not filled yet.
func isNil(w Widget) (yes bool) {
	defer func() {
		if recover() != nil {
			yes = true
		}
	}()
	return w == nil || w.control() == nil
}

// FormCell places one child of a Form. Width and Height are hints in points, zero is natural size.
// A child with no edge set sticks to the top left.
type FormCell struct {
	Left, Right, Top, Bottom Edge
	Width, Height            int
}

// Anchor sets the position of a child laid out by Form.
func Anchor(c FormCell) Option {
	return place(func(pl *placement) { pl.form, pl.formc = true, c })
}

func (c FormCell) data() (*swt.FormData, error) {
	d := swt.NewFormData()
	var errs [4]error
	d.Left, errs[0] = c.Left.attachment(swt.LEFT)
	d.Right, errs[1] = c.Right.attachment(swt.RIGHT)
	d.Top, errs[2] = c.Top.attachment(swt.TOP)
	d.Bottom, errs[3] = c.Bottom.attachment(swt.BOTTOM)
	if err := errors.Join(errs[:]...); err != nil {
		return nil, err
	}
	d.Width, d.Height = int32(c.Width), int32(c.Height)
	if c.Width == 0 {
		d.Width = swt.DEFAULT
	}
	if c.Height == 0 {
		d.Height = swt.DEFAULT
	}
	return d, nil
}

// PagesView shows one page at a time at the full size of the container, like a tab folder without tabs.
type PagesView struct {
	base
	c     *swt.Composite
	stack *swt.StackLayout
}

// Pages creates a container whose children are pages, one shown at a time; the first is shown
// until Show picks another.
// Options: Border, children.
func Pages(opts ...Option) Option {
	return &node{"Pages", func(h host) Widget {
		c := swt.NewCompositeParentStyle(h.into(), resolve(swt.NONE, opts))
		s := swt.NewStackLayout()
		c.SetLayout(s)
		g := &PagesView{base{&c.Control}, c, s}
		finish(g, sBorder, opts)
		return g
	}}
}

// Add appends pages; a page added while none is shown is shown.
func (g *PagesView) Add(pages ...Option) {
	add(g, pages)
	if !g.c.IsDisposed() {
		g.c.Layout()
	}
}

func (g *PagesView) into() *swt.Composite { return g.c }

func (g *PagesView) added(w Widget) {
	c := w.control()
	if g.Current() == -1 {
		g.stack.TopControl = c
	}
	c.AddDisposeListener(disposeFunc(func() { g.gone(c) }))
}

// gone runs while page c is disposed and still a child. When c is the shown page, the page before it
// takes its place, or the next one when c is the first, as in a CTabs.
func (g *PagesView) gone(c *swt.Control) {
	if g.stack.TopControl != c {
		return
	}
	pages := g.c.GetChildren()
	switch i := slices.Index(pages, c); {
	case i > 0:
		g.stack.TopControl = pages[i-1]
	case i == 0 && len(pages) > 1:
		g.stack.TopControl = pages[1]
	default:
		g.stack.TopControl = nil
	}
}

// Show makes page i, counted from 0 in the order the pages were added, the visible one. Dispose on a
// page removes it: the pages after it move down by one, and when it was shown the page before it is
// shown, or the next one when it was the first.
func (g *PagesView) Show(i int) {
	pages := g.c.GetChildren()
	if i < 0 || i >= len(pages) {
		panic(fmt.Sprintf("gowt: Pages.Show(%d) with %d pages", i, len(pages)))
	}
	g.stack.TopControl = pages[i]
	g.c.Layout()
}

// Current returns the index of the visible page, -1 if there is none.
func (g *PagesView) Current() int {
	for i, c := range g.c.GetChildren() {
		if c == g.stack.TopControl {
			return i
		}
	}
	return -1
}

// Unwrap returns the underlying composite; its layout is an swt.StackLayout.
func (g *PagesView) Unwrap() *swt.Composite { return g.c }

// AsComposite returns the underlying composite, as Panel does, for the look package.
func (g *PagesView) AsComposite() *swt.Composite { return g.c }

// Placement options. They describe where a child sits and how it grows without naming the parent's
// layout, and add up: GrowX(), Span(2) give one cell that spans two columns and fills its width.
// When several set the same field, the later one wins. An option the parent's layout cannot use is
// reported to App.OnError naming the widget and the layout, and the widget keeps its natural size.
// Grow, GrowX and GrowY are also accepted, and change nothing, in a Split, Fill, Pages, Tabs or
// CTabs: those give every child its whole part anyway.

// Grow fills the cell and takes the free width and height of a VBox, HBox or Grid.
func Grow() Option {
	return fillOpt("Grow", func(c *GridCell) { c.Align, c.VAlign, c.GrowX, c.GrowY = AlignFill, AlignFill, true, true })
}

// GrowX fills the width of the cell and takes the free width of an HBox or Grid row.
func GrowX() Option {
	return fillOpt("GrowX", func(c *GridCell) { c.Align, c.GrowX = AlignFill, true })
}

// GrowY fills the height of the cell and takes the free height of a VBox or Grid column.
func GrowY() Option {
	return fillOpt("GrowY", func(c *GridCell) { c.VAlign, c.GrowY = AlignFill, true })
}

// Span makes the child cover cols columns of the Grid.
func Span(cols int) Option { return gridOpt("Span", func(c *GridCell) { c.SpanX = cols }) }

// Place aligns the child inside its cell: Place(AlignEnd, AlignCenter) puts it at the right, centred vertically.
func Place(h, v Align) Option {
	return gridOpt("Place", func(c *GridCell) { c.Align, c.VAlign = h, v })
}

// Indent moves the child x points right and y points down inside its cell; the cell grows by as much.
func Indent(x, y int) Option {
	return gridOpt("Indent", func(c *GridCell) { c.IndentX, c.IndentY = x, y })
}

// Cell sets every field of the child's grid cell at once; placement options after it change single fields.
func Cell(c GridCell) Option { return gridOpt("Cell", func(g *GridCell) { *g = c }) }

// Size asks for a width and height in points, zero keeps the natural one. It works in a VBox, HBox,
// Grid, Flow and Form.
func Size(w, h int) Option {
	return place(func(pl *placement) {
		pl.size = true
		pl.cell.Width, pl.cell.Height = w, h
		pl.formc.Width, pl.formc.Height = w, h
	})
}

func fillOpt(name string, f func(*GridCell)) Option {
	return place(func(pl *placement) {
		if pl.fill == "" {
			pl.fill = name
		}
		f(&pl.cell)
	})
}

func gridOpt(name string, f func(*GridCell)) Option {
	return place(func(pl *placement) {
		if pl.grid == "" {
			pl.grid = name
		}
		f(&pl.cell)
	})
}

// placement collects the placement options of one widget; finish turns it into layout data.
type placement struct {
	grid  string // the first option only a VBox, HBox or Grid can use, "" if none
	fill  string // the first of Grow, GrowX, GrowY, "" if none
	size  bool   // Size given
	form  bool   // Anchor given
	cell  GridCell
	formc FormCell
}

// isGrid reports whether a layoutName is laid out by a GridLayout.
func isGrid(layout string) bool { return layout == "Grid" || layout == "VBox" || layout == "HBox" }

// fillsAll reports whether a layoutName gives every child its whole part, so that Grow is already true.
func fillsAll(layout string) bool {
	switch layout {
	case "Split", "Fill", "Pages", "Tabs", "CTabs":
		return true
	}
	return false
}

// data returns the layout data for widget in a parent laid out by layout, a layoutName.
func (pl *placement) data(widget, layout string) (any, error) {
	switch {
	case pl.grid != "" && !isGrid(layout):
		return nil, misplaced(pl.grid, "a VBox, HBox or Grid", widget, layout)
	case pl.fill != "" && !isGrid(layout) && !fillsAll(layout):
		return nil, misplaced(pl.fill, "a VBox, HBox or Grid", widget, layout)
	case pl.form && layout != "Form":
		return nil, misplaced("Anchor", "a Form", widget, layout)
	}
	switch {
	case isGrid(layout):
		return pl.cell.data(), nil
	case layout == "Flow":
		return rowData(pl.cell.Width, pl.cell.Height), nil
	case layout == "Form":
		d, err := pl.formc.data()
		if err != nil {
			return nil, fmt.Errorf("gowt: Anchor on %s: %w", widget, err)
		}
		return d, nil
	case fillsAll(layout) && !pl.size:
		return nil, nil
	}
	return nil, misplaced("Size", "a VBox, HBox, Grid, Flow or Form", widget, layout)
}

func misplaced(opt, need, widget, layout string) error {
	if layout == "" {
		layout = "a parent without a layout"
	}
	return fmt.Errorf("gowt: %s on %s needs %s parent, not %s", opt, widget, need, layout)
}

// layoutName names the layout of c as the facade does ("Grid", "VBox", ...), "" for none.
func layoutName(c *swt.Composite) string {
	if c == nil {
		return ""
	}
	if c.GetLayout() == nil {
		// A TabFolder places its pages itself.
		if nameOf(&c.Control) == "Tabs" {
			return "Tabs"
		}
		return ""
	}
	switch l := c.GetLayout().Impl().(type) {
	case *boxLayout:
		if l.row {
			return "HBox"
		}
		return "VBox"
	case *swt.GridLayout:
		return "Grid"
	case *swt.RowLayout:
		return "Flow"
	case *swt.FormLayout:
		return "Form"
	case *swt.FillLayout:
		return "Fill"
	case *swt.StackLayout:
		return "Pages"
	case *swt.SashFormLayout:
		return "Split"
	case *swt.CTabFolderLayout:
		return "CTabs"
	default:
		return strings.TrimPrefix(fmt.Sprintf("%T", l), "*swt.")
	}
}
