package gfx

import (
	"bytes"
	"encoding/binary"
	"math"
	"slices"
	"sync"
)

// OpKind tells which drawing command an Op is.
type OpKind uint8

// The operands of each kind are in the fields of Op named after it.
const (
	OpSave         OpKind = iota
	OpRestore             //
	OpTranslate           // A, B
	OpScale               // A, B
	OpRotate              // A
	OpTransform           // M
	OpFill                // Path, Color
	OpFillGrad            // Path, Grad
	OpStroke              // Path, Style, Color
	OpStrokeGrad          // Path, Style, Grad
	OpShadowBox           // rounded rect A, B, C, D (x, y, w, h), E radius, Shadow, Color fill
	OpText                // Text, Font, A, B baseline origin, Color
	OpImage               // Image, A, B, C, D destination rect
	OpClipRect            // A, B, C, D
	OpClipRRect           // A, B, C, D, E radius
	OpClipPath            // Path
	OpBeginLayer          // A alpha: everything up to OpEndLayer is composited as one with this alpha
	OpEndLayer            //
	OpFillShadow          // Path, Color fill, Shadow: the shadow of the filled shape, then the shape
	OpStrokeShadow        // Path, Style, Color, Shadow: the shadow of the stroked outline, then the stroke
	OpTextShadow          // as OpText, plus Shadow under the glyphs
)

// Cap is the shape of the ends of a stroked line.
type Cap uint8

// Join is the shape of the corners of a stroked path.
type Join uint8

// The line caps: flat at the end, round, or square past the end by half the width.
const (
	CapButt Cap = iota
	CapRound
	CapSquare
)

// The line joins: sharp (up to MiterLimit), round, or cut off.
const (
	JoinMiter Join = iota
	JoinRound
	JoinBevel
)

// StrokeStyle describes a stroke. The zero value is a hairline with butt caps and miter joins; set
// Width.
type StrokeStyle struct {
	Width      float64
	Cap        Cap
	Join       Join
	MiterLimit float64
	Dash       []float64
	DashPhase  float64
}

// Shadow is a blurred drop shadow offset by (DX, DY).
type Shadow struct {
	DX, DY, Blur float64
	Color        Color
}

// Font is a face, a size in points and a tracking.
//
// Name is an exact face name (a PostScript name such as "IBMPlexSans-SmBld", or a family); when it is
// set, Family, Weight and Italic are not read. Without a Name the backend picks the face of Family
// whose weight is closest to Weight (CSS scale 100..900, 0 means 400) and whose slant matches
// Italic, among installed fonts and those registered with RegisterFonts. An empty Family or one the
// backend does not have is the default sans face of the backend, at that weight and slant.
//
// Spacing is added after every character, in points, in drawing and in measuring: the width of a
// line is its natural width plus Spacing for each rune, the last one included, as in CSS.
type Font struct {
	Name    string
	Family  string
	Weight  int
	Italic  bool
	Size    float64
	Spacing float64
}

// Image is a raster a backend can draw. *Raster is the portable one; a backend may add its own.
type Image interface{ Size() (w, h float64) }

// Op is one decoded command. Fields not named by the kind are stale; Path, Style and Grad
// point into the Iter and are valid until its next Next.
type Op struct {
	Kind          OpKind
	A, B, C, D, E float64
	M             Affine
	Color         Color
	Path          *Path
	Style         *StrokeStyle
	Grad          *Gradient
	Shadow        Shadow
	Text          string
	Font          Font
	Image         Image
}

// Picture is a recorded command list with its bounds in the coordinates it was recorded in.
// Commands live in one byte stream: a kind byte, then float32 coordinates, 8-bit colors and
// varint indices. It is immutable after Finish and safe to replay from several goroutines.
type Picture struct {
	Bounds Rect
	// Lod, when set, replaces this picture where its larger side is under LodBelow device pixels.
	// It should look like p at that size; a backend fades it out over the band above LodBelow.
	Lod      *Picture
	LodBelow float64

	data []byte
	res  *picRes // nil for a picture without text or images
}

