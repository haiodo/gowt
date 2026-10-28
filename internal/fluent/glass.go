//go:build windows

package fluent

import (
	"unsafe"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// Capsule is the radius that makes the shorter side a half circle, as look.Capsule.
const Capsule = -1

// surface is the rounded input surface that look.SetGlass draws on Windows: no Mica, a WinUI TextBox
// fill with a corner radius, an optional hairline and an accent ring while the focus is inside.
type surface struct {
	c      *swt.Composite
	on     bool
	radius float64 // points
	border bool
	ring   bool
	hooked bool
	labels map[uintptr]bool // adopted STATIC children
}

var surfaces = map[*swt.Composite]*surface{}

func surfaceOf(c *swt.Composite) *surface {
	s := surfaces[c]
	if s == nil {
		s = &surface{c: c, radius: 12}
		surfaces[c] = s
		c.AddDisposeListener(disposer(func() { delete(surfaces, c) }))
	}
	return s
}

// Glass turns the surface on or off; on, the Text fields inside c lose their frame and take the fill.
func Glass(c *swt.Composite, on bool) {
	s := surfaceOf(c)
	s.on = on
	s.hook()
	if on {
		s.adoptChildren()
	}
	c.Redraw()
}

func GlassRadius(c *swt.Composite, r float64) {
	s := surfaceOf(c)
	s.radius = r
	if s.on {
		s.adoptChildren()
	}
	c.Redraw()
}

// GlassBorder sets the hairline of the surface of c, a container or a push button.
func GlassBorder(c *swt.Control, on bool) {
	if b := ButtonOf(c); b != nil {
		Button(b, false, false, on)
		return
	}
	cc := compositeOf(c)
	s := surfaceOf(cc)
	s.border = on
	s.hook()
	cc.Redraw()
}

func GlassFocusRing(ctl *swt.Control, on bool) {
	c := compositeOf(ctl)
	s := surfaceOf(c)
	s.ring = on
	s.hook()
	c.Redraw()
}

// ButtonOf returns c as a push button, or nil.
func ButtonOf(c *swt.Control) *swt.Button {
	if c.GetName() != "Button" || c.GetStyle()&(swt.CHECK|swt.RADIO|swt.TOGGLE|swt.ARROW) != 0 {
		return nil
	}
	// swt.Button starts with its Control.
	return (*swt.Button)(unsafe.Pointer(c))
}

// compositeOf is for the controls of the glass calls that are no button: a Composite starts with its Control.
func compositeOf(c *swt.Control) *swt.Composite { return (*swt.Composite)(unsafe.Pointer(c)) }

func (s *surface) hook() {
	if s.hooked {
		return
	}
	s.hooked = true
	// A plain Composite sends no Paint events, so the surface is the erased background.
	attach(uintptr(s.c.Handle), s.proc)
	s.c.AddListener(swt.Resize, newListener(func(*swt.Event) { s.redraw() }))
	d := s.c.GetDisplay()
	// The focus control is the new one only after the event: look at it on the next turn.
	f := newListener(func(*swt.Event) {
		if s.ring {
			d.AsyncExec(jrt.NewRunnable(func() {
				if !s.c.IsDisposed() {
					s.redraw()
				}
			}))
		}
	})
	d.AddFilter(swt.FocusIn, f)
	d.AddFilter(swt.FocusOut, f)
	s.c.AddDisposeListener(disposer(func() {
		d.RemoveFilter(swt.FocusIn, f)
		d.RemoveFilter(swt.FocusOut, f)
	}))
}

// adoptChildren frames nothing inside the surface: its Text fields lose their own frame and take
// the colour of the fill.
func (s *surface) adoptChildren() {
	pal := Current(s.c.GetDisplay())
	col := swt.NewColorRedGreenBlue(int32(pal.Fill.R), int32(pal.Fill.G), int32(pal.Fill.B))
	var walk func(cc *swt.Composite)
	walk = func(cc *swt.Composite) {
		for _, ch := range cc.GetChildren() {
			switch ch.GetName() {
			case "Text":
				Unframe(ch, s)
				if !ch.HasCustomBackground() || frames[uintptr(ch.Handle)] != nil {
					ch.SetBackgroundWithColor(col)
				}
			case "Label":
				s.adoptLabel(ch)
			}
		}
	}
	walk(s.c)
}

// adoptLabel makes a STATIC child erase with the slice of the surface under it; proc draws its
// text on a transparent background (WM_CTLCOLORSTATIC).
func (s *surface) adoptLabel(l *swt.Control) {
	if s.labels == nil {
		s.labels = map[uintptr]bool{}
	}
	s.labels[uintptr(l.Handle)] = true
	attach(uintptr(l.Handle), func(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
		if msg != wmEraseBkgnd || !s.on || wp == 0 {
			return 0, false
		}
		var cr rect
		call(pGetClientRect, hwnd, uintptr(unsafe.Pointer(&cr)))
		if cr.right <= 0 || cr.bottom <= 0 {
			return 1, true
		}
		drawBuffered(s.c.GetDisplay(), wp, cr.right, cr.bottom, 0, func(p *painter) { s.paintSlice(p, hwnd, cr.right, cr.bottom) })
		return 1, true
	})
	l.Redraw()
}

// redraw repaints the surface and its children, which paint a slice of it.
func (s *surface) redraw() {
	s.c.Redraw()
	for _, ch := range s.c.GetChildren() {
		ch.Redraw()
	}
}

// surfaceOver is the glass surface that is the parent of c, or nil.
func surfaceOver(c *swt.Control) *surface {
	p := c.GetParent()
	if p == nil {
		return nil
	}
	if s := surfaces[p]; s != nil && s.on {
		return s
	}
	return nil
}

// paintSlice paints the part of the surface under the w by h client area of the child hwnd.
func (s *surface) paintSlice(p *painter, hwnd uintptr, w, h int32) {
	var sr rect
	so, org := point{}, point{}
	sh := uintptr(s.c.Handle)
	call(pGetClientRect, sh, uintptr(unsafe.Pointer(&sr)))
	call(pClientToScreen, sh, uintptr(unsafe.Pointer(&so)))
	call(pClientToScreen, hwnd, uintptr(unsafe.Pointer(&org)))
	s.paint(p, w, h, org.x-so.x, org.y-so.y, sr.right, sr.bottom)
}

func (s *surface) proc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
	if msg == wmPaint && s.on && !s.c.IsListening(swt.Paint) {
		// SWT's default paint overwrites the erased surface between the children with the parent colour.
		var ps [72]byte // PAINTSTRUCT
		call(pBeginPaint, hwnd, uintptr(unsafe.Pointer(&ps[0])))
		call(pEndPaint, hwnd, uintptr(unsafe.Pointer(&ps[0])))
		return 0, true
	}
	if msg == wmCtlColorStatic && s.on && s.labels[lp] {
		// SWT sets the text colour; the background stays as the child erased it.
		def()
		call(pSetBkMode, wp, 1)               // TRANSPARENT
		return call(pGetStockObject, 5), true // NULL_BRUSH
	}
	if msg != wmEraseBkgnd || !s.on || wp == 0 {
		return 0, false
	}
	var cr rect
	call(pGetClientRect, hwnd, uintptr(unsafe.Pointer(&cr)))
	w, h := cr.right, cr.bottom
	if w <= 0 || h <= 0 {
		return 1, true
	}
	s.draw(wp, w, h, 0, 0, w, h)
	return 1, true
}

