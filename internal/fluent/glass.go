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
	s.c.AddListener(swt.Resize, listener(func(*swt.Event) { s.c.Redraw() }))
	d := s.c.GetDisplay()
	// The focus control is the new one only after the event: look at it on the next turn.
	f := listener(func(*swt.Event) {
		if s.ring {
			d.AsyncExec(jrt.NewRunnable(func() {
				if !s.c.IsDisposed() {
					s.c.Redraw()
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
			if ch.GetName() == "Text" {
				Unframe(ch, s)
				if !ch.HasCustomBackground() || frames[uintptr(ch.Handle)] != nil {
					ch.SetBackgroundWithColor(col)
				}
			}
		}
	}
	walk(s.c)
}

func (s *surface) proc(hwnd, msg, wp, lp uintptr, def func() uintptr) (uintptr, bool) {
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
	d := s.c.GetDisplay()
	pal := Current(d)
	bg := parentBackground(&s.c.Control)
	r := s.radiusPx()
	ring := s.ring && s.hasFocus()
	drawBuffered(d, dst, w, h, func(p *painter) {
		p.fillRect(bg, 0, 0, w, h)
		p.fillRound(pal.Fill, -ox, -oy, cw, ch, r)
		if s.border {
			p.strokeRound(pal.StrongStroke(bg).over(0.5, pal.Stroke(bg)), -ox, -oy, cw, ch, r, 1)
		}
		if ring {
			p.strokeRound(pal.Accent, 1-ox, 1-oy, cw-2, ch-2, max(r-1, 0), 2)
		}
	})
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
