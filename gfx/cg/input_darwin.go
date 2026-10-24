//go:build darwin

package cg

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/haiodo/gowt/internal/cocoa"
)

// SWT hands a scroll or a pinch to the app as integers and drops the trackpad phase and the
// momentum; these functions read the NSEvent itself.

var (
	msg2       func(recv, sel, a, b uintptr) uintptr
	msg4       func(recv, sel, a, b, c, d uintptr) uintptr
	msgPtRet   func(recv, sel uintptr) cgPoint
	msgConvert func(recv, sel uintptr, p cgPoint, v uintptr) cgPoint

	inputOnce sync.Once
	isel      struct {
		typ, window, locationInWindow, scrollingDeltaX, scrollingDeltaY, hasPrecise, phase, momentumPhase,
		magnification, modifierFlags, convertPoint, isFlipped, bounds, addMonitor, removeMonitor, defaultCenter,
		addObserver, removeObserver, object, buttonNumber, respondsTo, windowNumber, pressedButtons uintptr
	}
	clsNSEvent, clsNotifCenter, clsNSCursor uintptr
)

func initInput() {
	inputOnce.Do(func() {
		initHost()
		for _, f := range []any{&msg2, &msg4, &msgPtRet, &msgConvert} {
			purego.RegisterLibFunc(f, purego.RTLD_DEFAULT, "objc_msgSend")
		}
		s := func(n string) uintptr { return uintptr(cocoa.OSSel_registerName(n)) }
		isel.typ, isel.window, isel.locationInWindow = s("type"), s("window"), s("locationInWindow")
		isel.scrollingDeltaX, isel.scrollingDeltaY, isel.hasPrecise = s("scrollingDeltaX"), s("scrollingDeltaY"), s("hasPreciseScrollingDeltas")
		isel.phase, isel.momentumPhase, isel.magnification, isel.modifierFlags = s("phase"), s("momentumPhase"), s("magnification"), s("modifierFlags")
		isel.convertPoint, isel.isFlipped, isel.bounds = s("convertPoint:fromView:"), s("isFlipped"), s("bounds")
		isel.addMonitor, isel.removeMonitor = s("addLocalMonitorForEventsMatchingMask:handler:"), s("removeMonitor:")
		isel.defaultCenter, isel.addObserver, isel.removeObserver = s("defaultCenter"), s("addObserverForName:object:queue:usingBlock:"), s("removeObserver:")
		isel.object, isel.buttonNumber, isel.respondsTo = s("object"), s("buttonNumber"), s("respondsToSelector:")
		isel.windowNumber, isel.pressedButtons = s("windowNumber"), s("pressedMouseButtons")
		clsNSEvent = uintptr(cocoa.OSObjc_lookUpClass("NSEvent"))
		clsNotifCenter = uintptr(cocoa.OSObjc_lookUpClass("NSNotificationCenter"))
		clsNSCursor = uintptr(cocoa.OSObjc_lookUpClass("NSCursor"))
	})
}

// InputKind tells which native event an InputEvent came from.
type InputKind uint8

const (
	InputScroll InputKind = iota
	InputMagnify
	InputSmartMagnify
	// InputMouseUp reports a button release anywhere, also outside the view and the window; it is
	// never consumed. X, Y may lie outside the view.
	InputMouseUp
)

// InputEvent is a scroll, pinch or smart-zoom (two-finger double tap) event over a view.
type InputEvent struct {
	Kind InputKind
	// X, Y: the pointer in the view's points, y down; they lie inside the view.
	X, Y float64
	// DX, DY: content shift of a scroll, points when Precise, lines for a wheel. The sign
	// follows natural scrolling: positive DY moves the content down.
	DX, DY  float64
	Precise bool
	// Phase and Momentum are raw NSEventPhase masks: 1 began, 2 stationary, 4 changed, 8 ended,
	// 16 cancelled, 32 may begin. Momentum is nonzero while the content coasts after the fingers left.
	Phase, Momentum uint64
	// Magnification: the relative zoom change of this pinch event.
	Magnification float64
	Flags         uint64 // NSEventModifierFlags
	Button        int    // InputMouseUp: SWT numbering, 1 left, 2 middle, 3 right
}

