package shot

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"

	xdraw "golang.org/x/image/draw"
)

// Finish brings the PNG to GOWT_SNAP_SIZE=<w>x<h> (outer size in points) at 2 px per point; a smaller capture
// scale is scaled up. The capture is stretched to the target when within 3% of it, else cropped or padded.
func Finish(path string) error {
	var w, h int
	if _, err := fmt.Sscanf(os.Getenv("GOWT_SNAP_SIZE"), "%dx%d", &w, &h); err != nil || w <= 0 || h <= 0 {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	out, scale, stretched := fit(src, w, h)
	fmt.Fprintf(os.Stderr, "gowt: snap capture scale %g, exported at 2x as %dx%d, captured %dx%d, stretched %v\n",
		scale, out.Bounds().Dx(), out.Bounds().Dy(), src.Bounds().Dx(), src.Bounds().Dy(), stretched)
	f, err = os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}

// fit returns img as 2w x 2h, the capture scale (image width over w, to a quarter) and whether it was stretched.
func fit(img image.Image, w, h int) (*image.NRGBA, float64, bool) {
	b := img.Bounds()
	scale := math.Max(1, math.Round(float64(b.Dx())/float64(w)*4)/4)
	out := image.NewNRGBA(image.Rect(0, 0, 2*w, 2*h))
	if b.Dx() == 2*w && b.Dy() == 2*h {
		draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
		return out, scale, false
	}
	sw, sh := float64(b.Dx())*2/scale, float64(b.Dy())*2/scale
	if math.Abs(sw-float64(2*w)) <= 0.03*float64(2*w) && math.Abs(sh-float64(2*h)) <= 0.03*float64(2*h) {
		xdraw.CatmullRom.Scale(out, out.Bounds(), img, b, xdraw.Src, nil)
		return out, scale, true
	}
	scaled := image.NewNRGBA(image.Rect(0, 0, int(math.Round(sw)), int(math.Round(sh))))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), img, b, xdraw.Src, nil)
	draw.Draw(out, out.Bounds(), scaled, image.Point{}, draw.Src)
	return out, scale, false
}
