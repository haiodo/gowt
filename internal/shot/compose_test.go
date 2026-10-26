package shot

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeSolid(t *testing.T, path string, w, h int, c color.NRGBA) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func pixel(t *testing.T, path string, x, y int) color.NRGBA {
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
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

// A popover is drawn where its frame says, scaled to the window's pixels per point, and the canvas grows
// when it reaches past the window.
func TestCompose(t *testing.T) {
	dir := t.TempDir()
	base, over := filepath.Join(dir, "base.png"), filepath.Join(dir, "over.png")
	writeSolid(t, base, 200, 100, color.NRGBA{255, 255, 255, 255}) // 100 x 50 points at scale 2
	writeSolid(t, over, 20, 10, color.NRGBA{255, 0, 0, 255})       // 10 x 5 points at scale 2
	fr, err := Compose(base, Frame{100, 200, 100, 50}, over, Frame{110, 240, 10, 5})
	if err != nil || fr != (Frame{100, 200, 100, 50}) {
		t.Fatalf("frame %+v, err %v", fr, err)
	}
	if c := pixel(t, base, 20, 80); c != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("pixel (20,80) is %v, want the popover's red", c)
	}
	if c := pixel(t, base, 19, 80); c != (color.NRGBA{255, 255, 255, 255}) {
		t.Errorf("pixel (19,80) is %v, want the window's white", c)
	}

	fr, err = Compose(base, fr, over, Frame{95, 190, 10, 5})
	if err != nil || fr != (Frame{95, 190, 105, 60}) {
		t.Fatalf("the popover left of and above the window: frame %+v, err %v", fr, err)
	}
	if c := pixel(t, base, 0, 0); c != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("pixel (0,0) is %v, want the popover's red", c)
	}
	if c := pixel(t, base, 10, 20); c != (color.NRGBA{255, 255, 255, 255}) {
		t.Errorf("pixel (10,20) is %v, want the window's white", c)
	}
}