// NSEventType values and masks.
const (
	nsLeftMouseUp  = 2
	nsRightMouseUp = 4
	nsOtherMouseUp = 26
	nsScrollWheel  = 22
	nsMagnify      = 30
	nsSmartMagnify = 32
	nsMonitorMasks = 1<<nsScrollWheel | 1<<nsMagnify | 1<<nsSmartMagnify | 1<<nsLeftMouseUp | 1<<nsRightMouseUp | 1<<nsOtherMouseUp
	// the traced types: press and drag of the three buttons
	nsTraceMasks   = 1<<1 | 1<<3 | 1<<25 | 1<<6 | 1<<7 | 1<<27
	notifUTF8      = 0x08000100
	notifBacking   = "NSWindowDidChangeBackingPropertiesNotification"
	notifChangeScr = "NSWindowDidChangeScreenNotification"
)

// swtButton maps NSEvent.buttonNumber (0 left, 1 right, 2 middle) to SWT's numbering.
func swtButton(n int) int {
	switch n {
	case 0:
		return 1
	case 1:
		return 3
	case 2:
		return 2
	}
	return n + 1
}

// Tracing is set by GFXDEMO_TRACE: the monitor and the view print every mouse event they see.
var Tracing = os.Getenv("GFXDEMO_TRACE") != ""