// Size is the memory the picture holds, without Lod and images.
func (p *Picture) Size() int {
	n := len(p.data) + 88
	if p.res != nil {
		n += 72 + len(p.res.fonts)*64 + len(p.res.images)*16
		for _, s := range p.res.strs {
			n += len(s) + 16
		}
	}
	return n
}

// picRes holds what a command stream refers to by index.
type picRes struct {
	strs   []string
	fonts  []Font
	images []Image
}

// Equal reports whether q draws exactly what p draws, Lod included. Scene.SetPicture uses it to
// ignore a picture that did not change, so a node moved with its content is not redrawn.
func (p *Picture) Equal(q *Picture) bool {
	if p == q {
		return true
	}
	if p == nil || q == nil || p.Bounds != q.Bounds || p.LodBelow != q.LodBelow || !bytes.Equal(p.data, q.data) {
		return false
	}
	if (p.res == nil) != (q.res == nil) || (p.Lod == nil) != (q.Lod == nil) {
		return false
	}
	if p.res != nil && (!slices.Equal(p.res.strs, q.res.strs) || !slices.Equal(p.res.fonts, q.res.fonts) || !slices.Equal(p.res.images, q.res.images)) {
		return false
	}
	return p.Lod == nil || p.Lod.Equal(q.Lod)
}

// Len is the number of bytes of the command stream.
func (p *Picture) Len() int { return len(p.data) }

// TextMetrics are the extents of a laid-out line in points; Ascent and Descent are positive.
type TextMetrics struct{ Width, Ascent, Descent float64 }

// TextMeasurer reports the size of text as the backend draws it.
type TextMeasurer interface {
	MeasureText(s string, f Font) TextMetrics
}

// EstimateText is the measurer used until a backend sets one: 0.6 em per rune, ascent 0.8 em, descent 0.25 em.
type EstimateText struct{}

// MeasureText estimates the size of s from the number of runes; Spacing counts for each of them.
func (EstimateText) MeasureText(s string, f Font) TextMetrics {
	n := 0
	for range s {
		n++
	}
	return TextMetrics{float64(n) * (f.Size*0.6 + f.Spacing), f.Size * 0.8, f.Size * 0.25}
}

// DefaultMeasurer is what NewRecorder uses. Importing a backend (gfx/cg, gfx/cairo, gfx/d2d, and
// so gfx/view and gfx/offscreen) replaces it with that backend's measurer, so text bounds match
// what is drawn whatever the order of recording and of creating a renderer. Without a backend it is
// EstimateText.
var DefaultMeasurer TextMeasurer = EstimateText{}

// LazyMeasurer returns a TextMeasurer that calls open on its first use, once, and measures with the
// result; when open fails it estimates with EstimateText. Backends use it so that a renderer is
// made only if text is recorded.
func LazyMeasurer(open func() (TextMeasurer, error)) TextMeasurer {
	return &lazyMeasurer{open: open}
}

type lazyMeasurer struct {
	once sync.Once
	open func() (TextMeasurer, error)
	m    TextMeasurer
}

func (l *lazyMeasurer) MeasureText(s string, f Font) TextMetrics {
	l.once.Do(func() {
		var err error
		if l.m, err = l.open(); err != nil || l.m == nil {
			l.m = EstimateText{}
		}
	})
	return l.m.MeasureText(s, f)
}

// Recorder builds a Picture and tracks its bounds through the transform and clip stack.
type Recorder struct {
	pic   Picture
	res   picRes
	ctm   Affine
	clip  Rect // visible area in picture coordinates
	stack []recState
	m     TextMeasurer
	ls    int // open layers
}

type recState struct {
	ctm  Affine
	clip Rect
}

