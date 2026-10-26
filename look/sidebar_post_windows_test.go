//go:build windows

package look_test

import (
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

// postBlocked brings the test window to the foreground: SendInput goes to the foreground window,
// so a window another app covers gets neither mouse nor keys.
func postBlocked(app *gowt.App) string {
	d := app.Unwrap()
	shells := d.GetShells()
	if len(shells) == 0 {
		return "no test window"
	}
	s := shells[0]
	s.ForceActive()
	for range 10 {
		for d.ReadAndDispatch() {
		}
		if win32.OSGetForegroundWindow() == s.Handle {
			return ""
		}
		time.Sleep(50 * time.Millisecond)
	}
	return "the test window is not the foreground window: input posted with SendInput would go elsewhere"
}

// postMouse posts a left-button event at x, y in display points through SendInput.
func postMouse(app *gowt.App, typ, x, y int32) bool {
	e := swt.NewEvent()
	e.Type, e.X, e.Y, e.Button = typ, x, y, 1
	return app.Unwrap().Post(e)
}
