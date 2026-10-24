//go:build windows

package fluent

import (
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

const (
	wmChangeUIState = 0x127
	uisSet          = 1
	uisfHideFocus   = 1
)

// Table hooks EraseItem so SWT paints the Explorer selection once per row, not per cell (the seams
// between columns), drops the dotted focus rectangle until a key is used, and draws images of right
// and centre columns aligned.
func Table(t *swt.Table) {
	t.AddListener(swt.MouseDown, listener(func(*swt.Event) {
		if !t.IsDisposed() {
			call(pSendMessage, uintptr(t.Handle), wmChangeUIState, uisSet|uisfHideFocus<<16, 0)
		}
	}))
	t.AddListener(swt.EraseItem, listener(func(e *swt.Event) {
		if call(pSendMessage, uintptr(t.Handle), wmQueryUIState, 0, 0)&uisfHideFocus != 0 {
			e.Detail &^= swt.FOCUSED
		}
		if movedImage(t, e) {
			e.Detail &^= swt.FOREGROUND
		}
	}))
	t.AddListener(swt.PaintItem, listener(func(e *swt.Event) {
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
