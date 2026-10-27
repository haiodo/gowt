//go:build darwin

package view

import (
	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/jrt"
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
	v.r.down = v.endEdit
	v.r.size = func() (float64, float64) {
		w, h := v.cv.Size()
		return float64(w), float64(h)
	}
	if cg.Tracing {
		v.r.trace = cg.Tracef
	}
	pt := func(m g.Mouse) gfx.Point { return gfx.Point{X: float64(m.X), Y: float64(m.Y)} }
	v.cv.OnMouseDown(func(m g.Mouse) {
		cg.Tracef("swt MouseDown (%d,%d) button=%d", m.X, m.Y, m.Button)
		v.r.pointer(PointerDown, pt(m), m.Button, m.Count, m.Mods)
		v.pollRelease()
	})
	v.cv.OnMouseUp(func(m g.Mouse) {
		cg.Tracef("swt MouseUp (%d,%d) button=%d", m.X, m.Y, m.Button)
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
			cg.Tracef("swt key down=%v code=%d", down, e.KeyCode)
			if down && e.KeyCode == int32(swt.ESC) && v.r.held != 0 {
				v.r.abort()
				return
			}
			if v.onKey != nil {
				focusTrace("view key down=%v code=%d char=%q: handler begin", down, e.KeyCode, rune(e.Character))
				v.onKey(&Key{Down: down, Code: int(e.KeyCode), Char: rune(e.Character), Mods: modsOfMask(e.StateMask)})
				focusTrace("view key down=%v: handler end, doit=%v", down, e.Doit)
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

// pollRelease watches the hardware button state while a button is held and reports the release
// when no event brought it. 20 ms is the latency ceiling of that fallback.
func (v *View) pollRelease() {
	if v.polling || v.r.held == 0 {
		return
	}
	v.polling = true
	display := v.cv.Unwrap().GetDisplay()
	var tick jrt.Runnable
	tick = jrt.NewRunnable(func() {
		if v.gone.Load() {
			return
		}
		v.r.settle(func(b int) bool {
			down := cg.ButtonDown(b)
			if !down {
				cg.Tracef("poll: button %d is up while held, releasing", b)
			}
			return down
		})
		if v.r.held == 0 {
			v.polling = false
			return
		}
		display.TimerExec(20, tick)
	})
	display.TimerExec(20, tick)
}

// modsOfFlags reads NSEventModifierFlags: shift 1<<17, control 1<<18, option 1<<19, command 1<<20.
func modsOfFlags(f uint64) g.Mods {
	return g.Mods{Shift: f&(1<<17) != 0, Ctrl: f&(1<<18) != 0, Alt: f&(1<<19) != 0, Cmd: f&(1<<20) != 0}
}

func (v *View) native(e cg.InputEvent) bool {
	pos, m := gfx.Point{X: e.X, Y: e.Y}, modsOfFlags(e.Flags)
	// a control over the view gets the scroll and the pinch itself
	if e.Kind != cg.InputMouseUp && v.covered(pos) {
		return false
	}
	switch e.Kind {
	case cg.InputScroll:
		v.r.scroll(pos, e.DX, e.DY, e.Precise, phaseOf(e.Phase), phaseOf(e.Momentum), m, e.Time)
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
