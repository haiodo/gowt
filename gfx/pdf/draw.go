package pdf

import (
	"bytes"
	"fmt"
	"math"

	"github.com/haiodo/gowt/gfx"
)

// page holds the content of one page; Draw may be called several times.
type page struct {
	d    *Document
	w, h float64
	buf  bytes.Buffer
}

var _ gfx.Page = (*page)(nil)

// Draw paints the items; see gfx.Page.
func (p *page) Draw(scale float64, origin gfx.Point, items []gfx.Item) {
	rp := replayer{d: p.d, out: &p.buf, ctm: gfx.Identity()}
	rp.save()
	rp.cm(gfx.Affine{1, 0, 0, -1, 0, p.h})
	rp.cm(gfx.Affine{scale, 0, 0, scale, -scale * origin.X, -scale * origin.Y})
	for i := range items {
		rp.item(&items[i])
	}
	rp.restore()
}

type layer struct {
	out   *bytes.Buffer
	alpha float64
	ctm   gfx.Affine
	stack []gfx.Affine
}

type replayer struct {
	d      *Document
	out    *bytes.Buffer
	ctm    gfx.Affine
	stack  []gfx.Affine
	layers []layer
	it     gfx.Iter
}

func (rp *replayer) printf(f string, a ...any) { fmt.Fprintf(rp.out, f, a...) }

func (rp *replayer) save() {
	rp.out.WriteString("q\n")
	rp.stack = append(rp.stack, rp.ctm)
}

func (rp *replayer) restore() {
	rp.out.WriteString("Q\n")
	if n := len(rp.stack); n > 0 {
		rp.ctm = rp.stack[n-1]
		rp.stack = rp.stack[:n-1]
	}
}

func (rp *replayer) cm(m gfx.Affine) {
	rp.printf("%s %s %s %s %s %s cm\n", num(m[0]), num(m[1]), num(m[2]), num(m[3]), num(m[4]), num(m[5]))
	rp.ctm = rp.ctm.Mul(m)
}

func (rp *replayer) item(it *gfx.Item) {
	for i := range it.Clips {
		c := &it.Clips[i]
		rp.save()
		rp.cm(c.M)
		switch c.Kind {
		case gfx.ClipRect:
			rp.rect(c.Rect.Min.X, c.Rect.Min.Y, c.Rect.W(), c.Rect.H())
			rp.out.WriteString("W n\n")
		case gfx.ClipRRect:
			var rr gfx.Path
			rr.RoundRect(c.Rect.Min.X, c.Rect.Min.Y, c.Rect.W(), c.Rect.H(), c.Radius)
			rp.path(&rr)
			rp.out.WriteString("W n\n")
		case gfx.ClipPath:
			rp.path(c.Path)
			rp.clipOp(c.Path)
		}
		if inv, ok := c.M.Invert(); ok {
			rp.cm(inv)
		}
	}
	rp.save()
	rp.cm(it.M)
	if it.Alpha < 1 {
		rp.beginLayer(it.Alpha)
	}
	rp.replay(it.Pic)
	if it.Alpha < 1 {
		rp.endLayer()
	}
	rp.restore()
	for range it.Clips {
		rp.restore()
	}
}

func (rp *replayer) clipOp(p *gfx.Path) {
	if p.EvenOdd {
		rp.out.WriteString("W* n\n")
	} else {
		rp.out.WriteString("W n\n")
	}
}

func (rp *replayer) rect(x, y, w, h float64) {
	rp.printf("%s %s %s %s re\n", num(x), num(y), num(w), num(h))
}

func (rp *replayer) path(p *gfx.Path) {
	var cx, cy, sx, sy float64
	pts := p.Pts
	for _, v := range p.Verbs {
		switch v {
		case gfx.MoveTo:
			cx, cy = pts[0].X, pts[0].Y
			sx, sy = cx, cy
			rp.printf("%s %s m\n", num(cx), num(cy))
			pts = pts[1:]
		case gfx.LineTo:
			cx, cy = pts[0].X, pts[0].Y
			rp.printf("%s %s l\n", num(cx), num(cy))
			pts = pts[1:]
		case gfx.QuadTo:
			x1, y1, x2, y2 := pts[0].X, pts[0].Y, pts[1].X, pts[1].Y
			rp.printf("%s %s %s %s %s %s c\n", num(cx+2.0/3*(x1-cx)), num(cy+2.0/3*(y1-cy)),
				num(x2+2.0/3*(x1-x2)), num(y2+2.0/3*(y1-y2)), num(x2), num(y2))
			cx, cy = x2, y2
			pts = pts[2:]
		case gfx.CubicTo:
			rp.printf("%s %s %s %s %s %s c\n", num(pts[0].X), num(pts[0].Y), num(pts[1].X), num(pts[1].Y), num(pts[2].X), num(pts[2].Y))
			cx, cy = pts[2].X, pts[2].Y
			pts = pts[3:]
		case gfx.ClosePath:
			rp.out.WriteString("h\n")
			cx, cy = sx, sy
		}
	}
}

