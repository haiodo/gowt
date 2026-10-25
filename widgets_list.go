package gowt

import "github.com/haiodo/gowt/swt"

// ComboView is a drop-down (or Simple) list with an edit field; ReadOnly() makes it a pure chooser.
type ComboView struct {
	base
	c *swt.Combo
}

// Combo creates a combo box with the given items.
// Options: ReadOnly, Simple, Value, Selected, OnSelect, OnChange, Bind (string: text, int: index).
func Combo(items []string, opts ...Option) Option {
	return leaf("Combo", sReadOnly|sSimple, opts, func(p *swt.Composite) Widget {
		c := swt.NewCombo(p, resolve(swt.NONE, opts))
		if len(items) > 0 { // SWT rejects a nil items slice
			c.SetItems(items)
		}
		return &ComboView{base{&c.Control}, c}
	})
}

// Text returns the edit field content, also when it matches no item.
func (c *ComboView) Text() string { return c.c.GetText() }

// SetText replaces the edit field content.
func (c *ComboView) SetText(s string) { c.c.SetText(s) }

// Items returns the list entries.
func (c *ComboView) Items() []string { return c.c.GetItems() }

// SetItems replaces the list entries. Lays the parents out when the size changes: the width follows
// the widest entry.
func (c *ComboView) SetItems(s []string) { sizing(&c.c.Control, func() { c.c.SetItems(s) }) }

// Index is the selected item, -1 if the text matches none.
func (c *ComboView) Index() int { return int(c.c.GetSelectionIndex()) }

// Select selects item i.
func (c *ComboView) Select(i int) { c.c.Select(int32(i)) }

// Unwrap returns the underlying swt.Combo for the full API.
func (c *ComboView) Unwrap() *swt.Combo { return c.c }

// OnSelect runs f with the chosen item index.
func (c *ComboView) OnSelect(f func(index int)) {
	onSelect(c.c, func(*swt.SelectionEvent) { f(c.Index()) })
}

// OnChange runs f with the new text after every edit.
func (c *ComboView) OnChange(f func(text string)) {
	c.c.AddModifyListener(&modifier{func() { f(c.c.GetText()) }})
}

// ListView is a single or multi (MultiSelect) selection list of strings.
type ListView struct {
	base
	l *swt.List
}

// List creates a list with the given items.
// Options: MultiSelect, NoBorder, Selected, OnSelect, OnActivate, Bind (int).
func List(items []string, opts ...Option) Option {
	return leaf("List", sMultiSelect|sNoBorder, opts, func(p *swt.Composite) Widget {
		l := swt.NewList(p, resolve(swt.V_SCROLL|swt.BORDER, opts))
		scrollOverlay(l)
		if len(items) > 0 {
			l.SetItems(items)
		}
		return &ListView{base{&l.Control}, l}
	})
}

// Items returns the list entries.
func (l *ListView) Items() []string { return l.l.GetItems() }

// SetItems replaces the list entries.
func (l *ListView) SetItems(s []string) { l.l.SetItems(s) }

// Add appends an entry.
func (l *ListView) Add(s string) { l.l.Add(s) }

// Remove deletes entry i.
func (l *ListView) Remove(i int) { l.l.Remove(int32(i)) }

// Clear deletes all entries and with them the selection.
func (l *ListView) Clear() { l.l.RemoveAll() }

// Len counts all entries, selected or not.
func (l *ListView) Len() int { return int(l.l.GetItemCount()) }

// Index is the first selected item, -1 if none.
func (l *ListView) Index() int { return int(l.l.GetSelectionIndex()) }

// Selected returns all selected item indices.
func (l *ListView) Selected() []int { return toInts(l.l.GetSelectionIndices()) }

// Select replaces the selection with items i; no i clears it.
func (l *ListView) Select(i ...int) { l.l.SetSelectionIndices(toInt32s(i)) }

// Deselect clears the selection and calls no OnSelect, as SWT's deselectAll sends no Selection.
func (l *ListView) Deselect() { l.l.DeselectAll() }

// SetItem replaces the text of item i, keeping the selection.
func (l *ListView) SetItem(i int, s string) { l.l.SetItem(int32(i), s) }

// Unwrap returns the underlying swt.List for the full API.
func (l *ListView) Unwrap() *swt.List { return l.l }

