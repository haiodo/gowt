//go:build linux

package cairo

import (
	"math"
	"os"
	"sync"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

var _ gfx.Renderer = (*Renderer)(nil)

// Options select the variants of the renderer.
type Options struct {
	LOD bool // skip text, shadow and dashes that are unreadable at the current scale
	// FontDir is registered with RegisterFonts before the first text is laid out; the environment
	// variable GFX_FONT_DIR stands in when it is empty.
	FontDir string
}

// Renderer holds the font map and the caches shared by all Targets; it is safe for concurrent use.
// Pango calls are serialized, drawing into separate Targets is not.
type Renderer struct {
	opts    Options
	fm      uintptr // PangoFontMap
	fopts   uintptr // cairo_font_options_t
	mctx    uintptr // context for measuring
	pmu     sync.Mutex
	descs   sync.Map // fontKey -> uintptr
	metrics sync.Map // lineKey -> gfx.TextMetrics
	images  sync.Map // *gfx.Raster -> *Image
}

// The measurer of recordings made before any renderer exists (see gfx.DefaultMeasurer).
func init() {
	gfx.DefaultMeasurer = gfx.LazyMeasurer(func() (gfx.TextMeasurer, error) { return NewRenderer(Options{}) })
}

// NewRenderer loads cairo and Pango. It fails when the libraries are missing; it needs no display.
func NewRenderer(o Options) (*Renderer, error) {
	if err := bind(); err != nil {
		return nil, err
	}
	if o.FontDir == "" {
		o.FontDir = os.Getenv("GFX_FONT_DIR")
	}
	if o.FontDir != "" {
		_, _ = RegisterFonts(o.FontDir) // a missing directory leaves the system fonts
	}
	r := &Renderer{opts: o, fm: fmNew(), fopts: fontOptionsCreate()}
	fontOptionsSetAA(r.fopts, antialiasGray)
	fontOptionsSetHint(r.fopts, hintStyleNone)
	fontOptionsSetMetrics(r.fopts, hintMetricsOff)
	fontsMu.Lock()
	fontMaps[r.fm] = struct{}{}
	fontsMu.Unlock()
	r.mctx = r.newContext()
	return r, nil
}

// Image is a cairo image surface snapshot of a Target. Size is in points.
type Image struct {
	surf     uintptr
	w, h     float64
	wpx, hpx int
}

// Size returns the size of the image in points.
func (i *Image) Size() (float64, float64) { return i.w, i.h }

// Release drops the surface reference of the image.
func (i *Image) Release() {
	if i.surf != 0 {
		surfaceDestroy(i.surf)
		i.surf = 0
	}
}

// Target is an offscreen premultiplied BGRA bitmap (cairo ARGB32 on a little-endian CPU).
type Target struct {
	r      *Renderer
	surf   uintptr
	cr     uintptr
	w, h   int
	data   uintptr
	stride int
	scale  float64
	eff    float64 // scale times the scale of the item transform being drawn
	dashed bool
	it     gfx.Iter

	layerAlpha []float64
	screen     float64 // device pixels per screen point of a Shadow.Screen; 0 is 1
}

// SetScreenScale sets the device pixels one screen point of a gfx.Shadow with Screen is.
func (t *Target) SetScreenScale(v float64) { t.screen = v }

func (t *Target) screenScale() float64 {
	if t.screen > 0 {
		return t.screen
	}
	return 1
}

// NewTarget makes an offscreen bitmap of wpx x hpx device pixels for gfx.
func (r *Renderer) NewTarget(wpx, hpx int) gfx.Target {
	surf := imageSurfaceCreate(formatARGB32, int32(wpx), int32(hpx))
	cr := create(surf)
	t := &Target{r: r, surf: surf, cr: cr, w: wpx, h: hpx, scale: 1, eff: 1}
	t.data = imageSurfaceGetData(surf)
	t.stride = int(imageSurfaceGetStride(surf))
	save(cr)
	return t
}

// Close frees the surface.
func (t *Target) Close() {
	if t.cr != 0 {
		destroy(t.cr)
		surfaceDestroy(t.surf)
		t.cr, t.surf = 0, 0
	}
}

// Size returns the size in pixels.
func (t *Target) Size() (int, int) { return t.w, t.h }

// Pixels returns the bitmap memory (BGRA, premultiplied, rows top to bottom, 4*W bytes each); it is
// valid until Close.
func (t *Target) Pixels() []byte {
	surfaceFlush(t.surf)
	return unsafe.Slice((*byte)(ptr(t.data)), t.stride*t.h)
}

// Begin clears the target to bg and maps the scene point origin to the top left pixel at scale.
func (t *Target) Begin(scale float64, origin gfx.Point, bg gfx.Color) {
	cr := t.cr
	restore(cr)
	save(cr)
	if bg.A == 0 {
		setOperator(cr, operatorClear)
	} else {
		setOperator(cr, operatorSource)
		setSourceRGBA(cr, bg.R, bg.G, bg.B, bg.A)
	}
	paint(cr)
	setOperator(cr, operatorOver)
	scaleBy(cr, scale)
	translate(cr, -origin.X, -origin.Y)
	t.scale, t.eff, t.dashed = scale, scale, false
}

func scaleBy(cr uintptr, s float64) { scale(cr, s, s) }

// Snapshot copies the current pixels into an Image of the given size in points.
func (t *Target) Snapshot(wpt, hpt float64) gfx.Image {
	surfaceFlush(t.surf)
	s := imageSurfaceCreate(formatARGB32, int32(t.w), int32(t.h))
	surfaceFlush(s)
	dst := unsafe.Slice((*byte)(ptr(imageSurfaceGetData(s))), int(imageSurfaceGetStride(s))*t.h)
	copy(dst, unsafe.Slice((*byte)(ptr(t.data)), t.stride*t.h))
	surfaceMarkDirty(s)
	return &Image{surf: s, w: wpt, h: hpt, wpx: t.w, hpx: t.h}
}

func isTranslation(m gfx.Affine) bool { return m[0] == 1 && m[1] == 0 && m[2] == 0 && m[3] == 1 }

func (t *Target) concat(m gfx.Affine) {
	if isTranslation(m) {
		if m[4] != 0 || m[5] != 0 {
			translate(t.cr, m[4], m[5])
		}
		return
	}
	cm := matrix{m[0], m[1], m[2], m[3], m[4], m[5]}
	transform(t.cr, &cm)
}

// Draw paints the items in order: each under its clips, through its transform, at its alpha. Items
// that follow one another under the same clips share the clip state.
func (t *Target) Draw(items []gfx.Item) {
	cr := t.cr
	var stack []gfx.Clip
	for i := range items {
		it := &items[i]
		p := 0
		for p < len(stack) && p < len(it.Clips) && sameClip(&stack[p], &it.Clips[p]) {
			p++
		}
		for len(stack) > p {
			restore(cr)
			stack = stack[:len(stack)-1]
		}
		for j := p; j < len(it.Clips); j++ {
			save(cr)
			t.clip(&it.Clips[j])
			stack = append(stack, it.Clips[j])
		}
		save(cr)
		t.concat(it.M)
		t.eff = t.scale * it.M.Scale()
		if it.Alpha < 1 {
			pushGroup(cr)
		}
		t.replay(it.Pic)
		if it.Alpha < 1 {
			popGroupToSource(cr)
			paintWithAlpha(cr, it.Alpha)
		}
		restore(cr)
	}
	for range stack {
		restore(cr)
	}
	t.eff = t.scale
}

func sameClip(a, b *gfx.Clip) bool {
	return a.M == b.M && a.Kind == b.Kind && a.Rect == b.Rect && a.Radius == b.Radius && a.Path == b.Path
}

func (t *Target) clip(c *gfx.Clip) {
	cr := t.cr
	var saved matrix
	getMatrix(cr, &saved)
	t.concat(c.M)
	switch c.Kind {
	case gfx.ClipRect:
		rectangle(cr, c.Rect.Min.X, c.Rect.Min.Y, c.Rect.W(), c.Rect.H())
		setFillRule(cr, fillRuleWinding)
		clip(cr)
	case gfx.ClipRRect:
		var rr gfx.Path
		rr.RoundRect(c.Rect.Min.X, c.Rect.Min.Y, c.Rect.W(), c.Rect.H(), c.Radius)
		t.addPath(&rr)
		setFillRule(cr, fillRuleWinding)
		clip(cr)
	case gfx.ClipPath:
		t.addPath(c.Path)
		t.setRule(c.Path)
		clip(cr)
	}
	setMatrix(cr, &saved)
}

func (t *Target) setRule(p *gfx.Path) {
	if p.EvenOdd {
		setFillRule(t.cr, fillRuleEvenOdd)
	} else {
		setFillRule(t.cr, fillRuleWinding)
	}
}

// lodFadeBand is how far above LodBelow a Lod picture still shows through, as a factor of LodBelow.
const lodFadeBand = 1.3

func (t *Target) replay(p *gfx.Picture) {
	cr := t.cr
	if t.r.opts.LOD {
		if w, h := p.Bounds.W()*t.eff, p.Bounds.H()*t.eff; w < 3 && h < 3 {
			setSourceRGBA(cr, 0.75, 0.75, 0.78, 1)
			rectangle(cr, p.Bounds.Min.X, p.Bounds.Min.Y, math.Max(p.Bounds.W(), 1/t.eff), math.Max(p.Bounds.H(), 1/t.eff))
			setFillRule(cr, fillRuleWinding)
			fill(cr)
			return
		}
		if p.Lod != nil {
			size := math.Max(p.Bounds.W(), p.Bounds.H()) * t.eff
			switch {
			case size < p.LodBelow:
				p = p.Lod
			case size < p.LodBelow*lodFadeBand:
				t.replayOps(p.Lod)
				save(cr)
				pushGroup(cr)
				t.replayOps(p)
				popGroupToSource(cr)
				paintWithAlpha(cr, (size-p.LodBelow)/(p.LodBelow*(lodFadeBand-1)))
				restore(cr)
				return
			}
		}
	}
	t.replayOps(p)
}

func (t *Target) replayOps(p *gfx.Picture) {
	cr := t.cr
	save(cr)
	t.dashed = false
	layers := 0
	t.it.Reset(p)
	for t.it.Next() {
		op := t.it.Op()
		switch op.Kind {
		case gfx.OpSave:
			save(cr)
		case gfx.OpRestore:
			restore(cr)
		case gfx.OpTranslate:
			translate(cr, op.A, op.B)
		case gfx.OpScale:
			scale(cr, op.A, op.B)
		case gfx.OpRotate:
			rotate(cr, op.A)
		case gfx.OpTransform:
			t.concat(op.M)
		case gfx.OpFill:
			setSourceRGBA(cr, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
			t.addPath(op.Path)
			t.setRule(op.Path)
			fill(cr)
		case gfx.OpFillGrad:
			save(cr)
			t.addPath(op.Path)
			t.setRule(op.Path)
			t.useGradient(op.Grad)
			fill(cr)
			restore(cr)
		case gfx.OpStroke:
			duty := t.strokeStyle(op.Style)
			setSourceRGBA(cr, op.Color.R, op.Color.G, op.Color.B, op.Color.A*duty)
			t.addPath(op.Path)
			stroke(cr)
		case gfx.OpStrokeGrad:
			save(cr)
			t.strokeStyle(op.Style)
			t.addPath(op.Path)
			t.useGradient(op.Grad)
			stroke(cr)
			restore(cr)
			t.dashed = false
		case gfx.OpShadowBox:
			t.shadowBox(op)
		case gfx.OpFillShadow:
			t.shadowPath(op, false)
		case gfx.OpStrokeShadow:
			t.shadowPath(op, true)
		case gfx.OpText:
			t.text(op.Text, op)
		case gfx.OpTextShadow:
			t.shadowText(op)
		case gfx.OpImage:
			t.image(op)
		case gfx.OpClipRect:
			rectangle(cr, op.A, op.B, op.C, op.D)
			setFillRule(cr, fillRuleWinding)
			clip(cr)
		case gfx.OpClipRRect:
			var rr gfx.Path
			rr.RoundRect(op.A, op.B, op.C, op.D, op.E)
			t.addPath(&rr)
			setFillRule(cr, fillRuleWinding)
			clip(cr)
		case gfx.OpClipPath:
			t.addPath(op.Path)
			t.setRule(op.Path)
			clip(cr)
		case gfx.OpBeginLayer:
			save(cr)
			pushGroup(cr)
			t.layerAlpha = append(t.layerAlpha, op.A)
			layers++
		case gfx.OpEndLayer:
			if layers > 0 {
				layers--
				t.endLayer()
			}
		}
	}
	for ; layers > 0; layers-- {
		t.endLayer()
	}
	restore(cr)
}

func (t *Target) endLayer() {
	n := len(t.layerAlpha) - 1
	a := t.layerAlpha[n]
	t.layerAlpha = t.layerAlpha[:n]
	popGroupToSource(t.cr)
	paintWithAlpha(t.cr, a)
	restore(t.cr)
}

// useGradient makes g the source, extended past both ends.
func (t *Target) useGradient(g *gfx.Gradient) {
	if len(g.Stops) == 0 {
		setSourceRGBA(t.cr, 0, 0, 0, 0)
		return
	}
	var pat uintptr
	if g.Kind == gfx.RadialGradient {
		pat = patternCreateRadial(g.P0.X, g.P0.Y, g.R0, g.P1.X, g.P1.Y, g.R1)
	} else {
		pat = patternCreateLinear(g.P0.X, g.P0.Y, g.P1.X, g.P1.Y)
	}
	for _, st := range g.Stops {
		patternAddColorStop(pat, st.Offset, st.Color.R, st.Color.G, st.Color.B, st.Color.A)
	}
	patternSetExtend(pat, extendPad)
	setSource(t.cr, pat)
	patternDestroy(pat) // the context keeps its own reference
}

func (t *Target) addPath(p *gfx.Path) {
	cr := t.cr
	newPath(cr)
	var cx, cy, sx, sy float64
	pts := p.Pts
	for _, v := range p.Verbs {
		switch v {
		case gfx.MoveTo:
			cx, cy = pts[0].X, pts[0].Y
			sx, sy = cx, cy
			moveTo(cr, cx, cy)
			pts = pts[1:]
		case gfx.LineTo:
			cx, cy = pts[0].X, pts[0].Y
			lineTo(cr, cx, cy)
			pts = pts[1:]
		case gfx.QuadTo:
			// cairo has cubics only: raise the degree
			x1, y1, x2, y2 := pts[0].X, pts[0].Y, pts[1].X, pts[1].Y
			curveTo(cr, cx+2.0/3*(x1-cx), cy+2.0/3*(y1-cy), x2+2.0/3*(x1-x2), y2+2.0/3*(y1-y2), x2, y2)
			cx, cy = x2, y2
			pts = pts[2:]
		case gfx.CubicTo:
			curveTo(cr, pts[0].X, pts[0].Y, pts[1].X, pts[1].Y, pts[2].X, pts[2].Y)
			cx, cy = pts[2].X, pts[2].Y
			pts = pts[3:]
		case gfx.ClosePath:
			closePath(cr)
			cx, cy = sx, sy
		}
	}
}

// strokeStyle sets the stroke state and returns the share of the line that stays inked: a dash
// dropped for being unreadable is paid for by that share of the alpha.
func (t *Target) strokeStyle(st *gfx.StrokeStyle) float64 {
	cr := t.cr
	w := st.Width
	if t.r.opts.LOD && w*t.eff < 1 {
		w = 1 / t.eff // hairline floor: a thin line stays one device pixel wide
	}
	setLineWidth(cr, w)
	setLineCap(cr, int32(st.Cap))
	setLineJoin(cr, int32(st.Join))
	if st.MiterLimit > 0 {
		setMiterLimit(cr, st.MiterLimit)
	}
	duty := 1.0
	switch {
	case len(st.Dash) > 0 && !(t.r.opts.LOD && t.eff < 0.4):
		setDash(cr, &st.Dash[0], int32(len(st.Dash)), st.DashPhase)
		t.dashed = true
	case t.dashed:
		setDash(cr, nil, 0, 0)
		t.dashed = false
	}
	if len(st.Dash) > 0 && !t.dashed {
		duty = dashDuty(st.Dash)
	}
	return duty
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

// image draws a Raster or a Snapshot into the rectangle A, B, C, D.
func (t *Target) image(op *gfx.Op) {
	img := t.r.surfaceOf(op.Image)
	if img == nil || op.C <= 0 || op.D <= 0 {
		return
	}
	cr := t.cr
	save(cr)
	rectangle(cr, op.A, op.B, op.C, op.D)
	setFillRule(cr, fillRuleWinding)
	clip(cr)
	translate(cr, op.A, op.B)
	scale(cr, op.C/float64(img.wpx), op.D/float64(img.hpx))
	setSourceSurface(cr, img.surf, 0, 0)
	patternSetExtend(getSource(cr), extendPad) // bilinear sampling must not fade the outer pixels
	patternSetFilter(getSource(cr), filterGood)
	paint(cr)
	restore(cr)
}

func (r *Renderer) surfaceOf(img gfx.Image) *Image {
	switch v := img.(type) {
	case *Image:
		return v
	case *gfx.Raster:
		if c, ok := r.images.Load(v); ok {
			return c.(*Image)
		}
		if v.W <= 0 || v.H <= 0 {
			return nil
		}
		s := imageSurfaceCreate(formatARGB32, int32(v.W), int32(v.H))
		stride := int(imageSurfaceGetStride(s))
		dst := unsafe.Slice((*byte)(ptr(imageSurfaceGetData(s))), stride*v.H)
		for y := 0; y < v.H; y++ {
			row := dst[y*stride : y*stride+v.W*4]
			src := v.Pix[y*v.W*4 : (y+1)*v.W*4]
			for i := 0; i+3 < len(src); i += 4 {
				a := uint32(src[i+3])
				row[i] = byte((uint32(src[i+2])*a + 127) / 255)
				row[i+1] = byte((uint32(src[i+1])*a + 127) / 255)
				row[i+2] = byte((uint32(src[i])*a + 127) / 255)
				row[i+3] = byte(a)
			}
		}
		surfaceMarkDirty(s)
		c := &Image{surf: s, w: float64(v.W), h: float64(v.H), wpx: v.W, hpx: v.H}
		actual, loaded := r.images.LoadOrStore(v, c)
		if loaded {
			c.Release()
		}
		return actual.(*Image)
	}
	return nil
}
