//go:build darwin

package cg

import (
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

func (r *Renderer) Layout(s string, f gfx.Font) Layout { return Layout{r.newLine(s, r.font(f))} }
func (l Layout) Release()                              { cfRelease(l.line) }

// Width is the typographic advance of the laid-out line in points.
func (l Layout) Width() float64 {
	var a, d, g float64
	return ctLineTypographicBounds(l.line, &a, &d, &g)
}

func (s *Surface) text(str string, op *gfx.Op) {
	if s.r.opts.LOD && op.Font.Size*s.eff < 4 {
		return
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
	cgFillColor(ctx, op.Color.R, op.Color.G, op.Color.B, op.Color.A)
	cgSetTextPos(ctx, 0, 0)
	ctLineDraw(line, ctx)
	cgRestoreGSt(ctx)
}
