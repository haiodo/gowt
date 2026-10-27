//go:build !darwin

package look_test

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/shot"
)

// panePixel captures w and returns the pixel at the middle of its lower half, inside the StyledText
// of the TestMaterialPane pane. GOWT_TEST_SHOTS=<dir> keeps the PNG.
func panePixel(w *gowt.Window) (gowt.RGB, error) {
	dir := os.Getenv("GOWT_TEST_SHOTS")
	if dir == "" {
		dir = os.TempDir()
	}
	path := filepath.Join(dir, "material.png")
	if err := shot.WindowPNG(w.Unwrap().Handle, path); err != nil {
		return gowt.RGB{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return gowt.RGB{}, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return gowt.RGB{}, err
	}
	b := img.Bounds()
	r, g, bl, _ := img.At((b.Min.X+b.Max.X)/2, b.Min.Y+b.Dy()*3/4).RGBA()
	return gowt.RGB{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8)}, nil
}

func far(a, b gowt.RGB, levels int) bool {
	d := func(x, y uint8) int { return max(int(x)-int(y), int(y)-int(x)) }
	return d(a.R, b.R) > levels || d(a.G, b.G) > levels || d(a.B, b.B) > levels
}

// paneColourProblem wants the pane pixel to be want and not the white of the content around it.
func paneColourProblem(w *gowt.Window, want, white gowt.RGB) string {
	got, err := panePixel(w)
	if err != nil {
		return err.Error()
	}
	if !far(got, white, 4) || far(got, want, 4) {
		return fmt.Sprintf("pane shows %v, want the material %v", got, want)
	}
	return ""
}

// shotHandle is what shot.Capture takes for w.
func shotHandle(w *gowt.Window) int64 { return w.Unwrap().Handle }