func (rp *replayer) beginLayer(alpha float64) {
	rp.layers = append(rp.layers, layer{out: rp.out, alpha: alpha, ctm: rp.ctm, stack: rp.stack})
	rp.out = &bytes.Buffer{}
	rp.ctm, rp.stack = gfx.Identity(), nil
}

func (rp *replayer) endLayer() {
	n := len(rp.layers) - 1
	l := rp.layers[n]
	rp.layers = rp.layers[:n]
	body := rp.out.Bytes()
	rp.out, rp.ctm, rp.stack = l.out, l.ctm, l.stack
	const big = 1000000
	form := rp.d.stream(fmt.Sprintf("/Type /XObject /Subtype /Form /BBox [-%[1]v -%[1]v %[1]v %[1]v] /Group << /S /Transparency /CS /DeviceRGB >> /Resources %d 0 R", big, resObj), body, true)
	rp.printf("q /%s gs /%s Do Q\n", rp.d.res.alpha(l.alpha), rp.d.res.xobject(form, "Fm"))
}

// replay draws p inside its own save level; the unbalanced saves and layers of the picture are closed.
func (rp *replayer) replay(p *gfx.Picture) {
	depth := 1 // save levels of this picture inside the current layer, the first one included
	var outer []int
	rp.save()
	rp.it.Reset(p)
	for rp.it.Next() {
		op := rp.it.Op()
		switch op.Kind {
		case gfx.OpSave:
			rp.save()
			depth++
		case gfx.OpRestore:
			if depth > 1 {
				rp.restore()
				depth--
			}
		case gfx.OpTranslate:
			rp.cm(gfx.TranslateBy(op.A, op.B))
		case gfx.OpScale:
			rp.cm(gfx.ScaleBy(op.A, op.B))
		case gfx.OpRotate:
			rp.cm(gfx.RotateBy(op.A))
		case gfx.OpTransform:
			rp.cm(op.M)
		case gfx.OpFill:
			rp.fill(op.Path, op.Color)
		case gfx.OpFillGrad:
			rp.fillGrad(op.Path, op.Grad)
		case gfx.OpStroke:
			rp.stroke(op.Path, op.Style, op.Color)
		case gfx.OpStrokeGrad:
			rp.strokeGrad(op.Path, op.Style, op.Grad)
		case gfx.OpShadowBox:
			var rr gfx.Path
			rr.RoundRect(op.A, op.B, op.C, op.D, op.E)
			rp.shadowFill(&rr, op.Shadow)
			rp.fill(&rr, op.Color)
		case gfx.OpFillShadow:
			rp.shadowFill(op.Path, op.Shadow)
			rp.fill(op.Path, op.Color)
		case gfx.OpStrokeShadow:
			rp.shadowStroke(op.Path, op.Style, op.Shadow)
			rp.stroke(op.Path, op.Style, op.Color)
		case gfx.OpText:
			rp.text(op, false)
		case gfx.OpTextShadow:
			rp.text(op, true)
		case gfx.OpImage:
			rp.image(op)
		case gfx.OpClipRect:
			rp.rect(op.A, op.B, op.C, op.D)
			rp.out.WriteString("W n\n")
		case gfx.OpClipRRect:
			var rr gfx.Path
			rr.RoundRect(op.A, op.B, op.C, op.D, op.E)
			rp.path(&rr)
			rp.out.WriteString("W n\n")
		case gfx.OpClipPath:
			rp.path(op.Path)
			rp.clipOp(op.Path)
		case gfx.OpBeginLayer:
			outer = append(outer, depth)
			rp.beginLayer(op.A)
			rp.save()
			depth = 1
		case gfx.OpEndLayer:
			if len(outer) > 0 {
				rp.closeLayer(depth)
				depth = outer[len(outer)-1]
				outer = outer[:len(outer)-1]
			}
		}
	}
	for len(outer) > 0 {
		rp.closeLayer(depth)
		depth = outer[len(outer)-1]
		outer = outer[:len(outer)-1]
	}
	for ; depth > 0; depth-- {
		rp.restore()
	}
}

// closeLayer ends the layer begun by OpBeginLayer, with depth save levels open inside it.
func (rp *replayer) closeLayer(depth int) {
	for ; depth > 0; depth-- {
		rp.restore()
	}
	rp.endLayer()
}

// color sets the fill (or stroke) color and, for a translucent one, the alpha through a state
// dictionary; the caller wraps the paint in save and restore.
func (rp *replayer) color(c gfx.Color, stroke bool) {
	op := "rg"
	if stroke {
		op = "RG"
	}
	rp.printf("%s %s %s %s\n", num(c.R), num(c.G), num(c.B), op)
	if c.A < 1 {
		rp.printf("/%s gs\n", rp.d.res.alpha(c.A))
	}
}

