package snapcmp

import (
	"image"
	"image/color"
	"testing"
)

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestIdentical(t *testing.T) {
	a := solid(10, 10, color.RGBA{200, 200, 200, 255})
	r := Compare(a, solid(10, 10, color.RGBA{200, 200, 200, 255}), 0)
	if r.Differing != 0 || !r.Within(0) {
		t.Fatalf("identical images differ: %+v", r)
	}
}

func TestAntialiasingBelowThreshold(t *testing.T) {
	a := solid(10, 10, color.RGBA{200, 200, 200, 255})
	b := solid(10, 10, color.RGBA{210, 195, 200, 255})
	if r := Compare(a, b, 16); r.Differing != 0 {
		t.Fatalf("delta 10 should be tolerated at threshold 16: %d", r.Differing)
	}
	if r := Compare(a, b, 5); r.Differing != 100 {
		t.Fatalf("delta 10 should count at threshold 5: %d", r.Differing)
	}
}

func TestCaretSizedChangeWithinFraction(t *testing.T) {
	a := solid(100, 100, color.RGBA{255, 255, 255, 255})
	b := solid(100, 100, color.RGBA{255, 255, 255, 255})
	for y := 10; y < 20; y++ {
		b.SetRGBA(50, y, color.RGBA{0, 0, 0, 255})
	}
	r := Compare(a, b, 16)
	if r.Differing != 10 || !r.Within(0.002) || r.Within(0.0005) {
		t.Fatalf("10 of 10000 px: %+v fraction %v", r, r.Fraction())
	}
	if r.Diff.At(50, 15) != (color.RGBA{255, 0, 0, 255}) {
		t.Fatal("diff image must mark the changed pixel red")
	}
}

func TestSizeMismatchFails(t *testing.T) {
	r := Compare(solid(10, 10, color.RGBA{}), solid(10, 11, color.RGBA{}), 16)
	if !r.SizeMismatch || r.Within(1) {
		t.Fatalf("size mismatch must fail: %+v", r)
	}
}
