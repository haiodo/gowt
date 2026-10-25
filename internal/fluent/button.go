//go:build windows

package fluent

import (
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

var (
	pBeginPaint              = user32.NewProc("BeginPaint")
	pEndPaint                = user32.NewProc("EndPaint")
	pCreateCompatDC          = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatBmp         = gdi32.NewProc("CreateCompatibleBitmap")
	pBitBlt                  = gdi32.NewProc("BitBlt")
	pDeleteObject            = gdi32.NewProc("DeleteObject")
	pDeleteDC                = gdi32.NewProc("DeleteDC")
	wmQueryUIState   uintptr = 0x129
)

// buttonLook is the state of a button painted by fluent: Subtle has no fill until the pointer is over
// it (WinUI "Subtle" button), otherwise the button has the standard fill; Border draws the hairline
// of a Subtle or Circle button; Circle makes the shape a circle or a capsule.
type buttonLook struct {
	b                      *swt.Button
	subtle, circle, border bool
	hot                    bool
	accent, danger         bool // Primary and Destructive: a solid accent or critical fill
}

var buttons = map[uintptr]*buttonLook{}

// Button repaints a push button in the WinUI 3 style. Each flag adds to what an earlier call set.
func Button(b *swt.Button, subtle, circle, border bool) {
	h := uintptr(b.Handle)
	if h == 0 || b.GetStyle()&(swt.PUSH) == 0 {
		return
	}
	l := buttons[h]
	if l == nil {
		l = &buttonLook{b: b}
		buttons[h] = l
		attach(h, l.proc)
		b.AddDisposeListener(disposer(func() { delete(buttons, h) }))
	}
	l.subtle = l.subtle || subtle
	l.circle = l.circle || circle
	l.border = l.border || border
	call(pInvalidateRect, h, 0, 1)
}

// ButtonKind gives a push button the solid fill of a WinUI accent button (primary) or of a critical
// one (destructive; with primary it is the same fill, the critical colour wins).
func ButtonKind(b *swt.Button, primary, destructive bool) {
	Button(b, false, false, false)
	if l := buttons[uintptr(b.Handle)]; l != nil {
		l.accent = l.accent || primary
		l.danger = l.danger || destructive
		call(pInvalidateRect, uintptr(b.Handle), 0, 1)
	}
}

// solid returns the fill and the text colour of an accent or critical button, and whether this is one.
func (l *buttonLook) solid(pal Palette, pushed bool, bg RGB) (fill, text RGB, ok bool) {
	var base RGB
	switch {
	case l.danger && pal.Dark:
		base, text = RGB{0xFF, 0x99, 0xA4}, RGB{0, 0, 0}
	case l.danger:
		base, text = RGB{0xC4, 0x2B, 0x1C}, RGB{255, 255, 255}
	case l.accent:
		base, text = pal.Accent, pal.OnAccent
	default:
		return RGB{}, RGB{}, false
	}
	a := 1.0
	if pushed {
		a = 0.8
	} else if l.hot {
		a = 0.9
	}
	return base.over(a, bg), text, true
}

func (l *buttonLook) proc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	switch msg {
	case wmEraseBkgnd:
		return 1, true
	case wmPaint:
		l.paint(hwnd)
		return 0, true
	case wmMouseMove:
		if !l.hot {
			l.hot = true
			tme := struct {
				size, flags uint32
				hwnd        uintptr
				hover       uint32
			}{24, tmeLeave, hwnd, 0}
			call(pTrackMouseEvent, uintptr(unsafe.Pointer(&tme)))
			call(pInvalidateRect, hwnd, 0, 0)
		}
	case wmMouseLeave:
		l.hot = false
		call(pInvalidateRect, hwnd, 0, 0)
	case wmSetFocus, wmKillFocus, wmEnable:
		r := def()
		call(pInvalidateRect, hwnd, 0, 0)
		return r, true
	}
	return 0, false
}