// NewRecorder returns a Recorder that measures text with DefaultMeasurer.
func NewRecorder() *Recorder {
	return &Recorder{pic: Picture{Bounds: EmptyRect()}, ctm: Identity(), clip: infRect(), m: DefaultMeasurer}
}

// NewRecorderWith is NewRecorder with m measuring text bounds.
func NewRecorderWith(m TextMeasurer) *Recorder {
	r := NewRecorder()
	r.m = m
	return r
}

func infRect() Rect {
	return Rect{Point{math.Inf(-1), math.Inf(-1)}, Point{math.Inf(1), math.Inf(1)}}
}

// Intersect returns the common area; the result is Empty when there is none.
func (r Rect) Intersect(o Rect) Rect {
	return Rect{Point{math.Max(r.Min.X, o.Min.X), math.Max(r.Min.Y, o.Min.Y)}, Point{math.Min(r.Max.X, o.Max.X), math.Min(r.Max.Y, o.Max.Y)}}
}

func (r *Recorder) op(k OpKind) { r.pic.data = append(r.pic.data, byte(k)) }

func (r *Recorder) f32(vs ...float64) {
	for _, v := range vs {
		r.pic.data = binary.LittleEndian.AppendUint32(r.pic.data, math.Float32bits(float32(v)))
	}
}

func (r *Recorder) uv(v int) { r.pic.data = binary.AppendUvarint(r.pic.data, uint64(v)) }

func (r *Recorder) color(c Color) {
	r.pic.data = append(r.pic.data, q8(c.R), q8(c.G), q8(c.B), q8(c.A))
}

func q8(v float64) byte { return byte(math.Round(math.Min(math.Max(v, 0), 1) * 255)) }

func (r *Recorder) path(p *Path) {
	n := len(p.Verbs) << 1
	if p.EvenOdd {
		n |= 1
	}
	r.uv(n)
	for _, v := range p.Verbs {
		r.pic.data = append(r.pic.data, byte(v))
	}
	for _, q := range p.Pts {
		r.f32(q.X, q.Y)
	}
}

func (r *Recorder) style(st StrokeStyle) {
	r.f32(st.Width, st.MiterLimit, st.DashPhase)
	r.pic.data = append(r.pic.data, byte(st.Cap), byte(st.Join))
	r.uv(len(st.Dash))
	r.f32(st.Dash...)
}

func (r *Recorder) grad(g *Gradient) {
	r.pic.data = append(r.pic.data, byte(g.Kind))
	r.f32(g.P0.X, g.P0.Y, g.P1.X, g.P1.Y, g.R0, g.R1)
	r.uv(len(g.Stops))
	for _, s := range g.Stops {
		r.f32(s.Offset)
		r.color(s.Color)
	}
}

// grow adds the local rectangle, as drawn through the current transform and clip, to the bounds.
func (r *Recorder) grow(local Rect) {
	r.pic.Bounds = r.pic.Bounds.Union(r.ctm.Bounds(local).Intersect(r.clip))
}

// Save pushes the transform and the clip.
func (r *Recorder) Save() {
	r.stack = append(r.stack, recState{r.ctm, r.clip})
	r.op(OpSave)
}

// Restore pops what Save pushed.
func (r *Recorder) Restore() {
	if n := len(r.stack); n > 0 {
		r.ctm, r.clip = r.stack[n-1].ctm, r.stack[n-1].clip
		r.stack = r.stack[:n-1]
	}
	r.op(OpRestore)
}

// Translate moves the origin of what follows by (x, y).
func (r *Recorder) Translate(x, y float64) {
	r.ctm = r.ctm.Mul(TranslateBy(x, y))
	r.op(OpTranslate)
	r.f32(x, y)
}

// Scale scales what follows by x and y about the origin.
func (r *Recorder) Scale(x, y float64) {
	r.ctm = r.ctm.Mul(ScaleBy(x, y))
	r.op(OpScale)
	r.f32(x, y)
}