// Tracef prints a line to stderr when Tracing is set.
func Tracef(format string, a ...any) {
	if Tracing {
		fmt.Fprintf(os.Stderr, "trace %s %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, a...))
	}
}

var dragTrace int

func traceEvent(ev, view uintptr) {
	t := send0(ev, isel.typ)
	if t == 6 || t == 7 || t == 27 {
		if dragTrace++; dragTrace%10 != 1 {
			return
		}
	}
	loc := msgPtRet(ev, isel.locationInWindow)
	Tracef("nsevent type=%d win=%d viewwin=%d loc=(%.0f,%.0f) button=%d pressed=%b", t,
		int64(send0(ev, isel.windowNumber)), int64(send0(send0(view, isel.window), isel.windowNumber)),
		loc.X, loc.Y, int64(send0(ev, isel.buttonNumber)), send0(clsNSEvent, isel.pressedButtons))
}

// ButtonDown reports whether the mouse button b (SWT numbering) is held right now, whatever
// events reached the app.
func ButtonDown(b int) bool {
	initInput()
	n := b - 1
	switch b {
	case 3:
		n = 1
	case 2:
		n = 2
	}
	return send0(clsNSEvent, isel.pressedButtons)>>uint(n)&1 != 0
}

func nsBool(v uintptr) bool { return v&0xff != 0 }

// readInput decodes ev; the point is in window coordinates, ok is false for other event types.
func readInput(ev uintptr) (e InputEvent, win cgPoint, ok bool) {
	switch send0(ev, isel.typ) {
	case nsScrollWheel:
		e.Kind = InputScroll
		e.DX, e.DY = msgF64Ret(ev, isel.scrollingDeltaX), msgF64Ret(ev, isel.scrollingDeltaY)
		e.Precise = nsBool(send0(ev, isel.hasPrecise))
		e.Phase, e.Momentum = uint64(send0(ev, isel.phase)), uint64(send0(ev, isel.momentumPhase))
	case nsMagnify:
		e.Kind = InputMagnify
		e.Magnification = msgF64Ret(ev, isel.magnification)
		e.Phase = uint64(send0(ev, isel.phase))
	case nsSmartMagnify:
		e.Kind = InputSmartMagnify
	case nsLeftMouseUp, nsRightMouseUp, nsOtherMouseUp:
		e.Kind = InputMouseUp
		e.Button = swtButton(int(send0(ev, isel.buttonNumber)))
	default:
		return e, win, false
	}
	e.Flags = uint64(send0(ev, isel.modifierFlags))
	return e, msgPtRet(ev, isel.locationInWindow), true
}

// viewPoint maps a window point to the view's points, y down; in is false outside the view.
func viewPoint(view uintptr, win cgPoint) (x, y float64, in bool) {
	p := msgConvert(view, isel.convertPoint, win, 0)
	b := msgRectRet(view, isel.bounds)
	if !nsBool(send0(view, isel.isFlipped)) {
		p.Y = b.H - p.Y
	}
	return p.X, p.Y, p.X >= 0 && p.Y >= 0 && p.X < b.W && p.Y < b.H
}

// MonitorInput calls fn on the main thread for scroll, pinch and smart-zoom events inside view.
// fn returns true to consume the event, so SWT does not also send its own MouseWheel or Gesture.
func MonitorInput(view uintptr, fn func(InputEvent) bool) (stop func()) {
	initInput()
	enableSmartMagnify(view)
	block := objc.NewBlock(func(_ objc.Block, ev objc.ID) objc.ID {
		if Tracing {
			traceEvent(uintptr(ev), view)
		}
		e, win, ok := readInput(uintptr(ev))
		if !ok || send0(uintptr(ev), isel.window) != send0(view, isel.window) {
			return ev
		}
		x, y, in := viewPoint(view, win)
		e.X, e.Y = x, y
		if e.Kind == InputMouseUp {
			fn(e)
			return ev
		}
		if !in {
			return ev
		}
		if fn(e) {
			return 0
		}
		return ev
	})
	masks := uintptr(nsMonitorMasks)
	if Tracing {
		masks |= nsTraceMasks
	}
	mon := msg2(clsNSEvent, isel.addMonitor, masks, uintptr(block))
	return func() {
		if mon != 0 {
			msgInt(clsNSEvent, isel.removeMonitor, mon)
			block.Release()
			mon = 0
		}
	}
}

// OnWindowChange calls fn, on the main thread, when the window of view moves to another screen or
// its backing scale changes. The returned function stops it.
func OnWindowChange(view uintptr, fn func()) (stop func()) {
	initInput()
	block := objc.NewBlock(func(_ objc.Block, note objc.ID) {
		if w := send0(view, isel.window); w != 0 && send0(uintptr(note), isel.object) == w {
			fn()
		}
	})
	center := send0(clsNotifCenter, isel.defaultCenter)
	var tokens []uintptr
	for _, name := range []string{notifBacking, notifChangeScr} {
		b := []byte(name)
		s := cfStringCreateWithBytes(0, &b[0], len(b), notifUTF8, false)
		tokens = append(tokens, msg4(center, isel.addObserver, s, 0, 0, uintptr(block)))
		cfRelease(s)
	}
	return func() {
		for _, t := range tokens {
			msgInt(center, isel.removeObserver, t)
		}
		tokens = nil
		block.Release()
	}
}

// SystemCursor returns the NSCursor class-method result for name, such as "openHandCursor", or 0
// if NSCursor has no such cursor. The cursors are shared objects: do not release them.
func SystemCursor(name string) uintptr {
	initInput()
	sel := uintptr(cocoa.OSSel_registerName(name))
	if !nsBool(msgInt(clsNSCursor, uintptr(cocoa.OSSel_registerName("respondsToSelector:")), sel)) {
		return 0
	}
	return send0(clsNSCursor, sel)
}

// enableSmartMagnify makes the view's class respond to smartMagnifyWithEvent:. AppKit sends the
// two finger double tap only to a responder that implements it; SWT's view classes do not. The
// monitor sees the event first, so the method itself does nothing.
func enableSmartMagnify(view uintptr) {
	sel := uintptr(cocoa.OSSel_registerName("smartMagnifyWithEvent:"))
	if view == 0 || nsBool(msgInt(view, isel.respondsTo, sel)) {
		return
	}
	smartOnce.Do(func() {
		purego.RegisterLibFunc(&objectGetClass, purego.RTLD_DEFAULT, "object_getClass")
		purego.RegisterLibFunc(&classAddMethod, purego.RTLD_DEFAULT, "class_addMethod")
		smartIMP = purego.NewCallback(func(self, sel, ev uintptr) {})
	})
	types := []byte("v@:@\x00")
	classAddMethod(objectGetClass(view), sel, smartIMP, &types[0])
}

var (
	smartOnce      sync.Once
	smartIMP       uintptr
	objectGetClass func(obj uintptr) uintptr
	classAddMethod func(cls, sel, imp uintptr, types *byte) bool
)
