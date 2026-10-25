//go:build darwin

package cg

import (
	"math"
	"os"
	"sync"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

// kCGImageAlphaPremultipliedFirst | kCGBitmapByteOrder32Little: BGRA bytes in memory,
// the layout CALayer and most GPUs take without a conversion.
const bitmapInfoBGRA = 2 | 2<<12

// Options select the variants the benchmarks compare.
type Options struct {
	TextCache    bool // keep CTLine objects per (font, string)
	ShadowSprite bool // draw box shadows from a cached 9-slice sprite instead of a live blur
	LOD          bool // skip text, shadow and dashes that are unreadable at the current scale
	// FontDir is registered with RegisterFonts for this process before the first text is laid out;
	// the environment variable GFX_FONT_DIR stands in when it is empty.
	FontDir string
}

// Renderer holds the caches shared by all Surfaces; it is safe for concurrent use.
type Renderer struct {
	opts    Options
	cs      uintptr
	fonts   sync.Map // fontKey -> uintptr
	lines   sync.Map // lineKey -> uintptr
	sprites sync.Map // spriteKey -> *shadowSprite
	images  sync.Map // *gfx.Raster -> *Image
	metrics sync.Map // lineKey -> gfx.TextMetrics
	skip    uint32   // bit per gfx.OpKind to leave out; benchmarks use it to split the cost
}

type fontKey struct {
	name string
	size float64
}

type lineKey struct {
	font uintptr
	s    string
}

func NewRenderer(o Options) *Renderer {
	bind()
	if o.FontDir == "" {
		o.FontDir = os.Getenv("GFX_FONT_DIR")
	}
	if o.FontDir != "" {
		_, _ = RegisterFonts(o.FontDir) // a missing directory leaves the system fonts
	}
	r := &Renderer{opts: o, cs: cgColorSpaceCreateDeviceRGB()}
	if _, ok := gfx.DefaultMeasurer.(gfx.EstimateText); ok {
		gfx.DefaultMeasurer = r // recordings from now on get CoreText bounds
	}
	return r
}

// Image is a CGImage snapshot of a Surface or a sprite piece. Size is in points.
type Image struct {
	ref  uintptr
	w, h float64
}

func (i *Image) Size() (float64, float64) { return i.w, i.h }

// Ref is the CGImageRef, for handing to CALayer.contents.
func (i *Image) Ref() uintptr { return i.ref }

// ColorRef returns a new CGColorRef (caller owns it) for CALayer color properties.
func ColorRef(c gfx.Color) uintptr {
	bind()
	return cgColorCreate(c.R, c.G, c.B, c.A)
}

func (i *Image) Release() {
	if i.ref != 0 {
		cgImageRelease(i.ref)
		i.ref = 0
	}
}

// Surface is an offscreen BGRA bitmap. One Surface must not be used by two goroutines at
// once; separate Surfaces of one Renderer may be.
type Surface struct {
	r      *Renderer
	ctx    uintptr
	W, H   int // pixels
	data   uintptr
	scale  float64
	dashed bool
	eff    float64 // scale times the scale of the item transform being drawn
	it     gfx.Iter
	layers int
}

func (r *Renderer) NewSurface(wpx, hpx int) *Surface {
	ctx := cgBitmapContextCreate(0, uintptr(wpx), uintptr(hpx), 8, uintptr(wpx*4), r.cs, bitmapInfoBGRA)
	if ctx == 0 {
		panic("gfx/cg: CGBitmapContextCreate failed")
	}
	s := &Surface{r: r, ctx: ctx, W: wpx, H: hpx, data: cgBitmapContextGetData(ctx), scale: 1}
	cgFontSmooth(ctx, false) // grayscale AA: LCD smoothing cannot be cached in a transparent tile
	cgInterpolate(ctx, 1)
	cgSaveGState(ctx) // the base state Begin returns to
	return s
}

func (s *Surface) Close() {
	if s.ctx != 0 {
		cgContextRelease(s.ctx)
		s.ctx = 0
	}
}

// Pixels returns the bitmap memory (BGRA, premultiplied, rows top to bottom, W*4 bytes each).
// It is valid until Close.
func (s *Surface) Pixels() []byte {
	return unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&s.data))), s.W*s.H*4)
}

// Begin clears the surface to bg and maps scene point (ox, oy) to the top left pixel at
// the given scale (device pixels per scene unit); y points down afterwards.
func (s *Surface) Begin(scale, ox, oy float64, bg gfx.Color) {
	cgRestoreGSt(s.ctx)
	cgSaveGState(s.ctx)
	full := cgRect{0, 0, float64(s.W), float64(s.H)}
	if bg.A == 0 {
		cgClearRect(s.ctx, full)
	} else {
		cgFillColor(s.ctx, bg.R, bg.G, bg.B, bg.A)
		cgFillRect(s.ctx, full)
	}
	cgTranslate(s.ctx, 0, float64(s.H))
	cgScale(s.ctx, scale, -scale)
	cgTranslate(s.ctx, -ox, -oy)
	s.scale, s.eff = scale, scale
	s.dashed = false
}

// BeginClipped is Begin for a partial redraw: only the scene rectangle dirty is cleared
// and drawn into, the rest of the bitmap is left as it was.
func (s *Surface) BeginClipped(scale, ox, oy float64, dirty gfx.Rect) {
	cgRestoreGSt(s.ctx)
	cgSaveGState(s.ctx)
	r := cgRect{(dirty.Min.X - ox) * scale, (dirty.Min.Y - oy) * scale, dirty.W() * scale, dirty.H() * scale}
	r.Y = float64(s.H) - r.Y - r.H
	cgClearRect(s.ctx, r)
	cgClipRect(s.ctx, r)
	cgTranslate(s.ctx, 0, float64(s.H))
	cgScale(s.ctx, scale, -scale)
	cgTranslate(s.ctx, -ox, -oy)
	s.scale, s.eff = scale, scale
	s.dashed = false
}

