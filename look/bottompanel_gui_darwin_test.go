package look_test

import (
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
)

// macOS 26: the panel's glass runs from the window's left edge, under the sidebar's glass, to the
// inspector; the panel itself stays right of the sidebar.
func TestBottomPanelUnderSidebar(t *testing.T) {
	if !cocoa.GlassAvailable() {
		t.Skip("macOS 26 glass only; older macOS has a flush sidebar")
	}
	bottomWindow(t, nil, func(app *gowt.App, w *gowt.Window, side, content, panel, insp *gowt.Panel) {
		r := w.Unwrap().GetClientArea()
		sb, pb, ib := side.Unwrap().GetBounds(), panel.Unwrap().GetBounds(), insp.Unwrap().GetBounds()
		f, below, ok := cocoa.BackdropFrame(panel.AsComposite().View, side.AsComposite().View)
		if !ok {
			t.Fatal("the panel has no backdrop")
		}
		want := [4]float64{float64(r.X), float64(pb.Y), float64(ib.X - r.X), float64(pb.Height)}
		if got := [4]float64{f.X, f.Y, f.Width, f.Height}; got != want {
			t.Errorf("panel glass %v, want %v: from the window edge under the sidebar %v to the inspector %v", got, want, sb, ib)
		}
		if !below {
			t.Error("the panel glass lies above the sidebar glass")
		}
		if pb.X < sb.X+sb.Width {
			t.Errorf("panel %v starts under the sidebar %v", pb, sb)
		}
		look.SetSidebarVisible(w, false)
		app.Flush()
		if f, _, _ := cocoa.BackdropFrame(panel.AsComposite().View, nil); f.X != float64(r.X) || f.Width != float64(panel.Unwrap().GetBounds().Width) {
			t.Errorf("sidebar hidden: panel glass %v, panel %v", f, panel.Unwrap().GetBounds())
		}
		look.SetBottomPanelVisible(w, false)
		app.Flush()
		if f, _, _ := cocoa.BackdropFrame(panel.AsComposite().View, nil); f.Height != 0 {
			t.Errorf("panel hidden: glass %v, want 0 high", f)
		}
	})
}

