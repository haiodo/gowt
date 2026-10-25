//go:build windows

package d2d

import (
	"math"

	"github.com/haiodo/gowt/gfx"
)

// A box shadow is a Gaussian blur of the rounded rectangle, computed in Go and drawn from a cached
// bitmap: big boxes from a 9-slice sprite (four corners, four stretched edges, a stretched core),
// small ones from one bitmap of the whole shadow. Direct2D's shadow effect would need a command
// list per box; the sprite costs nine DrawBitmap calls.

type spriteKey struct {
	radius, blur float64
	col          gfx.Color
	scale        float64
}

type shadowSprite struct {
	img  *Image
	q, m int // corner size and blur margin, in device pixels
	side int
}

func (s *shadowSprite) release() { s.img.Release() }

type wholeKey struct {
	w, h, radius, blur float64
	col                gfx.Color
	scale              float64
}

// rrCoverage is the antialiased coverage of the rounded rectangle [x0,x1]x[y0,y1] with radius r at a pixel center.
func rrCoverage(px, py, x0, y0, x1, y1, r float64) float64 {
	cx, cy := (x0+x1)/2, (y0+y1)/2
	qx := math.Abs(px-cx) - ((x1-x0)/2 - r)
	qy := math.Abs(py-cy) - ((y1-y0)/2 - r)
	d := math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
	return math.Min(math.Max(0.5-d, 0), 1)
}

// blurShadow returns the premultiplied BGRA pixels of the rounded rectangle [bx,bx+bw]x[by,by+bh]
// (device pixels) blurred with a Gaussian of standard deviation sigma, in a w x h bitmap, in color col.
func blurShadow(w, h int, bx, by, bw, bh, r, sigma float64, col gfx.Color) []byte {
	a := make([]float32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a[y*w+x] = float32(rrCoverage(float64(x)+0.5, float64(y)+0.5, bx, by, bx+bw, by+bh, r))
		}
	}
	if sigma > 0.01 {
		rad := int(math.Ceil(3 * sigma))
		k := make([]float32, 2*rad+1)
		var sum float32
		for i := range k {
			d := float64(i - rad)
			k[i] = float32(math.Exp(-d * d / (2 * sigma * sigma)))
			sum += k[i]
		}
		for i := range k {
			k[i] /= sum
		}
		tmp := make([]float32, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				var acc float32
				for i, kv := range k {
					if xx := x + i - rad; xx >= 0 && xx < w {
						acc += kv * a[y*w+xx]
					}
				}
				tmp[y*w+x] = acc
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				var acc float32
				for i, kv := range k {
					if yy := y + i - rad; yy >= 0 && yy < h {
						acc += kv * tmp[yy*w+x]
					}
				}
				a[y*w+x] = acc
			}
		}
	}
	out := make([]byte, w*h*4)
	for i, v := range a {
		al := float64(v) * col.A
		out[i*4] = byte(math.Round(col.B * al * 255))
		out[i*4+1] = byte(math.Round(col.G * al * 255))
		out[i*4+2] = byte(math.Round(col.R * al * 255))
		out[i*4+3] = byte(math.Round(al * 255))
	}
	return out
}

func (rp *replayer) sprite(radius, blur float64, col gfx.Color, scale float64) *shadowSprite {
	k := spriteKey{radius, blur, col, scale}
	if v, ok := rp.r.sprites.Load(k); ok {
		return v.(*shadowSprite)
	}
	m := int(math.Ceil(1.5 * blur * scale))
	q := int(math.Ceil(radius*scale)) + 2*m // keeps the 1 pixel strip clear of both blur ramps
	side := 2*q + 1
	inner := float64(side - 2*m)
	pix := blurShadow(side, side, float64(m), float64(m), inner, inner, radius*scale, blur/2*scale, col)
	bmp, err := newBitmap(rp.dc, side, side, pix, side*4, 0)
	if err != nil {
		return nil
	}
	sp := &shadowSprite{&Image{bmp, float64(side), float64(side)}, q, m, side}
	v, loaded := rp.r.sprites.LoadOrStore(k, sp)
	if loaded {
		sp.release()
	}
	return v.(*shadowSprite)
}

