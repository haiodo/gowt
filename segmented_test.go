package gowt

import (
	"strings"
	"testing"

	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

func TestSegmented(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		var picked []int
		mode := rx.New(0)
		s := put[*SegmentedView](w, Segmented([]Segment{{Text: "Fixed"}, {Text: "Hug", Tooltip: "Fit the content"}, {Text: "Fill"}},
			Selected(1), Small(), OnSelect(func(i int) { picked = append(picked, i) })))
		b := put[*SegmentedView](w, Segmented([]Segment{{Text: "A"}, {Text: "B"}}, Bind(mode), GrowX()))
		if len(*errs) != 0 {
			t.Fatalf("valid use reported %q", *errs)
		}
		if s.Index() != 1 || selectedSegment(s) != 1 {
			t.Errorf("Selected(1): index %d, native %d", s.Index(), selectedSegment(s))
		}
		s.Select(2)
		if len(picked) != 0 || selectedSegment(s) != 2 {
			t.Errorf("Select ran OnSelect %v or left the native selection at %d", picked, selectedSegment(s))
		}
		pressSegment(s, 0)
		pressSegment(s, 0) // the selected one again: no change
		if len(picked) != 1 || picked[0] != 0 || s.Index() != 0 || selectedSegment(s) != 0 {
			t.Errorf("click on 0: OnSelect %v, index %d, native %d", picked, s.Index(), selectedSegment(s))
		}
		s.Select(5)
		if s.Index() != 0 {
			t.Errorf("Select(5) out of range changed the index to %d", s.Index())
		}

		pressSegment(b, 1)
		if mode.Get() != 1 {
			t.Errorf("Bind: a click on 1 set the state to %d", mode.Get())
		}
		mode.Set(0)
		rx.Flush()
		if b.Index() != 0 || selectedSegment(b) != 0 {
			t.Errorf("Bind: state 0 selected %d (native %d)", b.Index(), selectedSegment(b))
		}
		if p := preferred(b.control()); p.X <= 0 || p.Y <= 0 {
			t.Errorf("preferred size %v", p)
		}
	})
}

func TestSegmentedMomentary(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		var pressed []int
		s := put[*SegmentedView](w, Segmented([]Segment{{Text: "+"}, {Text: "-"}}, Selected(0), Momentary(),
			OnSelect(func(i int) { pressed = append(pressed, i) })))
		if s.Index() != -1 || selectedSegment(s) != -1 {
			t.Errorf("Momentary kept the selection %d (native %d)", s.Index(), selectedSegment(s))
		}
		pressSegment(s, 1)
		pressSegment(s, 1)
		s.Select(0)
		if len(pressed) != 2 || pressed[0] != 1 || pressed[1] != 1 || s.Index() != -1 {
			t.Errorf("momentary clicks ran OnSelect with %v, index %d", pressed, s.Index())
		}
	})
}

func TestToggleButton(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		wrap := rx.New(false)
		var changed []bool
		b := put[*ButtonView](w, Button("Wrap", Toggle(), Bind(wrap), OnChange(func(on bool) { changed = append(changed, on) })))
		put[*ButtonView](w, Button("", Toggle(), Borderless(), Small(), Tooltip("Visible")))
		put[*ButtonView](w, Button("+", Borderless(), Small()))
		if len(*errs) != 0 {
			t.Fatalf("valid use reported %q", *errs)
		}
		if b.Unwrap().GetStyle()&swt.TOGGLE == 0 {
			t.Fatal("Toggle did not make a TOGGLE button")
		}
		wrap.Set(true)
		rx.Flush()
		if !b.Checked() {
			t.Error("Bind: state true left the toggle off")
		}
		b.SetChecked(false)
		notify(b.Unwrap(), swt.Selection)
		if wrap.Get() || len(changed) != 1 || changed[0] {
			t.Errorf("a click to off: state %v, OnChange %v", wrap.Get(), changed)
		}

		put[*LabelView](w, Label("l", Toggle()))
		put[*ButtonView](w, Button("b", Momentary()))
		put[*LabelView](w, Label("l", Small()))
		put[*ButtonView](w, Button("c", Check(), Borderless()))
		want := []string{
			"gowt: Toggle does not apply to Label",
			"gowt: Momentary applies to Segmented, not to Button",
			"gowt: Small applies to Segmented and Button, not to Label",
			"gowt: Borderless applies to a push or Toggle Button, not to Button",
		}
		if got := strings.Join(*errs, "\n"); got != strings.Join(want, "\n") {
			t.Errorf("errors:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
		}
	})
}
