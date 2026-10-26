//go:build linux

package view

import (
	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cairo"
	"github.com/haiodo/gowt/swt"
)

func offscreenRenderer() (gfx.Renderer, error) { return cairo.NewRenderer(cairo.Options{LOD: true}) }

// touchpadStep is how many view points one unit of a touchpad's smooth scroll delta moves the
// content: GDK reports touchpad deltas in units of about ten pixels.
const touchpadStep = 10.0

// scrollState tells the first event of a touchpad scroll from the rest.
type scrollState struct{ active bool }

// wireInput feeds the router from SWT's mouse, key and focus events and from the GTK scroll and zoom
// gesture events SWT drops the details of. Everything runs on the UI thread.
func (v *View) wireInput() {
	v.r = newRouter(v.comp, func(p gfx.Point) *gfx.Node {
		return v.scene.HitTest(p, 4/v.comp.Camera().Zoom)
	})
	v.r.pre = func(*Pointer) {
		if c := v.cv.Unwrap(); !c.IsFocusControl() {
			c.SetFocus()
		}
	}
	v.r.down = v.endEdit
	v.r.size = func() (float64, float64) {
		w, h := v.cv.Size()
		return float64(w), float64(h)
	}
	if Tracing {
		v.r.trace = Tracef
	}
	pt := func(m g.Mouse) gfx.Point { return gfx.Point{X: float64(m.X), Y: float64(m.Y)} }
	v.cv.OnMouseDown(func(m g.Mouse) {
		Tracef("swt MouseDown (%d,%d) button=%d", m.X, m.Y, m.Button)
		v.r.pointer(PointerDown, pt(m), m.Button, m.Count, m.Mods)
	})
	v.cv.OnMouseUp(func(m g.Mouse) {
		Tracef("swt MouseUp (%d,%d) button=%d", m.X, m.Y, m.Button)
		v.r.pointer(PointerUp, pt(m), m.Button, 0, m.Mods)
	})
	v.cv.OnMouseMove(func(m g.Mouse) { v.r.pointer(PointerMove, pt(m), 0, 0, m.Mods) })

	c := v.cv.Unwrap()
	c.AddFocusListener(swt.FocusListenerFocusLostAdapter(func(*swt.FocusEvent) {
		if v.r.held != 0 {
			v.r.pointer(PointerCancel, v.r.last, 0, 0, g.Mods{})
		}
	}))
	key := func(down bool) func(*swt.KeyEvent) {
		return func(e *swt.KeyEvent) {
			Tracef("swt key down=%v code=%d", down, e.KeyCode)
			if down && e.KeyCode == int32(swt.ESC) && v.r.held != 0 {
				v.r.abort()
				return
			}
			if v.onKey != nil {
				v.onKey(&Key{Down: down, Code: int(e.KeyCode), Char: rune(e.Character), Mods: modsOfMask(e.StateMask)})
			}
		}
	}
	c.AddKeyListener(swt.KeyListenerKeyPressedAdapter(key(true)))
	c.AddKeyListener(swt.KeyListenerKeyReleasedAdapter(key(false)))

	addEvents(v.handle, gdkScrollMask|gdkSmoothScrollMask|gdkTouchpadMask)
	v.signals = append(v.signals, connect(v.handle, "scroll-event", v.onScroll))

	// The gesture belongs to the widget; the extra reference keeps it alive after attach returns.
	zoom := objectRef(gestureZoomNew(v.handle))
	prev := 1.0
	v.signals = append(v.signals,
		connect(zoom, "begin", func(gesture, seq, data uintptr) {
			prev = 1
			v.magnify(gesture, 0, PhaseBegan)
		}),
		connect(zoom, "scale-changed", func(gesture, data uintptr) {
			// the scale arrives as a double argument; the delta is read back from the gesture
			s := zoomDelta(gesture)
			if prev > 0 && s > 0 {
				v.magnify(gesture, s/prev-1, PhaseChanged)
				prev = s
			}
		}),
		connect(zoom, "end", func(gesture, seq, data uintptr) { v.magnify(gesture, 0, PhaseEnded) }),
		connect(zoom, "cancel", func(gesture, seq, data uintptr) { v.magnify(gesture, 0, PhaseCancelled) }),
	)
}

func (v *View) magnify(gesture uintptr, delta float64, ph Phase) {
	if v.gone.Load() {
		return
	}
	var x, y float64
	pos := v.r.last
	if gestureCenter(gesture, &x, &y) != 0 {
		pos = gfx.Point{X: x, Y: y}
	}
	if v.covered(pos) {
		return
	}
	v.r.magnify(pos, delta, ph, v.r.lastMods)
}

// onScroll is the "scroll-event" handler. Its position is the last one SWT reported for the pointer,
// which is in canvas coordinates.
func (v *View) onScroll(widget, ev, data uintptr) uintptr {
	if v.gone.Load() {
		return 0
	}
	// a control over the view passes on the scroll it did not use; its window is a child of ours
	if w := eventWidget(ev); w != v.handle && isAncestor(w, v.handle) != 0 {
		return 0
	}
	var state uint32
	eventState(ev, &state)
	m := g.Mods{Shift: state&1 != 0, Ctrl: state&4 != 0, Alt: state&8 != 0}
	var dx, dy float64
	var dir int32
	precise := false
	ph := PhaseNone
	switch {
	case scrollDirection(ev, &dir) != 0 && dir != scrollSmooth:
		switch dir {
		case 0: // up: the content moves down
			dy = 1
		case 1:
			dy = -1
		case 2:
			dx = 1
		case 3:
			dx = -1
		}
	case scrollDeltas(ev, &dx, &dy) != 0:
		dx, dy = -dx, -dy
		if d := eventSourceDev(ev); d != 0 && deviceSource(d) == sourceTouchpad {
			precise = true
			dx, dy = dx*touchpadStep, dy*touchpadStep
			switch {
			case scrollIsStop(ev) != 0:
				ph, v.scrolls.active = PhaseEnded, false
			case !v.scrolls.active:
				ph, v.scrolls.active = PhaseBegan, true
			default:
				ph = PhaseChanged
			}
		}
	default:
		return 0
	}
	if dx == 0 && dy == 0 && ph != PhaseEnded {
		return 1
	}
	v.r.scroll(v.r.last, dx, dy, precise, ph, PhaseNone, m, 0)
	return 1
}

var cursorStyles = [...]int32{
	CursorArrow:      swt.CURSOR_ARROW,
	CursorHand:       swt.CURSOR_HAND,
	CursorGrab:       swt.CURSOR_SIZEALL,
	CursorGrabbing:   swt.CURSOR_SIZEALL,
	CursorCrosshair:  swt.CURSOR_CROSS,
	CursorText:       swt.CURSOR_IBEAM,
	CursorResizeH:    swt.CURSOR_SIZEWE,
	CursorResizeV:    swt.CURSOR_SIZENS,
	CursorNotAllowed: swt.CURSOR_NO,
}

// SetCursor sets the cursor over the view. The shapes are the system cursors; GTK has no open and
// closed hand in SWT's set, so the grab shapes are the move cursor.
func (v *View) SetCursor(c Cursor) {
	cur, ok := v.cursor[c]
	if !ok && int(c) < len(cursorStyles) {
		cur = swt.NewCursorDeviceStyle(v.cv.Unwrap().GetDisplay(), cursorStyles[c])
		if v.cursor == nil {
			v.cursor = map[Cursor]*swt.Cursor{}
		}
		v.cursor[c] = cur
	}
	v.cv.Unwrap().SetCursor(cur)
}
