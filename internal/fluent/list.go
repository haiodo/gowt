//go:build windows

package fluent

import (
	"unicode/utf16"
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

const (
	lbGetSel         = 0x187
	lbGetText        = 0x189
	lbGetTextLen     = 0x18A
	lbGetCount       = 0x18B
	lbGetTopIndex    = 0x18E
	lbGetItemRect    = 0x198
	lbGetCaretIndex  = 0x19F
	lbSetItemHeight  = 0x1A0
	lbGetItemHeight  = 0x1A1
	lbItemFromPoint  = 0x1A9
	tvmSetItemHeight = 0x111B
)

// Tree gives the rows of a Tree the pill selection of Table; rowDips is the row height, 0 keeps the native one.
func Tree(t *swt.Tree, rowDips int) {
	if h := uintptr(t.Handle); h != 0 && rowDips > 0 {
		call(pSendMessage, h, tvmSetItemHeight, uintptr(px(float64(rowDips), dpiOf(h))), 0)
	}
	t.AddListener(swt.EraseItem, newListener(func(e *swt.Event) {
		hide := call(pSendMessage, uintptr(t.Handle), wmQueryUIState, 0, 0)&uisfHideFocus != 0
		if e.Detail&(swt.SELECTED|swt.HOT) != 0 && e.Gc != nil && e.Item != nil {
			if e.Index == 0 {
				paintTreeRow(t, e, hide)
			}
			e.Detail &^= swt.SELECTED | swt.HOT | swt.FOCUSED
		}
		if hide {
			e.Detail &^= swt.FOCUSED
		}
	}))
}

var sidebarTrees = map[*swt.Tree]bool{}

// SidebarTree makes t a sidebar tree: the window background instead of the white of an input surface,
// and a pill that spans the Tree, the pane giving the side margin. Call it after Tree.
func SidebarTree(t *swt.Tree, rowDips int) {
	h := uintptr(t.Handle)
	if h == 0 {
		return
	}
	sidebarTrees[t] = true
	t.AddDisposeListener(disposer(func() { delete(sidebarTrees, t) }))
	if rowDips > 0 {
		call(pSendMessage, h, tvmSetItemHeight, uintptr(px(float64(rowDips), dpiOf(h))), 0)
	}
	sidebarBackground(&t.Control)
	call(pInvalidateRect, h, 0, 1)
}

func paintTreeRow(t *swt.Tree, e *swt.Event, hideFocus bool) {
	pal := Current(t.GetDisplay())
	bgc := t.GetBackground().GetRGB()
	bg := RGB{uint8(bgc.Red), uint8(bgc.Green), uint8(bgc.Blue)}
	total := t.GetClientArea().Width
	if t.GetColumnCount() > 0 {
		total = 0
		for i := int32(0); i < t.GetColumnCount(); i++ {
			total += t.GetColumnIndex(i).GetWidth()
		}
	}
	inset := int32(4)
	if sidebarTrees[t] {
		inset, total = 0, max(total, t.GetClientArea().Width)
	}
	x := inset
	if hb := t.GetHorizontalBar(); hb != nil {
		x -= hb.GetSelection()
	}
	y, h, w := e.Y+1, e.Height-2, total-2*inset
	sel, hot := e.Detail&swt.SELECTED != 0, e.Detail&swt.HOT != 0
	gc := e.Gc
	gc.SetAntialias(swt.ON)
	gc.SetClipping(x-inset, y-1, w+2*inset, h+2)
	gc.SetForeground(t.GetForeground())
	fill := rowFill(pal, bg, sel, hot, t.IsFocusControl())
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

// listLook paints a List (a list box) with the pill rows of Table; the native control keeps the
// items, the selection, the keyboard and the accessible object.
type listLook struct {
	l       *swt.List
	hot     int32 // item under the pointer, -1 for none
	sidebar bool  // in a window sidebar: the pane gives the side margin
}

// SidebarList makes l a sidebar list: the window background, a pill that spans the List and text
// sidebarIconPad inside it. Call it after List.
func SidebarList(l *swt.List) {
	h := uintptr(l.Handle)
	if lk := lists[h]; lk != nil {
		lk.sidebar = true
		sidebarBackground(&l.Control)
		call(pInvalidateRect, h, 0, 1)
	}
}

var lists = map[uintptr]*listLook{}

// List gives l the Windows 11 rows; rowDips is the row height, 0 keeps the native one.
func List(l *swt.List, rowDips int) {
	h := uintptr(l.Handle)
	if h == 0 {
		return
	}
	if rowDips > 0 {
		// A font change resets the row height, so a Resize listener sets it again.
		apply := func() {
			if want := uintptr(px(float64(rowDips), dpiOf(h))); call(pSendMessage, h, lbGetItemHeight, 0, 0) != want {
				call(pSendMessage, h, lbSetItemHeight, 0, want)
			}
		}
		apply()
		l.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { apply() }))
	}
	lk := &listLook{l: l, hot: -1}
	lists[h] = lk
	l.AddDisposeListener(disposer(func() { delete(lists, h) }))
	chain(h, lk.proc)
	call(pInvalidateRect, h, 0, 1)
}

