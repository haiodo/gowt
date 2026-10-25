//go:build linux

package cairo

import (
	"math"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

// shadowBox draws the box over its shadow. cairo has no blur: the shape goes into an A8 surface,
// is blurred in Go (Gaussian of sigma blur/2 as three box passes) and masks the shadow color.
func (t *Target) shadowBox(op *gfx.Op) {
	cr := t.cr
	x, y, w, h, rad := op.A, op.B, op.C, op.D, op.E
	sh := op.Shadow
	var rr gfx.Path
	rr.RoundRect(x, y, w, h, rad)
	if t.r.opts.LOD && t.eff < 0.5 {
		sh.Color.A *= math.Max(0, t.eff*4-1) // a shadow under half scale is a smudge: fade it out
	}
	if sh.Color.A > 0 && w > 0 && h > 0 {
		t.blurredShape(x+sh.DX, y+sh.DY, w, h, rad, sh.Blur/2, sh.Color)
	}
	setSourceRGBA(cr, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
	t.addPath(&rr)
	setFillRule(cr, fillRuleWinding)
	fill(cr)
}

func (t *Target) blurredShape(x, y, w, h, rad, sigma float64, col gfx.Color) {
	cr := t.cr
	var m matrix
	getMatrix(cr, &m)
	sd := sigma * math.Sqrt(math.Abs(m.xx*m.yy-m.xy*m.yx))
	margin := int(math.Ceil(3 * sd))
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range [4][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		dx, dy := m.xx*c[0]+m.xy*c[1]+m.x0, m.yx*c[0]+m.yy*c[1]+m.y0
		x0, y0, x1, y1 = math.Min(x0, dx), math.Min(y0, dy), math.Max(x1, dx), math.Max(y1, dy)
	}
	ox, oy := int(math.Floor(x0))-margin, int(math.Floor(y0))-margin
	bw, bh := int(math.Ceil(x1))+margin-ox, int(math.Ceil(y1))+margin-oy
	if bw <= 0 || bh <= 0 || bw > 1<<14 || bh > 1<<14 {
		return
	}
	mask := imageSurfaceCreate(formatA8, int32(bw), int32(bh))
	defer surfaceDestroy(mask)
	mcr := create(mask)
	mm := matrix{m.xx, m.yx, m.xy, m.yy, m.x0 - float64(ox), m.y0 - float64(oy)}
	setMatrix(mcr, &mm)
	setSourceRGBA(mcr, 0, 0, 0, 1)
	var rr gfx.Path
	rr.RoundRect(x, y, w, h, rad)
	sub := &Target{r: t.r, cr: mcr}
	sub.addPath(&rr)
	fill(mcr)
	destroy(mcr)
	surfaceFlush(mask)
	if sd > 0.3 {
		stride := int(imageSurfaceGetStride(mask))
		blurA8(unsafe.Slice((*byte)(ptr(imageSurfaceGetData(mask))), stride*bh), bw, bh, stride, sd)
		surfaceMarkDirty(mask)
	}
	save(cr)
	identityMatrix(cr)
	setSourceRGBA(cr, col.R, col.G, col.B, col.A)
	maskSurface(cr, mask, float64(ox), float64(oy))
	restore(cr)
}

// boxSizes returns the n box widths whose repeated blur approximates a Gaussian of sigma.
func boxSizes(sigma float64, n int) []int {
	ideal := math.Sqrt(12*sigma*sigma/float64(n) + 1)
	wl := int(math.Floor(ideal))
	if wl%2 == 0 {
		wl--
	}
	wu := wl + 2
	m := int(math.Round((12*sigma*sigma - float64(n*wl*wl) - float64(4*n*wl) - float64(3*n)) / float64(-4*wl-4)))
	sz := make([]int, n)
	for i := range sz {
		if i < m {
			sz[i] = wl
		} else {
			sz[i] = wu
		}
	}
	return sz
}

// blurA8 blurs the alpha plane in place; integer arithmetic keeps the bytes stable.
func blurA8(pix []byte, w, h, stride int, sigma float64) {
	tmp := make([]byte, max(w, h))
	for _, size := range boxSizes(sigma, 3) {
		r := (size - 1) / 2
		if r < 1 {
			continue
		}
		for y := 0; y < h; y++ {
			boxLine(pix[y*stride:], 1, w, r, tmp)
		}
		for x := 0; x < w; x++ {
			boxLine(pix[x:], stride, h, r, tmp)
		}
	}
}

// boxLine averages n samples spaced step apart over a window of 2r+1, edges counted as zero.
func boxLine(p []byte, step, n, r int, tmp []byte) {
	div := 2*r + 1
	sum := 0
	for i := 0; i <= r && i < n; i++ {
		sum += int(p[i*step])
	}
	for i := 0; i < n; i++ {
		tmp[i] = byte((sum + div/2) / div)
		if j := i + r + 1; j < n {
			sum += int(p[j*step])
		}
		if j := i - r; j >= 0 {
			sum -= int(p[j*step])
		}
	}
	for i := 0; i < n; i++ {
		p[i*step] = tmp[i]
	}
}
