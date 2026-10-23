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

type borderRead struct {
	width float64
	rgba  [4]float64
	ok    bool
}

func readBorder(v *cocoa.NSView) borderRead {
	w, c, ok := cocoa.GlassBorder(v)
	return borderRead{w, c, ok}
}

// A glass capsule and glass buttons with a border carry a stroke of 1/backingScale points in
// separatorColor on the glass layer, in light and dark appearance (the Settings event that follows a
// system switch sets it again), after a resize, and with GlassBorder before and after the other options.
func TestGlassBorderLayer(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if !cocoa.GlassAvailable() {
		t.Skip("needs macOS 26")
	}
	var scale float64
	var light, dark, resized, plainRead [4]borderRead
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("glass border")
			w.SetSize(700, 400)
			var sidebar, content, capsule *gowt.Panel
			var b1, b2, b3, plain *gowt.ButtonView
			w.Add(gowt.VBox(gowt.Ref(&sidebar), gowt.Label("sidebar")),
				gowt.VBox(gowt.Ref(&content), gowt.Padding(7),
					gowt.HBox(gowt.Ref(&capsule), gowt.GrowX(), gowt.Size(0, 40), gowt.Label("capsule")),
					gowt.HBox(
						gowt.Button("", look.GlassBorder(), look.GlassButton(), gowt.Circle(), gowt.Size(40, 40), gowt.Ref(&b1)),
						gowt.Button("", look.GlassButton(), gowt.Circle(), look.GlassBorder(), gowt.Size(40, 40), gowt.Ref(&b2)),
						gowt.Button("Go", gowt.Circle(), look.GlassBorder(), look.GlassButton(), gowt.Size(40, 40), gowt.Ref(&b3)),
						gowt.Button("Plain", look.GlassButton(), gowt.Ref(&plain)))))
			look.SetGlassBorder(capsule, true)
			look.SetGlassRadius(capsule, look.Capsule)
			look.SetGlass(capsule, true)
			look.SetSidebar(w, sidebar, content, 200)
			w.Show()
			views := func() [4]*cocoa.NSView {
				return [4]*cocoa.NSView{capsule.Unwrap().View, b1.Unwrap().View, b2.Unwrap().View, b3.Unwrap().View}
			}
			read := func() (r [4]borderRead) {
				for i, v := range views() {
					r[i] = readBorder(v)
				}
				return r
			}
			app.After(300*time.Millisecond, func() {
				scale = cocoa.BackingScale(w.Unwrap().View)
				light = read()
				plainRead[0] = readBorder(plain.Unwrap().View)
				w.Unwrap().SetDarkThemePreferred(true)
				w.Unwrap().GetDisplay().SendEvent(swt.Settings, nil)
				dark = read()
				w.SetSize(900, 500)
				app.After(300*time.Millisecond, func() {
					resized = read()
					w.Close()
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"capsule", "GlassBorder first", "GlassBorder after Circle", "GlassBorder between"}
	same := func(a, b [4]float64) bool {
		for i := range a {
			if math.Abs(a[i]-b[i]) > 0.01 {
				return false
			}
		}
		return true
	}
	for _, c := range []struct {
		what string
		got  [4]borderRead
		dark bool
	}{{"light", light, false}, {"dark", dark, true}, {"resized (dark)", resized, true}} {
		want := cocoa.SeparatorRGBA(c.dark)
		for i, r := range c.got {
			if !r.ok || math.Abs(r.width-1/scale) > 1e-9 || !same(r.rgba, want) {
				t.Errorf("%s, %s: width %v colour %v (glass %v), want 1/%v and %v", c.what, names[i], r.width, r.rgba, r.ok, scale, want)
			}
		}
	}
	if plainRead[0].width != 0 {
		t.Errorf("button without GlassBorder has a stroke of %v", plainRead[0].width)
	}
}
