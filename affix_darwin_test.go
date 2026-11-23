package gowt

import (
	"math"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func TestStyledMultilineField(t *testing.T) {
	fill := Adaptive{Light: RGB{240, 220, 200}, Dark: RGB{20, 40, 60}}
	guiRun(t, func(app *App, w *Window) {
		v := put[*TextView](w, Text(Multiline(), Border(), FieldFill(fill), FieldStroke(RGB{4, 5, 6}), Radius(7), Inset(6, 4), Hint("old")))
		if v.aff != nil || v.t.GetMessage() != "old" || v.t.GetDataKey("gowt.hint") == nil {
			t.Errorf("multiline hint: affix %+v, message %q", v.aff, v.t.GetMessage())
		}
		v.SetHint("new")
		if got := v.t.GetMessage(); got != "new" {
			t.Errorf("hint %q, want new", got)
		}
		if v.inset != [2]int{6, 4} {
			t.Errorf("inset %v, want [6 4]", v.inset)
		}
		notify(v.t, swt.FocusIn)
		v.SetText("first\nsecond")
		if got := v.t.GetText(); got != "first\nsecond" {
			t.Errorf("text %q", got)
		}
		for _, dark := range []bool{false, true} {
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
