package gowt

import "github.com/haiodo/gowt/swt"

// Combo is a drop-down (or Simple) list with an edit field; ReadOnly() makes it a pure chooser.
type Combo struct{ c *swt.Combo }

// Combo adds a combo box with the given items.
func (p *Panel) Combo(items []string, opts ...Option) *Combo {
	c := swt.NewCombo(p.c, resolve(swt.NONE, opts))
	if len(items) > 0 { // SWT rejects a nil items slice
		c.SetItems(items)
	}
	applyOpts(&c.Control, opts)
	return &Combo{c}
}

func (c *Combo) control() *swt.Control { return &c.c.Control }
func (c *Combo) Text() string          { return c.c.GetText() }
func (c *Combo) SetText(s string)      { c.c.SetText(s) }
func (c *Combo) Items() []string       { return c.c.GetItems() }
func (c *Combo) SetItems(s []string)   { c.c.SetItems(s) }

// Index is the selected item, -1 if the text matches none.
func (c *Combo) Index() int { return int(c.c.GetSelectionIndex()) }

// Select selects item i.
func (c *Combo) Select(i int)       { c.c.Select(int32(i)) }
func (c *Combo) Unwrap() *swt.Combo { return c.c }

// OnSelect runs f with the chosen item index.
func (c *Combo) OnSelect(f func(index int)) {
	onSelect(c.c, func(*swt.SelectionEvent) { f(c.Index()) })
}

// OnChange runs f with the new text after every edit.
func (c *Combo) OnChange(f func(text string)) {
	c.c.AddModifyListener(&modifier{func() { f(c.c.GetText()) }})
}

// List is a single or multi (MultiSelect) selection list of strings.
type List struct{ l *swt.List }

// List adds a list with the given items.
func (p *Panel) List(items []string, opts ...Option) *List {
	l := swt.NewList(p.c, resolve(swt.V_SCROLL|swt.BORDER, opts))
	if len(items) > 0 {
		l.SetItems(items)
	}
	applyOpts(&l.Control, opts)
	return &List{l}
}

func (l *List) control() *swt.Control { return &l.l.Control }
func (l *List) Items() []string       { return l.l.GetItems() }
func (l *List) SetItems(s []string)   { l.l.SetItems(s) }
func (l *List) Add(s string)          { l.l.Add(s) }
func (l *List) Remove(i int)          { l.l.Remove(int32(i)) }
func (l *List) Clear()                { l.l.RemoveAll() }
func (l *List) Len() int              { return int(l.l.GetItemCount()) }

// Index is the first selected item, -1 if none.
func (l *List) Index() int { return int(l.l.GetSelectionIndex()) }

// Selected returns all selected item indices.
func (l *List) Selected() []int { return toInts(l.l.GetSelectionIndices()) }

// SetSelection replaces the selection.
func (l *List) SetSelection(i ...int) { l.l.SetSelectionIndices(toInt32s(i)) }
func (l *List) Unwrap() *swt.List     { return l.l }

// OnSelect runs f with the first selected index.
func (l *List) OnSelect(f func(index int)) {
	onSelect(l.l, func(*swt.SelectionEvent) { f(l.Index()) })
}

// OnActivate runs f on double click or Enter.
func (l *List) OnActivate(f func(index int)) {
	onActivate(l.l, func(*swt.SelectionEvent) { f(l.Index()) })
}

// Table is a multi-column grid with a header; rows are added with Row. Check makes rows checkable.
type Table struct{ t *swt.Table }

// Table adds a table with a visible header and grid lines.
func (p *Panel) Table(opts ...Option) *Table {
	t := swt.NewTable(p.c, resolve(swt.V_SCROLL|swt.H_SCROLL|swt.FULL_SELECTION|swt.BORDER, opts))
	t.SetHeaderVisible(true)
	t.SetLinesVisible(true)
	applyOpts(&t.Control, opts)
	return &Table{t}
}

func (t *Table) control() *swt.Control { return &t.t.Control }
func (t *Table) ShowHeader(v bool)     { t.t.SetHeaderVisible(v) }
func (t *Table) ShowLines(v bool)      { t.t.SetLinesVisible(v) }
func (t *Table) Len() int              { return int(t.t.GetItemCount()) }
func (t *Table) Remove(i int)          { t.t.Remove(int32(i)) }
func (t *Table) Clear()                { t.t.RemoveAll() }
func (t *Table) Unwrap() *swt.Table    { return t.t }

// Index is the first selected row, -1 if none.
func (t *Table) Index() int { return int(t.t.GetSelectionIndex()) }

// Selected returns all selected row indices.
func (t *Table) Selected() []int { return toInts(t.t.GetSelectionIndices()) }

// SetSelection replaces the selection.
func (t *Table) SetSelection(i ...int) { t.t.SetSelectionIndices(toInt32s(i)) }

// RowAt returns row i.
func (t *Table) RowAt(i int) *TableRow { return &TableRow{t.t.GetItem(int32(i))} }

// Column appends a column; Right() or Center() set the alignment.
func (t *Table) Column(title string, width int, opts ...Option) *Column {
	c := swt.NewTableColumn(t.t, resolve(swt.NONE, opts))
	c.SetText(title)
	c.SetWidth(int32(width))
	return &Column{c}
}

// Row appends a row with one text per column.
func (t *Table) Row(cells ...string) *TableRow {
	it := swt.NewTableItem(t.t, swt.NONE)
	it.SetTexts(cells)
	return &TableRow{it}
}

