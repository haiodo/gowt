package gowt

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/swt"
)

// styledTextShot captures a window whose panel has the colour bg and holds one empty StyledText built
// from opts that fills it. The panel passes no background on (INHERIT_NONE): a GTK StyledText takes
// an inherited one by itself. GOWT_TEST_SHOTS=<dir> keeps the PNGs.
func styledTextShot(t *testing.T, name string, bg RGB, opts ...Option) (img image.Image, style int32, border int32, errs []error) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	dir := os.Getenv("GOWT_TEST_SHOTS")
	if dir == "" {
		dir = t.TempDir()
	}
	path := filepath.Join(dir, name+".png")
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			app.OnError(func(e error) { errs = append(errs, e) })
			var text *StyledTextView
			w := app.Window("styled")
			w.SetSize(320, 200)
			w.Add(VBox(Grow(), Padding(0), Background(bg),
				Init(func(p *Panel) { p.AsComposite().SetBackgroundMode(swt.INHERIT_NONE) }),
				StyledText(append(opts, ReadOnly(), Grow(), Ref(&text))...)))
			w.Show()
			app.After(400*time.Millisecond, func() {
				style = text.Unwrap().GetStyle()
				border = text.Unwrap().GetBorderWidth()
				if e := shot.WindowPNG(shotHandle(w.shell), path); e != nil {
					t.Error(e)
				}
				w.Close()
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
	if img, err = png.Decode(f); err != nil {
		t.Fatal(err)
	}
	return img, style, border, errs
}

// centreNear reports whether the pixel at the centre of img is within 10 levels of c in every channel:
// colour management on macOS moves a captured colour by a few levels.
func centreNear(img image.Image, c RGB) (RGB, bool) {
	b := img.Bounds()
	r, g, bl, _ := img.At((b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2).RGBA()
	got := RGB{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)}
	d := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	return got, d(got.R, c.R) <= 10 && d(got.G, c.G) <= 10 && d(got.B, c.B) <= 10
}

// A StyledText with NoBackground shows the panel colour under it, one without it paints its own.
// NoBorder is accepted, wins over Border and leaves no frame.
func TestStyledTextNoBackground(t *testing.T) {
	marker := RGB{R: 40, G: 160, B: 90}
	img, style, border, errs := styledTextShot(t, "nobackground", marker, Border(), NoBorder(), NoBackground(), Scrollbars())
	for _, e := range errs {
		t.Errorf("OnError: %v", e)
	}
	if got, ok := centreNear(img, marker); !ok {
		t.Errorf("StyledText with NoBackground shows %v, want the panel colour %v", got, marker)
	}
	if style&swt.BORDER != 0 || border != 0 {
		t.Errorf("StyledText with Border and NoBorder: BORDER style %v, border width %d, want none", style&swt.BORDER != 0, border)
	}
	img, _, _, _ = styledTextShot(t, "background", marker)
	if got, ok := centreNear(img, marker); ok {
		t.Errorf("StyledText without NoBackground shows the panel colour %v, want its own background", got)
	}
}
