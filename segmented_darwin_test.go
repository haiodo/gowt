package gowt

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

// pressSegment clicks segment i through the control's target and action.
func pressSegment(s *SegmentedView, i int) { s.seg.v.Press(i) }

func selectedSegment(s *SegmentedView) int { return s.seg.v.Selected() }

// The segments carry their labels and tooltips, Small and Momentary reach the native control, and the
// control fills the composite's width; a borderless toggle has the recessed bezel.
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
		if b := cocoa.ButtonBezelStyle(eye.b.View); b != 13 || cocoa.ControlSize(eye.b.View) != 1 {
			t.Errorf("borderless small toggle: bezel %d, size %d; want 13 and 1", b, cocoa.ControlSize(eye.b.View))
		}
		if cocoa.ButtonBordered(plus.b.View) {
			t.Error("a borderless push button draws its bezel")
		}
	})
}
