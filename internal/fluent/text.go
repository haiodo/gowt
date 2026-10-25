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
	surf  *surface // the surface that frames a Text with off set
	in    [4]int32 // left, top, right, bottom padding of the last NCCALCSIZE
}

var frames = map[uintptr]*textFrame{}

// Text gives t the WinUI 3 TextBox look: 4 px radius, hairline, strong bottom edge, accent line while
// focused. The native border goes; the frame is painted in the non-client area.
func Text(t *swt.Text) {
	h := uintptr(t.Handle)
	if h == 0 || frames[h] != nil {
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

// Unframe leaves the frame of a Text made by Text to the surface s around it; the text keeps clear of
// the rounded corners of s.
func Unframe(c *swt.Control, s *surface) {
	f := frames[uintptr(c.Handle)]
	if f == nil {
		return
	}
	f.off, f.surf = true, s
	frameChanged(uintptr(c.Handle))
}

// ExtraHeight is the height in points the frame takes from the client area of a multi-line Text.
func ExtraHeight(t *swt.Text) int {
	f := frames[uintptr(t.Handle)]
	if f == nil || !f.multi {
		return 0
	}
	z := t.GetAutoscalingZoom()
	return int(swt.DPIUtilPixelToPoint(2*f.vpad(uintptr(t.Handle)), z) + 1)
}

func (f *textFrame) vpad(hwnd uintptr) int32 {
	if f.off {
		return f.surf.radiusPx() * 3 / 10
	}
	return px(2, dpiOf(hwnd))
}

// applyColors puts the theme's input colours on a Text that has none of its own.
func (f *textFrame) applyColors() { inputColors(&f.t.Control) }

func inputColors(c *swt.Control) {
	pal := Current(c.GetDisplay())
	fill := pal.Fill
	c.SetBackgroundWithColor(swt.NewColorRedGreenBlue(int32(fill.R), int32(fill.G), int32(fill.B)))
	if pal.Dark {
		c.SetForeground(swt.NewColorRedGreenBlue(int32(pal.Text.R), int32(pal.Text.G), int32(pal.Text.B)))
	}
}

func (f *textFrame) proc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	if f.off && msg != wmNCCalcSize && msg != wmNCPaint {
		return 0, false
	}
	switch msg {
	case wmNCCalcSize:
		r := def()
		rc := (*rect)(ptr(lp))
		dpi := dpiOf(hwnd)
		left, right := px(10, dpi), px(6, dpi)
		if f.off {
			left = max(f.surf.radiusPx()*45/100, px(8, dpi))
			right = left
		}
		if style(hwnd)&wsVScroll != 0 {
			right = 0
		}
		rc.left += left
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
		f.in = [4]int32{left, top, right, bottom}
		return r, true
	case wmNCPaint:
		r := def()
		if f.off {
			f.paintInSurface(hwnd)
			return r, true
		}
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

func (f *textFrame) paint(hwnd uintptr) { paintInput(hwnd, &f.t.Control) }

// paintInput paints the frame of the single-line edit hwnd, which belongs to the control c, in its
// non-client area.
func paintInput(hwnd uintptr, c *swt.Control) {
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

	d := c.GetDisplay()
	pal := Current(d)
	enabled := c.IsEnabled()
	bg := parentBackground(c)
	fill := pal.Fill
	if !enabled {
		fill = pal.Black.over(0.04, bg)
	}
	r := px(4, dpi)
	drawBuffered(d, hdc, w, h, func(p *painter) {
		p.fillRect(bg, 0, 0, w, h)
		p.fillRound(fill, 0, 0, w, h, r)
		p.strokeRound(pal.Stroke(bg), 0, 0, w, h, r, 1)
		switch {
		case enabled && focused(hwnd):
			p.line(pal.Accent, r, h-1, w-1-r, h-1, 2)
		case enabled:
			p.line(pal.StrongStroke(bg), r, h-1, w-1-r, h-1, 1)
		}
	})
}

// paintInSurface fills the non-client area of a Text in a surface with the slice of the surface under it.
func (f *textFrame) paintInSurface(hwnd uintptr) {
	var wr, cr, sr rect
	call(pGetWindowRect, hwnd, uintptr(unsafe.Pointer(&wr)))
	call(pGetClientRect, hwnd, uintptr(unsafe.Pointer(&cr)))
	sh := uintptr(f.surf.c.Handle)
	call(pGetClientRect, sh, uintptr(unsafe.Pointer(&sr)))
	so, org := point{}, point{}
	call(pClientToScreen, sh, uintptr(unsafe.Pointer(&so)))
	call(pClientToScreen, hwnd, uintptr(unsafe.Pointer(&org)))
	w, h := wr.right-wr.left, wr.bottom-wr.top
	hdc := call(pGetWindowDC, hwnd)
	if hdc == 0 || w <= 0 || h <= 0 {
		return
	}
	defer call(pReleaseDC, hwnd, hdc)
	ox, oy := org.x-wr.left, org.y-wr.top
	call(pExcludeClipRect, hdc, uintptr(ox), uintptr(oy), uintptr(ox+cr.right), uintptr(oy+cr.bottom))
	f.surf.draw(hdc, w, h, wr.left-so.x, wr.top-so.y, sr.right, sr.bottom)
}

const (
	cbSetItemHeight = 0x153
	cbGetItemHeight = 0x154
)

// ComboHeight sizes the closed combo through its selection field: SWT spends the layout height on the
// drop-down list. The themed box is 6 px taller than the field; a font change resets the field, so a
// Resize listener sets it again.
func ComboHeight(c *swt.Combo, dips int) {
	h := uintptr(c.Handle)
	if h == 0 {
		return
	}
	apply := func() {
		want := uintptr(px(float64(dips), dpiOf(h)) - 6)
		if call(pSendMessage, h, cbGetItemHeight, ^uintptr(0), 0) != want {
			call(pSendMessage, h, cbSetItemHeight, ^uintptr(0), want)
		}
	}
	apply()
	c.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { apply() }))
}
