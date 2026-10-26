package shot

import (
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
)

// Frame is where a captured window lies on the screen, y down; points, or pixels on Windows.
type Frame struct{ X, Y, W, H float64 }

// Passive makes Capture leave the focus alone, where it would bring the window to the front and activate
// it (Windows); an open popover closes when its window is deactivated.
var Passive bool

// WindowPNG is Capture without the frame.
func WindowPNG(handle int64, path string) error {
	_, err := Capture(handle, path)
	return err
}

// Compose draws the capture at overPath onto the one at path where the frames say; the canvas grows
// where the second window reaches past the first. It returns the frame of the result.
func Compose(path string, base Frame, overPath string, over Frame) (Frame, error) {
	b, err := readPNG(path)
	if err != nil {
		return base, err
	}
	o, err := readPNG(overPath)
	if err != nil {
		return base, err
	}
	if base.W <= 0 || over.W <= 0 {
		return base, nil
	}
	k := float64(b.Bounds().Dx()) / base.W // pixels per unit
	if w := int(over.W*k + 0.5); w != o.Bounds().Dx() {
		o = scaleTo(o, w, int(over.H*k+0.5))
	}
	r := image.Rect(0, 0, b.Bounds().Dx(), b.Bounds().Dy())
	or := image.Rect(0, 0, o.Bounds().Dx(), o.Bounds().Dy()).Add(image.Pt(int(math.Round((over.X-base.X)*k)), int(math.Round((over.Y-base.Y)*k))))
	u := r.Union(or)
	out := image.NewNRGBA(image.Rect(0, 0, u.Dx(), u.Dy()))
	draw.Draw(out, r.Sub(u.Min), b, b.Bounds().Min, draw.Src)
	draw.Draw(out, or.Sub(u.Min), o, o.Bounds().Min, draw.Over)
	f, err := os.Create(path)
	if err != nil {
		return base, err
	}
	defer f.Close()
	if err := png.Encode(f, out); err != nil {
		return base, err
	}
	return Frame{base.X + float64(u.Min.X)/k, base.Y + float64(u.Min.Y)/k, float64(u.Dx()) / k, float64(u.Dy()) / k}, nil
}

func readPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

// scaleTo resizes img to w x h with the nearest pixel, for a popover on another display scale.
func scaleTo(img image.Image, w, h int) image.Image {
	src := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.Set(x, y, img.At(src.Min.X+x*src.Dx()/w, src.Min.Y+y*src.Dy()/h))
		}
	}
	return out
}