func (rp *replayer) drawPiece(bmp uintptr, dx, dy, dw, dh float64, sx, sy, sw, sh int) {
	vcall(rp.dc, d2DCDrawBitmap, ui(bmp), ref(&rectF{float32(dx), float32(dy), float32(dx + dw), float32(dy + dh)}), fl(1), ui(1),
		ref(&rectF{float32(sx), float32(sy), float32(sx + sw), float32(sy + sh)}))
}

// drawShadow paints the shadow of the box (x, y, w, h), already offset by the shadow vector.
func (rp *replayer) drawShadow(x, y, w, h, radius float64, sh gfx.Shadow) {
	sc := rp.eff
	if w > 2*(radius+1.5*sh.Blur) && h > 2*(radius+1.5*sh.Blur) {
		sp := rp.sprite(radius, sh.Blur, sh.Color, sc)
		if sp == nil {
			return
		}
		q, m := float64(sp.q)/sc, float64(sp.m)/sc
		x0, y0, x1, y1 := x-m, y-m, x+w+m, y+h+m
		b, qi, s := sp.img.bmp, sp.q, sp.side
		rp.drawPiece(b, x0, y0, q, q, 0, 0, qi, qi)
		rp.drawPiece(b, x1-q, y0, q, q, qi+1, 0, qi, qi)
		rp.drawPiece(b, x0, y1-q, q, q, 0, qi+1, qi, qi)
		rp.drawPiece(b, x1-q, y1-q, q, q, qi+1, qi+1, qi, qi)
		rp.drawPiece(b, x0+q, y0, x1-x0-2*q, q, qi, 0, 1, qi)
		rp.drawPiece(b, x0+q, y1-q, x1-x0-2*q, q, qi, qi+1, 1, qi)
		rp.drawPiece(b, x0, y0+q, q, y1-y0-2*q, 0, qi, qi, 1)
		rp.drawPiece(b, x1-q, y0+q, q, y1-y0-2*q, qi+1, qi, qi, 1)
		rp.drawPiece(b, x0+q, y0+q, x1-x0-2*q, y1-y0-2*q, qi, qi, 1, 1)
		_ = s
		return
	}
	k := wholeKey{w, h, radius, sh.Blur, sh.Color, sc}
	var img *Image
	if v, ok := rp.r.wholes.Load(k); ok {
		img = v.(*Image)
	} else {
		m := int(math.Ceil(1.5 * sh.Blur * sc))
		bw, bh := int(math.Ceil(w*sc)), int(math.Ceil(h*sc))
		pw, ph := bw+2*m, bh+2*m
		if pw > 4096 || ph > 4096 {
			return
		}
		pix := blurShadow(pw, ph, float64(m), float64(m), float64(bw), float64(bh), math.Min(radius*sc, math.Min(float64(bw), float64(bh))/2), sh.Blur/2*sc, sh.Color)
		bmp, err := newBitmap(rp.dc, pw, ph, pix, pw*4, 0)
		if err != nil {
			return
		}
		img = &Image{bmp, float64(pw), float64(ph)}
		if v, loaded := rp.r.wholes.LoadOrStore(k, img); loaded {
			img.Release()
			img = v.(*Image)
		}
	}
	pw, ph := int(img.w), int(img.h)
	m := float64((pw - int(math.Ceil(w*sc))) / 2)
	rp.drawPiece(img.bmp, x-m/sc, y-m/sc, float64(pw)/sc, float64(ph)/sc, 0, 0, pw, ph)
}

func (rp *replayer) shadowBox(op *gfx.Op) {
	x, y, w, h, r := op.A, op.B, op.C, op.D, op.E
	sh := op.Shadow
	if rp.r.opts.LOD && rp.eff < 0.5 {
		sh.Color.A *= math.Max(0, rp.eff*4-1) // a shadow under half scale is a smudge: fade it out to a quarter
	}
	if rp.ctm.Det() == 0 {
		return
	}
	rp.applyM()
	if sh.Color.A > 0 {
		rp.drawShadow(x+sh.DX, y+sh.DY, w, h, r, sh)
	}
	var rr gfx.Path
	rr.RoundRect(x, y, w, h, r)
	rp.fill(&rr, op.Color)
}