func (rp *replayer) fill(p *gfx.Path, c gfx.Color) {
	if c.A <= 0 || p.Empty() {
		return
	}
	rp.save()
	rp.color(c, false)
	rp.path(p)
	rp.fillOp(p)
	rp.restore()
}

func (rp *replayer) fillOp(p *gfx.Path) {
	if p.EvenOdd {
		rp.out.WriteString("f*\n")
	} else {
		rp.out.WriteString("f\n")
	}
}

func (rp *replayer) strokeStyle(st *gfx.StrokeStyle, width float64) {
	rp.printf("%s w %d J %d j\n", num(width), st.Cap, st.Join)
	if st.MiterLimit > 0 {
		rp.printf("%s M\n", num(st.MiterLimit))
	}
	if len(st.Dash) > 0 {
		rp.out.WriteString("[")
		for i, v := range st.Dash {
			if i > 0 {
				rp.out.WriteString(" ")
			}
			rp.out.WriteString(num(v))
		}
		rp.printf("] %s d\n", num(st.DashPhase))
	}
}

func (rp *replayer) stroke(p *gfx.Path, st *gfx.StrokeStyle, c gfx.Color) {
	if c.A <= 0 || p.Empty() {
		return
	}
	rp.save()
	rp.color(c, true)
	rp.strokeStyle(st, st.Width)
	rp.path(p)
	rp.out.WriteString("S\n")
	rp.restore()
}

func (rp *replayer) image(op *gfx.Op) {
	r, ok := op.Image.(*gfx.Raster)
	if !ok || r.W <= 0 || r.H <= 0 || op.C <= 0 || op.D <= 0 {
		return
	}
	rp.save()
	rp.cm(gfx.Affine{op.C, 0, 0, -op.D, op.A, op.B + op.D})
	rp.printf("/%s Do\n", rp.d.res.image(rp.d, r))
	rp.restore()
}

// shadowSteps stack translucent strokes of growing width under a shape: the edge fades out over
// the blur radius, which CoreGraphics takes as about twice the standard deviation.
const shadowSteps = 8

func shadowAlpha(a float64) float64 { return 1 - math.Pow(1-a, 1.0/shadowSteps) }

func (rp *replayer) shadowFill(p *gfx.Path, sh gfx.Shadow) {
	if sh.Color.A <= 0 || p.Empty() {
		return
	}
	rp.save()
	rp.cm(gfx.TranslateBy(sh.DX, sh.DY))
	if sh.Blur > 0 {
		step := gfx.Shadow{Color: sh.Color}
		step.Color.A = shadowAlpha(sh.Color.A)
		for j := 1; j <= shadowSteps; j++ {
			rp.stroke(p, &gfx.StrokeStyle{Width: 2 * sh.Blur * float64(j) / shadowSteps, Cap: gfx.CapRound, Join: gfx.JoinRound}, step.Color)
		}
	}
	rp.fill(p, sh.Color)
	rp.restore()
}

func (rp *replayer) shadowStroke(p *gfx.Path, st *gfx.StrokeStyle, sh gfx.Shadow) {
	if sh.Color.A <= 0 || p.Empty() {
		return
	}
	rp.save()
	rp.cm(gfx.TranslateBy(sh.DX, sh.DY))
	if sh.Blur > 0 {
		c := sh.Color
		c.A = shadowAlpha(sh.Color.A)
		for j := 1; j <= shadowSteps; j++ {
			s := *st
			s.Width = st.Width + 2*sh.Blur*float64(j)/shadowSteps
			rp.stroke(p, &s, c)
		}
	}
	rp.stroke(p, st, sh.Color)
	rp.restore()
}

func (rp *replayer) fillGrad(p *gfx.Path, g *gfx.Gradient) {
	pat, alpha, ok := rp.d.res.pattern(rp.d, g, rp.ctm)
	if !ok || p.Empty() {
		return
	}
	rp.save()
	if alpha < 1 {
		rp.printf("/%s gs\n", rp.d.res.alpha(alpha))
	}
	rp.printf("/Pattern cs /%s scn\n", pat)
	rp.path(p)
	rp.fillOp(p)
	rp.restore()
}

func (rp *replayer) strokeGrad(p *gfx.Path, st *gfx.StrokeStyle, g *gfx.Gradient) {
	pat, alpha, ok := rp.d.res.pattern(rp.d, g, rp.ctm)
	if !ok || p.Empty() {
		return
	}
	rp.save()
	if alpha < 1 {
		rp.printf("/%s gs\n", rp.d.res.alpha(alpha))
	}
	rp.printf("/Pattern CS /%s SCN\n", pat)
	rp.strokeStyle(st, st.Width)
	rp.path(p)
	rp.out.WriteString("S\n")
	rp.restore()
}
