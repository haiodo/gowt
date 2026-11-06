package gowt

import (
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/gtk"
)

// TestAccessibilityCommonControls reads the ATK identity of the common controls - the objects
// AT-SPI exposes to assistive technology: role, name, value, state, focus and actions.
func TestAccessibilityCommonControls(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		pressed := false
		btn := put[*ButtonView](w, Button("Save", OnClick(func() { pressed = true })))
		icon := put[*ButtonView](w, Button("", AccessibleLabel("Attach")))
		check := put[*ButtonView](w, Button("Wrap", Check()))
		text := put[*TextView](w, Text(Value("hello")))
		scale := put[*ScaleView](w, Scale(0, 100, Value(30)))
		w.Unwrap().Layout()

		bh := btn.Unwrap().Handle
		if got := gtk.AccessibleRoleName(bh); got != "push button" && got != "button" {
			t.Errorf("button role %q, want push button or button", got)
		}
		if got := gtk.AccessibleName(bh); got != "Save" {
			t.Errorf("button name %q, want Save", got)
		}

		ih := icon.Unwrap().Handle
		if got := gtk.AccessibleRoleName(ih); got != "push button" && got != "button" {
			t.Errorf("icon button role %q, want push button or button", got)
		}
		if got := gtk.AccessibleName(ih); got != "Attach" {
			t.Errorf("icon button name %q, want Attach (AccessibleLabel)", got)
		}

		check.SetChecked(true)
		ch := check.Unwrap().Handle
		if got := gtk.AccessibleRoleName(ch); got != "check box" {
			t.Errorf("check role %q, want check box", got)
		}
		if got := gtk.AccessibleName(ch); got != "Wrap" {
			t.Errorf("check name %q, want Wrap", got)
		}
		if !gtk.AccessibleState(ch, "checked") {
			t.Error("checked ATK state missing on checked check box")
		}

		th := text.Unwrap().Handle
		if got := gtk.AccessibleRoleName(th); got == "" {
			t.Error("text has no ATK role")
		}
		if got := gtk.AccessibleText(th); got != "hello" {
			t.Errorf("text ATK value %q, want hello", got)
		}

		sh := scale.Unwrap().Handle
		if got := gtk.AccessibleRoleName(sh); got != "slider" {
			t.Errorf("scale role %q, want slider", got)
		}
		if got, ok := gtk.AccessibleNumber(sh); !ok || got != 30 {
			t.Errorf("scale ATK value %v, %v, want 30, true", got, ok)
		}

		app.Async(func() {
			w.Unwrap().ForceActive()
			actions := gtk.AccessibleActionNames(bh)
			if len(actions) == 0 {
				t.Error("button has no ATK actions")
			} else if !gtk.AccessiblePerformAction(bh, actions[0]) {
				t.Errorf("ATK action %q failed", actions[0])
			}
			btn.Unwrap().GetDisplay().RunDeferredEvents()
			if !pressed {
				t.Error("ATK action did not press the button")
			}
			w.Unwrap().ForceActive()
			app.After(20*time.Millisecond, func() {
				focused := text.Unwrap().SetFocus()
				app.After(20*time.Millisecond, func() {
					if !gtk.AccessibleState(th, "focused") {
						t.Errorf("focused ATK state missing: SetFocus=%v GTK focus=%v", focused, gtk.GTKGtk_widget_is_focus(th))
					}
				})
			})
		})
	})
}
