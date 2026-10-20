package gowt

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt/internal/shot"
)

// hintShot captures a window holding one multi-line Text built from opts, after edit has run on it.
func hintShot(t *testing.T, name string, edit func(*TextView), opts ...Option) image.Image {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	path := filepath.Join(t.TempDir(), name+".png")
	var err error
	onMain(func() {
		err = Run(func(app *App) {
			var text *TextView
			w := app.Window("hint")
			w.SetSize(320, 120)
			w.Add(Text(append(opts, Multiline(), Border(), Grow(), Ref(&text))...))
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
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			r1, g1, b1, _ := a.At(x, y).RGBA()
			r2, g2, b2, _ := b.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 {
				n++
			}
		}
	}
	return n
}

// A blinking caret differs by a thin bar at most; the hint is far more pixels than that.
const caretPixels = 200

func TestMultilineHint(t *testing.T) {
	const hint = "Write a message to the whole team"
	plain := hintShot(t, "plain", nil)
	hinted := hintShot(t, "hinted", nil, Hint(hint))
	if d := pixelDiff(plain, hinted); d <= caretPixels {
		t.Errorf("hint not drawn: %d pixels differ from a field without it", d)
	}
	filled := hintShot(t, "filled", nil, Value("x"))
	if d := pixelDiff(filled, hintShot(t, "hintedFilled", nil, Hint(hint), Value("x"))); d > caretPixels {
		t.Errorf("hint drawn over text: %d pixels differ", d)
	}
	cleared := hintShot(t, "cleared", func(v *TextView) { v.SetText("x"); v.SetText("") }, Hint(hint))
	if d := pixelDiff(hinted, cleared); d > caretPixels {
		t.Errorf("hint not back after the text was cleared: %d pixels differ", d)
	}
}
