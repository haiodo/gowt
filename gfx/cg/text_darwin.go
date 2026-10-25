//go:build darwin

package cg

import (
	"math"
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

func (r *Renderer) font(f gfx.Font) uintptr {
	k := fontKey{f.Name, f.Size}
	if v, ok := r.fonts.Load(k); ok {
		return v.(uintptr)
	}
	name := cfString(f.Name)
	ref := ctFontCreateWithName(name, f.Size, 0)
	cfRelease(name)
	v, loaded := r.fonts.LoadOrStore(k, ref)
	if loaded {
		cfRelease(ref)
	}
	return v.(uintptr)
}

// newLine lays out one line of text in font; the color comes from the context at draw time,
// so a line is reusable for any fill color. The caller releases it.
func (r *Renderer) newLine(s string, font uintptr) uintptr {
	str := cfString(s)
	keys := [2]uintptr{kFontAttr, kFgFromCtx}
	vals := [2]uintptr{font, kTrue}
	dict := cfDictionaryCreate(0, &keys[0], &vals[0], 2, kKeyCB, kValCB)
	as := cfAttrStringCreate(0, str, dict)
	line := ctLineCreateWithAttrStr(as)
	cfRelease(as)
	cfRelease(dict)
	cfRelease(str)
	return line
}

// Layout creates an uncached line for s; Release frees it. Used by benchmarks and measuring.
type Layout struct{ line uintptr }

// Layout lays out one line of s in f.
func (r *Renderer) Layout(s string, f gfx.Font) Layout { return Layout{r.newLine(s, r.font(f))} }

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
		k := lineKey{font, str}
		if v, ok := r.lines.Load(k); ok {
			line = v.(uintptr)
		} else {
			l := r.newLine(str, font)
			v, loaded := r.lines.LoadOrStore(k, l)
			if loaded {
				cfRelease(l)
			}
			line = v.(uintptr)
		}
	} else {
		line = r.newLine(str, font)
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
