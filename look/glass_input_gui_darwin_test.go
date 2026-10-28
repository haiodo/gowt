package look_test

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/look"
)

// SetGlassRadius keeps a fixed radius, and a Capsule follows the shorter side when the panel is resized.
func TestGlassRadius(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if !cocoa.GlassAvailable() {
		t.Skip("needs macOS 26")
	}
	got := map[string]float64{}
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("radius")
			w.SetSize(400, 300)
			var row *gowt.Panel
			w.Add(gowt.VBox(gowt.Padding(10), gowt.Grow(),
				gowt.HBox(gowt.Ref(&row), gowt.GrowX(), gowt.Padding(0), gowt.Size(0, 44), gowt.Label("x"))))
			look.SetGlassRadius(row, 20) // before SetGlass
			look.SetGlass(row, true)
			w.Show()
			v := row.Unwrap().View
			app.After(150*time.Millisecond, func() {
				got["fixed"] = cocoa.GlassCornerRadius(v)
				look.SetGlassRadius(row, look.Capsule)
				got["capsule"] = cocoa.GlassCornerRadius(v)
				row.SetHeight(60)
				app.Flush()
				got["grown"] = cocoa.GlassCornerRadius(v)
				look.SetGlassRadius(row, 20)
				row.SetHeight(100)
				app.Flush()
				got["fixed grown"] = cocoa.GlassCornerRadius(v)
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]float64{"fixed": 20, "capsule": 22, "grown": 30, "fixed grown": 20} {
		if math.Abs(got[name]-want) > 0.5 {
			t.Errorf("%s: radius %.1f, want %.1f", name, got[name], want)
		}
	}
}

// A multi-line Text in glass draws no background, a single-line one keeps its field.
func TestGlassTextTransparent(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var multiAlpha, lateAlpha, singleAlpha int32
	var multiPix float64
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("glass text")
			w.SetSize(400, 200)
			var glass *gowt.Panel
			var multi, late, single *gowt.TextView
			w.Add(gowt.VBox(gowt.Ref(&glass), gowt.Grow(), gowt.Padding(12),
				gowt.Text(gowt.Multiline(), gowt.GrowX(), gowt.Size(0, 60), gowt.Ref(&multi)),
				gowt.Text(gowt.GrowX(), gowt.Ref(&single))))
			look.SetGlass(glass, true)
			w.Show()
			app.After(100*time.Millisecond, func() {
				glass.Add(gowt.Text(gowt.Multiline(), gowt.GrowX(), gowt.Size(0, 60), gowt.Ref(&late)))
				app.After(200*time.Millisecond, func() {
					multiAlpha = multi.Unwrap().GetBackground().GetAlpha()
					lateAlpha = late.Unwrap().GetBackground().GetAlpha()
					singleAlpha = single.Unwrap().GetBackground().GetAlpha()
					tv := multi.Unwrap().View
					sv := cocoa.NewNSViewOverload1(cocoa.Msg(tv.Id, "enclosingScrollView"))
					// Inside the clip view: a legacy scroller (a mouse, no trackpad) draws its track at the edge.
					clip := cocoa.NewNSScrollViewOverload1(sv.Id).ContentView().Frame()
					multiPix = alphaAt(sv, clip.X+clip.Width-4, clip.Y+clip.Height-4)
					w.Close()
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if multiAlpha != 0 || lateAlpha != 0 {
		t.Errorf("multi-line Text background alpha %d, added later %d, want 0", multiAlpha, lateAlpha)
	}
	if singleAlpha == 0 {
		t.Error("single-line Text lost its field in glass")
	}
	if multiPix > 0.01 {
		t.Errorf("multi-line Text paints alpha %.2f in its scroll view, want 0", multiPix)
	}
}
