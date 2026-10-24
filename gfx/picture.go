package gfx

import (
	"math"
	"unicode/utf8"
)

type OpKind uint8

const (
	OpSave OpKind = iota
	OpRestore
	OpTranslate // A, B
	OpScale     // A, B
	OpRotate    // A
	OpFill      // Idx path, Color
	OpStroke    // Idx path, Style, Color
	OpShadowBox // rounded rect A,B,C,D (x, y, w, h), E radius, Shadow, Color fill
	OpText      // Idx string, Font, A,B baseline origin, Color
	OpImage     // Idx image, A,B,C,D destination rect
)

type Cap uint8
type Join uint8

const (
	CapButt Cap = iota
	CapRound
	CapSquare
)

const (
	JoinMiter Join = iota
	JoinRound
	JoinBevel
)

type StrokeStyle struct {
	Width      float64
	Cap        Cap
	Join       Join
	MiterLimit float64
	Dash       []float64
	DashPhase  float64
}

type Shadow struct {
	DX, DY, Blur float64
	Color        Color
}

type Font struct {
	Name string
	Size float64
}

// Image is a raster a backend can draw; the concrete type belongs to the backend.
type Image interface{ Size() (w, h float64) }

type Op struct {
	Kind       OpKind
	Idx        int32
	Style      int32
	A, B, C, D float64
	E          float64
	Color      Color
	Shadow     Shadow
	Font       Font
}

// Picture is a recorded command list with its bounds in the coordinates it was recorded in.
// It is immutable after Finish and safe to replay from several goroutines.
type Picture struct {
	Ops    []Op
	Paths  []*Path
	Styles []StrokeStyle
	Strs   []string
	Images []Image
	Bounds Rect
	// Lod, when set, replaces this picture where its larger side is under LodBelow device pixels.
	Lod      *Picture
	LodBelow float64
}

// Recorder builds a Picture and tracks its bounds through the transform stack.
type Recorder struct {
	pic   Picture
	ctm   Affine
	stack []Affine
}

func NewRecorder() *Recorder {
	return &Recorder{pic: Picture{Bounds: EmptyRect()}, ctm: Identity()}
}

func (r *Recorder) add(o Op) { r.pic.Ops = append(r.pic.Ops, o) }

func (r *Recorder) grow(local Rect) {
	r.pic.Bounds = r.pic.Bounds.Union(r.ctm.Bounds(local))
}

func (r *Recorder) Save() {
	r.stack = append(r.stack, r.ctm)
	r.add(Op{Kind: OpSave})
}

func (r *Recorder) Restore() {
	if n := len(r.stack); n > 0 {
		r.ctm = r.stack[n-1]
		r.stack = r.stack[:n-1]
	}
	r.add(Op{Kind: OpRestore})
}

func (r *Recorder) Translate(x, y float64) {
	r.ctm = r.ctm.Mul(TranslateBy(x, y))
	r.add(Op{Kind: OpTranslate, A: x, B: y})
}

func (r *Recorder) Scale(x, y float64) {
	r.ctm = r.ctm.Mul(ScaleBy(x, y))
	r.add(Op{Kind: OpScale, A: x, B: y})
}

func (r *Recorder) Rotate(a float64) {
	r.ctm = r.ctm.Mul(RotateBy(a))
	r.add(Op{Kind: OpRotate, A: a})
}

func (r *Recorder) path(p *Path) int32 {
	r.pic.Paths = append(r.pic.Paths, p)
	return int32(len(r.pic.Paths) - 1)
}

// FillPath keeps p by reference; the caller must not change it afterwards.
func (r *Recorder) FillPath(p *Path, c Color) {
	r.grow(p.Bounds())
	r.add(Op{Kind: OpFill, Idx: r.path(p), Color: c})
}

func (r *Recorder) StrokePath(p *Path, st StrokeStyle, c Color) {
	// a miter join reaches up to the miter limit times the half width
	half := st.Width / 2 * math.Max(st.MiterLimit, 1.5)
	r.grow(p.Bounds().Expand(half))
	r.pic.Styles = append(r.pic.Styles, st)
	r.add(Op{Kind: OpStroke, Idx: r.path(p), Style: int32(len(r.pic.Styles) - 1), Color: c})
}

// FillShadowRRect fills a rounded rectangle with a blurred drop shadow under it.
func (r *Recorder) FillShadowRRect(x, y, w, h, radius float64, fill Color, sh Shadow) {
	reach := 1.5*sh.Blur + math.Max(math.Abs(sh.DX), math.Abs(sh.DY))
	r.grow(R(x, y, w, h).Expand(reach))
	r.add(Op{Kind: OpShadowBox, A: x, B: y, C: w, D: h, E: radius, Color: fill, Shadow: sh})
}

// Text draws s with its baseline origin at (x, y). The bounds are an estimate (0.75 em per
// rune); text measured by a backend is more exact but needs a font engine.
func (r *Recorder) Text(s string, x, y float64, f Font, c Color) {
	w := float64(utf8.RuneCountInString(s)) * f.Size * 0.75
	r.grow(R(x, y-f.Size, w, f.Size*1.4))
	r.pic.Strs = append(r.pic.Strs, s)
	r.add(Op{Kind: OpText, Idx: int32(len(r.pic.Strs) - 1), A: x, B: y, Font: f, Color: c})
}

func (r *Recorder) DrawImage(img Image, x, y, w, h float64) {
	r.grow(R(x, y, w, h))
	r.pic.Images = append(r.pic.Images, img)
	r.add(Op{Kind: OpImage, Idx: int32(len(r.pic.Images) - 1), A: x, B: y, C: w, D: h})
}

func (r *Recorder) Finish() *Picture {
	p := r.pic
	return &p
}
