package gowt

import (
	"testing"
	"time"
)

// GC.Size is the canvas client area at paint time, the same numbers CanvasView.Size gives.
func TestGCSize(t *testing.T) {
	var w0, h0, cw, ch, drawn int
	guiRun(t, func(app *App, w *Window) {
		cv := put[*CanvasView](w, Canvas(func(g *GC) {
			w0, h0 = g.Size()
			drawn++
		}, Size(160, 90)))
		w.Show()
		cv.Redraw()
		cv.Unwrap().Update()
		for i := 0; i < 20 && drawn == 0; i++ {
			for app.Unwrap().ReadAndDispatch() {
			}
			time.Sleep(10 * time.Millisecond)
		}
		cw, ch = cv.Size()
	})
	if drawn == 0 || w0 != 160 || h0 != 90 || cw != w0 || ch != h0 {
		t.Fatalf("painted %d times: GC.Size %dx%d, CanvasView.Size %dx%d, want 160x90", drawn, w0, h0, cw, ch)
	}
}
