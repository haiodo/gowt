//go:build windows

package fluent

import (
	"syscall"
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

var pFindWindowEx = user32.NewProc("FindWindowExW")

const (
	spinInset  = 4  // dips between the buttons and the frame
	spinButton = 22 // dips, width of the spin buttons
	spinGap    = 4  // dips between the text and the buttons

	spinUp, spinDown = 1, 2

	wmSetFont       = 0x30
	wmLButtonDblClk = 0x203

	// SpinnerExtra is the width in dips the framed field needs beyond the native Spinner.
	SpinnerExtra = 24
)

// spinnerLook is the state of a Spinner in the NumberBox look: the edit spans the whole control and
// paints the Text frame, the up-down control floats over its right end as two chevron buttons.
type spinnerLook struct {
	s         *swt.Spinner
	edit, ud  uintptr
	hot, down int // spinUp or spinDown, 0 for none
}

func child(parent uintptr, class string) uintptr {
	c, _ := syscall.UTF16PtrFromString(class)
	return call(pFindWindowEx, parent, 0, uintptr(unsafe.Pointer(c)), 0)
}

type modify func()

func (m modify) ModifyText(*swt.ModifyEvent) { m() }

// Spinner gives s the WinUI 3 NumberBox look with compact spin buttons. The edit and the up-down
// control stay native; only the frame and the buttons are painted.
func Spinner(s *swt.Spinner) {
	sh := uintptr(s.Handle)
	l := &spinnerLook{s: s, edit: child(sh, "Edit"), ud: child(sh, "msctls_updown32")}
	if sh == 0 || l.edit == 0 || l.ud == 0 {
		return
	}
	setStyle(sh, style(sh)&^wsBorder)
	setExStyle(sh, exStyle(sh)&^wsExEdge)
	if !s.HasCustomBackground() {
		inputColors(&s.Control)
	}
	attach(l.edit, l.editProc)
	attach(l.ud, l.udProc)
	chain(sh, l.proc)
	frameChanged(sh)
	l.layout(sh)
	frameChanged(l.edit)
	s.AddModifyListener(modify(func() { call(pInvalidateRect, l.ud, 0, 0) }))
}

func spinGeometry(dpi int) (inset, button, gap int32) {
	return px(spinInset, dpi), px(spinButton, dpi), px(spinGap, dpi)
}

// layout stretches the edit over the control and puts the buttons in its right end; SWT lays them out
// side by side on every resize.
func (l *spinnerLook) layout(hwnd uintptr) {
	var cr rect
	call(pGetClientRect, hwnd, uintptr(unsafe.Pointer(&cr)))
	inset, button, _ := spinGeometry(dpiOf(hwnd))
	vins := min(inset, cr.bottom/8) // a short field keeps room for two chevrons
	// An even height keeps the halves the control splits itself into equal to the painted ones.
	bh := (cr.bottom - 2*vins) &^ 1
	if bh < 2 {
		bh = cr.bottom &^ 1
	}
	const flags = swpNoZOrder | swpNoActivate
	call(pSetWindowPos, l.edit, 0, 0, 0, uintptr(cr.right), uintptr(cr.bottom), flags)
	call(pSetWindowPos, l.ud, 0, uintptr(cr.right-inset-button), uintptr((cr.bottom-bh)/2), uintptr(button), uintptr(bh), flags)
}

func (l *spinnerLook) proc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	switch msg {
	case wmSize, wmSetFont:
		r := def()
		l.layout(hwnd)
		frameChanged(l.edit)
		return r, true
	}
	return 0, false
}

