//go:build windows

package fluent

import (
	"syscall"
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

// textFrame is the state of a framed Text; off leaves the control without any frame or padding,
// for a Text inside a glass surface that draws the frame itself.
type textFrame struct {
	t      *swt.Text
	multi  bool
	off    bool
	surf   *surface // the surface that frames a Text with off set
	in     [4]int32 // left, top, right, bottom padding of the last NCCALCSIZE
	affix  *Affix
	drag   *int32 // the last client x of a scrub of the prefix, nil while none runs
	moved  bool   // the scrub has moved
	fill   *RGB   // FieldFill: the frame is this fill, without the hairline
	stroke *RGB   // FieldStroke: the border colour, already blended over the fill
	radius int    // its corner radius, dips
	padX   int    // extra text padding, dips
}

// Affix is what SetAffix shows inside the frame of a single-line Text around its value.
type Affix struct {
	Prefix, Suffix string
	Image          *swt.Image // drawn instead of Prefix
	// Drag makes the prefix a scrub handle: a drag reports each horizontal move in points, a click
	// without a move focuses the Text with its text selected.
	Drag func(dx float64, shift bool)
}

// affixGap is the space in dips between an affix and the value.
const affixGap = 6

// SetAffix shows a in the non-client area of a Text framed by Text, in its font and the WinUI
// secondary text colour; the client area, where the edit draws the value, keeps clear of it. It
// returns the dips the affixes take, which SWT's preferred width of the edit leaves out.
func SetAffix(t *swt.Text, a Affix) (width int) {
	h := uintptr(t.Handle)
	f := frames[h]
	if f == nil || f.multi {
		return 0
	}
	f.affix = &a
	frameChanged(h)
	lead, trail := f.affixWidths(h)
	return int(swt.DPIUtilPixelToPoint(lead+trail, int32(dpiOf(h)*100/96)))
}

// affixWidths returns the pixels the prefix and the suffix take, each with its gap.
func (f *textFrame) affixWidths(hwnd uintptr) (lead, trail int32) {
	a := f.affix
	if a == nil {
		return 0, 0
	}
	dpi := dpiOf(hwnd)
	if a.Image != nil {
		lead = px(float64(a.Image.GetBounds().Width), dpi) + px(affixGap, dpi)
	} else if a.Prefix != "" {
		lead = textWidth(hwnd, a.Prefix) + px(affixGap, dpi)
	}
	if a.Suffix != "" {
		trail = textWidth(hwnd, a.Suffix) + px(affixGap, dpi)
	}
	return lead, trail
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

// SetFieldFill makes the frame of a single-line Text the rounded fill of the control's background
// colour, without the hairline; radius is in dips. The caret, the focus accent and the affixes stay.
func SetFieldFill(t *swt.Text, radius int) {
	f := frames[uintptr(t.Handle)]
	if f == nil || f.multi {
		return
	}
	c := t.GetBackground()
	f.fill = &RGB{uint8(c.GetRed()), uint8(c.GetGreen()), uint8(c.GetBlue())}
	f.radius = radius
	redrawFrame(uintptr(t.Handle))
}

// SetFieldStroke draws a border of the RGBA colour around the frame of a single-line Text; the
// colour is blended over the fill, as the frame paints in plain RGB.
func SetFieldStroke(t *swt.Text, r, g, b, a uint8) {
	f := frames[uintptr(t.Handle)]
	if f == nil || f.multi {
		return
	}
	c := t.GetBackground()
	base := RGB{uint8(c.GetRed()), uint8(c.GetGreen()), uint8(c.GetBlue())}
	col := RGB{r, g, b}.over(float64(a)/255, base)
	f.stroke = &col
	redrawFrame(uintptr(t.Handle))
}

// SetFieldInset widens the padding between the frame of a Text and its text by x dips.
func SetFieldInset(t *swt.Text, x int) {
	f := frames[uintptr(t.Handle)]
	if f == nil || f.multi {
		return
	}
	f.padX = x
	frameChanged(uintptr(t.Handle))
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
		left, right := px(10+float64(f.padX), dpi), px(6+float64(f.padX), dpi)
		if lead, trail := f.affixWidths(hwnd); lead+trail > 0 {
			left += lead
			if trail > 0 {
				right = px(10, dpi) + trail
			}
		}
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
	case wmNCHitTest, wmSetCursor, wmNCLButtonDown, wmMouseMove, wmLButtonUp, wmCaptureChanged:
		return f.scrub(hwnd, msg, wp, lp, def)
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

func (f *textFrame) paint(hwnd uintptr) { paintInputAffix(hwnd, &f.t.Control, f) }

// paintInput paints the frame of the single-line edit hwnd, which belongs to the control c, in its
// non-client area.
func paintInput(hwnd uintptr, c *swt.Control) { paintInputAffix(hwnd, c, nil) }

// paintInputAffix is paintInput with the affixes of f, if f has any.
func paintInputAffix(hwnd uintptr, c *swt.Control, f *textFrame) {
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
	custom := f != nil && f.fill != nil
	if custom {
		fill, r = *f.fill, px(float64(f.radius), dpi)
	}
	var font uintptr
	if f != nil && f.affix != nil {
		font = call(pSendMessage, hwnd, wmGetFont, 0, 0)
	}
	drawBuffered(d, hdc, w, h, font, func(p *painter) {
		p.fillRect(bg, 0, 0, w, h)
		p.fillRound(fill, 0, 0, w, h, r)
		switch {
		case f != nil && f.stroke != nil:
			p.strokeRound(*f.stroke, 0, 0, w, h, r, px(1, dpi))
		case !custom:
			p.strokeRound(pal.Stroke(bg), 0, 0, w, h, r, 1)
		}
		switch {
		case enabled && focused(hwnd):
			p.line(pal.Accent, r, h-1, w-1-r, h-1, 2)
		case enabled && !custom && (f == nil || f.stroke == nil):
			p.line(pal.StrongStroke(bg), r, h-1, w-1-r, h-1, 1)
		}
		if font != 0 {
			// TextFillColorSecondary: black at 61% on light, white at 79% on dark.
			ink := pal.Black.over(0.61, fill)
			if pal.Dark {
				ink = pal.Black.over(0.79, fill)
			}
			if !enabled {
				ink = pal.Disabled
			}
			f.drawAffix(p, hwnd, w, ink)
		}
	})
}

// drawAffix draws the prefix where the value would start without it and the suffix where it would
// end, in the line of the value: the client area holds one line of the font from its top.
func (f *textFrame) drawAffix(p *painter, hwnd uintptr, w int32, ink RGB) {
	a := f.affix
	dpi := dpiOf(hwnd)
	x, top := px(10, dpi), f.in[1]
	th := textHeight(hwnd)
	z := p.gc.GetZoom()
	p.gc.SetForeground(p.color(ink))
	switch {
	case a.Image != nil:
		b := a.Image.GetBounds()
		ih := swt.DPIUtilPointToPixel(b.Height, z)
		p.gc.DrawImage(a.Image, swt.DPIUtilPixelToPoint(x, z), swt.DPIUtilPixelToPoint(top+(th-ih)/2, z))
	case a.Prefix != "":
		p.gc.DrawStringInPixels(a.Prefix, x, top, true)
	}
	if a.Suffix != "" {
		sw := p.gc.TextExtentInPixels(a.Suffix, swt.DRAW_TRANSPARENT).X
		p.gc.DrawStringInPixels(a.Suffix, w-px(10, dpi)-sw, top, true)
	}
}

// scrub runs a drag of the prefix: the press in the non-client area captures the mouse, each move
// reports its x distance in dips, a release without a move focuses the field with its text selected.
func (f *textFrame) scrub(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	if f.affix == nil || f.affix.Drag == nil || !f.t.IsEnabled() {
		return 0, false
	}
	switch msg {
	case wmNCHitTest:
		// The default hit test of a borderless child gives no code to the non-client area.
		if f.overPrefix(hwnd, int32(int16(lp)), int32(int16(lp>>16))) {
			return htBorder, true
		}
	case wmSetCursor:
		var pt point
		call(pGetCursorPos, uintptr(unsafe.Pointer(&pt)))
		if f.overPrefix(hwnd, pt.x, pt.y) {
			call(pSetCursor, call(pLoadCursor, 0, idcSizeWE))
			return 1, true
		}
	case wmNCLButtonDown:
		var pt = point{int32(int16(lp)), int32(int16(lp >> 16))}
		if !f.overPrefix(hwnd, pt.x, pt.y) {
			return 0, false
		}
		call(pScreenToClient, hwnd, uintptr(unsafe.Pointer(&pt)))
		f.drag, f.moved = &pt.x, false
		call(pSetCapture, hwnd)
		return 0, true
	case wmMouseMove:
		if f.drag == nil {
			return 0, false
		}
		x := int32(int16(lp))
		if dx := x - *f.drag; dx != 0 {
			*f.drag = x
			f.moved = true
			f.affix.Drag(float64(dx)*96/float64(dpiOf(hwnd)), wp&mkShift != 0)
		}
		return 0, true
	case wmLButtonUp:
		if f.drag == nil {
			return 0, false
		}
		f.drag = nil
		call(pReleaseCapture)
		if !f.moved {
			f.t.SetFocus()
			f.t.SelectAll()
		}
		return 0, true
	case wmCaptureChanged:
		f.drag = nil
	}
	return 0, false
}

// overPrefix reports whether the screen point x, y is on the frame left of the client area.
func (f *textFrame) overPrefix(hwnd uintptr, x, y int32) bool {
	var wr rect
	call(pGetWindowRect, hwnd, uintptr(unsafe.Pointer(&wr)))
	return y >= wr.top && y < wr.bottom && x >= wr.left && x < wr.left+f.in[0]
}

// textWidth is the width in pixels of s in the font of the window.
func textWidth(hwnd uintptr, s string) int32 {
	hdc := call(pGetDC, hwnd)
	if hdc == 0 {
		return 0
	}
	defer call(pReleaseDC, hwnd, hdc)
	old := call(pSelectObject, hdc, call(pSendMessage, hwnd, wmGetFont, 0, 0))
	defer call(pSelectObject, hdc, old)
	u, _ := syscall.UTF16FromString(s)
	var size point
	call(pGetTextExtentPoint32, hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&size)))
	return size.x
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