// OnSelect runs f with the first selected index.
func (l *ListView) OnSelect(f func(index int)) {
	onSelect(l.l, func(*swt.SelectionEvent) { f(l.Index()) })
}

// OnActivate runs f on double click or Enter.
func (l *ListView) OnActivate(f func(index int)) {
	onActivate(l.l, func(*swt.SelectionEvent) { f(l.Index()) })
}

// TableView is a multi-column grid with a header; rows are added with Row. Check makes rows checkable.
type TableView struct {
	base
	t *swt.Table
}

// Table creates a table with a visible header and grid lines.
// Options: Check, MultiSelect, NoBorder, Selected, OnSelect, OnActivate.
func Table(opts ...Option) Option {
	return leaf("Table", sCheck|sMultiSelect|sNoBorder, opts, func(p *swt.Composite) Widget {
		t := swt.NewTable(p, resolve(swt.V_SCROLL|swt.H_SCROLL|swt.FULL_SELECTION|swt.BORDER, opts))
		scrollOverlay(t)
		t.SetHeaderVisible(true)
		t.SetLinesVisible(true)
		fluentTable(t)
		return &TableView{base{&t.Control}, t}
	})
}

// ShowHeader shows or hides the column headers.
func (t *TableView) ShowHeader(v bool) { t.t.SetHeaderVisible(v) }

// ShowLines shows or hides the grid lines.
func (t *TableView) ShowLines(v bool) { t.t.SetLinesVisible(v) }

// ShowRowLines shows or hides thin lines between rows only. On macOS ShowLines also turns on
// vertical lines and alternating row colors, this does not; elsewhere it is ShowLines.
func (t *TableView) ShowRowLines(v bool) {
	if !rowLinesOnly(&t.t.Control, v) {
		t.t.SetLinesVisible(v)
	}
}

// Len counts rows, not columns.
func (t *TableView) Len() int { return int(t.t.GetItemCount()) }

// Remove deletes row i.
func (t *TableView) Remove(i int) {
	t.t.Remove(int32(i))
	contentChanged(&t.t.Control)
}

// Clear deletes all rows and with them the selection.
func (t *TableView) Clear() {
	t.t.RemoveAll()
	contentChanged(&t.t.Control)
}

// Batch runs f with drawing of the table suspended, so that many row changes repaint once.
func (t *TableView) Batch(f func()) {
	t.t.SetRedraw(false)
	defer t.t.SetRedraw(true)
	f()
}

// Unwrap returns the underlying swt.Table for the full API.
func (t *TableView) Unwrap() *swt.Table { return t.t }

// Index is the first selected row, -1 if none.
func (t *TableView) Index() int { return int(t.t.GetSelectionIndex()) }

// Selected returns all selected row indices.
func (t *TableView) Selected() []int { return toInts(t.t.GetSelectionIndices()) }

// Select replaces the selection with rows i; no i clears it.
func (t *TableView) Select(i ...int) { t.t.SetSelectionIndices(toInt32s(i)) }

// Deselect clears the selection and calls no OnSelect, as SWT's deselectAll sends no Selection.
func (t *TableView) Deselect() { t.t.DeselectAll0() }

// RowAt returns row i.
func (t *TableView) RowAt(i int) *TableRow { return &TableRow{t.t.GetItem(int32(i))} }

// Column appends a column; Right() or Center() set the alignment.
// Options: Right, Center.
func (t *TableView) Column(title string, width int, opts ...Option) *Column {
	itemOpts(t.t, "Table.Column", sRight|sCenter, false, opts)
	c := swt.NewTableColumn(t.t, resolve(swt.NONE, opts))
	c.SetText(title)
	c.SetWidth(int32(width))
	return &Column{c}
}

// Row appends a row with one text per column.
func (t *TableView) Row(cells ...string) *TableRow {
	it := swt.NewTableItem(t.t, swt.NONE)
	it.SetTexts(cells)
	contentChanged(&t.t.Control)
	return &TableRow{it}
}

// OnSelect runs f with the first selected row.
func (t *TableView) OnSelect(f func(index int)) {
	onSelect(t.t, func(*swt.SelectionEvent) { f(t.Index()) })
}

// OnActivate runs f on double click or Enter.
func (t *TableView) OnActivate(f func(index int)) {
	onActivate(t.t, func(*swt.SelectionEvent) { f(t.Index()) })
}

