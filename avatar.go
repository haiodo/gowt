package gowt

import (
	"math"

	"github.com/haiodo/gowt/swt"
)

// AvatarImage makes a round image size points wide: a disc of bg with the initials in white, the
// size of the text following size. The image is drawn at every zoom of the display, so it is sharp
// on a scaled screen; use it as the image of a Table or Tree cell, a Label or a Button.
func (a *App) AvatarImage(initials string, bg RGB, size int) *Image {
	return a.ImageFromProvider(&circleProvider{a: a, size: size, draw: func(gc *swt.GC, n int32) {
		gc.SetBackground(bg.color())
		gc.FillRectangle(0, 0, n, n)
		f := Font{Px: 0.4 * float64(n), Weight: Semibold}.from(a.display.GetSystemFont(), a.display)
		gc.SetFont(a.font(f))
		defer a.releaseFont(f)
		gc.SetForeground(RGB{255, 255, 255}.color())
		ext := gc.TextExtent(initials)
		gc.DrawText(initials, (n-ext.X)/2, (n-ext.Y)/2)
	}})
}

// CircleImage makes a round copy of img, size points wide: the middle square of img is scaled and
// clipped to a circle with a smooth edge. img stays as it is and may be disposed afterwards only
// when the copy is no longer shown, as the copy draws from it again at another zoom.
func (a *App) CircleImage(img *Image, size int) *Image {
	b := img.i.GetBounds()
	side := min(b.Width, b.Height)
	return a.ImageFromProvider(&circleProvider{a: a, size: size, draw: func(gc *swt.GC, n int32) {
		gc.SetInterpolation(swt.HIGH)
		gc.DrawImageImageSrcXSrcYSrcWidthSrcHeightDestXDestYDestWidthDestHeight(img.i, (b.Width-side)/2, (b.Height-side)/2, side, side, 0, 0, n, n)
	}})
}

// circleProvider renders a square with draw and cuts the corners with an alpha edge.
type circleProvider struct {
	a    *App
	size int
	draw func(gc *swt.GC, n int32)
}

func (p *circleProvider) GetImageData(zoom int32) *swt.ImageData {
	px := int32(p.size) * zoom / 100
	// A temporary image is made in points of the display: its pixels are n * deviceZoom / 100.
	n := (px*100 + swt.DPIUtilGetDeviceZoom() - 1) / swt.DPIUtilGetDeviceZoom()
	tmp := swt.NewImageDeviceWidthHeight(p.a.display, n, n)
	defer tmp.Dispose()
	gc := swt.NewGCDrawable(tmp)
	p.draw(gc, n)
	gc.Dispose()
	data := tmp.GetImageDataAtCurrentZoom()
	roundAlpha(data)
	return data
}

// roundAlpha sets the alpha of d to a disc inscribed in it, one pixel of anti-aliased edge.
func roundAlpha(d *swt.ImageData) {
	r := float64(min(d.Width, d.Height)) / 2
	for y := int32(0); y < d.Height; y++ {
		for x := int32(0); x < d.Width; x++ {
			dist := math.Hypot(float64(x)+0.5-r, float64(y)+0.5-r)
			d.SetAlpha(x, y, int32(255*math.Max(0, math.Min(1, r-dist+0.5))))
		}
	}
}
