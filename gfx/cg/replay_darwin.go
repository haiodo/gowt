//go:build darwin

package cg

import (
	"math"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

var _ gfx.Renderer = (*Renderer)(nil)

// NewTarget makes an offscreen bitmap of wpx x hpx device pixels for gfx.
func (r *Renderer) NewTarget(wpx, hpx int) gfx.Target { return target{r.NewSurface(wpx, hpx)} }

// target adapts a Surface to gfx.Target.
type target struct{ *Surface }

func (t target) Begin(scale float64, origin gfx.Point, bg gfx.Color) {
	t.Surface.Begin(scale, origin.X, origin.Y, bg)
}

func (t target) Size() (int, int) { return t.W, t.H }

func (t target) Snapshot(wpt, hpt float64) gfx.Image { return t.Surface.Snapshot(wpt, hpt) }

// MeasureText returns the CoreText extents of s, cached per font and string.
func (r *Renderer) MeasureText(s string, f gfx.Font) gfx.TextMetrics {
	font := r.font(f)
	k := lineKey{font, s, f.Spacing}
	if v, ok := r.metrics.Load(k); ok {
		return v.(gfx.TextMetrics)
	}
	line := r.newLine(s, font, f.Spacing)
	var a, d, l float64
	w := ctLineTypographicBounds(line, &a, &d, &l)
	cfRelease(line)
	m := gfx.TextMetrics{Width: w, Ascent: a, Descent: d}
	r.metrics.Store(k, m)
	return m
}

// lodFadeBand is how far above LodBelow a Lod picture still shows through, as a factor of LodBelow.
const lodFadeBand = 1.3

func isTranslation(m gfx.Affine) bool { return m[0] == 1 && m[1] == 0 && m[2] == 0 && m[3] == 1 }

func (s *Surface) concat(m gfx.Affine) {
	if isTranslation(m) {
		if m[4] != 0 || m[5] != 0 {
			cgTranslate(s.ctx, m[4], m[5])
		}
		return
	}
	cgConcat(s.ctx, cgAffine{m[0], m[1], m[2], m[3], m[4], m[5]})
}

// Draw paints the items in order: each under its clips, through its transform, at its alpha.
// Items that follow one another under the same clips share the clip state: a rounded or path clip
// costs CoreGraphics a mask the size of the clip, which is too much to set again for every item.
func (s *Surface) Draw(items []gfx.Item) {
	ctx := s.ctx
	var stack []gfx.Clip
	for i := range items {
		it := &items[i]
		p := 0
		for p < len(stack) && p < len(it.Clips) && sameClip(&stack[p], &it.Clips[p]) {
			p++
		}
		for len(stack) > p {
			cgRestoreGSt(ctx)
			stack = stack[:len(stack)-1]
		}
		for j := p; j < len(it.Clips); j++ {
			cgSaveGState(ctx)
			s.clip(&it.Clips[j])
			stack = append(stack, it.Clips[j])
		}
		cgSaveGState(ctx)
		s.concat(it.M)
		s.eff = s.scale * it.M.Scale()
		if it.Alpha < 1 {
			cgSetAlpha(ctx, it.Alpha)
			cgBeginLayer(ctx, 0)
		}
		s.replay(it.Pic)
		if it.Alpha < 1 {
			cgEndLayer(ctx)
		}
		cgRestoreGSt(ctx)
	}
	for range stack {
		cgRestoreGSt(ctx)
	}
	s.eff = s.scale
}

func sameClip(a, b *gfx.Clip) bool {
	return a.M == b.M && a.Kind == b.Kind && a.Rect == b.Rect && a.Radius == b.Radius && a.Path == b.Path
}

// clip intersects the clip with c, leaving the transform as it was.
func (s *Surface) clip(c *gfx.Clip) {
	ctx := s.ctx
	s.concat(c.M)
	switch c.Kind {
	case gfx.ClipRect:
		cgClipRect(ctx, cgRect{c.Rect.Min.X, c.Rect.Min.Y, c.Rect.W(), c.Rect.H()})
	case gfx.ClipRRect:
		var rr gfx.Path
		rr.RoundRect(c.Rect.Min.X, c.Rect.Min.Y, c.Rect.W(), c.Rect.H(), c.Radius)
		s.addPath(&rr)
		cgClip(ctx)
	case gfx.ClipPath:
		s.addPath(c.Path)
		s.clipFilled(c.Path)
	}
	if inv, ok := c.M.Invert(); ok {
		s.concat(inv)
	}
}

// Replay draws p translated by (dx, dy).
func (s *Surface) Replay(p *gfx.Picture, dx, dy float64) {
	cgSaveGState(s.ctx)
	cgTranslate(s.ctx, dx, dy)
	s.eff = s.scale
	s.replay(p)
	cgRestoreGSt(s.ctx)
}

func (s *Surface) replay(p *gfx.Picture) {
	ctx := s.ctx
	if s.r.opts.LOD {
		if w, h := p.Bounds.W()*s.eff, p.Bounds.H()*s.eff; w < 3 && h < 3 {
			cgFillColor(ctx, 0.75, 0.75, 0.78, 1)
			cgFillRect(ctx, cgRect{p.Bounds.Min.X, p.Bounds.Min.Y, math.Max(p.Bounds.W(), 1/s.eff), math.Max(p.Bounds.H(), 1/s.eff)})
			return
		}
		if p.Lod != nil {
			// the substitute fades out over the band above LodBelow, so the look does not jump
			size := math.Max(p.Bounds.W(), p.Bounds.H()) * s.eff
			switch {
			case size < p.LodBelow:
				p = p.Lod
			case size < p.LodBelow*lodFadeBand:
				s.replayOps(p.Lod)
				cgSaveGState(ctx)
				cgSetAlpha(ctx, (size-p.LodBelow)/(p.LodBelow*(lodFadeBand-1)))
				cgBeginLayer(ctx, 0)
				s.replayOps(p)
				cgEndLayer(ctx)
				cgRestoreGSt(ctx)
				return
			}
		}
	}
	s.replayOps(p)
}

func (s *Surface) replayOps(p *gfx.Picture) {
	ctx := s.ctx
	cgSaveGState(ctx)
	s.dashed = false
	layers := 0
	s.it.Reset(p)
	for s.it.Next() {
		op := s.it.Op()
		if s.r.skip>>op.Kind&1 != 0 {
			continue
		}
		switch op.Kind {
		case gfx.OpSave:
			cgSaveGState(ctx)
		case gfx.OpRestore:
			cgRestoreGSt(ctx)
		case gfx.OpTranslate:
			cgTranslate(ctx, op.A, op.B)
		case gfx.OpScale:
			cgScale(ctx, op.A, op.B)
		case gfx.OpRotate:
			cgRotate(ctx, op.A)
		case gfx.OpTransform:
			s.concat(op.M)
		case gfx.OpFill:
			s.fill(op.Path, op.Color)
		case gfx.OpFillGrad:
			cgSaveGState(ctx)
			s.addPath(op.Path)
			s.clipFilled(op.Path)
			s.gradient(op.Grad)
			cgRestoreGSt(ctx)
		case gfx.OpStroke:
			s.stroke(op.Path, op.Style, op.Color)
		case gfx.OpStrokeGrad:
			cgSaveGState(ctx)
			s.strokeStyle(op.Style)
			s.addPath(op.Path)
			cgStrokedPath(ctx)
			cgClip(ctx)
			s.gradient(op.Grad)
			cgRestoreGSt(ctx)
			s.dashed = false
		case gfx.OpShadowBox:
			s.shadowBox(op)
		case gfx.OpFillShadow:
			s.withShadow(op.Shadow, func() { s.fill(op.Path, op.Color) })
		case gfx.OpStrokeShadow:
			s.withShadow(op.Shadow, func() { s.stroke(op.Path, op.Style, op.Color) })
		case gfx.OpText:
			s.text(op.Text, op)
		case gfx.OpTextShadow:
			s.withShadow(op.Shadow, func() { s.text(op.Text, op) })
		case gfx.OpImage:
			img := s.r.cgImage(op.Image)
			if img == nil {
				continue
			}
			cgSaveGState(ctx)
			cgTranslate(ctx, op.A, op.B+op.D)
			cgScale(ctx, 1, -1)
			cgDrawImage(ctx, cgRect{0, 0, op.C, op.D}, img.ref)
			cgRestoreGSt(ctx)
		case gfx.OpClipRect:
			cgClipRect(ctx, cgRect{op.A, op.B, op.C, op.D})
		case gfx.OpClipRRect:
			var rr gfx.Path
			rr.RoundRect(op.A, op.B, op.C, op.D, op.E)
			s.addPath(&rr)
			cgClip(ctx)
		case gfx.OpClipPath:
			s.addPath(op.Path)
			s.clipFilled(op.Path)
		case gfx.OpBeginLayer:
			cgSaveGState(ctx)
			cgSetAlpha(ctx, op.A)
			cgBeginLayer(ctx, 0)
			layers++
		case gfx.OpEndLayer:
			if layers > 0 {
				layers--
				cgEndLayer(ctx)
				cgRestoreGSt(ctx)
			}
		}
	}
	for ; layers > 0; layers-- {
		cgEndLayer(ctx)
		cgRestoreGSt(ctx)
	}
	cgRestoreGSt(ctx)
}

// cgImage returns the CGImage behind img: a snapshot as it is, a Raster converted once.
func (r *Renderer) cgImage(img gfx.Image) *Image {
	switch v := img.(type) {
	case *Image:
		return v
	case *gfx.Raster:
		if c, ok := r.images.Load(v); ok {
			return c.(*Image)
		}
		ctx := cgBitmapContextCreate(0, uintptr(v.W), uintptr(v.H), 8, uintptr(v.W*4), r.cs, bitmapInfoBGRA)
		if ctx == 0 {
			return nil
		}
		addr := cgBitmapContextGetData(ctx)
		dst := unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&addr))), v.W*v.H*4)
		for i := 0; i+3 < len(v.Pix); i += 4 {
			a := uint32(v.Pix[i+3])
			dst[i] = byte((uint32(v.Pix[i+2])*a + 127) / 255)
			dst[i+1] = byte((uint32(v.Pix[i+1])*a + 127) / 255)
			dst[i+2] = byte((uint32(v.Pix[i])*a + 127) / 255)
			dst[i+3] = byte(a)
		}
		c := &Image{cgBitmapContextCreateImage(ctx), float64(v.W), float64(v.H)}
		cgContextRelease(ctx)
		actual, loaded := r.images.LoadOrStore(v, c)
		if loaded {
			c.Release()
		}
		return actual.(*Image)
	}
	return nil
}

