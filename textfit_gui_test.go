package gowt

import (
	"os"
	"strings"
	"testing"
	"time"
)

// An AutoGrow Text follows its visual lines between its limits, wrapping counted, and a long line does
// not widen the field past its window.
func TestAutoGrowText(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var errs []error
	got := map[string]int{}
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			w := app.Window("autogrow")
			w.SetSize(400, 300)
			var row *Panel
			var text *TextView
			w.Add(VBox(Padding(10), Grow(),
				HBox(Ref(&row), GrowX(), Padding(0),
					Text(Multiline(), Ref(&text), GrowX(), Inset(10, 6), AutoGrow(40, 100)))))
			w.Show()
			height := func() int { app.Flush(); return int(text.Unwrap().GetBounds().Height) }
			step := func(name, s string) {
				text.SetText(s)
				app.Flush()
				got[name] = height()
			}
			app.After(200*time.Millisecond, func() {
				got["empty"] = height()
				step("three", "a\nb\nc")
				got["line"] = int(text.Unwrap().GetLineHeight())
				step("long", strings.Repeat("word ", 400))
				width := int(text.Unwrap().GetClientArea().Width)
				got["wide"] = text.ContentHeight(width)
				got["narrow"] = text.ContentHeight(width / 3)
				got["window"] = int(w.Unwrap().GetClientArea().Width)
				got["field"] = int(text.Unwrap().GetBounds().Width)
				step("cleared", "")
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range errs {
		t.Errorf("OnError: %v", e)
	}
	if got["empty"] != 40 {
		t.Errorf("empty field %d high, want the minimum 40", got["empty"])
	}
	if got["narrow"] <= got["wide"] {
		t.Errorf("ContentHeight at 1/3 of the width %d, at full width %d: wrapping is not counted", got["narrow"], got["wide"])
	}
	t.Logf("three %d line %d", got["three"], got["line"])
	if got["three"] <= 40 || got["three"] >= 100 {
		t.Errorf("three lines: %d high, want between 40 and 100", got["three"])
	}
	if got["long"] != 100 {
		t.Errorf("long text: %d high, want the maximum 100", got["long"])
	}
	if got["field"] > got["window"] {
		t.Errorf("long text: field %d wide in a window %d wide", got["field"], got["window"])
	}
	if got["cleared"] != 40 {
		t.Errorf("cleared: %d high, want 40", got["cleared"])
	}
}
