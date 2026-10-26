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
	var rr gfx.Path
	rr.RoundRect(x, y, w, h, rad)
	if w > 0 && h > 0 {
		t.shadowUnder(gfx.R(x, y, w, h), op.Shadow, func(sub *Target) {
			sub.addPath(&rr)
			fill(sub.cr)
		})
	}
	setSourceRGBA(cr, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
	t.addPath(&rr)
	setFillRule(cr, fillRuleWinding)
	fill(cr)
}

// shadowPath draws the path of op, stroked or filled, over its shadow.
func (t *Target) shadowPath(op *gfx.Op, stroked bool) {
	cr := t.cr
	p, st := op.Path, op.Style
	if !stroked {
		st = nil
	}
	box := p.Bounds()
	if st != nil {
		limit := math.Max(st.MiterLimit, 10)
		box = box.Expand(st.Width / 2 * math.Max(limit, math.Sqrt2))
	}
	t.shadowUnder(box, op.Shadow, func(sub *Target) {
		sub.addPath(p)
		if st == nil {
			sub.setRule(p)
			fill(sub.cr)
			return
		}
		sub.strokeStyle(st)
		stroke(sub.cr)
	})
	if st == nil {
		setSourceRGBA(cr, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
		t.addPath(p)
		t.setRule(p)
		fill(cr)
		return
	}
	duty := t.strokeStyle(st)
	setSourceRGBA(cr, op.Color.R, op.Color.G, op.Color.B, op.Color.A*duty)
	t.addPath(p)
	stroke(cr)
}

// shadowText draws the text over its shadow.
func (t *Target) shadowText(op *gfx.Op) {
	m := t.r.MeasureText(op.Text, op.Font)
	f := op.Font.Size
	box := gfx.R(op.A-f*0.1, op.B-m.Ascent, m.Width+f*0.2, m.Ascent+m.Descent)
	mask := *op
	mask.Color = gfx.Color{A: 1}
	t.shadowUnder(box, op.Shadow, func(sub *Target) { sub.text(op.Text, &mask) })
	t.text(op.Text, op)
}

// shadowUnder paints the shadow sh of what draw puts in local, which bounds it before the offset.
func (t *Target) shadowUnder(local gfx.Rect, sh gfx.Shadow, draw func(sub *Target)) {
	if t.r.opts.LOD && t.eff < 0.5 {
		sh.Color.A *= math.Max(0, t.eff*4-1) // a shadow under half scale is a smudge: fade it out
	}
	if sh.Color.A > 0 {
		t.blurredShape(local, sh, draw)
	}
}

// blurredShape renders draw into an A8 mask under the current transform moved by the offset of sh
// in local coordinates, so the offset turns and mirrors with the transform, blurs it and paints
// sh.Color through it.
func (t *Target) blurredShape(local gfx.Rect, sh gfx.Shadow, draw func(sub *Target)) {
	cr := t.cr
	var m matrix
	getMatrix(cr, &m)
	sd := sh.Blur / 2 * math.Sqrt(math.Abs(m.xx*m.yy-m.xy*m.yx))
	margin := int(math.Ceil(3 * sd))
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range [4]gfx.Point{local.Min, {X: local.Max.X, Y: local.Min.Y}, {X: local.Min.X, Y: local.Max.Y}, local.Max} {
		lx, ly := c.X+sh.DX, c.Y+sh.DY
		dx, dy := m.xx*lx+m.xy*ly+m.x0, m.yx*lx+m.yy*ly+m.y0
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
	mm := matrix{m.xx, m.yx, m.xy, m.yy,
		m.x0 + m.xx*sh.DX + m.xy*sh.DY - float64(ox), m.y0 + m.yx*sh.DX + m.yy*sh.DY - float64(oy)}
	setMatrix(mcr, &mm)
	setSourceRGBA(mcr, 0, 0, 0, 1)
	setFillRule(mcr, fillRuleWinding)
	draw(&Target{r: t.r, cr: mcr, eff: t.eff})
	destroy(mcr)
	surfaceFlush(mask)
	if sd > 0.3 {
		stride := int(imageSurfaceGetStride(mask))
		blurA8(unsafe.Slice((*byte)(ptr(imageSurfaceGetData(mask))), stride*bh), bw, bh, stride, sd)
		surfaceMarkDirty(mask)
	}
	col := sh.Color
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
