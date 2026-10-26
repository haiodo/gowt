package gowt

import (
	"time"

	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

// postBlocked brings the test window to the foreground: SendInput goes to the foreground window.
func postBlocked(app *App) string {
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

// pointer converts display points to the pixels postMouse takes; call it on the UI thread.
func pointer(app *App, p swt.Point) swt.Point {
	q := app.Unwrap().TranslateToDisplayCoordinates(swt.NewPoint(p.X, p.Y))
	return swt.Point{X: q.X, Y: q.Y}
}

// postMouse sends a left-button event at x, y in screen pixels through SendInput. Unlike
// Display.Post it may run off the UI thread, which a drag needs: the OLE loop blocks that thread.
func postMouse(_ *App, typ, x, y int32) bool {
	in := win32.NewMOUSEINPUT()
	switch typ {
	case swt.MouseMove:
		vx, vy := win32.OSGetSystemMetrics(win32.OSSM_XVIRTUALSCREEN), win32.OSGetSystemMetrics(win32.OSSM_YVIRTUALSCREEN)
		w, h := win32.OSGetSystemMetrics(win32.OSSM_CXVIRTUALSCREEN), win32.OSGetSystemMetrics(win32.OSSM_CYVIRTUALSCREEN)
		in.DwFlags = win32.OSMOUSEEVENTF_MOVE | win32.OSMOUSEEVENTF_ABSOLUTE | win32.OSMOUSEEVENTF_VIRTUALDESK
		in.Dx = ((x-vx)*65535 + w - 2) / (w - 1)
		in.Dy = ((y-vy)*65535 + h - 2) / (h - 1)
	case swt.MouseDown:
		in.DwFlags = win32.OSMOUSEEVENTF_LEFTDOWN
	case swt.MouseUp:
		in.DwFlags = win32.OSMOUSEEVENTF_LEFTUP
	}
	input := win32.NewINPUT()
	input.Type = win32.OSINPUT_MOUSE
	input.Mi = in
	return win32.OSSendInput(1, input, win32.INPUTSizeof) != 0
}
