package look_test

import (
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/look"
)

// SetGlassTint is one call per panel: the glass behind a SetGlass panel gets the exact sRGB tint,
// a panel without it reports none, and a backdrop reinstall keeps the tint.
func TestGlassTintPanel(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var reduce bool
	onMain(func() { reduce = cocoa.ReduceTransparency() })
	if reduce {
		t.Skip("Reduce transparency: the panel glass is the NSVisualEffectView material")
	}
	if !cocoa.GlassAvailable() {
		t.Skip("needs macOS 26")
	}
	const (
		r, g, b, a = 210.0 / 255, 120.0 / 255, 40.0 / 255, 200.0 / 255
	)
	var tinted, reinstalled [4]float64
	var plainOK, tintedOK, reinstalledOK bool
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("glass tint")
			w.SetSize(400, 300)
			var plain, panel *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&plain), gowt.Label("plain")), gowt.VBox(gowt.Ref(&panel), gowt.Label("panel")))
			look.SetGlass(plain, true)
			look.SetGlass(panel, true)
			look.SetGlassTint(panel, gowt.RGBA{R: 210, G: 120, B: 40, A: 200})
			w.Show()
			app.After(300*time.Millisecond, func() {
				plainView := plain.Unwrap().View
				panelView := panel.Unwrap().View
				_, plainOK = cocoa.BackdropTint(plainView)
				tinted, tintedOK = cocoa.BackdropTint(panelView)
				// A reinstall reads the stored tint again, as setGlass does after a glass rebuild.
				cocoa.InstallPanelBackdrop(panelView, cocoa.BackdropGlass)
				reinstalled, reinstalledOK = cocoa.BackdropTint(panelView)
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if plainOK {
		t.Error("panel without SetGlassTint reports a tint")
	}
	for i, got := range []struct {
		rgba [4]float64
		ok   bool
	}{{tinted, tintedOK}, {reinstalled, reinstalledOK}} {
		if !got.ok {
			t.Fatalf("case %d: tint lost", i)
		}
		for j, want := range []float64{r, g, b, a} {
			if d := got.rgba[j] - want; d > 1e-4 || d < -1e-4 {
				t.Errorf("case %d: tint component %d = %v, want %v", i, j, got.rgba[j], want)
			}
		}
	}
}