// gradient paints g over the current clip.
func (s *Surface) gradient(g *gfx.Gradient) {
	n := len(g.Stops)
	if n == 0 {
		return
	}
	comps := make([]float64, 0, n*4)
	locs := make([]float64, 0, n)
	for _, st := range g.Stops {
		comps = append(comps, st.Color.R, st.Color.G, st.Color.B, st.Color.A)
		locs = append(locs, st.Offset)
	}
	cg := cgGradCreate(s.r.cs, &comps[0], &locs[0], uintptr(n))
	if cg == 0 {
		return
	}
	const extend = 3 // draw before the start and after the end
	if g.Kind == gfx.RadialGradient {
		cgDrawRadial(s.ctx, cg, cgPoint{g.P0.X, g.P0.Y}, g.R0, cgPoint{g.P1.X, g.P1.Y}, g.R1, extend)
	} else {
		cgDrawLinear(s.ctx, cg, cgPoint{g.P0.X, g.P0.Y}, cgPoint{g.P1.X, g.P1.Y}, extend)
	}
	cgGradRelease(cg)
}

// clipFilled clips to the path already added, by the fill rule of p.
func (s *Surface) clipFilled(p *gfx.Path) {
	if p.EvenOdd {
		cgEOClip(s.ctx)
	} else {
		cgClip(s.ctx)
	}
}

