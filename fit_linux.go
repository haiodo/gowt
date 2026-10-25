package gowt

import "github.com/haiodo/gowt/swt"

// SWT's Pack gave about the header width here whatever the rows held (measured in the stand, cause
// not found), so the texts are measured with the control's font.
const (
	fitCellPad   = 12 // left and right cell padding of the GTK row
	fitImageGap  = 6
	fitHeaderPad = 28 // header padding and room for the sort arrow
	fitIndent    = 24 // expander and indent of one Tree level
)

// packColumn sizes c to its header and its widest cell.
func packColumn(c column) {
	switch col := c.(type) {
	case *swt.TableColumn:
		t := col.GetParent()
		idx := t.IndexOfColumn(col)
		m := newTextMeasure(&t.Control)
		defer m.done()
		w := m.text(col.GetText()) + fitHeaderPad
		for _, it := range t.GetItems() {
			w = max(w, m.cell(it.GetTextIndex(idx), it.GetImageIndex(idx), 0))
		}
		col.SetWidth(int32(w))
	case *swt.TreeColumn:
		t := col.GetParent()
		idx := t.IndexOfColumn(col)
		m := newTextMeasure(&t.Control)
		defer m.done()
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
		col.SetWidth(int32(w))
	default:
		c.Pack()
	}
}

type textMeasure struct{ gc *swt.GC }

func newTextMeasure(c *swt.Control) textMeasure {
	gc := swt.NewGCDrawable(c)
	gc.SetFont(c.GetFont())
	return textMeasure{gc}
}

func (m textMeasure) done() { m.gc.Dispose() }

func (m textMeasure) text(s string) int {
	if s == "" {
		return 0
	}
	return int(m.gc.TextExtent(s).X)
}

func (m textMeasure) cell(s string, img *swt.Image, level int) int {
	w := m.text(s) + fitCellPad + level*fitIndent
	if img != nil {
		w += int(img.GetBounds().Width) + fitImageGap
	}
	return w
}
