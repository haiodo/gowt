//go:build darwin

package cg

import (
	"math"
	"runtime"
	"unsafe"

	"github.com/haiodo/gowt/gfx"
)

const kCFStringEncodingUTF8 = 0x08000100

func cfString(s string) uintptr {
	if s == "" {
		return cfStringCreateWithBytes(0, nil, 0, kCFStringEncodingUTF8, false)
	}
	return cfStringCreateWithBytes(0, unsafe.StringData(s), len(s), kCFStringEncodingUTF8, false)
}

// font returns the CTFont for f. Name is taken as an exact face name; without it Family, Weight and
// Italic pick the closest face of the family, and the sans fallback stands in for a family
// CoreText does not have.
func (r *Renderer) font(f gfx.Font) uintptr {
	k := fontKey{f.Name, f.Family, f.Weight, f.Italic, f.Size}
	if v, ok := r.fonts.Load(k); ok {
		return v.(uintptr)
	}
	ref := newFont(f)
	v, loaded := r.fonts.LoadOrStore(k, ref)
	if loaded {
		cfRelease(ref)
	}
	return v.(uintptr)
}

func newFont(f gfx.Font) uintptr {
	if f.Name == "" && (f.Family != "" || f.Weight != 0 || f.Italic) {
		w := f.Weight
		if w == 0 {
			w = 400
		}
		if ref, ok := fontByFamily(f.Family, w, f.Italic, f.Size); ok {
			return ref
		}
		return sansFont(w, f.Italic, f.Size)
	}
	name := cfString(f.Name)
	ref := ctFontCreateWithName(name, f.Size, 0)
	cfRelease(name)
	return ref
}

// sansFont is the face of a family CoreText lacks: Helvetica Neue by family, else by PostScript name,
// else the system UI font. CTFontCreateWithName on a name it does not know can answer with a serif face.
func sansFont(weight int, italic bool, size float64) uintptr {
	if ref, ok := fontByFamily(fallbackFamily, weight, italic, size); ok {
		return ref
	}
	for _, n := range []string{sansName(weight, italic), "Helvetica"} {
		name := cfString(n)
		ref := ctFontCreateWithName(name, size, 0)
		cfRelease(name)
		if ref != 0 {
			return ref
		}
	}
	return ctUIFont(kCTFontUIFontSystem, size, 0)
}

func sansName(weight int, italic bool) string {
	n := "HelveticaNeue"
	switch {
	case weight >= 700:
		n += "-Bold"
	case weight >= 500:
		n += "-Medium"
	case weight <= 300:
		n += "-Light"
	}
	if italic {
		if n == "HelveticaNeue" {
			return n + "-Italic"
		}
		return n + "Italic"
	}
	return n
}

// newLine lays out one line of text in font; the color comes from the context at draw time,
// so a line is reusable for any fill color. spacing is added after every character. The caller
// releases it.
func (r *Renderer) newLine(s string, font uintptr, spacing float64) uintptr {
	str := cfString(s)
	keys := [3]uintptr{kFontAttr, kFgFromCtx, kKernAttr}
	vals := [3]uintptr{font, kTrue, 0}
	n := 2
	if spacing != 0 {
		vals[2] = cfNumberCreate(0, kCFNumberFloat64Type, unsafe.Pointer(&spacing))
		n = 3
	}
	dict := cfDictionaryCreate(0, &keys[0], &vals[0], n, kKeyCB, kValCB)
	as := cfAttrStringCreate(0, str, dict)
	line := ctLineCreateWithAttrStr(as)
	cfRelease(as)
	cfRelease(dict)
	if n == 3 {
		cfRelease(vals[2])
	}
	cfRelease(str)
	return line
}

// ctLine owns a cached CTLine: the cache may drop it while another Surface still draws it.
type ctLine struct{ ref uintptr }

func newCTLine(ref uintptr) *ctLine {
	l := &ctLine{ref}
	runtime.AddCleanup(l, cfRelease, ref)
	return l
}

// Layout creates an uncached line for s; Release frees it. Used by benchmarks and measuring.
type Layout struct{ line uintptr }

// Layout lays out one line of s in f.
func (r *Renderer) Layout(s string, f gfx.Font) Layout {
	return Layout{r.newLine(s, r.font(f), f.Spacing)}
}

// Release frees the line.
func (l Layout) Release() { cfRelease(l.line) }

// Width is the typographic advance of the laid-out line in points.
func (l Layout) Width() float64 {
	var a, d, g float64
	return ctLineTypographicBounds(l.line, &a, &d, &g)
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

func (s *Surface) skeleton(str string, op *gfx.Op, a float64) {
	m := s.r.MeasureText(str, op.Font)
	h := skeletonH * op.Font.Size
	var bar gfx.Path
	bar.RoundRect(op.A, op.B-skeletonMid*op.Font.Size-h/2, m.Width, h, h/2)
	cgFillColor(s.ctx, op.Color.R, op.Color.G, op.Color.B, op.Color.A*skeletonInk*a)
	s.addPath(&bar)
	cgFillPath(s.ctx)
}

func (s *Surface) text(str string, op *gfx.Op) {
	alpha := 1.0
	if px := op.Font.Size * s.eff; s.r.opts.LOD && px < skeletonHi {
		alpha = math.Max(0, (px-skeletonLo)/(skeletonHi-skeletonLo))
		s.skeleton(str, op, 1-alpha)
		if alpha == 0 {
			return
		}
	}
	r := s.r
	font := r.font(op.Font)
	var line uintptr
	if r.opts.TextCache {
		k := lineKey{font, str, op.Font.Spacing}
		l, ok := r.lines.Load(k)
		if !ok {
			l, _ = r.lines.LoadOrStore(k, newCTLine(r.newLine(str, font, op.Font.Spacing)))
		}
		line = l.ref
		defer runtime.KeepAlive(l)
	} else {
		line = r.newLine(str, font, op.Font.Spacing)
		defer cfRelease(line)
	}
	ctx := s.ctx
	cgSaveGState(ctx)
	cgTranslate(ctx, op.A, op.B)
	cgScale(ctx, 1, -1) // the surface is y-down, glyphs are y-up
	cgFillColor(ctx, op.Color.R, op.Color.G, op.Color.B, op.Color.A*alpha)
	cgSetTextPos(ctx, 0, 0)
	ctLineDraw(line, ctx)
	cgRestoreGSt(ctx)
}
