package look_test

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
)

// SetGlassFocusRing puts a keyboardFocusIndicatorColor ring on the glass layer while the Text inside
// has focus, and the separator stroke back on blur; the ring survives a resize and a switch to dark.
// gowt.OnFocus on the Text reports the same changes.
func TestGlassFocusRing(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if !cocoa.GlassAvailable() {
		t.Skip("needs macOS 26")
	}
	var scale float64
	var start, focused, resized, dark, blurred borderRead
	var events []bool
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("glass focus")
			w.SetSize(600, 300)
			var capsule *gowt.Panel
			var text, other *gowt.TextView
			w.Add(gowt.VBox(gowt.Padding(20), gowt.Gap(12),
				gowt.HBox(gowt.Ref(&capsule), gowt.GrowX(), gowt.Padding(0),
					gowt.Text(gowt.Multiline(), gowt.GrowX(), gowt.Size(0, 36), gowt.Inset(14, 8), gowt.Ref(&text),
						gowt.OnFocus(func(on bool) { events = append(events, on) }))),
				gowt.Text(gowt.Ref(&other))))
			look.SetGlassRadius(capsule, 18)
			look.SetGlass(capsule, true)
			look.SetGlassBorder(capsule, true)
			look.SetGlassFocusRing(capsule, true)
			w.Show()
			other.Focus()
			app.After(300*time.Millisecond, func() {
				scale = cocoa.BackingScale(w.Unwrap().View)
				start = readBorder(capsule.Unwrap().View)
				text.Focus()
				focused = readBorder(capsule.Unwrap().View)
				w.SetSize(800, 400)
				app.After(200*time.Millisecond, func() {
					resized = readBorder(capsule.Unwrap().View)
					w.Unwrap().SetDarkThemePreferred(true)
					w.Unwrap().GetDisplay().SendEvent(swt.Settings, nil)
					dark = readBorder(capsule.Unwrap().View)
					other.Focus()
					blurred = readBorder(capsule.Unwrap().View)
					w.Close()
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	same := func(a, b [4]float64) bool {
		for i := range a {
			if math.Abs(a[i]-b[i]) > 0.01 {
				return false
			}
		}
		return true
	}
	for _, c := range []struct {
		what  string
		got   borderRead
		width float64
		rgba  [4]float64
	}{
		{"unfocused", start, 1 / scale, cocoa.SeparatorRGBA(false)},
		{"focused", focused, 2, cocoa.FocusRingRGBA(false)},
		{"focused, resized", resized, 2, cocoa.FocusRingRGBA(false)},
		{"focused, dark", dark, 2, cocoa.FocusRingRGBA(true)},
		{"blurred", blurred, 1 / scale, cocoa.SeparatorRGBA(true)},
	} {
		if !c.got.ok || math.Abs(c.got.width-c.width) > 1e-9 || !same(c.got.rgba, c.rgba) {
			t.Errorf("%s: width %v colour %v (glass %v), want %v and %v", c.what, c.got.width, c.got.rgba, c.got.ok, c.width, c.rgba)
		}
	}
	if len(events) < 2 || !events[len(events)-2] || events[len(events)-1] {
		t.Errorf("OnFocus events %v, want ... true, false", events)
	}
}
