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
	k := lineKey{font, s}
	if v, ok := r.metrics.Load(k); ok {
		return v.(gfx.TextMetrics)
	}
	line := r.newLine(s, font)
	var a, d, l float64
	w := ctLineTypographicBounds(line, &a, &d, &l)
	cfRelease(line)
	m := gfx.TextMetrics{Width: w, Ascent: a, Descent: d}
	r.metrics.Store(k, m)
	return m
}

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
func (s *Surface) Draw(items []gfx.Item) {
	ctx := s.ctx
	for i := range items {
		it := &items[i]
		cgSaveGState(ctx)
		for j := range it.Clips {
			c := &it.Clips[j]
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
				cgClip(ctx)
			}
			if inv, ok := c.M.Invert(); ok {
				s.concat(inv)
			}
		}
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
	s.eff = s.scale
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
		if p.Lod != nil && math.Max(p.Bounds.W(), p.Bounds.H())*s.eff < p.LodBelow {
			p = p.Lod
		}
	}
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
			cgFillColor(ctx, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
			s.addPath(op.Path)
			cgFillPath(ctx)
		case gfx.OpFillGrad:
			cgSaveGState(ctx)
			s.addPath(op.Path)
			cgClip(ctx)
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
		case gfx.OpText:
			s.text(op.Text, op)
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
			cgClip(ctx)
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

func (s *Surface) strokeStyle(st *gfx.StrokeStyle) {
	ctx := s.ctx
	cgLineWidth(ctx, st.Width)
	cgLineCap(ctx, int32(st.Cap))
	cgLineJoin(ctx, int32(st.Join))
	if st.MiterLimit > 0 {
		cgMiterLimit(ctx, st.MiterLimit)
	}
	switch {
	case len(st.Dash) > 0 && !(s.r.opts.LOD && s.eff < 0.4):
		cgLineDash(ctx, st.DashPhase, &st.Dash[0], uintptr(len(st.Dash)))
		s.dashed = true
	case s.dashed:
		cgLineDash(ctx, 0, nil, 0)
		s.dashed = false
	}
}

func (s *Surface) stroke(p *gfx.Path, st *gfx.StrokeStyle, c gfx.Color) {
	cgStrokeColor(s.ctx, c.R, c.G, c.B, c.A)
	s.strokeStyle(st)
	s.addPath(p)
	cgStrokePath(s.ctx)
}

func (s *Surface) shadowBox(op *gfx.Op) {
	ctx := s.ctx
	x, y, w, h, r := op.A, op.B, op.C, op.D, op.E
	sh := op.Shadow
	var rr gfx.Path
	rr.RoundRect(x, y, w, h, r)
	if s.r.opts.LOD && s.eff < 0.5 {
		sh.Color.A = 0 // a shadow under half scale is a smudge
	}
	if sh.Color.A > 0 && s.r.opts.ShadowSprite && w > 2*(r+1.5*sh.Blur) && h > 2*(r+1.5*sh.Blur) {
		s.r.sprite(r, sh.Blur, sh.Color, s.eff, math.Max(math.Abs(sh.DX), math.Abs(sh.DY)), op.Color.A >= 1).draw(s, x+sh.DX, y+sh.DY, w, h)
	} else if sh.Color.A > 0 {
		col := cgColorCreate(sh.Color.R, sh.Color.G, sh.Color.B, sh.Color.A)
		cgSaveGState(ctx)
		// the shadow offset lives in unflipped device pixels, so y goes negative
		cgSetShadow(ctx, cgSize{sh.DX * s.eff, -sh.DY * s.eff}, sh.Blur*s.eff, col)
		cgFillColor(ctx, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
		s.addPath(&rr)
		cgFillPath(ctx)
		cgRestoreGSt(ctx)
		cgColorRelase(col)
		return
	}
	cgFillColor(ctx, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
	s.addPath(&rr)
	cgFillPath(ctx)
}
