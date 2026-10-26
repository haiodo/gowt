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

// OnResize reports the first size once, also when it is known before the loop runs, and then every
// size layout gives later; the numbers are CanvasView.Size's.
func TestCanvasOnResize(t *testing.T) {
	type size struct{ w, h int }
	var got []size
	guiRun(t, func(app *App, w *Window) {
		cv := put[*CanvasView](w, Canvas(nil, Grow()))
		cv.OnResize(func(cw, ch int) { got = append(got, size{cw, ch}) })
		w.SetSize(300, 200)
		app.After(100*time.Millisecond, func() { w.SetSize(520, 420) })
	})
	if len(got) < 2 || got[0].w <= 0 || got[0].h <= 0 || got[len(got)-1].w <= got[0].w || got[len(got)-1].h <= got[0].h {
		t.Fatalf("reports %v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i] == got[i-1] {
			t.Errorf("report %d repeats %v", i, got[i])
		}
	}
}
