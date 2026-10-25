//go:build windows

package fluent

import (
	"math"
	"time"
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

var (
	pSetTimer  = user32.NewProc("SetTimer")
	pKillTimer = user32.NewProc("KillTimer")
)

const (
	wmTimer          = 0x113
	wmLButtonDown    = 0x201
	wmLButtonUp      = 0x202
	wmCaptureChanged = 0x215

	pbmSetRange   = 0x401
	pbmSetPos     = 0x402
	pbmDeltaPos   = 0x403
	pbmStepIt     = 0x405
	pbmSetRange32 = 0x406
	pbmGetRange   = 0x407
	pbmGetPos     = 0x408
	pbmSetState   = 0x410
	pbmGetState   = 0x411
	pbsMarquee    = 0x8

	tbmGetPos         = 0x400
	tbmGetRangeMin    = 0x401
	tbmGetRangeMax    = 0x402
	tbmGetThumbRect   = 0x419
	tbmSetThumbLength = 0x420
	tbsAutoTicks      = 0x1
	tbsVert           = 0x2
	tbsNoTicks        = 0x10
	tbsFixedLength    = 0x40

	marqueeTimer = 0x7001
	marqueeTick  = 16
	marqueeLoop  = 2000 * time.Millisecond
)

// bufferedPaint runs fn on an off-screen copy of the client area of hwnd, with the font of the window
// selected, and copies it to the window in one go.
func bufferedPaint(hwnd uintptr, fn func(hdc uintptr, w, h int32)) {
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
	fn(mem, w, h)
	call(pBitBlt, hdc, 0, 0, uintptr(w), uintptr(h), mem, 0, 0, 0xCC0020)
}

// progressLook paints a ProgressBar as the WinUI 3 bar: a hairline track and a 3 px rounded indicator.
type progressLook struct {
	b     *swt.ProgressBar
	start time.Time
}

// Progress gives a horizontal ProgressBar the Windows 11 look, with its own animation for the
// indeterminate bar; the native control keeps the value, range and state.
func Progress(b *swt.ProgressBar) {
	h := uintptr(b.Handle)
	if h == 0 || b.GetStyle()&swt.VERTICAL != 0 {
		return
	}
	l := &progressLook{b: b, start: time.Now()}
	chain(h, l.proc)
	if style(h)&pbsMarquee != 0 {
		call(pSetTimer, h, marqueeTimer, marqueeTick, 0)
	}
	call(pInvalidateRect, h, 0, 1)
}

func (l *progressLook) proc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	switch msg {
	case wmEraseBkgnd:
		return 1, true
	case wmPaint:
		bufferedPaint(hwnd, func(hdc uintptr, w, h int32) { l.draw(hwnd, hdc, w, h) })
		return 0, true
	case wmTimer:
		if wp == marqueeTimer {
			call(pInvalidateRect, hwnd, 0, 0)
			return 0, true
		}
	case pbmSetPos, pbmDeltaPos, pbmStepIt, pbmSetRange, pbmSetRange32, pbmSetState:
		r := def()
		call(pInvalidateRect, hwnd, 0, 0)
		return r, true
	}
	return 0, false
}

func (l *progressLook) draw(hwnd, hdc uintptr, w, h int32) {
	d := l.b.GetDisplay()
	pal := Current(d)
	dpi := dpiOf(hwnd)
	bg := parentBackground(&l.b.Control)
	p := newPainter(d, hdc)
	defer p.done()
	p.fillRect(bg, 0, 0, w, h)

	track, bar := px(1, dpi), px(3, dpi)
	if bar%2 != track%2 {
		track++
	}
	ty, by := (h-track)/2, (h-bar)/2
	p.fillRound(pal.StrongStroke(bg), 0, ty, w, track, track/2)

	fill := pal.Accent
	switch call(pSendMessage, hwnd, pbmGetState, 0, 0) {
	case 2: // PBST_ERROR
		fill = RGB{0xC4, 0x2B, 0x1C}
		if pal.Dark {
			fill = RGB{0xFF, 0x99, 0xA4}
		}
	case 3: // PBST_PAUSED
		fill = RGB{0x9D, 0x5D, 0x00}
		if pal.Dark {
			fill = RGB{0xFC, 0xE1, 0x00}
		}
	}
	if !l.b.IsEnabled() {
		fill = pal.Black.over(0.36, bg)
	}

	if style(hwnd)&pbsMarquee != 0 {
		t := float64(time.Since(l.start)%marqueeLoop) / float64(marqueeLoop)
		t = t * t * (3 - 2*t) // smoothstep: the bar slows at both ends
		seg := float64(w) * 0.4
		x0 := -seg + (float64(w)+seg)*t
		x1 := x0 + seg
		x0, x1 = math.Max(x0, 0), math.Min(x1, float64(w))
		if x1-x0 >= 1 {
			p.fillRound(fill, int32(x0), by, int32(x1-x0), bar, bar/2)
		}
		return
	}
	lo := int32(call(pSendMessage, hwnd, pbmGetRange, 1, 0))
	hi := int32(call(pSendMessage, hwnd, pbmGetRange, 0, 0))
	pos := int32(call(pSendMessage, hwnd, pbmGetPos, 0, 0))
	if hi <= lo || pos <= lo {
		return
	}
	fw := int32(float64(w) * float64(min(pos, hi)-lo) / float64(hi-lo))
	if fw < bar {
		fw = bar
	}
	p.fillRound(fill, 0, by, fw, bar, bar/2)
}