// Snapshot returns the current bitmap as an image of the given size in points.
func (s *Surface) Snapshot(wpt, hpt float64) *Image {
	return &Image{cgBitmapContextCreateImage(s.ctx), wpt, hpt}
}

// Composite draws img with its top left at (x, y) in device pixels, ignoring the CTM set by
// Begin; it is the CPU compositor used by the tile benchmarks.
func (s *Surface) Composite(img *Image, x, y, wpx, hpx float64) {
	cgDrawImage(s.ctx, cgRect{x, float64(s.H) - y - hpx, wpx, hpx}, img.ref)
}

// shadowSprite is a rounded-rectangle shadow cut into corner and edge pieces; a shadow of
// any larger size is eight draws (four corners, four stretched edges) plus a center fill
// when the box fill does not hide it.
type shadowSprite struct {
	q          float64   // corner piece size in points
	m          float64   // margin the blur adds around the shape
	reach      float64   // edge band thickness: q, or less when an opaque fill hides the rest
	corner     [4]*Image // tl, tr, bl, br
	edge       [4]*Image // top, bottom, left, right
	centerFill gfx.Color
}

type spriteKey struct {
	r, blur float64
	col     gfx.Color
	scale   float64
	reach   float64
}

// sprite returns the cached pieces; opaque is true when the box fill covers everything but
// the outer reach of the shadow.
func (r *Renderer) sprite(radius, blur float64, col gfx.Color, scale, offset float64, opaque bool) *shadowSprite {
	m := math.Ceil(1.5 * blur)
	q := radius + 2*m
	reach := q
	if opaque {
		reach = math.Min(q, m+math.Ceil(offset))
	}
	k := spriteKey{radius, blur, col, scale, reach}
	if v, ok := r.sprites.Load(k); ok {
		return v.(*shadowSprite)
	}
	v, _ := r.sprites.LoadOrStore(k, r.buildSprite(radius, blur, col, scale, reach))
	return v.(*shadowSprite)
}

func (r *Renderer) buildSprite(radius, blur float64, col gfx.Color, scale, reach float64) *shadowSprite {
	m := math.Ceil(1.5 * blur)
	q := radius + 2*m // keeps the 1-point strip clear of both blur ramps
	side := 2*q + 1   // points; the middle point is the stretchable strip
	px := int(math.Ceil(side * scale))
	surf := r.NewSurface(px, px)
	defer surf.Close()
	ctx := surf.ctx
	// draw the shape far outside the bitmap and bring only its shadow back with the offset
	const far = 4096.0
	c := cgColorCreate(col.R, col.G, col.B, col.A)
	cgSetShadow(ctx, cgSize{far, 0}, blur*scale, c)
	cgFillColor(ctx, 0, 0, 0, 1)
	var rr gfx.Path
	sp := float64(px) / side // exact pixels per point after rounding
	rr.RoundRect(-far+m*sp, m*sp, (side-2*m)*sp, (side-2*m)*sp, radius*sp)
	surf.addPath(&rr)
	cgFillPath(ctx)
	cgColorRelase(c)
	img := cgBitmapContextCreateImage(ctx)
	defer cgImageRelease(img)

	// the y-down context shows an image upside down, so the pieces for the top of the
	// destination are cut from the bottom of the image and the other way round
	piece := func(x, y, w, h float64) *Image {
		return &Image{cgImageCreateWithImageInRct(img, cgRect{x * sp, y * sp, w * sp, h * sp}), w, h}
	}
	out := &shadowSprite{q: q, m: m, reach: reach, centerFill: col}
	out.corner = [4]*Image{piece(0, q+1, q, q), piece(q+1, q+1, q, q), piece(0, 0, q, q), piece(q+1, 0, q, q)}
	out.edge = [4]*Image{piece(q, side-reach, 1, reach), piece(q, 0, 1, reach), piece(0, q, reach, 1), piece(side-reach, q, reach, 1)}
	return out
}

// draw paints the shadow of the box (x, y, w, h), already offset by the shadow vector.
func (sp *shadowSprite) draw(s *Surface, x, y, w, h float64) {
	ctx := s.ctx
	q, m, t := sp.q, sp.m, sp.reach
	x0, y0, x1, y1 := x-m, y-m, x+w+m, y+h+m
	put := func(img *Image, dx, dy, dw, dh float64) { cgDrawImage(ctx, cgRect{dx, dy, dw, dh}, img.ref) }
	put(sp.corner[0], x0, y0, q, q)
	put(sp.corner[1], x1-q, y0, q, q)
	put(sp.corner[2], x0, y1-q, q, q)
	put(sp.corner[3], x1-q, y1-q, q, q)
	put(sp.edge[0], x0+q, y0, x1-x0-2*q, t)
	put(sp.edge[1], x0+q, y1-t, x1-x0-2*q, t)
	put(sp.edge[2], x0, y0+q, t, y1-y0-2*q)
	put(sp.edge[3], x1-t, y0+q, t, y1-y0-2*q)
	if t >= q { // the flat core shows through a translucent fill
		c := sp.centerFill
		cgFillColor(ctx, c.R, c.G, c.B, c.A)
		cgFillRect(ctx, cgRect{x0 + q, y0 + q, x1 - x0 - 2*q, y1 - y0 - 2*q})
	}
}