// SortDir is the sort indicator drawn in a column header. It only draws the arrow: sorting the rows is up to the app.
type SortDir int

// The sort indicator values.
const (
	SortNone SortDir = iota
	SortAscending
	SortDescending
)

func (d SortDir) swt() int32 {
	switch d {
	case SortAscending:
		return swt.UP
	case SortDescending:
		return swt.DOWN
	}
	return swt.NONE
}

// SetSort draws the sort indicator dir in the header of col; col nil or SortNone clears it.
func (t *TableView) SetSort(col *Column, dir SortDir) {
	if col == nil || dir == SortNone {
		t.t.SetSortDirection(swt.NONE)
		t.t.SetSortColumn(nil)
		return
	}
	t.t.SetSortColumn(col.c.(*swt.TableColumn))
	t.t.SetSortDirection(dir.swt())
}

// Mods are the modifier keys held during a mouse press. On macOS a Ctrl-click arrives as Button 1 with Ctrl set.
type Mods struct{ Ctrl, Shift, Alt, Cmd bool }

func modsOf(e *swt.MouseEvent) Mods {
	m := e.StateMask
	return Mods{m&swt.CTRL != 0, m&swt.SHIFT != 0, m&swt.ALT != 0, m&swt.COMMAND != 0}
}

// TableClick is a mouse press on a Table. Row and Col are -1 when the point is not on a row.
type TableClick struct {
	Row, Col int // row index and column index of the cell under the pointer
	Button   int // SWT's numbering: 1 left, 2 middle, 3 right
	Count    int // 1 for a single press, 2 for the second press of a double click
	X, Y     int // pointer position in the table's coordinates
	Mods         // keys held down
}

// colAtX maps x (control coordinates, scrolled by scroll) to a column index given the column widths
// in display order; order[i] is the column index shown i-th. It returns -1 for a point left of the
// first column or past the last one, so empty space is never a cell.
func colAtX(widths, order []int, scroll, x int) int {
	x += scroll
	if x < 0 {
		return -1
	}
	for i, w := range widths {
		if x < w {
			return order[i]
		}
		x -= w
	}
	return -1
}

// probeXs lists the x positions to try with GetItemPoint: the pointer's, then the right end of the
// first column, because macOS finds no item on a tree's expander arrow.
func probeXs(x, firstWidth, clientWidth int) []int {
	if firstWidth <= 0 {
		firstWidth = clientWidth
	}
	return []int{x, firstWidth - 2}
}

// CellAt returns the row and column under the point (x, y) in the table's coordinates, or -1, -1
// when no row is there. Col is -1 for a point in a row band but left of the first column or right of the last one.
func (t *TableView) CellAt(x, y int) (row, col int) {
	n := int(t.t.GetColumnCount())
	order := toInts(t.t.GetColumnOrder())
	widths := make([]int, n)
	for i, c := range order {
		widths[i] = int(t.t.GetColumnIndex(int32(c)).GetWidth())
	}
	first := 0
	if n > 0 {
		first = widths[0]
	}
	for _, px := range probeXs(x, first, int(t.t.GetClientArea().Width)) {
		if it := t.t.GetItemPoint(swt.NewPoint(int32(px), int32(y))); it != nil {
			return int(t.t.IndexOfItem(it)), cellCol(n, widths, order, hscroll(t.t.GetHorizontalBar()), x)
		}
	}
	return -1, -1
}

// cellCol is colAtX, except that a table without columns has one implicit column 0.
func cellCol(n int, widths, order []int, scroll, x int) int {
	if n == 0 {
		return 0
	}
	return colAtX(widths, order, scroll, x)
}

func hscroll(b *swt.ScrollBar) int {
	if b == nil {
		return 0
	}
	return int(b.GetSelection())
}

// OnCellClick runs f on every mouse press, so a handler can ignore right clicks (Button != 1) and
// double clicks (Count > 1). The cell comes from the hit test, not from the selection, which may not have moved yet.
func (t *TableView) OnCellClick(f func(TableClick)) {
	t.t.AddMouseListener(swt.MouseListenerMouseDownAdapter(func(e *swt.MouseEvent) {
		row, col := t.CellAt(int(e.X), int(e.Y))
		f(TableClick{Row: row, Col: col, Button: int(e.Button), Count: int(e.Count), X: int(e.X), Y: int(e.Y), Mods: modsOf(e)})
	}))
}