// OnSelect runs f with the first selected row.
func (t *Table) OnSelect(f func(index int)) {
	onSelect(t.t, func(*swt.SelectionEvent) { f(t.Index()) })
}

// OnActivate runs f on double click or Enter.
func (t *Table) OnActivate(f func(index int)) {
	onActivate(t.t, func(*swt.SelectionEvent) { f(t.Index()) })
}

// Column is a table column.
type Column struct{ c *swt.TableColumn }

func (c *Column) SetTitle(s string)        { c.c.SetText(s) }
func (c *Column) SetWidth(w int)           { c.c.SetWidth(int32(w)) }
func (c *Column) Pack()                    { c.c.Pack() }
func (c *Column) Unwrap() *swt.TableColumn { return c.c }

// OnClick runs f when the header is clicked, e.g. to sort.
func (c *Column) OnClick(f func()) {
	onSelect(c.c, func(*swt.SelectionEvent) { f() })
}

// TableRow is a table row. Rows are re-wrapped on every lookup: keep per-row state in SetData.
type TableRow struct{ it *swt.TableItem }

func (r *TableRow) Text(col int) string          { return r.it.GetTextIndex(int32(col)) }
func (r *TableRow) SetText(col int, s string)    { r.it.SetTextIndexString(int32(col), s) }
func (r *TableRow) Checked() bool                { return r.it.GetChecked() }
func (r *TableRow) SetChecked(v bool)            { r.it.SetChecked(v) }
func (r *TableRow) SetImage(col int, img *Image) { r.it.SetImageIndexImage(int32(col), img.i) }
func (r *TableRow) Data() any                    { return r.it.GetData() }
func (r *TableRow) SetData(v any)                { r.it.SetData(v) }
func (r *TableRow) Unwrap() *swt.TableItem       { return r.it }

// Tree is a hierarchy of Nodes. Check makes nodes checkable, MultiSelect allows several selected.
type Tree struct{ t *swt.Tree }

// Tree adds a tree.
func (p *Panel) Tree(opts ...Option) *Tree {
	t := swt.NewTree(p.c, resolve(swt.V_SCROLL|swt.H_SCROLL|swt.BORDER, opts))
	applyOpts(&t.Control, opts)
	return &Tree{t}
}

func (t *Tree) control() *swt.Control { return &t.t.Control }
func (t *Tree) Clear()                { t.t.RemoveAll() }
func (t *Tree) Unwrap() *swt.Tree     { return t.t }

// Node appends a root node.
func (t *Tree) Node(text string) *Node {
	it := swt.NewTreeItem(t.t, swt.NONE)
	it.SetTextIndexString(0, text)
	return &Node{it}
}

// Roots returns the top-level nodes.
func (t *Tree) Roots() []*Node { return nodes(t.t.GetItems()) }

// Selected returns the selected nodes.
func (t *Tree) Selected() []*Node { return nodes(t.t.GetSelection()) }

// Select selects n alone.
func (t *Tree) Select(n *Node) { t.t.SetSelection(n.it) }

// OnSelect runs f with the first selected node, nil if none.
func (t *Tree) OnSelect(f func(n *Node)) {
	onSelect(t.t, func(*swt.SelectionEvent) { f(first(t.t.GetSelection())) })
}

// OnActivate runs f on double click or Enter.
func (t *Tree) OnActivate(f func(n *Node)) {
	onActivate(t.t, func(*swt.SelectionEvent) { f(first(t.t.GetSelection())) })
}

// OnExpand and OnCollapse run f with the node that changed. The event carries the item as a
// bare *swt.Widget that cannot be downcast, so the node is found by walking the tree.
func (t *Tree) OnExpand(f func(n *Node)) {
	t.t.AddTreeListener(swt.TreeListenerTreeExpandedAdapter(func(e *swt.TreeEvent) { f(t.find(e.Item)) }))
}

func (t *Tree) OnCollapse(f func(n *Node)) {
	t.t.AddTreeListener(swt.TreeListenerTreeCollapsedAdapter(func(e *swt.TreeEvent) { f(t.find(e.Item)) }))
}

func (t *Tree) find(w *swt.Widget) *Node {
	var walk func([]*swt.TreeItem) *Node
	walk = func(items []*swt.TreeItem) *Node {
		for _, it := range items {
			if &it.Widget == w {
				return &Node{it}
			}
			if n := walk(it.GetItems()); n != nil {
				return n
			}
		}
		return nil
	}
	return walk(t.t.GetItems())
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
	return &Node{it}
}

func (n *Node) Text() string          { return n.it.GetTextIndex(0) }
func (n *Node) SetText(s string)      { n.it.SetTextIndexString(0, s) }
func (n *Node) Expanded() bool        { return n.it.GetExpanded() }
func (n *Node) SetExpanded(v bool)    { n.it.SetExpanded(v) }
func (n *Node) Checked() bool         { return n.it.GetChecked() }
func (n *Node) SetChecked(v bool)     { n.it.SetChecked(v) }
func (n *Node) SetImage(img *Image)   { n.it.SetImageIndexImage(0, img.i) }
func (n *Node) Children() []*Node     { return nodes(n.it.GetItems()) }
func (n *Node) Data() any             { return n.it.GetData() }
func (n *Node) SetData(v any)         { n.it.SetData(v) }
func (n *Node) Remove()               { n.it.Dispose() }
func (n *Node) Unwrap() *swt.TreeItem { return n.it }
