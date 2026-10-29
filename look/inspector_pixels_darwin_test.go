package look_test

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/look"
	"github.com/haiodo/gowt/swt"
)

// Controls on the sidebar and inspector glass get an alpha-0 background, and none may fill its bounds
// with black: a check box drew its bezel in that colour at full opacity. GOWT_PIXELS_DIR keeps the PNG.
func TestPanesNoBlackFill(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var controls []*swt.Control
	probe := gowt.Custom(func(c *swt.Control) { controls = append(controls, c) })
	path := filepath.Join(t.TempDir(), "panes.png")
	if dir := os.Getenv("GOWT_PIXELS_DIR"); dir != "" { // relative to the package directory, where go test runs
		path = filepath.Join(dir, "panes.png")
	}
	rects := map[string][4]int32{} // window points, from the window's top left corner
	var width int32
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			w := app.Window("panes")
			var side, content, insp *gowt.Panel
			w.Add(
				gowt.VBox(gowt.Ref(&side), gowt.List([]string{"a", "b"}, gowt.GrowX()),
					gowt.Label("sidebar label", probe),
					gowt.Switch("sidebar switch", gowt.GrowX(), probe)),
				gowt.VBox(gowt.Ref(&content), gowt.Label("content")),
				gowt.VBox(gowt.Ref(&insp), gowt.Padding(16), gowt.Gap(10),
					gowt.Label("inspector label", probe),
					gowt.Grid(2, gowt.GrowX(), gowt.Label("Kind"), gowt.Label("Pane", probe)),
					gowt.Switch("Shared with others", gowt.GrowX(), probe),
					gowt.Switch("Locked", gowt.GrowX(), gowt.Checked(true), probe),
					gowt.Button("Radio", gowt.Radio(), probe),
					gowt.Section(gowt.GrowX(), probe,
						gowt.Grid(2, gowt.Padding(6), gowt.Label("Row", gowt.GrowX()), gowt.Switch("", gowt.Checked(true)))),
					gowt.Scale(0, 100, gowt.Value(40), gowt.GrowX(), probe),
					gowt.Combo([]string{"Auto", "Light"}, gowt.ReadOnly(), gowt.Selected(0), probe),
					gowt.Text(gowt.GrowX(), gowt.Value("text"), probe),
					gowt.Button("Push", probe)))
			look.SetSidebar(w, side, content, 220)
			look.SetInspector(w, content, insp, 260)
			w.SetSize(900, 600)
			w.Show()
			app.After(500*time.Millisecond, func() {
				defer w.Close()
				s := w.Unwrap().GetBounds()
				width = s.Width
				for _, c := range controls {
					o, size := c.ToDisplay(0, 0), c.GetSize()
					rects[c.String()] = [4]int32{o.X - s.X, o.Y - s.Y, size.X, size.Y}
					switch c.Impl().(type) {
					case *swt.Text, *swt.Combo: // fields keep their native background on glass
						if a := c.GetBackground().GetAlpha(); a != 255 {
							t.Errorf("%s has background alpha %d, want its own field", c, a)
						}
					}
				}
				if err := shot.WindowPNG(w.Unwrap().View.Id, path); err != nil {
					t.Error(err)
				}
			})
		})
	})
	if err != nil {
		t.Fatal(err)
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
	scale := float64(img.Bounds().Dx()) / float64(width)
	for name, r := range rects {
		if got := blackShare(img, r, scale); got > 0.01 {
			t.Errorf("%s at %v: %.0f%% of its pixels are black", name, r, got*100)
		}
	}
	if len(rects) != 12 {
		t.Errorf("probed %d controls, want 12", len(rects))
	}
}