// Column is a column of a Table or a Tree.
type Column struct{ c column }

// column is what swt.TableColumn and swt.TreeColumn have in common.
type column interface {
	selector
	SetText(string)
	SetWidth(int32)
	SetResizable(bool)
	Pack()
	AsItem() *swt.Item
}

// SetResizable allows or forbids resizing the column by dragging the header edge.
func (c *Column) SetResizable(v bool) { c.c.SetResizable(v) }

// SetTitle replaces the header text.
func (c *Column) SetTitle(s string) { c.c.SetText(s) }

// SetWidth sets the width in points.
func (c *Column) SetWidth(w int) { c.c.SetWidth(int32(w)) }

// Pack resizes the column to fit its content.
func (c *Column) Pack() { c.c.Pack() }

// Unwrap returns the underlying item for the full API; its Impl() is the *swt.TableColumn or *swt.TreeColumn.
func (c *Column) Unwrap() *swt.Item { return c.c.AsItem() }

// OnClick runs f when the header is clicked, e.g. to sort.
func (c *Column) OnClick(f func()) {
	onSelect(c.c, func(*swt.SelectionEvent) { f() })
}

// TableRow is a table row. Rows are re-wrapped on every lookup: keep per-row state in SetData.
type TableRow struct{ it *swt.TableItem }

// TextAt returns the text of column col.
func (r *TableRow) TextAt(col int) string { return r.it.GetTextIndex(int32(col)) }

// SetTextAt sets the text of column col.
func (r *TableRow) SetTextAt(col int, s string) { r.it.SetTextIndexString(int32(col), s) }

// Checked reports the check box of a Check table.
func (r *TableRow) Checked() bool { return r.it.GetChecked() }

// SetChecked sets the check box of a Check table.
func (r *TableRow) SetChecked(v bool) { r.it.SetChecked(v) }

// SetImageAt sets the icon of column col; the row does not own img.
func (r *TableRow) SetImageAt(col int, img *Image) { r.it.SetImageIndexImage(int32(col), img.i) }

// Data returns the value stored by SetData.
func (r *TableRow) Data() any { return r.it.GetData() }

// SetData stores an application value with the row.
func (r *TableRow) SetData(v any) { r.it.SetData(v) }

// Unwrap returns the underlying swt.TableItem for the full API.
func (r *TableRow) Unwrap() *swt.TableItem { return r.it }

// TreeView is a hierarchy of Nodes. Check makes nodes checkable, MultiSelect allows several selected.
type TreeView struct {
	base
	t *swt.Tree
}

// Tree creates a tree. Rows select as a whole (CellSelect() opts out) so that a click in any column finds its node. With
// Column, the first column holds the tree and the header is shown until ShowHeader(false).
// Options: Check, MultiSelect, NoBorder, CellSelect, OnSelect, OnActivate.
func Tree(opts ...Option) Option {
	return leaf("Tree", sCheck|sMultiSelect|sNoBorder|sCellSelect, opts, func(p *swt.Composite) Widget {
		t := swt.NewTree(p, resolve(swt.V_SCROLL|swt.H_SCROLL|swt.BORDER|swt.FULL_SELECTION, opts))
		scrollOverlay(t)
		// The natural height follows the visible rows: lay out again after an expand or collapse.
		t.AddTreeListener(swt.TreeListenerTreeExpandedAdapter(func(*swt.TreeEvent) { contentChanged(&t.Control) }))
		t.AddTreeListener(swt.TreeListenerTreeCollapsedAdapter(func(*swt.TreeEvent) { contentChanged(&t.Control) }))
		return &TreeView{base{&t.Control}, t}
	})
}

// Column appends a column; Right() or Center() set the alignment. The first column holds the node
// text and the expand arrows. The first call shows the header.
// Options: Right, Center.
func (t *TreeView) Column(title string, width int, opts ...Option) *Column {
	itemOpts(t.t, "Tree.Column", sRight|sCenter, false, opts)
	c := swt.NewTreeColumn(t.t, resolve(swt.NONE, opts))
	c.SetText(title)
	c.SetWidth(int32(width))
	if t.t.GetColumnCount() == 1 {
		t.t.SetHeaderVisible(true)
	}
	return &Column{c}
}

