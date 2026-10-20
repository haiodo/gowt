package look_test

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt"
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
