package gowt

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/shot"
)

// hintShot captures a window holding one read-only multi-line Text built from opts, after edit has run
// on it. Read-only: a blinking caret would differ between two shots. GOWT_TEST_SHOTS=<dir> keeps the PNGs.
func hintShot(t *testing.T, name string, edit func(*TextView), opts ...Option) image.Image {
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
			var text *TextView
			w := app.Window("hint")
			w.SetSize(320, 120)
			w.Add(Text(append(opts, Multiline(), ReadOnly(), Border(), Grow(), Ref(&text))...))
			w.Show()
			if edit != nil {
				edit(text)
			}
			app.After(400*time.Millisecond, func() {
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
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// pixelDiff counts the pixels that differ between a and b.
func pixelDiff(a, b image.Image) int {
	if a.Bounds() != b.Bounds() {
		return a.Bounds().Dx() * a.Bounds().Dy()
	}
	n := 0
	r := a.Bounds()
	if runtime.GOOS == "windows" {
		// The DWM shadow and corners blend with whatever is behind the window: skip the frame and title bar.
		r = image.Rect(r.Min.X+12, r.Min.Y+40, r.Max.X-12, r.Max.Y-12)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			r1, g1, b1, _ := a.At(x, y).RGBA()
			r2, g2, b2, _ := b.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 {
				n++
			}
		}
	}
	return n
}

// Two shots of the same field differ by a few pixels at most; a hint by far more.
const (
	samePixels = 20
	hintPixels = 200
)

func TestMultilineHint(t *testing.T) {
	const hint = "Write a message to the whole team"
	plain := hintShot(t, "plain", nil)
	hinted := hintShot(t, "hinted", nil, Hint(hint))
	if d := pixelDiff(plain, hinted); d <= hintPixels {
		t.Errorf("hint not drawn: %d pixels differ from a field without it", d)
	}
	filled := hintShot(t, "filled", nil, Value("x"))
	if d := pixelDiff(filled, hintShot(t, "hintedFilled", nil, Hint(hint), Value("x"))); d > samePixels {
		t.Errorf("hint drawn over text: %d pixels differ", d)
	}
	cleared := hintShot(t, "cleared", func(v *TextView) { v.SetText("x"); v.SetText("") }, Hint(hint))
	if d := pixelDiff(hinted, cleared); d > samePixels {
		t.Errorf("hint not back after the text was cleared: %d pixels differ", d)
	}
}
