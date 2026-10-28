package gowt

import (
	"testing"

	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// snap rounds to the step and its decimals, clamps, and never gives -0.
func TestNumberSnap(t *testing.T) {
	for _, c := range []struct {
		min, max, step, in, want float64
		text                     string
	}{
		{0, 100, 1, 12.4, 12, "12"},
		{0, 100, 1, 12.5, 13, "13"},
		{0, 100, 1, 140, 100, "100"},
		{-10, 10, 1, -0.2, 0, "0"},
		{0, 1, 0.1, 0.1 + 0.2, 0.3, "0.3"},
		{0, 100, 0.25, 7.3, 7.25, "7.25"},
		{0, 100, 0.5, 12, 12, "12"},
		{-50, 50, 2, 3, 4, "4"},
	} {
		n := &NumberView{min: c.min, max: c.max, step: c.step}
		got := n.snap(c.in)
		if got != c.want || n.format(got) != c.text {
			t.Errorf("snap(%v) with step %v in [%v, %v] = %v %q, want %v %q", c.in, c.step, c.min, c.max, got, n.format(got), c.want, c.text)
		}
	}
}

// Typing, the arrow keys and a scrub change the value and run OnChange; SetValue does not.
// Bind follows a float64 and an int state both ways.
func TestNumberField(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		f := rx.New(10.0)
		i := rx.New(5)
		var changes []float64
		n := put[*NumberView](w, Number(0, 100, Prefix("X"), Suffix("px"), Small(), Bind(f),
			OnChange(func(v float64) { changes = append(changes, v) })))
		m := put[*NumberView](w, Number(0, 10, Step(0.5), Bind(i)))
		if n.Value() != 10 || n.Unwrap().GetText() != "10" {
			t.Errorf("bound start %v %q, want 10", n.Value(), n.Unwrap().GetText())
			return
		}
		n.Unwrap().SetText("42.6 px")
		notify(n.Unwrap(), swt.DefaultSelection)
		if n.Value() != 43 || f.Get() != 43 || n.Unwrap().GetText() != "43" {
			t.Errorf("typed 42.6 px: value %v, state %v, text %q, want 43", n.Value(), f.Get(), n.Unwrap().GetText())
		}
		n.Unwrap().SetText("abc")
		notify(n.Unwrap(), swt.FocusOut)
		if n.Value() != 43 || n.Unwrap().GetText() != "43" {
			t.Errorf("typed abc: value %v, text %q, want 43 back", n.Value(), n.Unwrap().GetText())
		}
		up := swt.NewEvent()
		up.KeyCode = swt.ARROW_UP
		up.StateMask = swt.SHIFT
		n.Unwrap().NotifyListeners(swt.KeyDown, up)
		if n.Value() != 53 || up.Doit {
			t.Errorf("Shift+Up: value %v, doit %v, want 53 and the key consumed", n.Value(), up.Doit)
		}
		n.scrub(0.6, false)
		n.scrub(0.6, false)
		n.scrub(-3, true)
		if n.Value() != 34 {
			t.Errorf("scrub 0.6+0.6 then -3 with Shift: value %v, want 54 then 34", n.Value())
		}
		if want := []float64{43, 53, 54, 34}; len(changes) != len(want) {
			t.Errorf("OnChange got %v, want %v", changes, want)
		}
		f.Set(7)
		app.Flush()
		if n.Value() != 7 || len(changes) != 4 {
			t.Errorf("state set to 7: value %v, %d changes, want 7 and no OnChange", n.Value(), len(changes))
		}
		m.scrub(3, false)
		if i.Get() != 7 || m.Unwrap().GetText() != "6.5" {
			t.Errorf("int state after a scrub of 3 half steps from 5: %v, text %q, want 7 (6.5 rounded), 6.5", i.Get(), m.Unwrap().GetText())
		}
	})
}

// Prefix and Small refuse a multi-line Text; Step refuses a Text.
func TestAffixMisfit(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		var errs []error
		app.OnError(func(err error) { errs = append(errs, err) })
		put[*TextView](w, Text(Multiline(), Prefix("X")))
		put[*TextView](w, Text(Search(), Small()))
		put[*TextView](w, Text(Step(1)))
		put[*TextView](w, Text(Prefix("X"), Suffix("%"), Small()))
		if len(errs) != 3 {
			t.Errorf("errors %v, want three", errs)
		}
	})
}
