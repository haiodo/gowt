//go:build windows

package d2d

import (
	"math"
	"strconv"
	"strings"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

type matrix [6]float32

func toMatrix(m gfx.Affine) matrix {
	return matrix{float32(m[0]), float32(m[1]), float32(m[2]), float32(m[3]), float32(m[4]), float32(m[5])}
}

type gradStop struct {
	pos   float32
	color colorF
}

type layerParams1 struct {
	bounds    rectF
	mask      uintptr
	maskAA    uint32
	maskM     matrix
	opacity   float32
	opacityBr uintptr
	options   uint32
}

const (
	popAxis uint8 = iota + 1
	popLayer
)

type pop struct {
	kind uint8
	res  uintptr // geometry kept alive until the layer is popped
}

type frame struct {
	ctm   gfx.Affine
	pops  []pop
	layer bool
}

// replayer plays Pictures on one Target's device context. D2D has no save and restore of clips, so
// the frames record what each Save pushed and Restore pops it again.
type replayer struct {
	t     *Target
	r     *Renderer
	dc    uintptr
	it    gfx.Iter
	ctm   gfx.Affine
	set   matrix
	setOK bool
	eff   float64 // target scale times the scale of the item transform being drawn
	stack []frame
	col   gfx.Color
	colOK bool
	// openClips are the clips of the items drawn so far that the next item shares
	openClips []gfx.Clip
	openPops  []pop
	// hold keeps what owns the cached bitmaps and layouts of the batch reachable until EndDraw
	hold []any
}

func (rp *replayer) init(t *Target) {
	rp.t, rp.r, rp.dc = t, t.r, t.dc
}

func (rp *replayer) setColor(c gfx.Color) {
	if rp.colOK && rp.col == c {
		return
	}
	rp.col, rp.colOK = c, true
	vcall(rp.t.brush, d2SolidBrushSetColor, ref(&colorF{float32(c.R), float32(c.G), float32(c.B), float32(c.A)}))
}

func (rp *replayer) applyM() {
	m := toMatrix(rp.ctm)
	if rp.setOK && m == rp.set {
		return
	}
	rp.set, rp.setOK = m, true
	vcall(rp.dc, d2DCSetTransform, ref(&m))
}

func (rp *replayer) setM(m gfx.Affine) {
	rp.ctm = m
}

// draw paints the items in order: each under its clips, through its transform, at its alpha.
// Items that follow one another under the same clips share the clip layers.
func (rp *replayer) draw(items []gfx.Item) {
	rp.setOK, rp.colOK = false, false
	rp.openClips, rp.openPops = rp.openClips[:0], rp.openPops[:0]
	base := rp.t.base
	for i := range items {
		it := &items[i]
		p := 0
		for p < len(rp.openClips) && p < len(it.Clips) && sameClip(&rp.openClips[p], &it.Clips[p]) {
			p++
		}
		for len(rp.openClips) > p {
			rp.popOne(rp.openPops[len(rp.openPops)-1])
			rp.openPops = rp.openPops[:len(rp.openPops)-1]
			rp.openClips = rp.openClips[:len(rp.openClips)-1]
		}
		for j := p; j < len(it.Clips); j++ {
			rp.openPops = append(rp.openPops, rp.pushClip(&it.Clips[j], base.Mul(it.Clips[j].M)))
			rp.openClips = append(rp.openClips, it.Clips[j])
		}
		rp.ctm = base.Mul(it.M)
		rp.eff = rp.t.scale * it.M.Scale()
		var lp pop
		if it.Alpha < 1 {
			lp = rp.pushLayer(0, it.Alpha)
		}
		rp.replay(it.Pic)
		if it.Alpha < 1 {
			rp.popOne(lp)
		}
	}
	for len(rp.openPops) > 0 {
		rp.popOne(rp.openPops[len(rp.openPops)-1])
		rp.openPops = rp.openPops[:len(rp.openPops)-1]
	}
	rp.openClips = rp.openClips[:0]
	rp.eff = rp.t.scale
}

func sameClip(a, b *gfx.Clip) bool {
	return a.M == b.M && a.Kind == b.Kind && a.Rect == b.Rect && a.Radius == b.Radius && a.Path == b.Path
}

func (rp *replayer) popOne(p pop) {
	switch p.kind {
	case popAxis:
		vcall(rp.dc, d2DCPopAxisAlignedClip)
	case popLayer:
		vcall(rp.dc, d2DCPopLayer)
		release(p.res)
	}
}

// pushLayer composites everything up to the pop at alpha, under the geometry mask if there is one.
// The mask is in the space of the current transform.
func (rp *replayer) pushLayer(mask uintptr, alpha float64) pop {
	rp.applyM()
	lp := &layerParams1{
		bounds:  rectF{-math.MaxFloat32, -math.MaxFloat32, math.MaxFloat32, math.MaxFloat32},
		mask:    mask,
		maskM:   matrix{1, 0, 0, 1, 0, 0},
		opacity: float32(alpha),
	}
	vcall(rp.dc, d2DCPushLayer1, ref(lp), ui(0))
	return pop{popLayer, mask}
}

func axisAligned(m gfx.Affine) bool { return m[1] == 0 && m[2] == 0 }

// pushClip intersects the clip with c under ctm and returns what undoes it.
func (rp *replayer) pushClip(c *gfx.Clip, ctm gfx.Affine) pop {
	save := rp.ctm
	rp.ctm = ctm
	defer func() { rp.ctm = save }()
	switch c.Kind {
	case gfx.ClipRect:
		return rp.clipRect(c.Rect, ctm)
	case gfx.ClipRRect:
		var rr gfx.Path
		rr.RoundRect(c.Rect.Min.X, c.Rect.Min.Y, c.Rect.W(), c.Rect.H(), c.Radius)
		return rp.clipPath(&rr)
	case gfx.ClipPath:
		return rp.clipPath(c.Path)
	}
	return pop{}
}

func (rp *replayer) clipRect(r gfx.Rect, ctm gfx.Affine) pop {
	if axisAligned(ctm) {
		rp.applyM()
		vcall(rp.dc, d2DCPushAxisAlignedClip, ref(&rectF{float32(r.Min.X), float32(r.Min.Y), float32(r.Max.X), float32(r.Max.Y)}), ui(0))
		return pop{kind: popAxis}
	}
	var p gfx.Path
	p.Rect(r.Min.X, r.Min.Y, r.W(), r.H())
	return rp.clipPath(&p)
}

func (rp *replayer) clipPath(p *gfx.Path) pop {
	g, err := rp.r.geometry(p)
	if err != nil {
		return pop{}
	}
	return rp.pushLayer(g, 1)
}

// replay draws p at the current transform, with the level of detail for the current scale.
func (rp *replayer) replay(p *gfx.Picture) {
	if rp.r.opts.LOD {
		if w, h := p.Bounds.W()*rp.eff, p.Bounds.H()*rp.eff; w < 3 && h < 3 {
			rp.setColor(gfx.Color{R: 0.75, G: 0.75, B: 0.78, A: 1})
			rp.applyM()
			x, y := p.Bounds.Min.X, p.Bounds.Min.Y
			vcall(rp.dc, d2DCFillRect, ref(&rectF{float32(x), float32(y), float32(x + math.Max(p.Bounds.W(), 1/rp.eff)), float32(y + math.Max(p.Bounds.H(), 1/rp.eff))}), ui(rp.t.brush))
			return
		}
		if p.Lod != nil {
			// the substitute fades out over the band above LodBelow, so the look does not jump
			size := math.Max(p.Bounds.W(), p.Bounds.H()) * rp.eff
			switch {
			case size < p.LodBelow:
				p = p.Lod
			case size < p.LodBelow*lodFadeBand:
				rp.replayOps(p.Lod)
				lp := rp.pushLayer(0, (size-p.LodBelow)/(p.LodBelow*(lodFadeBand-1)))
				rp.replayOps(p)
				rp.popOne(lp)
				return
			}
		}
	}
	rp.replayOps(p)
}

// lodFadeBand is how far above LodBelow a Lod picture still shows through, as a factor of LodBelow.
const lodFadeBand = 1.3

func (rp *replayer) replayOps(p *gfx.Picture) {
	depth := len(rp.stack)
	rp.stack = append(rp.stack, frame{ctm: rp.ctm})
	layers := 0
	rp.it.Reset(p)
	for rp.it.Next() {
		op := rp.it.Op()
		switch op.Kind {
		case gfx.OpSave:
			rp.stack = append(rp.stack, frame{ctm: rp.ctm})
		case gfx.OpRestore:
			if len(rp.stack) > depth+1 && !rp.stack[len(rp.stack)-1].layer {
				rp.restore()
			}
		case gfx.OpTranslate:
			rp.ctm = rp.ctm.Mul(gfx.TranslateBy(op.A, op.B))
		case gfx.OpScale:
			rp.ctm = rp.ctm.Mul(gfx.ScaleBy(op.A, op.B))
		case gfx.OpRotate:
			rp.ctm = rp.ctm.Mul(gfx.RotateBy(op.A))
		case gfx.OpTransform:
			rp.ctm = rp.ctm.Mul(op.M)
		case gfx.OpFill:
			rp.fill(op.Path, op.Color)
		case gfx.OpFillGrad:
			rp.fillGrad(op.Path, op.Grad)
		case gfx.OpStroke:
			rp.stroke(op.Path, op.Style, op.Color, 0)
		case gfx.OpStrokeGrad:
			br := rp.gradBrush(op.Grad)
			if br != 0 {
				rp.stroke(op.Path, op.Style, gfx.Color{}, br)
				release(br)
			}
		case gfx.OpShadowBox:
			rp.shadowBox(op)
		case gfx.OpFillShadow:
			rp.shadowFill(op)
		case gfx.OpStrokeShadow:
			rp.shadowStroke(op)
		case gfx.OpText:
			rp.text(op.Text, op)
		case gfx.OpTextShadow:
			rp.shadowText(op)
		case gfx.OpImage:
			rp.image(op)
		case gfx.OpClipRect:
			rp.addPop(rp.clipRect(gfx.R(op.A, op.B, op.C, op.D), rp.ctm))
		case gfx.OpClipRRect:
			var rr gfx.Path
			rr.RoundRect(op.A, op.B, op.C, op.D, op.E)
			rp.addPop(rp.clipPath(&rr))
		case gfx.OpClipPath:
			rp.addPop(rp.clipPath(op.Path))
		case gfx.OpBeginLayer:
			rp.stack = append(rp.stack, frame{ctm: rp.ctm, layer: true})
			rp.addPop(rp.pushLayer(0, op.A))
			layers++
		case gfx.OpEndLayer:
			if layers > 0 {
				layers--
				for len(rp.stack) > depth+1 {
					l := rp.stack[len(rp.stack)-1].layer
					rp.restore()
					if l {
						break
					}
				}
			}
		}
	}
	for len(rp.stack) > depth {
		rp.restore()
	}
}

func (rp *replayer) addPop(p pop) {
	if p.kind != 0 {
		f := &rp.stack[len(rp.stack)-1]
		f.pops = append(f.pops, p)
	}
}

// restore pops the top frame: its clips come off in reverse and the transform returns to the saved one.
func (rp *replayer) restore() {
	f := rp.stack[len(rp.stack)-1]
	for i := len(f.pops) - 1; i >= 0; i-- {
		rp.popOne(f.pops[i])
	}
	rp.stack = rp.stack[:len(rp.stack)-1]
	rp.ctm = f.ctm
}

func (rp *replayer) fill(p *gfx.Path, c gfx.Color) {
	if rp.ctm.Det() == 0 {
		return
	}
	rp.setColor(c)
	rp.applyM()
	if r, ok := rectOf(p); ok {
		vcall(rp.dc, d2DCFillRect, ref(&r), ui(rp.t.brush))
		return
	}
	g, err := rp.r.geometry(p)
	if err != nil {
		return
	}
	vcall(rp.dc, d2DCFillGeometry, ui(g), ui(rp.t.brush), ui(0))
	release(g)
}

// rectOf recognizes the axis-aligned rectangle that gfx.Path.Rect writes.
func rectOf(p *gfx.Path) (rectF, bool) {
	if len(p.Verbs) != 5 || p.Verbs[0] != gfx.MoveTo || p.Verbs[4] != gfx.ClosePath {
		return rectF{}, false
	}
	for _, v := range p.Verbs[1:4] {
		if v != gfx.LineTo {
			return rectF{}, false
		}
	}
	a, b, c, d := p.Pts[0], p.Pts[1], p.Pts[2], p.Pts[3]
	if a.Y == b.Y && b.X == c.X && c.Y == d.Y && d.X == a.X && a.X < b.X && a.Y < d.Y {
		return rectF{float32(a.X), float32(a.Y), float32(c.X), float32(c.Y)}, true
	}
	return rectF{}, false
}

func (rp *replayer) fillGrad(p *gfx.Path, g *gfx.Gradient) {
	br := rp.gradBrush(g)
	if br == 0 {
		return
	}
	defer release(br)
	rp.applyM()
	geo, err := rp.r.geometry(p)
	if err != nil {
		return
	}
	vcall(rp.dc, d2DCFillGeometry, ui(geo), ui(br), ui(0))
	release(geo)
}

type linearProps struct{ start, end point2F }
type radialProps struct {
	center, offset point2F
	rx, ry         float32
}

// gradBrush returns a new gradient brush for g in the current user space, or 0.
func (rp *replayer) gradBrush(g *gfx.Gradient) uintptr {
	n := len(g.Stops)
	if n == 0 {
		return 0
	}
	stops := make([]gradStop, n)
	for i, s := range g.Stops {
		stops[i] = gradStop{float32(s.Offset), colorF{float32(s.Color.R), float32(s.Color.G), float32(s.Color.B), float32(s.Color.A)}}
	}
	var coll uintptr
	// gamma 2.2 interpolates the stored values as they are, as CoreGraphics does; extend mode clamp
	if hr := vcall(rp.dc, d2DCCreateGradStops, buf(stops), ui(uintptr(n)), ui(0), ui(0), ref(&coll)); failed(hr) {
		return 0
	}
	defer release(coll)
	var br uintptr
	if g.Kind == gfx.RadialGradient {
		pr := &radialProps{point2F{float32(g.P1.X), float32(g.P1.Y)}, point2F{float32(g.P0.X - g.P1.X), float32(g.P0.Y - g.P1.Y)}, float32(g.R1), float32(g.R1)}
		vcall(rp.dc, d2DCCreateRadialBrush, ref(pr), ui(0), ui(coll), ref(&br))
	} else {
		pr := &linearProps{point2F{float32(g.P0.X), float32(g.P0.Y)}, point2F{float32(g.P1.X), float32(g.P1.Y)}}
		vcall(rp.dc, d2DCCreateLinearBrush, ref(pr), ui(0), ui(coll), ref(&br))
	}
	return br
}

type strokeProps struct {
	startCap, endCap, dashCap, join uint32
	miter                           float32
	dashStyle                       uint32
	dashOffset                      float32
}

type strokeKey struct {
	cap, join uint8
	miter     float64
	dash      string
	phase     float64
	width     float64
}

// strokeStyle returns the cached ID2D1StrokeStyle for st at width w, with the dash on or off, and
// the share of the line that stays inked: a dash dropped for being unreadable is paid for by that
// share of the alpha.
func (rp *replayer) strokeStyle(st *gfx.StrokeStyle, w float64, dashed bool) (uintptr, float64) {
	cap := [...]uint32{0, 2, 1}[min(int(st.Cap), 2)] // butt, round, square -> FLAT, ROUND, SQUARE
	join := [...]uint32{3, 2, 1}[min(int(st.Join), 2)]
	miter := st.MiterLimit
	if miter < 1 {
		miter = 10
	}
	k := strokeKey{cap: uint8(cap), join: uint8(join), miter: miter}
	var dashes []float32
	if dashed {
		var sb strings.Builder
		for _, d := range st.Dash {
			sb.WriteString(strconv.FormatFloat(d, 'g', -1, 64))
			sb.WriteByte(',')
			dashes = append(dashes, float32(d/w))
		}
		k.dash, k.phase, k.width = sb.String(), st.DashPhase, w
	}
	if v, ok := rp.r.strokes.Load(k); ok {
		return v.(uintptr), 1
	}
	props := &strokeProps{startCap: cap, endCap: cap, dashCap: cap, join: join, miter: float32(miter)}
	if dashed {
		props.dashStyle = 5 // CUSTOM
		props.dashOffset = float32(st.DashPhase / w)
	}
	var s uintptr
	if hr := vcall(rp.r.factory, d2FactoryCreateStrokeStyle, ref(props), buf(dashes), ui(uintptr(len(dashes))), ref(&s)); failed(hr) {
		return 0, 1
	}
	if v, loaded := rp.r.strokes.LoadOrStore(k, s); loaded {
		release(s)
		s = v.(uintptr)
	}
	return s, 1
}

// dashDuty is the inked share of a dash pattern; an odd pattern repeats with on and off swapped.
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

// stroke draws the outline of p; br is a gradient brush, or 0 for the solid color c.
func (rp *replayer) stroke(p *gfx.Path, st *gfx.StrokeStyle, c gfx.Color, br uintptr) {
	if rp.ctm.Det() == 0 {
		return
	}
	w := st.Width
	if rp.r.opts.LOD && w*rp.eff < 1 {
		w = 1 / rp.eff // hairline floor: a thin line stays one device pixel wide instead of washing out
	}
	if w <= 0 {
		return
	}
	dashed := len(st.Dash) > 0 && !(rp.r.opts.LOD && rp.eff < 0.4)
	duty := 1.0
	if len(st.Dash) > 0 && !dashed {
		duty = dashDuty(st.Dash)
	}
	style, _ := rp.strokeStyle(st, w, dashed)
	if br == 0 {
		c.A *= duty
		rp.setColor(c)
		br = rp.t.brush
	}
	rp.applyM()
	g, err := rp.r.geometry(p)
	if err != nil {
		return
	}
	vcall(rp.dc, d2DCDrawGeometry, ui(g), ui(br), fl(w), ui(style))
	release(g)
}

type bezierSeg struct{ p1, p2, p3 point2F }
type quadSeg struct{ p1, p2 point2F }

// geometry builds an ID2D1PathGeometry from p; the caller releases it.
func (r *Renderer) geometry(p *gfx.Path) (uintptr, error) {
	var geo uintptr
	if hr := vcall(r.factory, d2FactoryCreatePathGeometry, ref(&geo)); failed(hr) {
		return 0, hrErr("CreatePathGeometry", hr)
	}
	var sink uintptr
	if hr := vcall(geo, d2GeomOpen, ref(&sink)); failed(hr) {
		release(geo)
		return 0, hrErr("Open", hr)
	}
	fill := uintptr(1) // WINDING
	if p.EvenOdd {
		fill = 0 // ALTERNATE
	}
	vcall(sink, d2SinkSetFillMode, ui(fill))
	pts := p.Pts
	var start, cur gfx.Point
	open, began := false, false
	end := func(closed bool) {
		if began {
			c := uintptr(0)
			if closed {
				c = 1
			}
			vcall(sink, d2SinkEndFigure, ui(c))
		}
		began = false
	}
	begin := func() {
		if !began {
			vcall(sink, d2SinkBeginFigure, pt2(cur.X, cur.Y), ui(0))
			began = true
		}
	}
	pt := func(q gfx.Point) point2F { return point2F{float32(q.X), float32(q.Y)} }
	for _, v := range p.Verbs {
		switch v {
		case gfx.MoveTo:
			end(false)
			start, cur, open = pts[0], pts[0], true
			pts = pts[1:]
		case gfx.LineTo:
			if !open {
				break
			}
			begin()
			vcall(sink, d2SinkAddLine, pt2(pts[0].X, pts[0].Y))
			cur = pts[0]
			pts = pts[1:]
		case gfx.QuadTo:
			if !open {
				break
			}
			begin()
			vcall(sink, d2SinkAddQuad, ref(&quadSeg{pt(pts[0]), pt(pts[1])}))
			cur = pts[1]
			pts = pts[2:]
		case gfx.CubicTo:
			if !open {
				break
			}
			begin()
			vcall(sink, d2SinkAddBezier, ref(&bezierSeg{pt(pts[0]), pt(pts[1]), pt(pts[2])}))
			cur = pts[2]
			pts = pts[3:]
		case gfx.ClosePath:
			end(true)
			cur = start // a segment after a close starts a new figure at the start point
		}
		if !open {
			// verbs before the first MoveTo have no start point: skip their points
			switch v {
			case gfx.LineTo:
				pts = pts[1:]
			case gfx.QuadTo:
				pts = pts[2:]
			case gfx.CubicTo:
				pts = pts[3:]
			}
		}
	}
	end(false)
	hr := vcall(sink, d2SinkClose)
	release(sink)
	if failed(hr) {
		release(geo)
		return 0, hrErr("Close", hr)
	}
	return geo, nil
}

func (rp *replayer) image(op *gfx.Op) {
	img := rp.r.bitmapOf(rp.dc, op.Image)
	if img == nil {
		return
	}
	rp.applyM()
	vcall(rp.dc, d2DCDrawBitmap, ui(img.bmp), ref(&rectF{float32(op.A), float32(op.B), float32(op.A + op.C), float32(op.B + op.D)}), fl(1), ui(1), ui(0))
	rp.hold = append(rp.hold, op.Image) // a Raster's bitmap is released with the Raster
}

// bitmapOf returns the bitmap behind img: a snapshot as it is, a Raster converted once.
func (r *Renderer) bitmapOf(dc uintptr, img gfx.Image) *Image {
	switch v := img.(type) {
	case *Image:
		return v
	case *gfx.Raster:
		if c, ok := r.images.Load(v); ok {
			return c
		}
		dst := make([]byte, v.W*v.H*4)
		for i := 0; i+3 < len(v.Pix); i += 4 {
			a := uint32(v.Pix[i+3])
			dst[i] = byte((uint32(v.Pix[i+2])*a + 127) / 255)
			dst[i+1] = byte((uint32(v.Pix[i+1])*a + 127) / 255)
			dst[i+2] = byte((uint32(v.Pix[i])*a + 127) / 255)
			dst[i+3] = byte(a)
		}
		bmp, err := newBitmap(dc, v.W, v.H, dst, v.W*4, 0)
		if err != nil {
			return nil
		}
		c := &Image{bmp, float64(v.W), float64(v.H)}
		actual, loaded := r.images.LoadOrStore(v, c, (*Image).Release)
		if loaded {
			c.Release()
		}
		return actual
	}
	return nil
}

// Under LOD, text smaller than skeletonHi device pixels turns into a rounded bar the size of the
// line, in skeletonInk of the text color: the block keeps its outline and its weight. Between
// skeletonLo and skeletonHi the glyphs and the bar cross-fade.
const (
	skeletonHi  = 6.0
	skeletonLo  = 4.0
	skeletonH   = 0.5  // bar height, in em
	skeletonMid = 0.3  // bar center above the baseline, in em
	skeletonInk = 0.45 // bar alpha against the text color
)

func (rp *replayer) skeleton(str string, op *gfx.Op, a float64) {
	m := rp.r.MeasureText(str, op.Font)
	h := skeletonH * op.Font.Size
	var bar gfx.Path
	bar.RoundRect(op.A, op.B-skeletonMid*op.Font.Size-h/2, m.Width, h, h/2)
	c := op.Color
	c.A *= skeletonInk * a
	rp.fill(&bar, c)
}

func (rp *replayer) text(str string, op *gfx.Op) {
	alpha := 1.0
	if px := op.Font.Size * rp.eff; rp.r.opts.LOD && px < skeletonHi {
		alpha = math.Max(0, (px-skeletonLo)/(skeletonHi-skeletonLo))
		rp.skeleton(str, op, 1-alpha)
		if alpha == 0 {
			return
		}
	}
	if str == "" || rp.ctm.Det() == 0 {
		return
	}
	lay := rp.r.text.layout(str, op.Font, rp.r.opts.TextCache)
	if lay == nil {
		return
	}
	if rp.r.opts.TextCache {
		rp.hold = append(rp.hold, lay)
	} else {
		defer release(lay.ptr)
	}
	c := op.Color
	c.A *= alpha
	rp.setColor(c)
	rp.applyM()
	// NO_SNAP keeps the glyph origin where the picture put it instead of rounding it to a pixel
	vcall(rp.dc, d2DCDrawTextLayout, pt2(op.A, op.B-lay.Ascent), ui(lay.ptr), ui(rp.t.brush), ui(1))
}

var _ = unsafe.Pointer(nil)
