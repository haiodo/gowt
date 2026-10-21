package gowt

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

// Circle and Borderless set the AppKit bezel, which stays through the layout pass that sizes the
// button (SWT used to put the push bezel back there); the accessibility label reaches the view.
func TestIconButtonBezel(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		circle := put[*ButtonView](w, Button("", Circle(), Size(40, 40), AccessibleLabel("Attach")))
		plain := put[*ButtonView](w, Button("", Borderless(), Size(40, 40)))
		w.Unwrap().Layout()
		cv, pv := circle.Unwrap().View, plain.Unwrap().View
		if got := cocoa.ButtonBezelStyle(cv); got != 7 {
			t.Errorf("Circle: bezel style %d, want 7 (circular)", got)
		}
		if shape := cocoa.ButtonBorderShape(cv); cocoa.GlassAvailable() && shape != 3 {
			t.Errorf("Circle: border shape %d, want 3 (circle)", shape)
		}
		if cocoa.ButtonBordered(pv) || !cocoa.ButtonBordered(cv) {
			t.Errorf("bordered: Borderless %v, Circle %v", cocoa.ButtonBordered(pv), cocoa.ButtonBordered(cv))
		}
		if got := cocoa.AccessibilityLabel(cv); got != "Attach" {
			t.Errorf("accessibility label %q", got)
		}
	})
}