// A Text on the inspector glass: a borderless one shows the glass behind it, not the opaque
// textBackgroundColor bar it drew in dark (30,30,30); a field with a border keeps the native fill.
func TestInspectorTextOnGlass(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	for _, mode := range []struct {
		name string
		look gowt.Appearance
	}{{"dark", gowt.AppearanceDark}, {"light", gowt.AppearanceLight}} {
		t.Run(mode.name, func(t *testing.T) {
			fields := map[string][4]int32{}
			bordered := map[string]bool{}
			var ctl []*swt.Control
			var names []string
			probe := func(name string, border bool) gowt.Option {
				return gowt.Custom(func(c *swt.Control) {
					ctl, names = append(ctl, c), append(names, name)
					bordered[name] = border
				})
			}
			path := filepath.Join(t.TempDir(), "fields.png")
			if dir := os.Getenv("GOWT_PIXELS_DIR"); dir != "" {
				path = filepath.Join(dir, "fields-"+mode.name+".png")
			}
			var width int32
			var ref [4]int32    // the middle of the content pane: dark in dark, light in light
			var backdrop string // the material behind the inspector and the appearances, at the capture
			late := path[:len(path)-len(".png")] + "-late.png"
			var err error
			onMain(func() {
				err = gowt.Run(func(app *gowt.App) {
					app.SetAppearance(mode.look)
					w := app.Window("fields " + mode.name)
					var content, insp *gowt.Panel
					w.Add(
						gowt.VBox(gowt.Ref(&content), gowt.Label("content")),
						gowt.VBox(gowt.Ref(&insp), gowt.Padding(16), gowt.Gap(16),
							gowt.Text(gowt.GrowX(), gowt.Value("Root"), probe("single-line", false)),
							gowt.Text(gowt.GrowX(), gowt.Border(), gowt.Value("Root"), probe("single-line Border", true)),
							gowt.Text(gowt.Multiline(), gowt.Border(), gowt.GrowX(), gowt.Size(0, 60), probe("multi-line Border", true))))
					look.SetInspector(w, content, insp, 260)
					w.SetSize(700, 400)
					w.Show()
					app.After(500*time.Millisecond, func() {
						// A second capture a second later tells a glass that is slow to follow the
						// appearance from one that never does.
						defer app.After(time.Second, func() {
							defer w.Close()
							defer app.SetAppearance(gowt.AppearanceSystem) // later tests run in the system appearance
							if err := shot.WindowPNG(w.Unwrap().View.Id, late); err != nil {
								t.Error(err)
							}
						})
						name, _, _ := cocoa.BackdropOf(insp.Unwrap().View)
						backdrop = fmt.Sprintf("%s, its appearance dark %v, NSApp dark %v", name,
							cocoa.BackdropDark(insp.Unwrap().View), cocoa.EffectiveAppearanceDark())
						s := w.Unwrap().GetBounds()
						width = s.Width
						for i, c := range ctl {
							o, size := c.ToDisplay(0, 0), c.GetSize()
							fields[names[i]] = [4]int32{o.X - s.X, o.Y - s.Y, size.X, size.Y}
						}
						o, size := content.Unwrap().ToDisplay(0, 0), content.Unwrap().GetSize()
						ref = [4]int32{o.X - s.X + size.X/2 - 10, o.Y - s.Y + size.Y/2 - 10, 20, 20}
						if err := shot.WindowPNG(w.Unwrap().View.Id, path); err != nil {
							t.Error(err)
						}
					})
				})
			})
			if err != nil {
				t.Fatal(err)
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
			scale := float64(img.Bounds().Dx()) / float64(width)
			if l := meanLuma(img, ref, scale); (l < 128) != (mode.look == gowt.AppearanceDark) {
				t.Skipf("the capture does not show the %s appearance: content pane luma %.0f", mode.name, l)
			}
			for name, r := range fields {
				// The right half of the field, clear of "Root" and of the border, against a strip of glass
				// in the gap above it.
				field := meanLuma(img, [4]int32{r[0] + r[2]/2, r[1] + 5, r[2]/2 - 6, r[3] - 10}, scale)
				glass := meanLuma(img, [4]int32{r[0], r[1] - 10, r[2], 6}, scale)
				t.Logf("%s at %v: field luma %.0f, glass %.0f", name, r, field, glass)
				// A field without a border shows the glass: it must take no fill of its own. One with
				// a border keeps the native NSTextField fill, which is darker than the glass in Dark
				// Aqua by design; neither may be a black bar.
				if !bordered[name] && field < glass-8 {
					t.Errorf("%s at %v: field luma %.0f is darker than the glass above it, %.0f", name, r, field, glass)
				}
				if field < 8 {
					t.Errorf("%s at %v: field luma %.0f is a black fill", name, r, field)
				}
			}
			if t.Failed() {
				t.Logf("backdrop %s; glass luma a second later: %s", backdrop, lateGlass(late, fields, scale))
			}
			if len(fields) != 3 {
				t.Errorf("probed %d fields, want 3", len(fields))
			}
		})
	}
}

// lateGlass is the glass luma above each field in the capture at path.
func lateGlass(path string, fields map[string][4]int32, scale float64) string {
	f, err := os.Open(path)
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return err.Error()
	}
	var out []string
	for name, r := range fields {
		out = append(out, fmt.Sprintf("%s %.0f", name, meanLuma(img, [4]int32{r[0], r[1] - 10, r[2], 6}, scale)))
	}
	return strings.Join(out, ", ")
}

// meanLuma is the mean Rec. 601 luma, 0..255, of r (window points) at scale pixels per point.
func meanLuma(img image.Image, r [4]int32, scale float64) float64 {
	px := image.Rect(int(float64(r[0])*scale), int(float64(r[1])*scale),
		int(float64(r[0]+r[2])*scale), int(float64(r[1]+r[3])*scale)).Intersect(img.Bounds())
	if px.Empty() {
		return 0
	}
	sum := 0.0
	for y := px.Min.Y; y < px.Max.Y; y++ {
		for x := px.Min.X; x < px.Max.X; x++ {
			cr, cg, cb, _ := img.At(x, y).RGBA()
			sum += (0.299*float64(cr) + 0.587*float64(cg) + 0.114*float64(cb)) / 257
		}
	}
	return sum / float64(px.Dx()*px.Dy())
}

// blackShare is the share of opaque near-black pixels in r (window points), at scale pixels per point.
func blackShare(img image.Image, r [4]int32, scale float64) float64 {
	px := image.Rect(int(float64(r[0])*scale), int(float64(r[1])*scale),
		int(float64(r[0]+r[2])*scale), int(float64(r[1]+r[3])*scale)).Intersect(img.Bounds())
	if px.Empty() {
		return 0
	}
	black := 0
	for y := px.Min.Y; y < px.Max.Y; y++ {
		for x := px.Min.X; x < px.Max.X; x++ {
			cr, cg, cb, ca := img.At(x, y).RGBA()
			if ca > 0xc000 && max(cr, cg, cb) < 0x0900 {
				black++
			}
		}
	}
	return float64(black) / float64(px.Dx()*px.Dy())
}