// Rotate rotates what follows by a radians about the origin.
func (r *Recorder) Rotate(a float64) {
	r.ctm = r.ctm.Mul(RotateBy(a))
	r.op(OpRotate)
	r.f32(a)
}

// Transform multiplies the current transform by m: m applies first.
func (r *Recorder) Transform(m Affine) {
	r.ctm = r.ctm.Mul(m)
	r.op(OpTransform)
	r.f32(m[:]...)
}

// ClipRect limits what follows, until the enclosing Restore, to rc.
func (r *Recorder) ClipRect(rc Rect) {
	r.clip = r.clip.Intersect(r.ctm.Bounds(rc))
	r.op(OpClipRect)
	r.f32(rc.Min.X, rc.Min.Y, rc.W(), rc.H())
}

// ClipRRect is ClipRect with corners of the given radius.
func (r *Recorder) ClipRRect(rc Rect, radius float64) {
	r.clip = r.clip.Intersect(r.ctm.Bounds(rc))
	r.op(OpClipRRect)
	r.f32(rc.Min.X, rc.Min.Y, rc.W(), rc.H(), radius)
}

// ClipPath limits what follows to the inside of p, by its fill rule.
func (r *Recorder) ClipPath(p *Path) {
	r.clip = r.clip.Intersect(r.ctm.Bounds(p.Bounds()))
	r.op(OpClipPath)
	r.path(p)
}

// BeginLayer starts a group composited as one at the given alpha; end it with EndLayer.
func (r *Recorder) BeginLayer(alpha float64) {
	r.ls++
	r.op(OpBeginLayer)
	r.f32(alpha)
}

// EndLayer closes the group BeginLayer opened and draws it with its alpha.
func (r *Recorder) EndLayer() {
	if r.ls > 0 {
		r.ls--
		r.op(OpEndLayer)
	}
}

// FillPath copies p into the picture.
func (r *Recorder) FillPath(p *Path, c Color) {
	r.grow(p.Bounds())
	r.op(OpFill)
	r.path(p)
	r.color(c)
}

// FillGradient fills p with g.
func (r *Recorder) FillGradient(p *Path, g *Gradient) {
	r.grow(p.Bounds())
	r.op(OpFillGrad)
	r.path(p)
	r.grad(g)
}

func (r *Recorder) strokeBounds(p *Path, st StrokeStyle) Rect {
	return p.Bounds().Expand(st.Width / 2 * strokeReach(p, st))
}

// StrokePath strokes p with st in color c.
func (r *Recorder) StrokePath(p *Path, st StrokeStyle, c Color) {
	r.grow(r.strokeBounds(p, st))
	r.op(OpStroke)
	r.path(p)
	r.style(st)
	r.color(c)
}

// StrokeGradient strokes p with st, painted by g.
func (r *Recorder) StrokeGradient(p *Path, st StrokeStyle, g *Gradient) {
	r.grow(r.strokeBounds(p, st))
	r.op(OpStrokeGrad)
	r.path(p)
	r.style(st)
	r.grad(g)
}

// strokeReach is how many half widths a stroke of p can reach beyond the path: the longest miter
// spike, or the square cap diagonal.
func strokeReach(p *Path, st StrokeStyle) float64 {
	reach := 1.0
	if st.Cap == CapSquare {
		reach = math.Sqrt2
	}
	if st.Join != JoinMiter {
		return reach
	}
	limit := st.MiterLimit
	if limit <= 0 {
		limit = 10
	}
	join := func(a, b *seg) {
		// miter length over width is 1/sin(half the angle between the two legs)
		cos := -a.tangent(1).Dot(b.tangent(0))
		if cos >= 1 {
			reach = math.Max(reach, limit)
			return
		}
		if s := math.Sqrt((1 - cos) / 2); s > 0 {
			reach = math.Max(reach, math.Min(1/s, limit))
		}
	}
	var prev, first seg
	have, sub, closed := false, -1, false
	p.each(func(s *seg, sb int, closing bool) {
		if sb != sub {
			if closed && have {
				join(&prev, &first)
			}
			sub, have, closed = sb, false, false
		}
		if have {
			join(&prev, s)
		} else {
			first = *s
		}
		prev, have, closed = *s, true, closing
	})
	if closed && have {
		join(&prev, &first)
	}
	return reach
}