// ShowRowLines shows or hides thin lines between rows only, see Table.ShowRowLines.
func (t *TreeView) ShowRowLines(v bool) {
	if !rowLinesOnly(&t.t.Control, v) {
		t.t.SetLinesVisible(v)
	}
}

// ShowHeader shows or hides the column headers.
func (t *TreeView) ShowHeader(v bool) { t.t.SetHeaderVisible(v) }

// SetSort draws the sort indicator dir in the header of col; col nil or SortNone clears it.
func (t *TreeView) SetSort(col *Column, dir SortDir) {
	if col == nil || dir == SortNone {
		t.t.SetSortDirection(swt.NONE)
		t.t.SetSortColumn(nil)
		return
	}
	t.t.SetSortColumn(col.c.(*swt.TreeColumn))
	t.t.SetSortDirection(dir.swt())
}

// TreeClick is a mouse press on a Tree. Node is nil and Col is -1 when the point is not on a node.
type TreeClick struct {
	Node   *Node
	Col    int // column index of the cell under the pointer
	Button int // SWT's numbering: 1 left, 2 middle, 3 right
	Count  int // 1 for a single press, 2 for the second press of a double click
	X, Y   int // pointer position in the tree's coordinates
	Mods       // keys held down
}

// CellAt returns the node and column under the point (x, y) in the tree's coordinates, or nil, -1
// when no node is there. The expander arrow and the space right of the text map to the cell; col is -1
// for a point in the row's band but right of the last column.
func (t *TreeView) CellAt(x, y int) (*Node, int) {
	n := int(t.t.GetColumnCount())
	order := toInts(t.t.GetColumnOrder())
	widths := make([]int, n)
	for i, c := range order {
		widths[i] = int(t.t.GetColumnIndex(int32(c)).GetWidth())
	}
	first := 0
	if n > 0 {
		first = widths[0]
	}
	for _, px := range probeXs(x, first, int(t.t.GetClientArea().Width)) {
		if it := t.t.GetItemPoint(swt.NewPoint(int32(px), int32(y))); it != nil {
			return &Node{it}, cellCol(n, widths, order, hscroll(t.t.GetHorizontalBar()), x)
		}
	}
	return nil, -1
}

// OnCellClick runs f on every mouse press, so a handler can ignore right clicks (Button != 1) and
// double clicks (Count > 1). The cell comes from the hit test, not from the selection, which may not have moved yet.
func (t *TreeView) OnCellClick(f func(TreeClick)) {
	t.t.AddMouseListener(swt.MouseListenerMouseDownAdapter(func(e *swt.MouseEvent) {
		n, col := t.CellAt(int(e.X), int(e.Y))
		f(TreeClick{Node: n, Col: col, Button: int(e.Button), Count: int(e.Count), X: int(e.X), Y: int(e.Y), Mods: modsOf(e)})
	}))
}

// Batch runs f with drawing of the tree suspended, so that many node changes repaint once.
func (t *TreeView) Batch(f func()) {
	t.t.SetRedraw(false)
	defer t.t.SetRedraw(true)
	f()
}

// Clear deletes all nodes, children included.
func (t *TreeView) Clear() {
	t.t.RemoveAll()
	contentChanged(&t.t.Control)
}

// Unwrap returns the underlying swt.Tree for the full API.
func (t *TreeView) Unwrap() *swt.Tree { return t.t }

// Node appends a root node.
func (t *TreeView) Node(text string) *Node {
	it := swt.NewTreeItem(t.t, swt.NONE)
	it.SetTextIndexString(0, text)
	contentChanged(&t.t.Control)
	return &Node{it}
}

// Roots returns the top-level nodes.
func (t *TreeView) Roots() []*Node { return nodes(t.t.GetItems()) }

// Selected returns the selected nodes.
func (t *TreeView) Selected() []*Node { return nodes(t.t.GetSelection()) }

// Select selects n alone; nil clears the selection. It calls no OnSelect, as SWT's setSelection sends no Selection.
func (t *TreeView) Select(n *Node) {
	if n == nil {
		t.Deselect()
		return
	}
	t.t.SetSelection(n.it)
}

