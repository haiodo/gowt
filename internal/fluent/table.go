//go:build windows

package fluent

import (
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

// Table hooks EraseItem so SWT paints the Explorer selection once per row, not per cell (the seams
// between columns), drops the dotted focus rectangle until a key is used, and draws images of right
// and centre columns aligned.
func Table(t *swt.Table) {
	t.AddListener(swt.EraseItem, newListener(func(e *swt.Event) {
		hide := call(pSendMessage, uintptr(t.Handle), wmQueryUIState, 0, 0)&uisfHideFocus != 0
		if e.Detail&(swt.SELECTED|swt.HOT) != 0 && e.Gc != nil && e.Item != nil {
			if e.Index == 0 {
				paintRow(t, e, hide)
			}
			e.Detail &^= swt.SELECTED | swt.HOT | swt.FOCUSED
		}
		if hide {
			e.Detail &^= swt.FOCUSED
		}
		if movedImage(t, e) {
			e.Detail &^= swt.FOREGROUND
		}
	}))
	t.AddListener(swt.PaintItem, newListener(func(e *swt.Event) {
		if movedImage(t, e) {
			paintAligned(t, e)
		}
	}))
}

// movedImage reports whether the cell of e has an image in a column that is not left-aligned.
func movedImage(t *swt.Table, e *swt.Event) bool {
	if e.Index <= 0 || e.Item == nil || e.Index >= t.GetColumnCount() {
		return false
	}
	if t.GetColumnIndex(e.Index).GetAlignment() == swt.LEFT {
		return false
	}
	return tableItem(e).GetImageIndex(e.Index) != nil
}

// tableItem is the item of an event of a Table: a TableItem starts with its Item and Widget.
func tableItem(e *swt.Event) *swt.TableItem { return (*swt.TableItem)(unsafe.Pointer(e.Item)) }

func paintAligned(t *swt.Table, e *swt.Event) {
	it := tableItem(e)
	img := it.GetImageIndex(e.Index)
	cell := it.GetBoundsIndex(e.Index)
	txt := it.GetTextIndex(e.Index)
	gc := e.Gc
	ib := img.GetBounds()
	const pad, gap = 6, 4
	var tw, th int32
	if txt != "" {
		x := gc.TextExtent(txt)
		tw, th = x.X, x.Y
	}
	total := ib.Width + tw
	if tw > 0 {
		total += gap
	}
	x := cell.X + cell.Width - pad - total
	if t.GetColumnIndex(e.Index).GetAlignment() == swt.CENTER {
		x = cell.X + (cell.Width-total)/2
	}
	gc.SetClipping(cell.X, cell.Y, cell.Width, cell.Height)
	gc.DrawImage(img, x, cell.Y+(cell.Height-ib.Height)/2)
	if tw > 0 {
		gc.DrawStringStringXYIsTransparent(txt, x+ib.Width+gap, cell.Y+(cell.Height-th)/2, true)
	}
}

// paintRow draws the selection and hover of the row as WinUI 3 does: one rounded pill over the whole row,
// drawn once from the first cell; GDI+ does not keep the cell clip, and the later cells draw over it.
func paintRow(t *swt.Table, e *swt.Event, hideFocus bool) {
	pal := Current(t.GetDisplay())
	bgc := t.GetBackground().GetRGB()
	bg := RGB{uint8(bgc.Red), uint8(bgc.Green), uint8(bgc.Blue)}
	var left, total int32
	for i := int32(0); i < t.GetColumnCount(); i++ {
		w := t.GetColumnIndex(i).GetWidth()
		if i < e.Index {
			left += w
		}
		total += w
	}
	if t.GetColumnCount() == 0 {
		total = t.GetClientArea().Width
	}
	x := e.X - left + 4 // left is 0 for the first cell
	y, h, w := e.Y+1, e.Height-2, total-8
	sel, hot := e.Detail&swt.SELECTED != 0, e.Detail&swt.HOT != 0
	var fill RGB
	switch {
	case sel && t.IsFocusControl():
		a := 0.20
		if hot {
			a = 0.28
		}
		fill = pal.Accent.over(a, bg)
	case sel:
		fill = pal.Black.over(0.09, bg)
	default:
		fill = pal.Black.over(0.05, bg)
	}
	gc := e.Gc
	gc.SetAntialias(swt.ON)
	gc.SetClipping(x-4, y-1, w+8, h+2)
	// SWT takes the colour of the text of the selected row from the GC after the event.
	gc.SetForeground(t.GetForeground())
	c := swt.NewColorRedGreenBlue(int32(fill.R), int32(fill.G), int32(fill.B))
	defer c.Dispose()
	gc.SetBackground(c)
	gc.FillRoundRectangle(x, y, w, h, 8, 8)
	if sel && e.Detail&swt.FOCUSED != 0 && !hideFocus {
		o := swt.NewColorRedGreenBlue(int32(pal.Accent.R), int32(pal.Accent.G), int32(pal.Accent.B))
		defer o.Dispose()
		gc.SetForeground(o)
		gc.SetLineWidth(1)
		gc.DrawRoundRectangle(x, y, w-1, h-1, 8, 8)
	}
}