// FillShadowRRect fills a rounded rectangle with a blurred drop shadow under it.
func (r *Recorder) FillShadowRRect(x, y, w, h, radius float64, fill Color, sh Shadow) {
	r.grow(R(x, y, w, h).Expand(sh.reach()))
	r.op(OpShadowBox)
	r.f32(x, y, w, h, radius, sh.DX, sh.DY, sh.Blur)
	r.color(fill)
	r.color(sh.Color)
}

func (r *Recorder) shadow(sh Shadow) {
	r.f32(sh.DX, sh.DY, sh.Blur)
	r.color(sh.Color)
}

func (sh Shadow) reach() float64 {
	return 1.5*sh.Blur + math.Max(math.Abs(sh.DX), math.Abs(sh.DY))
}

// FillShadowPath fills p with a blurred drop shadow of its shape under it. The offset of the shadow
// is in the coordinates of p, so it turns and mirrors with the transform.
func (r *Recorder) FillShadowPath(p *Path, c Color, sh Shadow) {
	r.grow(p.Bounds().Expand(sh.reach()))
	r.op(OpFillShadow)
	r.path(p)
	r.color(c)
	r.shadow(sh)
}

// StrokeShadowPath strokes p with a blurred drop shadow of the stroke under it.
func (r *Recorder) StrokeShadowPath(p *Path, st StrokeStyle, c Color, sh Shadow) {
	r.grow(r.strokeBounds(p, st).Expand(sh.reach()))
	r.op(OpStrokeShadow)
	r.path(p)
	r.style(st)
	r.color(c)
	r.shadow(sh)
}

// TextShadow is Text with a blurred drop shadow of the glyphs under it.
func (r *Recorder) TextShadow(s string, x, y float64, f Font, c Color, sh Shadow) {
	r.text(OpTextShadow, s, x, y, f, c, sh)
}

// Text draws s with its baseline origin at (x, y); the bounds come from the recorder's measurer.
func (r *Recorder) Text(s string, x, y float64, f Font, c Color) {
	r.text(OpText, s, x, y, f, c, Shadow{})
}

func (r *Recorder) text(k OpKind, s string, x, y float64, f Font, c Color, sh Shadow) {
	m := r.m.MeasureText(s, f)
	// a glyph can lean a little past its advance
	box := R(x-f.Size*0.1, y-m.Ascent, m.Width+f.Size*0.2, m.Ascent+m.Descent)
	if k == OpTextShadow {
		box = box.Expand(sh.reach())
	}
	r.grow(box)
	r.res.strs = append(r.res.strs, s)
	fi := -1
	for i, ff := range r.res.fonts {
		if ff == f {
			fi = i
			break
		}
	}
	if fi < 0 {
		r.res.fonts = append(r.res.fonts, f)
		fi = len(r.res.fonts) - 1
	}
	r.op(k)
	r.f32(x, y)
	r.uv(len(r.res.strs) - 1)
	r.uv(fi)
	r.color(c)
	if k == OpTextShadow {
		r.shadow(sh)
	}
}

// DrawImage draws img stretched into the rectangle at (x, y) of size w by h.
func (r *Recorder) DrawImage(img Image, x, y, w, h float64) {
	r.grow(R(x, y, w, h))
	r.res.images = append(r.res.images, img)
	r.op(OpImage)
	r.f32(x, y, w, h)
	r.uv(len(r.res.images) - 1)
}

