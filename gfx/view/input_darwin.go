//go:build darwin

package view

import (
	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// wireInput feeds the router from SWT's mouse, key and focus events and from the native scroll
// and pinch events SWT drops the details of. Everything runs on the UI thread.
func (v *View) wireInput() {
	v.r = newRouter(v.comp, func(p gfx.Point) *gfx.Node {
		return v.scene.HitTest(p, 4/v.comp.Camera().Zoom)
	})
	v.r.pre = func(*Pointer) {
		if c := v.cv.Unwrap(); !c.IsFocusControl() {
			c.SetFocus()
		}
	}
	pt := func(m g.Mouse) gfx.Point { return gfx.Point{X: float64(m.X), Y: float64(m.Y)} }
	v.cv.OnMouseDown(func(m g.Mouse) { v.r.pointer(PointerDown, pt(m), m.Button, m.Count, m.Mods) })
	v.cv.OnMouseUp(func(m g.Mouse) { v.r.pointer(PointerUp, pt(m), m.Button, 0, m.Mods) })
	v.cv.OnMouseMove(func(m g.Mouse) { v.r.pointer(PointerMove, pt(m), 0, 0, m.Mods) })

	c := v.cv.Unwrap()
	c.AddFocusListener(swt.FocusListenerFocusLostAdapter(func(*swt.FocusEvent) {
		if v.r.held != 0 {
			v.r.pointer(PointerCancel, v.r.last, 0, 0, g.Mods{})
		}
	}))
	key := func(down bool) func(*swt.KeyEvent) {
		return func(e *swt.KeyEvent) {
			if v.onKey != nil {
				v.onKey(&Key{Down: down, Code: int(e.KeyCode), Char: rune(e.Character), Mods: modsOfMask(e.StateMask)})
			}
		}
	}
	c.AddKeyListener(swt.KeyListenerKeyPressedAdapter(key(true)))
	c.AddKeyListener(swt.KeyListenerKeyReleasedAdapter(key(false)))

	v.stops = append(v.stops,
		cg.MonitorInput(v.nsview, v.native),
		cg.OnWindowChange(v.nsview, func() { v.resize.Store(true) }),
	)
}

func modsOfMask(m int32) g.Mods {
	return g.Mods{Ctrl: m&swt.CTRL != 0, Shift: m&swt.SHIFT != 0, Alt: m&swt.ALT != 0, Cmd: m&swt.COMMAND != 0}
}

// modsOfFlags reads NSEventModifierFlags: shift 1<<17, control 1<<18, option 1<<19, command 1<<20.
func modsOfFlags(f uint64) g.Mods {
	return g.Mods{Shift: f&(1<<17) != 0, Ctrl: f&(1<<18) != 0, Alt: f&(1<<19) != 0, Cmd: f&(1<<20) != 0}
}

func (v *View) native(e cg.InputEvent) bool {
	pos, m := gfx.Point{X: e.X, Y: e.Y}, modsOfFlags(e.Flags)
	switch e.Kind {
	case cg.InputScroll:
		v.r.scroll(pos, e.DX, e.DY, e.Precise, phaseOf(e.Phase), phaseOf(e.Momentum), m)
	case cg.InputMagnify:
		v.r.magnify(pos, e.Magnification, phaseOf(e.Phase), m)
	case cg.InputSmartMagnify:
		v.r.smartZoom(pos, m)
	case cg.InputMouseUp:
		// SWT queues the release and may not deliver it when it happens outside the window; the
		// router drops the SWT one that follows
		v.r.pointer(PointerUp, pos, e.Button, 0, m)
		return false
	}
	return true
}

// OnPointer sets f to run for every press, move, release and cancel, with the scene point and the
// node under the pointer. While a button is held the moves and the release keep coming outside
// the view. A Ctrl-click is Button 1 with Mods.Ctrl.
func (v *View) OnPointer(f func(*Pointer)) { v.r.onPointer = f }

// OnScroll sets f to run for scroll gestures and wheel events. f returns true when it used the
// event; otherwise the view pans (zooms with Cmd or Ctrl held).
func (v *View) OnScroll(f func(*Scroll) bool) { v.r.onScroll = f }

// OnMagnify sets f to run for pinch steps. f returns true when it used the event; otherwise the
// view zooms about the pointer.
func (v *View) OnMagnify(f func(*Magnify) bool) { v.r.onMagnify = f }

// OnSmartZoom sets f to run for the two finger double tap. f returns true when it used the event;
// otherwise the view toggles between zoom 1 and the zoom before.
func (v *View) OnSmartZoom(f func(*SmartZoom) bool) { v.r.onSmart = f }

// OnKey sets f to run for key presses and releases while the view has the keyboard focus. A press
// on the view gives it the focus.
func (v *View) OnKey(f func(*Key)) { v.onKey = f }

// Pan moves the content by (dx, dy) view points.
func (v *View) Pan(dx, dy float64) { v.r.pan(dx, dy) }

// SetZoomLimits bounds the zoom the built-in pinch, wheel zoom and smart zoom reach. The default
// is 0.02 to 64.
func (v *View) SetZoomLimits(lo, hi float64) { v.r.minZ, v.r.maxZ = lo, hi }

var cursorNames = [...]string{
	CursorArrow:      "arrowCursor",
	CursorHand:       "pointingHandCursor",
	CursorGrab:       "openHandCursor",
	CursorGrabbing:   "closedHandCursor",
	CursorCrosshair:  "crosshairCursor",
	CursorText:       "IBeamCursor",
	CursorResizeH:    "resizeLeftRightCursor",
	CursorResizeV:    "resizeUpDownCursor",
	CursorNotAllowed: "operationNotAllowedCursor",
}

// SetCursor sets the cursor over the view. The shapes are the system cursors; CursorResizeH and
// CursorResizeV are the only resize cursors.
func (v *View) SetCursor(c Cursor) {
	cur, ok := v.cursor[c]
	if !ok {
		if int(c) < len(cursorNames) {
			if h := cg.SystemCursor(cursorNames[c]); h != 0 {
				cur = swt.CursorCocoa_new(v.cv.Unwrap().GetDisplay(), cocoa.NewNSCursorOverload1(int64(h)))
			}
		}
		if v.cursor == nil {
			v.cursor = map[Cursor]*swt.Cursor{}
		}
		v.cursor[c] = cur
	}
	v.cv.Unwrap().SetCursor(cur)
}
