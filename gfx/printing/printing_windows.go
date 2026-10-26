//go:build windows

package printing

import (
	"sync"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/d2d"
	"github.com/haiodo/gowt/swt"
)

var (
	rendOnce sync.Once
	rend     *d2d.Renderer
	rendErr  error
)

// bandRows is the height of one bitmap sent to the printer, in dots: 4096 dots at 600 dpi is a
// band of 28 MB, which a 32-bit page of any size is cut into.
const bandRows = 1024

func nativePage(p *gowt.PrintPage) (gfx.Page, error) {
	rendOnce.Do(func() { rend, rendErr = d2d.NewRenderer(d2d.Options{}) })
	if rendErr != nil {
		return nil, rendErr
	}
	return bandPage{rend, p.Native()}, nil
}

type bandPage struct {
	r *d2d.Renderer
	n gowt.NativeContext
}

// Draw renders the page in bands at the printer's resolution and draws each as an image.
func (b bandPage) Draw(scale float64, origin gfx.Point, items []gfx.Item) {
	f := b.n.UnitsPerPoint
	w, h := int(b.n.Width), int(b.n.Height)
	for y := 0; y < h; y += bandRows {
		rows := min(bandRows, h-y)
		t := b.r.NewTarget(w, rows)
		t.Begin(f*scale, gfx.Point{X: origin.X, Y: origin.Y + float64(y)/(f*scale)}, gfx.Color{})
		t.Draw(items)
		r := gfx.FromBGRA(t.Pixels(), w, rows)
		t.Close()
		if img := swtImage(b.n.GC, r); img != nil {
			b.n.GC.DrawImage(img, 0, int32(y))
			img.Dispose()
		}
	}
}

// swtImage makes a device image of r, or nil when r is fully transparent.
func swtImage(gc *swt.GC, r *gfx.Raster) *swt.Image {
	pad := (r.W*3 + 3) &^ 3
	data := make([]int8, pad*r.H)
	alpha := make([]int8, r.W*r.H)
	seen := false
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			src := r.Pix[(y*r.W+x)*4:]
			o := y*pad + x*3
			data[o], data[o+1], data[o+2] = int8(src[2]), int8(src[1]), int8(src[0]) // B G R in the bytes of the masks below
			alpha[y*r.W+x] = int8(src[3])
			seen = seen || src[3] != 0
		}
	}
	if !seen {
		return nil
	}
	id := swt.NewImageDataWidthHeightDepthPaletteScanlinePadData(int32(r.W), int32(r.H), 24, swt.NewPaletteDataRedMaskGreenMaskBlueMask(0xFF, 0xFF00, 0xFF0000), 4, data)
	id.AlphaData = alpha
	return swt.NewImageDeviceData(gc.GetDevice(), id)
}
