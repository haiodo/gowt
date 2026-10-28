package look_test

import (
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
)

// On the inspector glass, light and dark: segmented controls draw, a toggle that is on stands out from
// one that is off, nothing fills black. GOWT_PIXELS_DIR keeps inspector-controls-<mode>.png.
func TestInspectorControlsOnGlass(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	for _, mode := range []struct {
		name string
		look gowt.Appearance
	}{{"light", gowt.AppearanceLight}, {"dark", gowt.AppearanceDark}} {
		t.Run(mode.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "controls.png")
			if dir := os.Getenv("GOWT_PIXELS_DIR"); dir != "" {
				path = filepath.Join(dir, "inspector-controls-"+mode.name+".png")
			}
			rects, width := captureInspectorControls(t, mode.look, path)
			img := decodePNG(t, path)
			scale := float64(img.Bounds().Dx()) / float64(width)
			for name, r := range rects {
				if got := blackShare(img, r, scale); got > 0.01 {
					t.Errorf("%s at %v: %.0f%% of its pixels are black", name, r, got*100)
				}
			}
			for _, name := range []string{"sizing", "align", "points"} {
				if lo, hi := lumaRange(img, rects[name], scale); hi-lo < 30 {
					t.Errorf("segmented %s: luma %0.f..%0.f, nothing drawn", name, lo, hi)
				}
			}
			for _, pair := range [][2]string{{"wrap on", "clip off"}, {"eye on", "eye off"}} {
				on, off := meanLuma(img, rects[pair[0]], scale), meanLuma(img, rects[pair[1]], scale)
				t.Logf("%s %.0f, %s %.0f", pair[0], on, pair[1], off)
				if d := on - off; d < 6 && d > -6 {
					t.Errorf("%s (%.0f) looks like %s (%.0f)", pair[0], on, pair[1], off)
				}
			}
		})
	}
}

// captureInspectorControls captures a window whose inspector holds the controls; it returns their
// rectangles in window points and the window width.
func captureInspectorControls(t *testing.T, mode gowt.Appearance, path string) (map[string][4]int32, int32) {
	t.Helper()
	ctl := map[string]*swt.Control{}
	probe := func(name string) gowt.Option {
		return gowt.Custom(func(c *swt.Control) { ctl[name] = c })
	}
	rects := map[string][4]int32{}
	var width int32
	var errs []error
	var shotErr, err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.SetAppearance(mode)
			app.OnError(func(e error) { errs = append(errs, e) })
			img := func(name string) *gowt.Image { return icons.Get(app, name, icons.Size(13)) }
			icon := func(name string) gowt.Option {
				return gowt.Init(func(b *gowt.ButtonView) { b.SetImage(img(name)) })
			}
			w := app.Window("inspector controls")
			var content, insp *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&content), gowt.Label("content")),
				gowt.VBox(gowt.Ref(&insp), gowt.Padding(gowt.Spacing.MD), gowt.Gap(gowt.Spacing.SM),
					gowt.HBox(gowt.GrowX(), gowt.Gap(0),
						gowt.Label("Layout", gowt.Typeface(gowt.Type.BodyStrong)),
						gowt.Spacer(),
						gowt.Button("", gowt.Toggle(), gowt.Borderless(), gowt.Small(), gowt.Checked(true), icon("search"), probe("eye on")),
						gowt.Button("", gowt.Toggle(), gowt.Borderless(), gowt.Small(), icon("search"), probe("eye off")),
						gowt.Button("", gowt.Borderless(), gowt.Small(), icon("add"), probe("add"))),
					gowt.Segmented([]gowt.Segment{{Text: "Fixed"}, {Text: "Hug"}, {Text: "Fill"}}, gowt.Small(), gowt.GrowX(), gowt.Selected(1), probe("sizing")),
					gowt.Segmented([]gowt.Segment{{Image: img("back"), Tooltip: "Left"}, {Image: img("menu"), Tooltip: "Center"}, {Image: img("forward"), Tooltip: "Right"}},
						gowt.GrowX(), gowt.Selected(0), probe("align")),
					gowt.Segmented([]gowt.Segment{{Text: "+"}, {Text: "-"}}, gowt.Momentary(), gowt.Small(), probe("points")),
					gowt.HBox(gowt.GrowX(),
						gowt.Button("Wrap", gowt.Toggle(), gowt.Small(), gowt.Checked(true), icon("menu"), probe("wrap on")),
						gowt.Button("Clip", gowt.Toggle(), gowt.Small(), icon("cut"), probe("clip off")))))
			look.SetInspector(w, content, insp, 260)
			w.SetSize(700, 420)
			w.Show()
			app.After(700*time.Millisecond, func() {
				defer w.Close()
				defer app.SetAppearance(gowt.AppearanceSystem) // later tests run in the system appearance
				s := w.Unwrap().GetBounds()
				width = s.Width
				for name, c := range ctl {
					o, size := c.ToDisplay(0, 0), c.GetSize()
					rects[name] = [4]int32{o.X - s.X, o.Y - s.Y, size.X, size.Y}
				}
				shotErr = shot.WindowPNG(shotHandle(w), path)
			})
		})
	})
	for _, e := range append(errs, err, shotErr) {
		if e != nil {
			t.Fatal(e)
		}
	}
	if len(rects) != 8 {
		t.Fatalf("probed %d controls, want 8", len(rects))
	}
	return rects, width
}

// lumaRange is the lowest and highest Rec. 601 luma in r (window points) at scale pixels per point.
func lumaRange(img image.Image, r [4]int32, scale float64) (lo, hi float64) {
	px := image.Rect(int(float64(r[0])*scale), int(float64(r[1])*scale),
		int(float64(r[0]+r[2])*scale), int(float64(r[1]+r[3])*scale)).Intersect(img.Bounds())
	lo = 255
	for y := px.Min.Y; y < px.Max.Y; y++ {
		for x := px.Min.X; x < px.Max.X; x++ {
			cr, cg, cb, _ := img.At(x, y).RGBA()
			l := (0.299*float64(cr) + 0.587*float64(cg) + 0.114*float64(cb)) / 257
			lo, hi = min(lo, l), max(hi, l)
		}
	}
	return lo, hi
}