func (s *Surface) addPath(p *gfx.Path) {
	ctx := s.ctx
	cgBeginPath(ctx)
	pts := p.Pts
	for _, v := range p.Verbs {
		switch v {
		case gfx.MoveTo:
			cgMoveTo(ctx, pts[0].X, pts[0].Y)
			pts = pts[1:]
		case gfx.LineTo:
			cgLineTo(ctx, pts[0].X, pts[0].Y)
			pts = pts[1:]
		case gfx.QuadTo:
			cgQuadTo(ctx, pts[0].X, pts[0].Y, pts[1].X, pts[1].Y)
			pts = pts[2:]
		case gfx.CubicTo:
			cgCurveTo(ctx, pts[0].X, pts[0].Y, pts[1].X, pts[1].Y, pts[2].X, pts[2].Y)
			pts = pts[3:]
		case gfx.ClosePath:
			cgClosePath(ctx)
		}
	}
}

// strokeStyle sets the stroke state and returns the share of the line that stays inked: a dash
// dropped for being unreadable is paid for by that share of the alpha.
func (s *Surface) strokeStyle(st *gfx.StrokeStyle) float64 {
	ctx := s.ctx
	w := st.Width
	if s.r.opts.LOD && w*s.eff < 1 {
		w = 1 / s.eff // hairline floor: a thin line stays one device pixel wide instead of washing out
	}
	cgLineWidth(ctx, w)
	cgLineCap(ctx, int32(st.Cap))
	cgLineJoin(ctx, int32(st.Join))
	if st.MiterLimit > 0 {
		cgMiterLimit(ctx, st.MiterLimit)
	}
	duty := 1.0
	switch {
	case len(st.Dash) > 0 && !(s.r.opts.LOD && s.eff < 0.4):
		cgLineDash(ctx, st.DashPhase, &st.Dash[0], uintptr(len(st.Dash)))
		s.dashed = true
	case s.dashed:
		cgLineDash(ctx, 0, nil, 0)
		s.dashed = false
	}
	if len(st.Dash) > 0 && !s.dashed {
		duty = dashDuty(st.Dash)
	}
	return duty
}

