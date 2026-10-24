//go:build windows

package fluent

import (
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

// textFrame is the state of a framed Text; off leaves the control without any frame or padding,
// for a Text inside a glass surface that draws the frame itself.
type textFrame struct {
	t     *swt.Text
	multi bool
	off   bool
}

var frames = map[uintptr]*textFrame{}

// Text gives t the WinUI 3 TextBox look: 4 px radius, hairline, strong bottom edge, accent line while
// focused. The native border goes; the frame is painted in the non-client area. Search fields keep SWT's look.
func Text(t *swt.Text) {
	h := uintptr(t.Handle)
	if h == 0 || t.GetStyle()&swt.SEARCH != 0 || frames[h] != nil {
		return
	}
	f := &textFrame{t: t, multi: t.GetStyle()&swt.MULTI != 0}
	frames[h] = f
	setStyle(h, style(h)&^wsBorder)
	setExStyle(h, exStyle(h)&^wsExEdge)
	if !t.HasCustomBackground() {
		f.applyColors()
	}
	frameChanged(h)
	attach(h, f.proc)
	t.AddDisposeListener(disposer(func() { delete(frames, h) }))
}

// Unframe takes the frame and the padding off a Text made by Text; c that is no such Text is ignored.
func Unframe(c *swt.Control) {
	f := frames[uintptr(c.Handle)]
	if f == nil || f.off {
		return
	}
	f.off = true
	frameChanged(uintptr(c.Handle))
}

// ExtraHeight is the height in points the frame takes from the client area of a multi-line Text.
func ExtraHeight(t *swt.Text) int {
	f := frames[uintptr(t.Handle)]
	if f == nil || f.off || !f.multi {
		return 0
	}
	z := t.GetAutoscalingZoom()
	return int(swt.DPIUtilPixelToPoint(2*f.vpad(uintptr(t.Handle)), z) + 1)
}

func (f *textFrame) vpad(hwnd uintptr) int32 { return px(2, dpiOf(hwnd)) }

// applyColors puts the theme's input colours on a Text that has none of its own.
func (f *textFrame) applyColors() {
	pal := Current(f.t.GetDisplay())
	fill := pal.Fill
	f.t.SetBackgroundWithColor(swt.NewColorRedGreenBlue(int32(fill.R), int32(fill.G), int32(fill.B)))
	if pal.Dark {
		f.t.SetForeground(swt.NewColorRedGreenBlue(int32(pal.Text.R), int32(pal.Text.G), int32(pal.Text.B)))
	}
}

func (f *textFrame) proc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	if f.off {
		return 0, false
	}
	switch msg {
	case wmNCCalcSize:
		r := def()
		rc := (*rect)(ptr(lp))
		dpi := dpiOf(hwnd)
		right := px(6, dpi)
		if style(hwnd)&wsVScroll != 0 {
			right = 0
		}
		rc.left += px(10, dpi)
		rc.right -= right
		vp := f.vpad(hwnd)
		top, bottom := vp, vp
		if !f.multi {
			if extra := (rc.bottom - rc.top) - textHeight(hwnd); extra > 2*vp {
				top = extra / 2
				bottom = extra - top
			}
		}
		rc.top += top
		rc.bottom -= bottom
		return r, true
	case wmNCPaint:
		r := def()
		f.paint(hwnd)
		return r, true
	case wmSetFocus, wmKillFocus, wmEnable:
		r := def()
		redrawFrame(hwnd)
		return r, true
	case wmMouseWheel:
		// Without a scroll bar the edit ignores the wheel.
		if style(hwnd)&wsVScroll == 0 && f.multi {
			lines := -int(int16(wp>>16)) * 3 / 120
			call(pSendMessage, hwnd, emLineScroll, 0, uintptr(int64(lines)))
			return 0, true
		}
	}
	return 0, false
}

func (f *textFrame) paint(hwnd uintptr) {
	var wr, cr rect
	call(pGetWindowRect, hwnd, uintptr(unsafe.Pointer(&wr)))
	call(pGetClientRect, hwnd, uintptr(unsafe.Pointer(&cr)))
	org := point{}
	call(pClientToScreen, hwnd, uintptr(unsafe.Pointer(&org)))
	w, h := wr.right-wr.left, wr.bottom-wr.top
	if w <= 0 || h <= 0 {
		return
	}
	hdc := call(pGetWindowDC, hwnd)
	if hdc == 0 {
		return
	}
	defer call(pReleaseDC, hwnd, hdc)
	dpi := dpiOf(hwnd)
	ox, oy := org.x-wr.left, org.y-wr.top
	call(pExcludeClipRect, hdc, uintptr(ox), uintptr(oy), uintptr(ox+cr.right), uintptr(oy+cr.bottom))
	if style(hwnd)&wsVScroll != 0 {
		sb := call(pGetSystemMetricsDpi, 2, uintptr(dpi)) // SM_CXVSCROLL
		call(pExcludeClipRect, hdc, uintptr(w)-sb, 0, uintptr(w), uintptr(h))
	}

	d := f.t.GetDisplay()
	pal := Current(d)
	enabled := f.t.IsEnabled()
	bg := parentBackground(&f.t.Control)
	fill := pal.Fill
	if !enabled {
		fill = pal.Black.over(0.04, bg)
	}
	r := px(4, dpi)
	p := newPainter(d, hdc)
	defer p.done()
	p.fillRect(bg, 0, 0, w, h)
	p.fillRound(fill, 0, 0, w, h, r)
	p.strokeRound(pal.Stroke(bg), 0, 0, w, h, r, 1)
	switch {
	case enabled && focused(hwnd):
		p.line(pal.Accent, r, h-1, w-1-r, h-1, 2)
	case enabled:
		p.line(pal.StrongStroke(bg), r, h-1, w-1-r, h-1, 1)
	}
}