func (l *buttonLook) paint(hwnd uintptr) {
	var ps [72]byte // PAINTSTRUCT
	hdc := call(pBeginPaint, hwnd, uintptr(unsafe.Pointer(&ps[0])))
	if hdc == 0 {
		return
	}
	defer call(pEndPaint, hwnd, uintptr(unsafe.Pointer(&ps[0])))
	var cr rect
	call(pGetClientRect, hwnd, uintptr(unsafe.Pointer(&cr)))
	w, h := cr.right, cr.bottom
	if w <= 0 || h <= 0 {
		return
	}
	mem := call(pCreateCompatDC, hdc)
	bmp := call(pCreateCompatBmp, hdc, uintptr(w), uintptr(h))
	if mem == 0 || bmp == 0 {
		return
	}
	defer call(pDeleteDC, mem)
	defer call(pDeleteObject, bmp)
	call(pSelectObject, mem, bmp)
	call(pSelectObject, mem, call(pSendMessage, hwnd, wmGetFont, 0, 0))

	l.draw(hwnd, mem, w, h)
	call(pBitBlt, hdc, 0, 0, uintptr(w), uintptr(h), mem, 0, 0, 0xCC0020)
}

func (l *buttonLook) draw(hwnd, hdc uintptr, w, h int32) {
	d := l.b.GetDisplay()
	pal := Current(d)
	dpi := dpiOf(hwnd)
	bg := parentBackground(&l.b.Control)
	surf := surfaceOver(&l.b.Control)
	if surf != nil {
		bg = pal.Fill
	}
	enabled := l.b.IsEnabled()
	pushed := call(pSendMessage, hwnd, bmGetState, 0, 0)&bstPushed != 0
	tint := RGB{249, 249, 249}
	if pal.Dark {
		tint = RGB{255, 255, 255}
	}

	var fill RGB
	var filled bool
	solidFill, solidText, solid := l.solid(pal, pushed, bg)
	solid = solid && enabled
	switch {
	case solid:
		fill, filled = solidFill, true
	case !enabled:
		if !l.subtle {
			fill, filled = pal.Black.over(0.04, bg), true
		}
	case l.subtle && pushed:
		fill, filled = pal.Black.over(0.09, bg), true
	case l.subtle && l.hot:
		fill, filled = pal.Black.over(0.06, bg), true
	case l.subtle:
	case pal.Dark:
		a := 0.06
		if pushed {
			a = 0.03
		} else if l.hot {
			a = 0.09
		}
		fill, filled = tint.over(a, bg), true
	default:
		a := 0.70
		if pushed {
			a = 0.30
		} else if l.hot {
			a = 0.50
		}
		fill, filled = tint.over(a, bg), true
	}
	r := px(4, dpi)
	if l.circle {
		r = min(w, h) / 2
	}
	p := newPainter(d, hdc)
	defer p.done()
	if surf != nil {
		surf.paintSlice(p, hwnd, w, h)
	} else {
		p.fillRect(bg, 0, 0, w, h)
	}
	if filled {
		p.fillRound(fill, 0, 0, w, h, r)
	}
	if (!l.subtle || l.border) && !solid {
		p.strokeRound(pal.Stroke(bg), 0, 0, w, h, r, 1)
	}

	txt := l.b.GetText()
	img := l.b.GetImage()
	fg := pal.Text
	if solid {
		fg = solidText
	} else if !enabled {
		fg = pal.Disabled
	}
	z := p.gc.GetZoom()
	var iw, ih, tw, th int32
	if img != nil {
		b := img.GetBounds()
		iw, ih = swt.DPIUtilPointToPixel(b.Width, z), swt.DPIUtilPointToPixel(b.Height, z)
	}
	if txt != "" {
		e := p.gc.TextExtentInPixels(txt, swt.DRAW_TRANSPARENT)
		tw, th = e.X, e.Y
	}
	gap := int32(0)
	if iw > 0 && tw > 0 {
		gap = px(8, dpi)
	}
	x := (w - (iw + gap + tw)) / 2
	if iw > 0 {
		p.gc.DrawImage(img, swt.DPIUtilPixelToPoint(x, z), swt.DPIUtilPixelToPoint((h-ih)/2, z))
		x += iw + gap
	}
	if tw > 0 {
		p.gc.SetForeground(p.color(fg))
		p.gc.DrawStringInPixels(txt, x, (h-th)/2, true)
	}
	if focused(hwnd) && call(pSendMessage, hwnd, wmQueryUIState, 0, 0)&1 == 0 {
		in := px(2, dpi)
		p.strokeRound(pal.Text, in, in, w-2*in, h-2*in, max(r-in, 0), px(2, dpi))
	}
}
