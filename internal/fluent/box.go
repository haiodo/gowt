//go:build windows

package fluent

import (
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

type disposer func()

func (d disposer) WidgetDisposed(*swt.DisposeEvent) { d() }

// listener is a pointer type: the event tables compare listeners, and comparing func values panics.
type listener struct{ f func(*swt.Event) }

func newListener(f func(*swt.Event)) *listener { return &listener{f} }

func (l *listener) HandleEvent(e *swt.Event) { l.f(e) }

// FlatBorder replaces the 3D edge (WS_EX_CLIENTEDGE) by a flat 1 px hairline with square corners.
func FlatBorder(c *swt.Control) {
	h := uintptr(c.Handle)
	if h == 0 || handlers[h] != nil || exStyle(h)&wsExEdge == 0 && style(h)&wsBorder == 0 {
		return
	}
	setExStyle(h, exStyle(h)&^wsExEdge)
	setStyle(h, style(h)|wsBorder)
	frameChanged(h)
	attach(h, func(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
		if msg != wmNCPaint {
			return 0, false
		}
		r := def()
		paintRing(c, hwnd)
		return r, true
	})
}

func paintRing(c *swt.Control, hwnd uintptr) {
	var wr rect
	call(pGetWindowRect, hwnd, uintptr(unsafe.Pointer(&wr)))
	w, h := wr.right-wr.left, wr.bottom-wr.top
	hdc := call(pGetWindowDC, hwnd)
	if hdc == 0 || w < 3 || h < 3 {
		return
	}
	defer call(pReleaseDC, hwnd, hdc)
	call(pExcludeClipRect, hdc, 1, 1, uintptr(w-1), uintptr(h-1))
	d := c.GetDisplay()
	pal := Current(d)
	drawBuffered(d, hdc, w, h, func(p *painter) {
		p.gc.SetAntialias(swt.OFF)
		p.fillRect(pal.Stroke(parentBackground(c)), 0, 0, w, h)
	})
}