// Finish returns the picture; the Recorder must not be used afterwards.
func (r *Recorder) Finish() *Picture {
	for ; r.ls > 0; r.ls-- {
		r.op(OpEndLayer)
	}
	p := r.pic
	p.data = append([]byte(nil), p.data...) // drop the growth slack
	if len(r.res.strs) > 0 || len(r.res.images) > 0 {
		p.res = &picRes{append([]string(nil), r.res.strs...), append([]Font(nil), r.res.fonts...), append([]Image(nil), r.res.images...)}
	}
	return &p
}

// Iter decodes a Picture command by command. The zero Iter is ready after Reset; reusing one
// keeps replays free of allocations.
type Iter struct {
	p     *Picture
	pos   int
	op    Op
	path  Path
	style StrokeStyle
	grad  Gradient
}

// Reset points the iterator at the start of p.
func (it *Iter) Reset(p *Picture) {
	it.p, it.pos = p, 0
}

func (it *Iter) f32() float64 {
	v := math.Float32frombits(binary.LittleEndian.Uint32(it.p.data[it.pos:]))
	it.pos += 4
	return float64(v)
}

func (it *Iter) uv() int {
	v, n := binary.Uvarint(it.p.data[it.pos:])
	it.pos += n
	return int(v)
}

func (it *Iter) color() Color {
	d := it.p.data[it.pos : it.pos+4]
	it.pos += 4
	return Color{float64(d[0]) / 255, float64(d[1]) / 255, float64(d[2]) / 255, float64(d[3]) / 255}
}

func (it *Iter) readPath() *Path {
	n := it.uv()
	it.path.EvenOdd = n&1 != 0
	n >>= 1
	it.path.Verbs = it.path.Verbs[:0]
	it.path.Pts = it.path.Pts[:0]
	np := 0
	for i := 0; i < n; i++ {
		v := Verb(it.p.data[it.pos+i])
		it.path.Verbs = append(it.path.Verbs, v)
		switch v {
		case MoveTo, LineTo:
			np++
		case QuadTo:
			np += 2
		case CubicTo:
			np += 3
		}
	}
	it.pos += n
	for i := 0; i < np; i++ {
		x := it.f32()
		it.path.Pts = append(it.path.Pts, Point{x, it.f32()})
	}
	return &it.path
}

func (it *Iter) readStyle() *StrokeStyle {
	st := &it.style
	st.Width, st.MiterLimit, st.DashPhase = it.f32(), it.f32(), it.f32()
	st.Cap, st.Join = Cap(it.p.data[it.pos]), Join(it.p.data[it.pos+1])
	it.pos += 2
	n := it.uv()
	st.Dash = st.Dash[:0]
	for i := 0; i < n; i++ {
		st.Dash = append(st.Dash, it.f32())
	}
	return st
}

func (it *Iter) readGrad() *Gradient {
	g := &it.grad
	g.Kind = GradientKind(it.p.data[it.pos])
	it.pos++
	g.P0.X, g.P0.Y, g.P1.X, g.P1.Y, g.R0, g.R1 = it.f32(), it.f32(), it.f32(), it.f32(), it.f32(), it.f32()
	n := it.uv()
	g.Stops = g.Stops[:0]
	for i := 0; i < n; i++ {
		off := it.f32()
		g.Stops = append(g.Stops, Stop{off, it.color()})
	}
	return g
}

func (it *Iter) readShadow(o *Op) {
	o.Shadow.DX, o.Shadow.DY, o.Shadow.Blur = it.f32(), it.f32(), it.f32()
	o.Shadow.Color = it.color()
}

