package gowt

import "testing"

// "a" 1 unit, emoji 2 units (surrogate pair), CJK 1 unit.
const mixed = "a😀中b"

func TestRuneUnitConversion(t *testing.T) {
	runeToUnit := []int{0, 1, 3, 4, 5}
	for r, want := range runeToUnit {
		if got := runesToUnits(mixed, r); got != want {
			t.Errorf("runesToUnits(%d) = %d, want %d", r, got, want)
		}
		if got := unitsToRunes(mixed, want); got != r {
			t.Errorf("unitsToRunes(%d) = %d, want %d", want, got, r)
		}
	}
	if got := unitsToRunes(mixed, 2); got != 1 {
		t.Errorf("unit inside a surrogate pair maps to %d, want the rune that starts it (1)", got)
	}
	if runesToUnits(mixed, 99) != 5 || unitsToRunes(mixed, 99) != 4 || runesToUnits("", 3) != 0 || runesToUnits(mixed, -1) != 0 {
		t.Error("out of range indexes must clamp")
	}
}

func TestStyledTextRunes(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		st := w.StyledText(Scrollbars())
		st.SetText(mixed)
		st.SetSelection(1, 2)
		if s, l := st.Selection(); s != 1 || l != 2 || st.SelectedText() != "😀中" {
			t.Errorf("selection %d,%d %q", s, l, st.SelectedText())
		}
		st.SetCaret(3)
		if st.Caret() != 3 {
			t.Errorf("caret %d", st.Caret())
		}
		// Editing text with surrogate pairs panics in swt's DefaultContent gap buffer (jrt.Substring
		// slice bounds), so Replace is checked on BMP text only.
		st.SetText("a中b")
		st.Replace(1, 1, "XY")
		if st.Text() != "aXYb" {
			t.Errorf("replace: %q", st.Text())
		}
		st.SetText("a中b")
		red := RGB{255, 0, 0}
		st.SetStyle(1, 1, TextStyle{Foreground: &red, Bold: true})
		got := st.Styles()
		if len(got) != 1 || got[0].Start != 1 || got[0].Length != 1 || !got[0].Style.Bold || *got[0].Style.Foreground != red {
			t.Errorf("styles %+v", got)
		}
		st.ClearStyles()
		if len(st.Styles()) != 0 {
			t.Error("ClearStyles left styles")
		}
	})
}

func TestClipboardRoundTrip(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		c := app.Clipboard()
		if err := c.SetText("héllo 😀"); err != nil {
			t.Skip("clipboard not writable here:", err)
		}
		if s, ok := c.Text(); !ok || s != "héllo 😀" {
			t.Errorf("text %q %v", s, ok)
		}
		c.Clear()
		if _, ok := c.Text(); ok {
			t.Error("text after Clear")
		}
	})
}