// scaleLook paints a Scale as the WinUI 3 slider. The native trackbar keeps input, keyboard and the
// accessible object; its thumb is sized to the drawn one so hit tests agree.
type scaleLook struct {
	s       *swt.Scale
	hot     bool
	pressed bool
}

// Scale gives a horizontal Scale the Windows 11 look.
func Scale(s *swt.Scale) {
	h := uintptr(s.Handle)
	if h == 0 || s.GetStyle()&swt.VERTICAL != 0 {
		return
	}
	setStyle(h, style(h)&^tbsAutoTicks|tbsNoTicks|tbsFixedLength)
	call(pSendMessage, h, tbmSetThumbLength, uintptr(px(20, dpiOf(h))), 0)
	l := &scaleLook{s: s}
	chain(h, l.proc)
	call(pInvalidateRect, h, 0, 1)
}

func (l *scaleLook) proc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	switch msg {
	case wmEraseBkgnd:
		return 1, true
	case wmPaint:
		bufferedPaint(hwnd, func(hdc uintptr, w, h int32) { l.draw(hwnd, hdc, w, h) })
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
	case wmLButtonDown:
		l.pressed = true
		r := def()
		call(pInvalidateRect, hwnd, 0, 0)
		return r, true
	case wmLButtonUp, wmCaptureChanged:
		l.pressed = false
		r := def()
		call(pInvalidateRect, hwnd, 0, 0)
		return r, true
	case wmSetFocus, wmKillFocus, wmEnable:
		r := def()
		call(pInvalidateRect, hwnd, 0, 0)
		return r, true
	}
	return 0, false
}

func (l *scaleLook) draw(hwnd, hdc uintptr, w, h int32) {
	d := l.s.GetDisplay()
	pal := Current(d)
	dpi := dpiOf(hwnd)
	bg := parentBackground(&l.s.Control)
	p := newPainter(d, hdc)
	defer p.done()
	p.fillRect(bg, 0, 0, w, h)

	var tr rect
	call(pSendMessage, hwnd, tbmGetThumbRect, 0, uintptr(unsafe.Pointer(&tr)))
	cx, cy := (tr.left+tr.right)/2, h/2
	half := px(10, dpi)
	th := px(4, dpi)
	enabled := l.s.IsEnabled()
	accent, off := pal.Accent, pal.Black.over(0.44, bg)
	if !enabled {
		accent = pal.Black.over(0.36, bg)
	}
	x0, x1 := half, w-half
	if x1 <= x0 {
		return
	}
	p.fillRound(off, x0, cy-th/2, x1-x0, th, th/2)
	if cx > x0 {
		p.fillRound(accent, x0, cy-th/2, cx-x0, th, th/2)
	}

	rim := pal.Stroke(bg)
	face := RGB{255, 255, 255}
	if pal.Dark {
		face = RGB{0x45, 0x45, 0x45}
		rim = pal.Black.over(0.08, face)
	}
	p.fillRound(rim, cx-half, cy-half, 2*half, 2*half, half)
	p.fillRound(face, cx-half+1, cy-half+1, 2*half-2, 2*half-2, half-1)
	dot := px(6, dpi)
	switch {
	case !enabled:
		dot = px(5, dpi)
	case l.pressed:
		dot = px(5, dpi)
	case l.hot:
		dot = px(7, dpi)
	}
	p.fillRound(accent, cx-dot, cy-dot, 2*dot, 2*dot, dot)
	if focused(hwnd) && call(pSendMessage, hwnd, wmQueryUIState, 0, 0)&uisfHideFocus == 0 {
		o := half + px(2, dpi)
		p.strokeRound(pal.Text, cx-o, cy-o, 2*o, 2*o, o, px(2, dpi))
	}
}