// draw paints the surface of size cw by ch into a w by h buffer on dst, the buffer's origin at (ox, oy)
// of the surface: a Text inside it paints its non-client area from the same shape.
func (s *surface) draw(dst uintptr, w, h, ox, oy, cw, ch int32) {
	drawBuffered(s.c.GetDisplay(), dst, w, h, 0, func(p *painter) { s.paint(p, w, h, ox, oy, cw, ch) })
}

func (s *surface) paint(p *painter, w, h, ox, oy, cw, ch int32) {
	pal := Current(s.c.GetDisplay())
	bg := parentBackground(&s.c.Control)
	r := s.radiusPx()
	p.fillRect(bg, 0, 0, w, h)
	p.fillRound(pal.Fill, -ox, -oy, cw, ch, r)
	if s.border {
		p.strokeRound(pal.StrongStroke(bg).over(0.5, pal.Stroke(bg)), -ox, -oy, cw, ch, r, 1)
	}
	if s.ring && s.hasFocus() {
		p.strokeRound(pal.Accent, 1-ox, 1-oy, cw-2, ch-2, max(r-1, 0), 2)
	}
}

// radiusPx is the corner radius in device pixels; Capsule is half the shorter side.
func (s *surface) radiusPx() int32 {
	z := s.c.GetAutoscalingZoom()
	if s.radius >= 0 {
		return swt.DPIUtilPointToPixel(int32(s.radius+0.5), z)
	}
	sz := s.c.GetSize()
	return min(swt.DPIUtilPointToPixel(sz.X, z), swt.DPIUtilPointToPixel(sz.Y, z)) / 2
}

func (s *surface) hasFocus() bool {
	f := s.c.GetDisplay().GetFocusControl()
	for f != nil {
		if f == &s.c.Control {
			return true
		}
		p := f.GetParent()
		if p == nil {
			return false
		}
		f = &p.Control
	}
	return false
}
