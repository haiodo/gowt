//go:build linux

package cairo

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/haiodo/gowt/gfx"
)

const pangoScale = 1024

// RegisterFonts adds the font files of dir (or the one file) to this process's fontconfig and
// returns how many were new. Call it before the first Renderer lays text out.
func RegisterFonts(dir string) (int, error) {
	if err := bind(); err != nil {
		return 0, err
	}
	st, err := os.Stat(dir)
	if err != nil {
		return 0, err
	}
	files := []string{dir}
	if st.IsDir() {
		ents, err := os.ReadDir(dir)
		if err != nil {
			return 0, err
		}
		files = files[:0]
		for _, e := range ents {
			switch strings.ToLower(filepath.Ext(e.Name())) {
			case ".otf", ".ttf", ".ttc", ".otc":
				files = append(files, filepath.Join(dir, e.Name()))
			}
		}
	}
	fontsMu.Lock()
	defer fontsMu.Unlock()
	n := 0
	for _, f := range files {
		abs, err := filepath.Abs(f)
		if err != nil || fontsDone[abs] {
			continue
		}
		if fcAppFontAddFile(0, abs) != 0 {
			fontsDone[abs] = true
			n++
		}
	}
	if n > 0 && fcConfigChanged != nil {
		for fm := range fontMaps {
			fcConfigChanged(fm)
		}
	}
	return n, nil
}

var (
	fontsMu   sync.Mutex
	fontsDone = map[string]bool{}
	fontMaps  = map[uintptr]struct{}{}
)

// face is the Pango description a gfx font name resolves to.
type face struct {
	family string
	weight int32
	style  int32
}

// resolve maps a font name to a face: a PostScript name ("IBMPlexSans-SmBld", what the macOS backend
// takes) by fontconfig's postscriptname, else a family with a "-Bold" style suffix read as weight.
func resolve(name string) face {
	if f, ok := matchPostScript(name); ok {
		return f
	}
	fam, style, _ := strings.Cut(name, "-")
	f := face{family: fam, weight: 400}
	if style == "" {
		f.family = name
		return f
	}
	s := strings.ToLower(style)
	switch {
	case strings.Contains(s, "bold") && !strings.Contains(s, "semi") && !strings.Contains(s, "demi"):
		f.weight = 700
	case strings.Contains(s, "semi"), strings.Contains(s, "demi"), strings.Contains(s, "smbld"):
		f.weight = 600
	case strings.Contains(s, "medium"), strings.Contains(s, "medm"):
		f.weight = 500
	case strings.Contains(s, "light"):
		f.weight = 300
	}
	if strings.Contains(s, "italic") {
		f.style = 2
	} else if strings.Contains(s, "oblique") {
		f.style = 1
	}
	return f
}

func matchPostScript(name string) (face, bool) {
	pat := fcPatternCreate()
	defer fcPatternDestroy(pat)
	fcPatternAddString(pat, "postscriptname", name)
	fcConfigSubstitute(0, pat, 0)
	fcDefaultSubstitute(pat)
	var res int32
	m := fcFontMatch(0, pat, &res)
	if m == 0 {
		return face{}, false
	}
	defer fcPatternDestroy(m)
	var ps, fam uintptr
	if fcPatternGetString(m, "postscriptname", 0, &ps) != 0 || cstr(ps) != name || fcPatternGetString(m, "family", 0, &fam) != 0 {
		return face{}, false
	}
	f := face{family: cstr(fam), weight: 400}
	var w, sl int32
	if fcPatternGetInteger(m, "weight", 0, &w) == 0 {
		f.weight = pangoWeight(w)
	}
	if fcPatternGetInteger(m, "slant", 0, &sl) == 0 {
		switch {
		case sl >= 110:
			f.style = 1
		case sl >= 100:
			f.style = 2
		}
	}
	return f, true
}

// pangoWeight converts a fontconfig weight (0..215) to a Pango one (100..1000) by the points both
// scales define.
func pangoWeight(w int32) int32 {
	pts := [][2]int32{{0, 100}, {40, 200}, {50, 300}, {80, 400}, {100, 500}, {180, 600}, {200, 700}, {205, 800}, {210, 900}, {215, 1000}}
	for i := 1; i < len(pts); i++ {
		if w <= pts[i][0] {
			a, b := pts[i-1], pts[i]
			return a[1] + (b[1]-a[1])*(w-a[0])/(b[0]-a[0])
		}
	}
	return 1000
}

