package look_test

import (
	"fmt"
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

// The examples/materialpanel layout: after a typed character, one capture over two blink periods differs
// from a capture with the caret hidden in a thin column of contrasting pixels. On macOS the caret was white,
// unseen on the light material; GOWT_SNAP_SCREEN captures what the window server composites (may need
// Screen Recording permission). GOWT_TEST_SHOTS=<dir> keeps the PNGs.
func TestMaterialCaret(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	t.Setenv("GOWT_SNAP_SCREEN", "1") // read on macOS only
	dir := os.Getenv("GOWT_TEST_SHOTS")
	if dir == "" {
		dir = t.TempDir()
	}
	const shown = 12
	var errs []error
	var typed string
	var focused bool
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			app.SetAppearance(gowt.AppearanceLight)
			app.OnError(func(e error) { errs = append(errs, e) })
			white := gowt.RGB{R: 255, G: 255, B: 255}
			w := app.Window("caret")
			w.SetSize(480, 360)
			var pane *gowt.Panel
			var text *gowt.StyledTextView
			w.Add(gowt.VBox(gowt.Grow(), gowt.Padding(0), gowt.Background(white),
				gowt.Split(gowt.Vertical(), gowt.Grow(),
					gowt.Canvas(func(g *gowt.GC) {}, gowt.Background(white)),
					gowt.VBox(gowt.Ref(&pane), gowt.Padding(0),
						gowt.StyledText(gowt.NoBackground(), gowt.NoBorder(), gowt.Grow(), gowt.Ref(&text), gowt.Focused(),
							gowt.Init(func(t *gowt.StyledTextView) { t.SetText("func main() {}") }))))))
			look.SetMaterial(pane, true)
			w.Show()
			capture := func(name string) {
				if e := shot.WindowPNG(shotHandle(w), filepath.Join(dir, name+".png")); e != nil {
					errs = append(errs, e)
				}
			}
			app.After(500*time.Millisecond, func() {
				st := text.Unwrap()
				focused = st.IsFocusControl()
				st.SetCaretOffset(st.GetCharCount())
				ev := swt.NewEvent()
				ev.Character, ev.Doit = 'x', true
				st.NotifyListeners(swt.KeyDown, ev)
				typed = st.GetText()
				caret := st.GetCaret()
				caret.SetVisible(false)
				app.After(300*time.Millisecond, func() {
					capture("caret-off")
					caret.SetVisible(true)
					var next func(i int)
					next = func(i int) {
						capture(fmt.Sprintf("caret-on%d", i))
						if i == shown-1 {
							w.Close()
							return
						}
						app.After(100*time.Millisecond, func() { next(i + 1) })
					}
					app.After(50*time.Millisecond, func() { next(0) })
				})
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range errs {
		t.Fatalf("OnError or capture: %v", e)
	}
	if !focused {
		t.Fatal("the StyledText has no keyboard focus, so it shows no caret")
	}
	if typed != "func main() {}x" {
		t.Errorf("text after typing x: %q", typed)
	}
	off := decodePNG(t, filepath.Join(dir, "caret-off.png"))
	var seen []string
	for i := range shown {
		box, n := changed(off, decodePNG(t, filepath.Join(dir, fmt.Sprintf("caret-on%d.png", i))))
		if n >= 6 && box.Dy() >= 6 && box.Dx()*3 <= box.Dy() {
			t.Logf("caret in capture %d: %d px in %v", i, n, box)
			return
		}
		seen = append(seen, fmt.Sprintf("%d px in %v", n, box))
	}
	t.Errorf("no capture shows the caret as a thin contrasting column: %v", seen)
}

func decodePNG(t *testing.T, path string) image.Image {
	t.Helper()
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

// changed returns the box and the number of the pixels that differ by more than 60 levels in a channel
// between a and b, both flattened over white.
func changed(a, b image.Image) (box image.Rectangle, n int) {
	flat := func(img image.Image, x, y int) [3]int {
		r, g, bl, al := img.At(x, y).RGBA()
		over := func(c uint32) int { return int((c + 0xffff - al) >> 8) }
		return [3]int{over(r), over(g), over(bl)}
	}
	r := a.Bounds().Intersect(b.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			p, q := flat(a, x, y), flat(b, x, y)
			if max(p[0]-q[0], q[0]-p[0], p[1]-q[1], q[1]-p[1], p[2]-q[2], q[2]-p[2]) > 60 {
				box = box.Union(image.Rect(x, y, x+1, y+1))
				n++
			}
		}
	}
	return box, n
}
