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

// styledTextShot captures a window filled by a panel of the colour bg that holds, 40 points in, one
// empty StyledText built from opts. The panel passes no background on (INHERIT_NONE): a GTK
// StyledText takes an inherited one by itself. GOWT_TEST_SHOTS=<dir> keeps the PNGs.
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
			w := app.Window("styled", Padding(0))
			w.SetSize(320, 200)
			w.Add(VBox(Grow(), Padding(40), Background(bg),
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

// centreLikePanel compares the pixel at the centre of img, in the StyledText, with one 20 points from
// the left edge, in the panel padding: both pass the same colour management (macOS captures in
// Display P3, where sRGB 40,160,90 reads 80,158,97).
func centreLikePanel(img image.Image) (centre, panel RGB, same bool) {
	b := img.Bounds()
	at := func(x, y int) RGB {
		r, g, bl, _ := img.At(x, y).RGBA()
		return RGB{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)}
	}
	y := (b.Min.Y + b.Max.Y) / 2
	centre, panel = at((b.Min.X+b.Max.X)/2, y), at(b.Min.X+b.Dx()*20/320, y)
	d := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	return centre, panel, d(centre.R, panel.R) <= 4 && d(centre.G, panel.G) <= 4 && d(centre.B, panel.B) <= 4
}

// A StyledText with NoBackground shows the panel colour under it, one without it paints its own.
// NoBorder is accepted, wins over Border and leaves no frame.
func TestStyledTextNoBackground(t *testing.T) {
	marker := RGB{R: 40, G: 160, B: 90}
	img, style, border, errs := styledTextShot(t, "nobackground", marker, Border(), NoBorder(), NoBackground(), Scrollbars())
	for _, e := range errs {
		t.Errorf("OnError: %v", e)
	}
	if got, panel, ok := centreLikePanel(img); !ok {
		t.Errorf("StyledText with NoBackground shows %v, want the panel colour %v", got, panel)
	}
	if style&swt.BORDER != 0 || border != 0 {
		t.Errorf("StyledText with Border and NoBorder: BORDER style %v, border width %d, want none", style&swt.BORDER != 0, border)
	}
	img, _, _, _ = styledTextShot(t, "background", marker)
	if got, _, ok := centreLikePanel(img); ok {
		t.Errorf("StyledText without NoBackground shows the panel colour %v, want its own background", got)
	}
}
