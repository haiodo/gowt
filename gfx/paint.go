package gfx

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
)

type GradientKind uint8

const (
	LinearGradient GradientKind = iota
	RadialGradient
)

// Stop is a color at Offset 0..1 along a gradient.
type Stop struct {
	Offset float64
	Color  Color
}

// Gradient is linear from P0 to P1, or radial from the circle (P0, R0) to the circle (P1, R1).
// Its coordinates are in the space of the shape that uses it.
type Gradient struct {
	Kind   GradientKind
	P0, P1 Point
	R0, R1 float64
	Stops  []Stop
}

func NewLinearGradient(p0, p1 Point, stops ...Stop) *Gradient {
	return &Gradient{Kind: LinearGradient, P0: p0, P1: p1, Stops: stops}
}

func NewRadialGradient(c0 Point, r0 float64, c1 Point, r1 float64, stops ...Stop) *Gradient {
	return &Gradient{Kind: RadialGradient, P0: c0, R0: r0, P1: c1, R1: r1, Stops: stops}
}

// Raster is a portable image: non-premultiplied RGBA, 4 bytes per pixel, rows top to bottom.
// Backends convert it on first use and keep the converted copy by the *Raster, so a Raster
// must not change after it was drawn.
type Raster struct {
	W, H int
	Pix  []byte
}

func NewRaster(w, h int) *Raster { return &Raster{W: w, H: h, Pix: make([]byte, w*h*4)} }

func (r *Raster) Size() (float64, float64) { return float64(r.W), float64(r.H) }

// RasterFrom copies img.
func RasterFrom(img image.Image) *Raster {
	b := img.Bounds()
	n := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(n, n.Bounds(), img, b.Min, draw.Src)
	return &Raster{W: b.Dx(), H: b.Dy(), Pix: n.Pix}
}

// Image returns r as an image.NRGBA sharing the pixels.
func (r *Raster) Image() *image.NRGBA {
	return &image.NRGBA{Pix: r.Pix, Stride: r.W * 4, Rect: image.Rect(0, 0, r.W, r.H)}
}

func (r *Raster) PNG() ([]byte, error) {
	var b bytes.Buffer
	if err := png.Encode(&b, r.Image()); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// FromBGRA converts premultiplied BGRA pixels, the layout of the CoreGraphics bitmaps, to a Raster.
func FromBGRA(pix []byte, w, h int) *Raster {
	r := NewRaster(w, h)
	for i := 0; i+3 < len(pix) && i+3 < len(r.Pix); i += 4 {
		b, g, rr, a := pix[i], pix[i+1], pix[i+2], pix[i+3]
		if a != 0 && a != 255 {
			rr = byte(min(int(rr)*255/int(a), 255))
			g = byte(min(int(g)*255/int(a), 255))
			b = byte(min(int(b)*255/int(a), 255))
		}
		r.Pix[i], r.Pix[i+1], r.Pix[i+2], r.Pix[i+3] = rr, g, b, a
	}
	return r
}