type fontKey struct {
	name string
	size float64
}

// desc returns the PangoFontDescription for f, made once per name and size.
func (r *Renderer) desc(f gfx.Font) uintptr {
	k := fontKey{f.Name, f.Size}
	if v, ok := r.descs.Load(k); ok {
		return v.(uintptr)
	}
	fc := resolve(f.Name)
	d := fdNew()
	fdSetFamily(d, fc.family)
	fdSetWeight(d, fc.weight)
	fdSetStyle(d, fc.style)
	fdSetAbsSize(d, f.Size*pangoScale)
	v, _ := r.descs.LoadOrStore(k, d)
	return v.(uintptr)
}

// newContext makes a Pango context with the options that keep text identical at every scale:
// grayscale antialiasing, no hinting, fractional advances.
func (r *Renderer) newContext() uintptr {
	ctx := fmCreateContext(r.fm)
	ctxSetFontOptions(ctx, r.fopts)
	if ctxSetRound != nil {
		ctxSetRound(ctx, 0)
	}
	return ctx
}

// MeasureText returns the Pango extents of s: the logical width of the line, the baseline as the
// ascent and the rest of the logical height as the descent. Results are cached per font and string.
func (r *Renderer) MeasureText(s string, f gfx.Font) gfx.TextMetrics {
	k := lineKey{f, s}
	if v, ok := r.metrics.Load(k); ok {
		return v.(gfx.TextMetrics)
	}
	r.pmu.Lock()
	l := layoutNew(r.mctx)
	layoutSetFont(l, r.desc(f))
	layoutSetText(l, s, -1)
	var ink, logical pangoRect
	layoutExtents(l, &ink, &logical)
	base := layoutBaseline(l)
	objectUnref(l)
	r.pmu.Unlock()
	m := gfx.TextMetrics{
		Width:   float64(logical.w) / pangoScale,
		Ascent:  float64(base) / pangoScale,
		Descent: float64(logical.h-base) / pangoScale,
	}
	r.metrics.Store(k, m)
	return m
}

type lineKey struct {
	f gfx.Font
	s string
}

// LOD skeleton text: a bar in place of glyphs under skeletonHi device pixels, cross-faded down to
// skeletonLo. The numbers are those of gfx/cg.
const (
	skeletonHi  = 6.0
	skeletonLo  = 4.0
	skeletonH   = 0.5
	skeletonMid = 0.3
	skeletonInk = 0.45
)

func (s *Target) skeleton(str string, op *gfx.Op, a float64) {
	m := s.r.MeasureText(str, op.Font)
	h := skeletonH * op.Font.Size
	var bar gfx.Path
	bar.RoundRect(op.A, op.B-skeletonMid*op.Font.Size-h/2, m.Width, h, h/2)
	setSourceRGBA(s.cr, op.Color.R, op.Color.G, op.Color.B, op.Color.A*skeletonInk*a)
	s.addPath(&bar)
	setFillRule(s.cr, fillRuleWinding)
	fill(s.cr)
}

func (s *Target) text(str string, op *gfx.Op) {
	alpha := 1.0
	if px := op.Font.Size * s.eff; s.r.opts.LOD && px < skeletonHi {
		alpha = math.Max(0, (px-skeletonLo)/(skeletonHi-skeletonLo))
		s.skeleton(str, op, 1-alpha)
		if alpha == 0 {
			return
		}
	}
	r := s.r
	cr := s.cr
	r.pmu.Lock()
	defer r.pmu.Unlock()
	ctx := r.newContext()
	ctxUpdate(cr, ctx)
	l := layoutNew(ctx)
	layoutSetFont(l, r.desc(op.Font))
	layoutSetText(l, str, -1)
	base := float64(layoutBaseline(l)) / pangoScale
	save(cr)
	setSourceRGBA(cr, op.Color.R, op.Color.G, op.Color.B, op.Color.A*alpha)
	moveTo(cr, op.A, op.B-base)
	showLayout(cr, l)
	restore(cr)
	objectUnref(l)
	objectUnref(ctx)
}
