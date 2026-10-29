package gowt

import (
	"math"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/cocoa"
)

func TestFieldFillRepaint(t *testing.T) {
	fill := Adaptive{Light: RGB{240, 220, 200}, Dark: RGB{20, 40, 60}}
	guiRun(t, func(app *App, w *Window) {
		v := put[*TextView](w, Text(FieldFill(fill), Radius(7)))
		for _, dark := range []bool{false, true, false} {
			app.repaint(dark)
			got, ok := cocoa.FieldFillRGBA(v.t.View)
			want := fill.in(dark)
			if !ok || math.Abs(got[0]-float64(want.R)/255) > 0.01 || math.Abs(got[1]-float64(want.G)/255) > 0.01 || math.Abs(got[2]-float64(want.B)/255) > 0.01 {
				t.Errorf("dark %v: native fill %v (%v), want %v", dark, got, ok, want)
			}
			if radius := cocoa.FieldFillRadius(v.t.View); radius != 7 {
				t.Errorf("radius %v, want 7", radius)
			}
		}
	})
}

func TestStyledFieldHintPreservesInset(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		v := put[*TextView](w, Text(Inset(6, 0), Hint("old")))
		v.SetHint("new")
		asyncRan := false
		app.Async(func() { asyncRan = true })
		app.After(time.Second, func() {
			if !asyncRan {
				t.Error("visible hint starved Async callbacks")
				w.Close()
			}
		})
		if got := cocoa.FieldInset(v.t.View); got != 6 {
			t.Errorf("inset after SetHint %v, want 6", got)
		}
	})
}