// Next decodes the next command; it reports false at the end of the picture.
func (it *Iter) Next() bool {
	if it.pos >= len(it.p.data) {
		return false
	}
	o := &it.op
	o.Kind = OpKind(it.p.data[it.pos])
	it.pos++
	switch o.Kind {
	case OpTranslate, OpScale:
		o.A, o.B = it.f32(), it.f32()
	case OpRotate, OpBeginLayer:
		o.A = it.f32()
	case OpTransform:
		for i := range o.M {
			o.M[i] = it.f32()
		}
	case OpFill:
		o.Path = it.readPath()
		o.Color = it.color()
	case OpFillGrad:
		o.Path = it.readPath()
		o.Grad = it.readGrad()
	case OpStroke:
		o.Path = it.readPath()
		o.Style = it.readStyle()
		o.Color = it.color()
	case OpStrokeGrad:
		o.Path = it.readPath()
		o.Style = it.readStyle()
		o.Grad = it.readGrad()
	case OpShadowBox:
		o.A, o.B, o.C, o.D, o.E = it.f32(), it.f32(), it.f32(), it.f32(), it.f32()
		o.Shadow.DX, o.Shadow.DY, o.Shadow.Blur = it.f32(), it.f32(), it.f32()
		o.Color = it.color()
		o.Shadow.Color = it.color()
	case OpFillShadow:
		o.Path = it.readPath()
		o.Color = it.color()
		it.readShadow(o)
	case OpStrokeShadow:
		o.Path = it.readPath()
		o.Style = it.readStyle()
		o.Color = it.color()
		it.readShadow(o)
	case OpText, OpTextShadow:
		o.A, o.B = it.f32(), it.f32()
		o.Text = it.p.res.strs[it.uv()]
		o.Font = it.p.res.fonts[it.uv()]
		o.Color = it.color()
		if o.Kind == OpTextShadow {
			it.readShadow(o)
		}
	case OpImage:
		o.A, o.B, o.C, o.D = it.f32(), it.f32(), it.f32(), it.f32()
		o.Image = it.p.res.images[it.uv()]
	case OpClipRect:
		o.A, o.B, o.C, o.D = it.f32(), it.f32(), it.f32(), it.f32()
	case OpClipRRect:
		o.A, o.B, o.C, o.D, o.E = it.f32(), it.f32(), it.f32(), it.f32(), it.f32()
	case OpClipPath:
		o.Path = it.readPath()
	}
	return true
}

// Op returns the command Next decoded.
func (it *Iter) Op() *Op { return &it.op }

// FillRect fills rc.
func (r *Recorder) FillRect(rc Rect, c Color) { r.FillPath(RectPath(rc), c) }

// FillRRect fills rc with corners rounded by radius.
func (r *Recorder) FillRRect(rc Rect, radius float64, c Color) {
	r.FillPath(RoundRectPath(rc, radius), c)
}

// FillRRect4 fills rc with a radius per corner (top left, top right, bottom right, bottom left).
func (r *Recorder) FillRRect4(rc Rect, radii [4]float64, c Color) {
	r.FillPath(RoundRectPath4(rc, radii), c)
}

// FillEllipse fills the ellipse that fits in rc.
func (r *Recorder) FillEllipse(rc Rect, c Color) { r.FillPath(EllipsePath(rc), c) }

// StrokeRect strokes the outline of rc, centred on its edge.
func (r *Recorder) StrokeRect(rc Rect, st StrokeStyle, c Color) {
	r.StrokePath(RectPath(rc), st, c)
}

// StrokeRRect strokes the outline of rc with corners rounded by radius.
func (r *Recorder) StrokeRRect(rc Rect, radius float64, st StrokeStyle, c Color) {
	r.StrokePath(RoundRectPath(rc, radius), st, c)
}

// StrokeRRect4 strokes the outline of rc with a radius per corner (top left, top right, bottom right,
// bottom left).
func (r *Recorder) StrokeRRect4(rc Rect, radii [4]float64, st StrokeStyle, c Color) {
	r.StrokePath(RoundRectPath4(rc, radii), st, c)
}

// StrokeEllipse strokes the outline of the ellipse that fits in rc.
func (r *Recorder) StrokeEllipse(rc Rect, st StrokeStyle, c Color) {
	r.StrokePath(EllipsePath(rc), st, c)
}