// No window background shows around the floating sidebar, in light and dark. The sidebar glass casts a
// shadow, so each inset is compared with the same strip right of the sidebar edge, in its shadow too.
func TestBottomPanelNoGap(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	if !cocoa.GlassAvailable() {
		t.Skip("macOS 26 glass only; older macOS has a flush sidebar")
	}
	t.Setenv("GOWT_SNAP_SCREEN", "1") // glass shows only in what the window server composites
	dir := os.Getenv("GOWT_PIXELS_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	tint := gowt.Adaptive{Light: gowt.RGB{R: 200, G: 225, B: 255}, Dark: gowt.RGB{R: 20, G: 40, B: 80}}
	for _, mode := range []struct {
		name string
		look gowt.Appearance
		tint gowt.RGB
	}{{"light", gowt.AppearanceLight, tint.Light}, {"dark", gowt.AppearanceDark, tint.Dark}} {
		t.Run(mode.name, func(t *testing.T) {
			mean := captureNoGap(t, mode.look, tint, filepath.Join(dir, "bottom-"+mode.name+".png"))
			want := [3]float64{float64(mode.tint.R), float64(mode.tint.G), float64(mode.tint.B)}
			if d := rgbDistance(mean["content"], want); d > 6 {
				t.Errorf("content %.0f, want its background %v", mean["content"], mode.tint)
			}
			refs := map[string]string{"inset beside content": "content right of sidebar",
				"inset beside panel": "panel right of sidebar", "inset below sidebar": "panel right of sidebar"}
			for inset, ref := range refs {
				if d := rgbDistance(mean[inset], mean[ref]); d > 8 {
					t.Errorf("%s %.0f is %.0f levels from the %s %.0f", inset, mean[inset], d, ref, mean[ref])
				}
			}
		})
	}
}

// captureNoGap captures a sidebar window with a content in tint and a bottom panel, as the window
// server composites it (GOWT_SNAP_SCREEN may need Screen Recording permission), and returns the mean
// colour of the content, the panel and the insets around the sidebar.
func captureNoGap(t *testing.T, mode gowt.Appearance, tint gowt.Adaptive, path string) map[string][3]float64 {
	t.Helper()
	var width int32
	var rects map[string][4]int32
	var err, shotErr error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.SetAppearance(mode)
			w := app.Window("no gap")
			var side, content, panel *gowt.Panel
			w.Add(gowt.VBox(gowt.Ref(&side), gowt.List([]string{"a", "b"}, gowt.Grow())),
				gowt.VBox(gowt.Ref(&content), gowt.Background(tint), gowt.Label("content")),
				gowt.VBox(gowt.Ref(&panel), gowt.Padding(8), gowt.Label("panel")))
			look.SetSidebar(w, side, content, 220)
			look.SetBottomPanel(w, content, panel, 160)
			w.SetSize(900, 600)
			w.Show()
			app.After(800*time.Millisecond, func() {
				defer w.Close()
				defer app.SetAppearance(gowt.AppearanceSystem) // later tests run in the system appearance
				r := w.Unwrap().GetClientArea()
				sb, cb, pb := side.Unwrap().GetBounds(), content.Unwrap().GetBounds(), panel.Unwrap().GetBounds()
				width = r.Width
				in := sb.X - r.X // the sidebar inset, 7 points
				rects = map[string][4]int32{
					"content":              {cb.X + cb.Width/2, cb.Y + cb.Height/2, 40, 20},
					"inset beside content": {r.X + 1, cb.Y + cb.Height/2, in - 2, 20},
					"panel":                {pb.X + pb.Width/2, pb.Y + pb.Height/2, 40, 20},
					"inset beside panel":   {r.X + 1, pb.Y + pb.Height/2, in - 2, 20},
					"inset below sidebar":  {sb.X + sb.Width/2, r.Y + r.Height - in + 1, 40, in - 2},
					// in the sidebar's shadow as the insets are, the sash over the edge is transparent
					"content right of sidebar": {sb.X + sb.Width + 1, cb.Y + cb.Height/2, in - 2, 20},
					"panel right of sidebar":   {sb.X + sb.Width + 1, pb.Y + pb.Height/2, in - 2, 20},
				}
				shotErr = shot.WindowPNG(w.Unwrap().View.Id, path)
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if shotErr != nil {
		t.Fatal(shotErr)
	}
	img := decodePNG(t, path)
	scale := float64(img.Bounds().Dx()) / float64(width)
	mean := map[string][3]float64{}
	for name, r := range rects {
		mean[name] = meanRGB(img, r, scale)
		t.Logf("%s at %v: %.0f", name, r, mean[name])
	}
	return mean
}

// meanRGB is the mean colour, 0..255 per channel, of r (window points) at scale pixels per point.
func meanRGB(img image.Image, r [4]int32, scale float64) [3]float64 {
	px := image.Rect(int(float64(r[0])*scale), int(float64(r[1])*scale),
		int(float64(r[0]+r[2])*scale), int(float64(r[1]+r[3])*scale)).Intersect(img.Bounds())
	var sum [3]float64
	if px.Empty() {
		return sum
	}
	for y := px.Min.Y; y < px.Max.Y; y++ {
		for x := px.Min.X; x < px.Max.X; x++ {
			cr, cg, cb, _ := img.At(x, y).RGBA()
			sum[0] += float64(cr) / 257
			sum[1] += float64(cg) / 257
			sum[2] += float64(cb) / 257
		}
	}
	n := float64(px.Dx() * px.Dy())
	return [3]float64{sum[0] / n, sum[1] / n, sum[2] / n}
}

// rgbDistance is the largest channel difference between a and b.
func rgbDistance(a, b [3]float64) float64 {
	return max(a[0]-b[0], b[0]-a[0], a[1]-b[1], b[1]-a[1], a[2]-b[2], b[2]-a[2])
}