// Deselect clears the selection and calls no OnSelect, as SWT's deselectAll sends no Selection.
func (t *TreeView) Deselect() { t.t.DeselectAll0() }

// OnSelect runs f with the first selected node, nil if none.
func (t *TreeView) OnSelect(f func(n *Node)) {
	onSelect(t.t, func(*swt.SelectionEvent) { f(first(t.t.GetSelection())) })
}

// OnActivate runs f on double click or Enter.
func (t *TreeView) OnActivate(f func(n *Node)) {
	onActivate(t.t, func(*swt.SelectionEvent) { f(first(t.t.GetSelection())) })
}

// OnExpand and OnCollapse run f with the node that changed.
func (t *TreeView) OnExpand(f func(n *Node)) {
	t.t.AddTreeListener(swt.TreeListenerTreeExpandedAdapter(func(e *swt.TreeEvent) { f(eventNode(e)) }))
}

// OnCollapse runs f with the node that was collapsed; see OnExpand.
func (t *TreeView) OnCollapse(f func(n *Node)) {
	t.t.AddTreeListener(swt.TreeListenerTreeCollapsedAdapter(func(e *swt.TreeEvent) { f(eventNode(e)) }))
}

// eventNode wraps the item of e: a bare *swt.Widget whose Impl is the *swt.TreeItem.
func eventNode(e *swt.TreeEvent) *Node {
	if e.Item == nil {
		return nil
	}
	if it, ok := e.Item.Impl().(*swt.TreeItem); ok {
		return &Node{it}
	}
	return nil
}

func first(items []*swt.TreeItem) *Node {
	if len(items) == 0 {
		return nil
	}
	return &Node{items[0]}
}

func nodes(items []*swt.TreeItem) []*Node {
	r := make([]*Node, len(items))
	for i, it := range items {
		r[i] = &Node{it}
	}
	return r
}

// Node is a tree item. Nodes are re-wrapped on every lookup: keep per-node state in SetData.
type Node struct{ it *swt.TreeItem }

// Node appends a child node.
func (n *Node) Node(text string) *Node {
	it := swt.NewTreeItemParentItemStyle(n.it, swt.NONE)
	it.SetTextIndexString(0, text)
	contentChanged(&n.it.GetParent().Control)
	return &Node{it}
}

// Text returns the node text (the first column).
func (n *Node) Text() string { return n.it.GetTextIndex(0) }

// SetText replaces the node text.
func (n *Node) SetText(s string) { n.it.SetTextIndexString(0, s) }

// Expanded reports whether the children are shown.
func (n *Node) Expanded() bool { return n.it.GetExpanded() }

// SetExpanded shows or hides the children.
func (n *Node) SetExpanded(v bool) {
	n.it.SetExpanded(v)
	contentChanged(&n.it.GetParent().Control)
}

// Checked reports the check box of a Check tree.
func (n *Node) Checked() bool { return n.it.GetChecked() }

// SetChecked sets the check box of a Check tree.
func (n *Node) SetChecked(v bool) { n.it.SetChecked(v) }

// SetImage sets the icon; the node does not own img.
func (n *Node) SetImage(img *Image) { n.it.SetImageIndexImage(0, img.i) }

// TextAt returns the text of column col.
func (n *Node) TextAt(col int) string { return n.it.GetTextIndex(int32(col)) }

// SetTextAt sets the text of column col; SetText is column 0.
func (n *Node) SetTextAt(col int, s string) { n.it.SetTextIndexString(int32(col), s) }

// SetImageAt sets the icon of column col; the node does not own img.
func (n *Node) SetImageAt(col int, img *Image) { n.it.SetImageIndexImage(int32(col), img.i) }

// Children returns the direct child nodes.
func (n *Node) Children() []*Node { return nodes(n.it.GetItems()) }

// Data returns the value stored by SetData.
func (n *Node) Data() any { return n.it.GetData() }

// SetData stores an application value with the node.
func (n *Node) SetData(v any) { n.it.SetData(v) }

// Remove deletes the node with its children.
func (n *Node) Remove() {
	t := n.it.GetParent()
	n.it.Dispose()
	contentChanged(&t.Control)
}

// Unwrap returns the underlying swt.TreeItem for the full API.
func (n *Node) Unwrap() *swt.TreeItem { return n.it }
