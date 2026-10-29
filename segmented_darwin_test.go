package gowt

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

// pressSegment clicks segment i through the control's target and action.
func pressSegment(s *SegmentedView, i int) { s.seg.v.Press(i) }

func selectedSegment(s *SegmentedView) int { return s.seg.v.Selected() }

// The segments carry their labels and tooltips, Small and Momentary reach the native control, and the
// control fills the composite's width; a borderless toggle takes the accent tint while on.
func TestSegmentedNative(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		s := put[*SegmentedView](w, Segmented([]Segment{{Text: "Fixed"}, {Text: "Hug", Tooltip: "Fit the content"}}, Small(), Size(240, 0)))
		m := put[*SegmentedView](w, Segmented([]Segment{{Text: "+"}, {Text: "-"}}, Momentary()))
		eye := put[*ButtonView](w, Button("eye", Toggle(), Borderless(), Small()))
		plus := put[*ButtonView](w, Button("+", Borderless()))
		w.Unwrap().Layout()
		v := s.seg.v
		if v.SegmentCount() != 2 || v.SegmentLabel(0) != "Fixed" || v.SegmentToolTip(1) != "Fit the content" {
			t.Errorf("segments: %d, %q, tooltip %q", v.SegmentCount(), v.SegmentLabel(0), v.SegmentToolTip(1))
		}
		if v.ControlSize() != 1 || m.seg.v.ControlSize() != 0 {
			t.Errorf("control size: Small %d, regular %d", v.ControlSize(), m.seg.v.ControlSize())
		}
		if m.seg.v.TrackingMode() != 2 || v.TrackingMode() != 0 {
			t.Errorf("tracking: Momentary %d, select one %d", m.seg.v.TrackingMode(), v.TrackingMode())
		}
		if f := v.Frame(); int(f.Width) != int(s.c.GetSize().X) || f.Height <= 0 {
			t.Errorf("frame %+v in a composite of %v", f, s.c.GetSize())
		}
		if cocoa.ButtonBordered(eye.b.View) || cocoa.ControlSize(eye.b.View) != 1 || cocoa.ButtonTinted(eye.b.View) {
			t.Errorf("borderless small toggle, off: bordered %v, size %d, tinted %v", cocoa.ButtonBordered(eye.b.View), cocoa.ControlSize(eye.b.View), cocoa.ButtonTinted(eye.b.View))
		}
		eye.SetChecked(true)
		if !cocoa.ButtonTinted(eye.b.View) {
			t.Error("a borderless toggle set on has no accent tint")
		}
		eye.SetChecked(false)
		if cocoa.ButtonTinted(eye.b.View) {
			t.Error("a borderless toggle set off keeps the accent tint")
		}
		if cocoa.ButtonBordered(plus.b.View) {
			t.Error("a borderless push button draws its bezel")
		}
	})
}

// The Disclosure chevron is the native disclosure button: its bezel, its on state as the expansion,
// and the accessibility role VoiceOver reads.
func TestDisclosureNative(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		d := put[*DisclosureView](w, Disclosure("Layout", Label("body")))
		v := d.chevron.View
		if b := cocoa.ButtonBezelStyle(v); b != 5 {
			t.Errorf("chevron bezel %d, want 5 (disclosure)", b)
		}
		if role := cocoa.AccessibilityRole(v); role != "AXDisclosureTriangle" {
			t.Errorf("chevron role %q, want AXDisclosureTriangle", role)
		}
		if !d.chevron.GetSelection() {
			t.Error("an expanded Disclosure has its chevron off")
		}
		d.SetExpanded(false)
		if d.chevron.GetSelection() {
			t.Error("a folded Disclosure has its chevron on")
		}
		d.SetTitle("Fill")
		if got := d.title.GetText(); got != "Fill" {
			t.Errorf("title after SetTitle %q, want Fill", got)
		}
		if name := cocoa.AccessibilityLabel(v); name != "Fill" {
			t.Errorf("chevron label after SetTitle %q, want Fill", name)
		}
		if d.Expanded() {
			t.Error("SetTitle unfolded the body")
		}
	})
}