func (l *listLook) proc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	switch msg {
	case wmEraseBkgnd:
		return 1, true
	case wmPaint:
		bufferedPaint(hwnd, func(hdc uintptr, w, h int32) { l.draw(hwnd, hdc, w, h) })
		return 0, true
	case wmMouseMove:
		// The high word of LB_ITEMFROMPOINT is 1 outside the client area.
		r := call(pSendMessage, hwnd, lbItemFromPoint, 0, lp)
		hot := int32(r & 0xFFFF)
		if r>>16 != 0 {
			hot = -1
		}
		if hot != l.hot {
			l.hot = hot
			call(pInvalidateRect, hwnd, 0, 0)
		}
		tme := struct {
			size, flags uint32
			hwnd        uintptr
			hover       uint32
		}{24, tmeLeave, hwnd, 0}
		call(pTrackMouseEvent, uintptr(unsafe.Pointer(&tme)))
	case wmMouseLeave:
		l.hot = -1
		call(pInvalidateRect, hwnd, 0, 0)
	case wmSetFocus, wmKillFocus:
		r := def()
		call(pInvalidateRect, hwnd, 0, 0)
		return r, true
	}
	return 0, false
}

func (l *listLook) draw(hwnd, hdc uintptr, w, h int32) {
	d := l.l.GetDisplay()
	pal := Current(d)
	dpi := dpiOf(hwnd)
	bgc := l.l.GetBackground().GetRGB()
	bg := RGB{uint8(bgc.Red), uint8(bgc.Green), uint8(bgc.Blue)}
	fg := pal.Text
	if !l.l.IsEnabled() {
		fg = pal.Disabled
	}
	p := newPainter(d, hdc)
	defer p.done()
	p.fillRect(bg, 0, 0, w, h)

	focus := focused(hwnd)
	showFocus := focus && call(pSendMessage, hwnd, wmQueryUIState, 0, 0)&uisfHideFocus == 0
	caret := int32(call(pSendMessage, hwnd, lbGetCaretIndex, 0, 0))
	n := int32(call(pSendMessage, hwnd, lbGetCount, 0, 0))
	inset, pad := px(4, dpi), px(12, dpi)
	if l.sidebar {
		inset, pad = 0, px(sidebarIconPad, dpi)
	}
	for i := int32(call(pSendMessage, hwnd, lbGetTopIndex, 0, 0)); i >= 0 && i < n; i++ {
		var r rect
		call(pSendMessage, hwnd, lbGetItemRect, uintptr(i), uintptr(unsafe.Pointer(&r)))
		if r.top >= h {
			break
		}
		sel := int32(call(pSendMessage, hwnd, lbGetSel, uintptr(i), 0)) > 0
		hot := i == l.hot && l.l.IsEnabled()
		x, y, rw, rh := r.left+inset, r.top+px(1, dpi), r.right-r.left-2*inset, r.bottom-r.top-px(2, dpi)
		if sel || hot {
			p.fillRound(rowFill(pal, bg, sel, hot, focus), x, y, rw, rh, px(4, dpi))
		}
		if showFocus && i == caret {
			p.strokeRound(pal.Accent, x, y, rw, rh, px(4, dpi), 1)
		}
		n := call(pSendMessage, hwnd, lbGetTextLen, uintptr(i), 0)
		if n == 0 || int32(n) < 0 {
			continue
		}
		buf := make([]uint16, n+1)
		call(pSendMessage, hwnd, lbGetText, uintptr(i), uintptr(unsafe.Pointer(&buf[0])))
		txt := string(utf16.Decode(buf[:n]))
		e := p.gc.TextExtentInPixels(txt, swt.DRAW_TRANSPARENT)
		p.gc.SetForeground(p.color(fg))
		p.gc.DrawStringInPixels(txt, x+pad, r.top+(r.bottom-r.top-e.Y)/2, true)
	}
}
