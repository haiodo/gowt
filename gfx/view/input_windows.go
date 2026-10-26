//go:build windows

package view

import (
	"math"
	"syscall"
	"time"
	"unsafe"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

const (
	wmPaint       = 0x000F
	wmEraseBkgnd  = 0x0014
	wmSize        = 0x0005
	wmMouseWheel  = 0x020A
	wmMouseHWheel = 0x020E
	gwlpWndProc   = ^uintptr(3) // -4

	mkShift   = 0x0004
	mkControl = 0x0008

	vkShift   = 0x10
	vkControl = 0x11
	vkMenu    = 0x12

	imdtTouchpad = 4 // INPUT_MESSAGE_DEVICE_TYPE
	wheelDelta   = 120.0
	// a gesture is over when no wheel message came for this long
	wheelIdle = 120 * time.Millisecond
	// view points one wheel notch of a precision touchpad stands for, as three lines of lineStep
	pointsPerNotch = 3 * lineStep
	// zoom of one notch of a pinch the touchpad sends as Ctrl+wheel
	pinchPerNotch = 0.1
)

var (
	user32                = syscall.NewLazyDLL("user32.dll")
	procSetWindowLongPtr  = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtr  = user32.NewProc("GetWindowLongPtrW")
	procCallWindowProc    = user32.NewProc("CallWindowProcW")
	procValidateRect      = user32.NewProc("ValidateRect")
	procScreenToClient    = user32.NewProc("ScreenToClient")
	procGetKeyState       = user32.NewProc("GetKeyState")
	procGetAsyncKeyState  = user32.NewProc("GetAsyncKeyState")
	procGetInputSource    = user32.NewProc("GetCurrentInputMessageSource")
	procGetDpiForWindow   = user32.NewProc("GetDpiForWindow")
	procSystemParamsInfoW = user32.NewProc("SystemParametersInfoW")
)

// windowScale is the device pixels per view point of the window's monitor.
func windowScale(hwnd uintptr) float64 {
	if procGetDpiForWindow.Find() == nil {
		if dpi, _, _ := procGetDpiForWindow.Call(hwnd); dpi != 0 {
			return float64(dpi) / 96
		}
	}
	return 1
}

func keyDown(vk uintptr) bool {
	r, _, _ := procGetKeyState.Call(vk)
	return int16(r) < 0
}

func buttonDown(b int) bool {
	vk := [...]uintptr{1: 0x01, 2: 0x04, 3: 0x02}
	if b < 1 || b >= len(vk) {
		return false
	}
	r, _, _ := procGetAsyncKeyState.Call(vk[b])
	return int16(r) < 0
}

func wheelLines() float64 {
	var n uint32 = 3
	procSystemParamsInfoW.Call(0x0068, 0, uintptr(unsafe.Pointer(&n)), 0) // SPI_GETWHEELSCROLLLINES
	if n == 0 || n > 100 {                                                // 0: no scrolling, > 100: a page per notch (WHEEL_PAGESCROLL)
		n = 3
	}
	return float64(n)
}

// wheelState turns wheel messages into gestures. Windows sends no finger phase: a message after a
// quiet gap begins one and a quiet gap ends it.
type wheelState struct {
	last    time.Time
	gen     int
	mode    uint8 // 1 scroll, 2 pinch
	precise bool
	pos     gfx.Point
	mods    g.Mods
}

// wireInput feeds the router from SWT's mouse, key and focus events, and from the wheel messages of
// the canvas window. Everything runs on the UI thread.
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
	// SWT sets the mouse capture on a press, so the moves and the release keep coming outside the view
	v.cv.OnMouseDown(func(m g.Mouse) {
		Tracef("swt MouseDown (%d,%d) button=%d", m.X, m.Y, m.Button)
		v.r.pointer(PointerDown, pt(m), m.Button, m.Count, m.Mods)
		v.pollRelease()
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

	v.stops = append(v.stops, v.subclass())
}

// subclass replaces the window procedure of the canvas: it keeps GDI from painting over the swap
// chain and reads the wheel messages SWT would turn into integer lines.
func (v *View) subclass() (restore func()) {
	old, _, _ := procGetWindowLongPtr.Call(v.hwnd, gwlpWndProc)
	cb := syscall.NewCallback(func(hwnd, msg, wp, lp uintptr) uintptr {
		switch msg {
		case wmEraseBkgnd:
			return 1
		case wmPaint:
			procValidateRect.Call(hwnd, 0)
			v.host.Invalidate()
			v.resize.Store(true) // a frame to present again
			return 0
		case wmSize:
			v.resize.Store(true)
		case wmMouseWheel, wmMouseHWheel:
			if v.wheelMsg(msg == wmMouseHWheel, wp, lp) {
				return 0
			}
		}
		r, _, _ := procCallWindowProc.Call(old, hwnd, msg, wp, lp)
		return r
	})
	procSetWindowLongPtr.Call(v.hwnd, gwlpWndProc, cb)
	return func() { procSetWindowLongPtr.Call(v.hwnd, gwlpWndProc, old) }
}

func (v *View) isTouchpad() bool {
	if procGetInputSource.Find() != nil {
		return false
	}
	var src struct{ device, origin uint32 }
	if r, _, _ := procGetInputSource.Call(uintptr(unsafe.Pointer(&src))); r == 0 {
		return false
	}
	return src.device == imdtTouchpad
}

// wheelMsg handles WM_MOUSEWHEEL and WM_MOUSEHWHEEL; it reports whether the view took the message.
func (v *View) wheelMsg(horizontal bool, wp, lp uintptr) bool {
	delta := float64(int16(wp >> 16))
	keys := wp & 0xFFFF
	p := struct{ x, y int32 }{int32(int16(lp)), int32(int16(lp >> 16))}
	procScreenToClient.Call(v.hwnd, uintptr(unsafe.Pointer(&p)))
	pos := gfx.Point{X: float64(p.x) / v.dpr, Y: float64(p.y) / v.dpr}
	// a control over the view forwards the wheel it did not use to its parent, this window
	if v.covered(pos) {
		return false
	}
	mods := g.Mods{Shift: keys&mkShift != 0, Ctrl: keys&mkControl != 0, Alt: keyDown(vkMenu)}
	touchpad := v.isTouchpad()
	precise := touchpad || math.Mod(delta, wheelDelta) != 0
	// the touchpad sends a pinch as Ctrl+wheel without the Ctrl key being down
	pinch := touchpad && mods.Ctrl && !keyDown(vkControl)
	w := &v.wheel
	now := time.Now()
	mode := uint8(1)
	if pinch {
		mode = 2
	}
	phase := PhaseChanged
	if now.Sub(w.last) > wheelIdle || w.mode != mode {
		phase = PhaseBegan
	}
	w.last, w.mode, w.pos, w.mods, w.precise = now, mode, pos, mods, precise
	w.gen++
	v.armWheelEnd(w.gen)
	if pinch {
		v.r.magnify(pos, delta/wheelDelta*pinchPerNotch, phase, g.Mods{})
		return true
	}
	var dx, dy float64
	switch {
	case horizontal:
		dx = -delta / wheelDelta // a positive delta scrolls right: the content moves left
	default:
		dy = delta / wheelDelta
	}
	if precise {
		dx, dy = dx*pointsPerNotch, dy*pointsPerNotch
	} else {
		dx, dy = dx*wheelLines(), dy*wheelLines()
	}
	if mods.Shift && !horizontal && !mods.Ctrl {
		dx, dy = dy, 0 // Shift turns the wheel sideways, as in other Windows apps
	}
	ph := PhaseNone
	if precise {
		ph = phase
	}
	v.r.scroll(pos, dx, dy, precise, ph, PhaseNone, mods, 0)
	return true
}

// armWheelEnd ends the gesture gen when no newer wheel message came within wheelIdle.
func (v *View) armWheelEnd(gen int) {
	display := v.cv.Unwrap().GetDisplay()
	display.TimerExec(int32(wheelIdle/time.Millisecond), jrt.NewRunnable(func() {
		w := &v.wheel
		if v.gone.Load() || w.gen != gen {
			return
		}
		if w.mode == 2 {
			v.r.magnify(w.pos, 0, PhaseEnded, g.Mods{})
		} else if w.precise {
			v.r.scroll(w.pos, 0, 0, true, PhaseEnded, PhaseNone, w.mods, 0)
		}
		w.mode = 0
	}))
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
			down := buttonDown(b)
			if !down {
				Tracef("poll: button %d is up while held, releasing", b)
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

var cursorStyles = [...]int32{
	CursorArrow:      swt.CURSOR_ARROW,
	CursorHand:       swt.CURSOR_HAND,
	CursorGrab:       swt.CURSOR_HAND,
	CursorGrabbing:   swt.CURSOR_SIZEALL,
	CursorCrosshair:  swt.CURSOR_CROSS,
	CursorText:       swt.CURSOR_IBEAM,
	CursorResizeH:    swt.CURSOR_SIZEWE,
	CursorResizeV:    swt.CURSOR_SIZENS,
	CursorNotAllowed: swt.CURSOR_NO,
}

// SetCursor sets the cursor over the view. Windows has no open and closed hand: CursorGrab is the
// hand and CursorGrabbing the four-way arrow.
func (v *View) SetCursor(c Cursor) {
	if int(c) >= len(cursorStyles) {
		return
	}
	v.cv.Unwrap().SetCursor(v.cv.Unwrap().GetDisplay().GetSystemCursor(cursorStyles[c]))
}
