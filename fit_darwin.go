package gowt

import "github.com/haiodo/gowt/swt"

// SWT's Pack measures the items a view-based table has cached, so a column came out as wide as its
// header while its cells cut the text; the texts are measured with the control's font instead.
const (
	fitCellPad   = 10 // left and right cell padding of a Cocoa row
	fitImageGap  = 6
	fitHeaderPad = 24 // header padding and room for the sort arrow
	fitIndent    = 18 // expander and indent of one Tree level
)

// packColumn sizes c to its header and its widest cell.
func packColumn(c column) {
	switch col := c.(type) {
	case *swt.TableColumn:
		t := col.GetParent()
		idx := t.IndexOfColumn(col)
		m := newTextMeasure(&t.Control)
		w := m.text(col.GetText()) + fitHeaderPad
		for _, it := range t.GetItems() {
			w = max(w, m.cell(it.GetTextIndex(idx), it.GetImageIndex(idx), 0))
		}
		m.done()
		col.SetWidth(int32(w))
	case *swt.TreeColumn:
		t := col.GetParent()
		idx := t.IndexOfColumn(col)
		m := newTextMeasure(&t.Control)
		w := m.text(col.GetText()) + fitHeaderPad
		var walk func(items []*swt.TreeItem, depth int)
		walk = func(items []*swt.TreeItem, depth int) {
			for _, it := range items {
				lvl := 0
				if idx == 0 {
					lvl = depth + 1
				}
				w = max(w, m.cell(it.GetTextIndex(idx), it.GetImageIndex(idx), lvl))
				if it.GetExpanded() {
					walk(it.GetItems(), depth+1)
				}
			}
		}
		walk(t.GetItems(), 0)
		m.done()
		col.SetWidth(int32(w))
	default:
		c.Pack()
	}
}