func (l *spinnerLook) editProc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	switch msg {
	case wmNCCalcSize:
		r := def()
		rc := (*rect)(ptr(lp))
		dpi := dpiOf(hwnd)
		inset, button, gap := spinGeometry(dpi)
		rc.left += px(10, dpi)
		rc.right -= inset + button + gap
		top, bottom := px(2, dpi), px(2, dpi)
		if extra := (rc.bottom - rc.top) - textHeight(hwnd); extra > top+bottom {
			top = extra / 2
			bottom = extra - top
		}
		rc.top += top
		rc.bottom -= bottom
		return r, true
	case wmNCPaint:
		r := def()
		paintInput(hwnd, &l.s.Control)
		return r, true
	case wmSetFocus, wmKillFocus, wmEnable:
		r := def()
		redrawFrame(hwnd)
		call(pInvalidateRect, l.ud, 0, 0)
		return r, true
	}
	return 0, false
}

func (l *spinnerLook) udProc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	half := func() int {
		var cr rect
		call(pGetClientRect, hwnd, uintptr(unsafe.Pointer(&cr)))
		if int32(int16(lp>>16)) < cr.bottom/2 {
			return spinUp
		}
		return spinDown
	}
	switch msg {
	case wmEraseBkgnd:
		return 1, true
	case wmPaint:
		bufferedPaint(hwnd, func(hdc uintptr, w, h int32) { l.draw(hwnd, hdc, w, h) })
		return 0, true
	case wmMouseMove:
		if l.hot == 0 {
			tme := struct {
				size, flags uint32
				hwnd        uintptr
				hover       uint32
			}{24, tmeLeave, hwnd, 0}
			call(pTrackMouseEvent, uintptr(unsafe.Pointer(&tme)))
		}
		if h := half(); h != l.hot {
			l.hot = h
			call(pInvalidateRect, hwnd, 0, 0)
		}
	case wmMouseLeave:
		l.hot = 0
		call(pInvalidateRect, hwnd, 0, 0)
	case wmLButtonDown, wmLButtonDblClk:
		l.down = half()
		r := def()
		call(pInvalidateRect, hwnd, 0, 0)
		return r, true
	case wmLButtonUp, wmCaptureChanged:
		r := def()
		l.down = 0
		call(pInvalidateRect, hwnd, 0, 0)
		return r, true
	case wmEnable:
		r := def()
		call(pInvalidateRect, hwnd, 0, 0)
		return r, true
	}
	return 0, false
}

func (l *spinnerLook) draw(hwnd, hdc uintptr, w, h int32) {
	d := l.s.GetDisplay()
	pal := Current(d)
	dpi := dpiOf(hwnd)
	bg := parentBackground(&l.s.Control)
	enabled := l.s.IsEnabled()
	fill := pal.Fill
	if !enabled {
		fill = pal.Black.over(0.04, bg)
	}
	p := newPainter(d, hdc)
	defer p.done()
	p.fillRect(fill, 0, 0, w, h)

	r := px(3, dpi)
	sel, minV, maxV := l.s.GetSelection(), l.s.GetMinimum(), l.s.GetMaximum()
	wrap := l.s.GetStyle()&swt.WRAP != 0
	for _, dir := range []int{spinUp, spinDown} {
		y0, y1 := int32(0), h/2
		if dir == spinDown {
			y0, y1 = h/2, h
		}
		active := enabled && (wrap || dir == spinUp && sel < maxV || dir == spinDown && sel > minV)
		glyph := pal.Text.over(0.61, fill)
		switch {
		case !active:
			glyph = pal.Black.over(0.36, fill)
		case l.down == dir:
			glyph = pal.Text.over(0.45, fill)
			p.fillRound(pal.Black.over(0.04, fill), 0, y0, w, y1-y0, r)
		case l.hot == dir && l.down == 0:
			p.fillRound(pal.Black.over(0.06, fill), 0, y0, w, y1-y0, r)
		}
		cx, cy, hw := w/2, (y0+y1)/2, px(3.5, dpi)
		lw := px(1, dpi)
		a, b := cy+hw/2, cy-hw/2
		if dir == spinDown {
			a, b = b, a
		}
		p.line(glyph, cx-hw, a, cx, b, lw)
		p.line(glyph, cx, b, cx+hw, a, lw)
	}
}
