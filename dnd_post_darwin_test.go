package gowt

import (
	"github.com/ebitengine/purego"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// postBlocked says why posted events cannot reach the test window: CGEventPost drops them without
// the Accessibility permission. The app is activated, or the first click only wakes the window.
func postBlocked(*App) string {
	if sym, err := purego.Dlsym(purego.RTLD_DEFAULT, "CGPreflightPostEventAccess"); err == nil {
		var allowed func() bool
		purego.RegisterFunc(&allowed, sym)
		if !allowed() {
			return "no permission to post events: allow the terminal in System Settings > Privacy & Security > Accessibility"
		}
	}
	app := cocoa.NSApplicationSharedApplication()
	app.ActivateIgnoringOtherApps(true)
	if !app.IsActive() {
		return "the test app is not the frontmost app"
	}
	return ""
}

// kCGEventLeftMouseDragged; the cocoa package has no constant for it.
const leftMouseDragged = 6

var buttonDown bool

// pointer converts display points to the coordinates postMouse takes; call it on the UI thread.
func pointer(_ *App, p swt.Point) swt.Point { return p }

// postMouse posts a left-button event at x, y in display points; a move with the button down is a
// drag, as the system reports it (Display.Post posts every move as a plain move).
func postMouse(_ *App, typ, x, y int32) bool {
	kind := map[int32]int32{swt.MouseMove: cocoa.OSKCGEventMouseMoved, swt.MouseDown: cocoa.OSKCGEventLeftMouseDown,
		swt.MouseUp: cocoa.OSKCGEventLeftMouseUp}[typ]
	if typ == swt.MouseMove && buttonDown {
		kind = leftMouseDragged
	}
	buttonDown = typ == swt.MouseDown || buttonDown && typ != swt.MouseUp
	src := cocoa.OSCGEventSourceCreate(cocoa.OSKCGEventSourceStateHIDSystemState)
	ev := cocoa.OSCGEventCreateMouseEvent(src, kind, cocoa.CGPoint{X: float64(x), Y: float64(y)}, 0)
	if src != 0 {
		cocoa.OSCFRelease(src)
	}
	if ev == 0 {
		return false
	}
	cocoa.OSCGEventPost(cocoa.OSKCGHIDEventTap, ev)
	cocoa.OSCFRelease(ev)
	return true
}
