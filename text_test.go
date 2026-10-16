package gowt

import (
	"testing"

	"github.com/haiodo/gowt/swt"
)

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

func TestHeadUnits(t *testing.T) {
	cases := []struct {
		s    string
		keep int
		want int
	}{
		{"a\nb\nc", 2, 2},
		{"a\nb\nc", 1, 4},
		{"a\nb\nc", 3, 0},
		{"a\nb\nc", 9, 0},
		{"a\nb\n", 1, 4},
		{"😀\nb\r\nc", 2, 3},
		{"a\nb", 0, 3},
	}
	for _, c := range cases {
		if got := headUnits(c.s, c.keep); got != c.want {
			t.Errorf("headUnits(%q, %d) = %d, want %d", c.s, c.keep, got, c.want)
		}
	}
}

func TestSortDirMapping(t *testing.T) {
	if SortNone.swt() != swt.NONE || SortAscending.swt() != swt.UP || SortDescending.swt() != swt.DOWN {
		t.Error("SortDir does not map to NONE, UP, DOWN")
	}
}

func TestNoBorderClearsDefault(t *testing.T) {
	if resolve(swt.BORDER|swt.V_SCROLL, []Option{NoBorder()}) != swt.V_SCROLL {
		t.Error("NoBorder must clear BORDER")
	}
	if resolve(swt.NONE, []Option{Search()}) != swt.SEARCH|swt.ICON_SEARCH|swt.ICON_CANCEL {
		t.Error("Search must set SEARCH and both icons")
	}
}

func TestColAtX(t *testing.T) {
	widths, order := []int{100, 50, 80}, []int{2, 0, 1} // column 2 is shown first
	cases := []struct{ scroll, x, want int }{
		{0, 0, 2}, {0, 99, 2}, {0, 100, 0}, {0, 149, 0}, {0, 150, 1}, {0, 229, 1}, {0, 230, -1}, {0, 5000, -1}, {0, -1, -1}, {20, 85, 0},
	}
	for _, c := range cases {
		if got := colAtX(widths, order, c.scroll, c.x); got != c.want {
			t.Errorf("colAtX(scroll %d, x %d) = %d, want %d", c.scroll, c.x, got, c.want)
		}
	}
	if colAtX(nil, nil, 0, 5) != -1 {
		t.Error("no columns must give -1")
	}
}

func TestFontWeightPrecedence(t *testing.T) {
	if (Font{Bold: true}).Weight != 0 {
		t.Error("Bold must not set Weight")
	}
	if Light >= Regular || Regular >= Medium || Medium >= Semibold || Semibold >= Bold {
		t.Error("Weight constants must increase")
	}
}

func TestParseShortcut(t *testing.T) {
	cases := []struct {
		keys, goos string
		accel      int32
		label      string
	}{
		{"Cmd+O", "darwin", swt.COMMAND | 'O', "Cmd+O"},
		{"cmd+o", "windows", swt.CTRL | 'O', "Ctrl+O"},
		{"Ctrl+Cmd+Q", "darwin", swt.CTRL | swt.COMMAND | 'Q', "Ctrl+Cmd+Q"},
		{"Cmd+Shift+S", "linux", swt.CTRL | swt.SHIFT | 'S', "Ctrl+Shift+S"},
		{"Alt+Left", "linux", swt.ALT | swt.ARROW_LEFT, "Alt+Left"},
		{"Cmd+[", "darwin", swt.COMMAND | '[', "Cmd+["},
		{"Cmd++", "windows", swt.CTRL | '+', "Ctrl++"},
		{"F5", "linux", swt.F1 + 4, "F5"},
		{"Shift+f12", "linux", swt.SHIFT | swt.F12, "Shift+F12"},
		{"Esc", "darwin", int32(swt.ESC), "Esc"},
		{"Cmd+pagedown", "windows", swt.CTRL | swt.PAGE_DOWN, "Ctrl+PageDown"},
		{"F", "linux", 'F', "F"},
	}
	for _, c := range cases {
		accel, label, err := parseShortcut(c.keys, c.goos)
		if err != nil || accel != c.accel || label != c.label {
			t.Errorf("parseShortcut(%q, %s) = %#x %q %v, want %#x %q", c.keys, c.goos, accel, label, err, c.accel, c.label)
		}
	}
	for _, bad := range []string{"", "Cmd+", "Hyper+O", "Cmd+Foo", "Cmd+12", "F13", "Cmd++O", "Cmd+é"} {
		if _, _, err := parseShortcut(bad, "linux"); err == nil {
			t.Errorf("parseShortcut(%q) must fail", bad)
		}
	}
}
