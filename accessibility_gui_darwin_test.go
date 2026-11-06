package gowt

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

// TestAccessibilityCommonControls reads the AppKit accessibility surface of the common controls -
// the same NSAccessibility cell or view VoiceOver uses: role, title/label, value, enabled state,
// focus and the press action.
func TestAccessibilityCommonControls(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		btn := put[*ButtonView](w, Button("Save"))
		icon := put[*ButtonView](w, Button("", AccessibleLabel("Attach")))
		toggle := put[*ButtonView](w, Button("Wrap", Toggle()))
		text := put[*TextView](w, Text(Value("hello")))
		scale := put[*ScaleView](w, Scale(0, 100, Value(30)))
		w.Unwrap().Layout()

		bv := btn.Unwrap().View
		if got := cocoa.AXStringAttribute(bv, "AXRole"); got != "AXButton" {
			t.Errorf("button role %q, want AXButton", got)
		}
		if got := cocoa.AXStringAttribute(bv, "AXTitle"); got != "Save" {
			t.Errorf("button title %q, want Save", got)
		}
		if !cocoa.AXBoolAttribute(bv, "AXEnabled") {
			t.Error("button AXEnabled false")
		}
		if !hasAction(cocoa.AXActionNames(bv), "AXPress") {
			t.Errorf("button actions %v, want AXPress", cocoa.AXActionNames(bv))
		}

		iv := icon.Unwrap().View
		if got := cocoa.AXStringAttribute(iv, "AXRole"); got != "AXButton" {
			t.Errorf("icon button role %q, want AXButton", got)
		}
		if got := cocoa.AccessibilityLabel(iv); got != "Attach" {
			t.Errorf("icon button label %q, want Attach", got)
		}

		tv := toggle.Unwrap().View
		if got := cocoa.AXIntAttribute(tv, "AXValue"); got != 0 {
			t.Fatalf("toggle AXValue %d before press, want 0", got)
		}
		cocoa.AXPerformAction(tv, "AXPress")
		if !toggle.Checked() {
			t.Error("AXPress did not toggle the button")
		}
		if got := cocoa.AXIntAttribute(tv, "AXValue"); got != 1 {
			t.Errorf("toggle AXValue %d after press, want 1", got)
		}

		if got := cocoa.AXStringAttribute(text.Unwrap().View, "AXRole"); got != "AXTextField" {
			t.Errorf("text role %q, want AXTextField", got)
		}
		if got := cocoa.AXStringAttribute(text.Unwrap().View, "AXValue"); got != "hello" {
			t.Errorf("text AXValue %q, want hello", got)
		}
		app.Async(func() {
			text.Unwrap().SetFocus()
			if !cocoa.AXBoolAttribute(text.Unwrap().View, "AXFocused") {
				t.Error("text AXFocused false after SetFocus")
			}
		})

		sv := scale.Unwrap().View
		if got := cocoa.AXStringAttribute(sv, "AXRole"); got != "AXSlider" {
			t.Errorf("scale role %q, want AXSlider", got)
		}
		if got := cocoa.AXIntAttribute(sv, "AXValue"); got != 30 {
			t.Errorf("scale AXValue %d, want 30", got)
		}
	})
}

func hasAction(actions []string, want string) bool {
	for _, a := range actions {
		if a == want {
			return true
		}
	}
	return false
}