// dashDuty is the inked share of a dash pattern; CoreGraphics repeats an odd pattern with on and off swapped.
func dashDuty(d []float64) float64 {
	var on, all float64
	for i, v := range d {
		all += v
		if i%2 == 0 {
			on += v
		}
	}
	if len(d)%2 == 1 {
		return 0.5
	}
	if all <= 0 {
		return 1
	}
	return on / all
}

func (s *Surface) fill(p *gfx.Path, c gfx.Color) {
	cgFillColor(s.ctx, c.R, c.G, c.B, c.A)
	s.addPath(p)
	if p.EvenOdd {
		cgEOFillPath(s.ctx)
	} else {
		cgFillPath(s.ctx)
	}
}

func (s *Surface) stroke(p *gfx.Path, st *gfx.StrokeStyle, c gfx.Color) {
	duty := s.strokeStyle(st)
	cgStrokeColor(s.ctx, c.R, c.G, c.B, c.A*duty)
	s.addPath(p)
	cgStrokePath(s.ctx)
}

// shadowState sets the shadow of sh on the context; clearShadow ends it. CoreGraphics reads the
// offset in device pixels and ignores the transform, so it is taken through the current one: a
// content flipped or turned by a transform keeps its shadow on the side the node's own axes say.
func (s *Surface) shadowState(sh gfx.Shadow) {
	m := cgGetCTM(s.ctx)
	off := cgSize{m.A*sh.DX + m.C*sh.DY, m.B*sh.DX + m.D*sh.DY}
	col := cgColorCreate(sh.Color.R, sh.Color.G, sh.Color.B, sh.Color.A)
	cgSetShadow(s.ctx, off, sh.Blur*math.Sqrt(math.Abs(m.A*m.D-m.B*m.C)), col)
	cgColorRelase(col)
}

func (s *Surface) clearShadow() { cgSetShadow(s.ctx, cgSize{}, 0, 0) }

// withShadow runs draw with the shadow of sh under what it paints.
func (s *Surface) withShadow(sh gfx.Shadow, draw func()) {
	if s.r.opts.LOD && s.eff < 0.5 {
		sh.Color.A *= math.Max(0, s.eff*4-1)
	}
	if sh.Color.A <= 0 {
		draw()
		return
	}
	s.shadowState(sh)
	draw()
	s.clearShadow()
}

func (s *Surface) shadowBox(op *gfx.Op) {
	ctx := s.ctx
	x, y, w, h, r := op.A, op.B, op.C, op.D, op.E
	sh := op.Shadow
	var rr gfx.Path
	rr.RoundRect(x, y, w, h, r)
	if s.r.opts.LOD && s.eff < 0.5 {
		sh.Color.A *= math.Max(0, s.eff*4-1) // a shadow under half scale is a smudge: fade it out to a quarter
	}
	if sh.Color.A > 0 && s.r.opts.ShadowSprite && w > 2*(r+1.5*sh.Blur) && h > 2*(r+1.5*sh.Blur) {
		s.r.sprite(r, sh.Blur, sh.Color, s.eff, math.Max(math.Abs(sh.DX), math.Abs(sh.DY)), op.Color.A >= 1).draw(s, x+sh.DX, y+sh.DY, w, h)
	} else if sh.Color.A > 0 {
		cgSaveGState(ctx)
		s.shadowState(sh)
		cgFillColor(ctx, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
		s.addPath(&rr)
		cgFillPath(ctx)
		cgRestoreGSt(ctx)
		return
	}
	cgFillColor(ctx, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
	s.addPath(&rr)
	cgFillPath(ctx)
}
