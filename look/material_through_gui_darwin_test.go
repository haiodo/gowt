package look_test

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
)

// The SetMaterial glass is translucent: over a red panel, captured as the window server composites it
// (GOWT_SNAP_SCREEN, may need Screen Recording permission), the middle of the pane is reddish, not the
// flat grey of an opaque fill. GOWT_TEST_SHOTS=<dir> keeps the PNG.
func TestMaterialShowsThrough(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if !cocoa.GlassAvailable() {
		t.Skip("macOS 26 glass only; the older material blends the desktop behind the window")
	}
	t.Setenv("GOWT_SNAP_SCREEN", "1")
	dir := os.Getenv("GOWT_TEST_SHOTS")
	if dir == "" {
		dir = t.TempDir()
	}
	path := filepath.Join(dir, "material-through.png")
	var shotErr, err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("material", gowt.Padding(0))
			w.SetSize(400, 300)
			var pane *gowt.Panel
			w.Add(gowt.VBox(gowt.Grow(), gowt.Padding(40), gowt.Background(gowt.RGB{R: 220, G: 30, B: 30}),
				gowt.VBox(gowt.Ref(&pane), gowt.Grow(), gowt.Label("pane"))))
			look.SetMaterial(pane, true)
			w.Show()
			app.After(800*time.Millisecond, func() {
				shotErr = shot.WindowPNG(w.Unwrap().View.Id, path)
				w.Close()
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if shotErr != nil {
		t.Fatal(shotErr)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	r, g, bl, _ := img.At((b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2+b.Dy()/6).RGBA()
	if r>>8 < g>>8+25 || r>>8 < bl>>8+25 {
		t.Errorf("pane middle %d,%d,%d over red: want the red to show through the glass", r>>8, g>>8, bl>>8)
	}
}
