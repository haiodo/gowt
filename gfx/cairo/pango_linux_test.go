//go:build linux

package cairo

import (
	"os"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

// TestMeasureAgainstPango lays the same text out through a second font map, with no gfx code in
// between, and compares the logical extents and the baseline.
func TestMeasureAgainstPango(t *testing.T) {
	r := newRenderer(t, Options{})
	fm := fmNew()
	defer objectUnref(fm)
	ctx := fmCreateContext(fm)
	defer objectUnref(ctx)
	ctxSetFontOptions(ctx, r.fopts)
	if ctxSetRound != nil {
		ctxSetRound(ctx, 0)
	}
	for _, f := range []gfx.Font{{Name: "DejaVu Sans", Size: 13}, {Name: "DejaVuSans-Bold", Size: 24.5}, {Name: "DejaVu Serif", Size: 9}} {
		for _, s := range []string{"Hello", "Жуков Ёлкин", "iiii WWWW", ""} {
			l := layoutNew(ctx)
			fc := resolve(f.Name)
			d := fdNew()
			fdSetFamily(d, fc.family)
			fdSetWeight(d, fc.weight)
			fdSetStyle(d, fc.style)
			fdSetAbsSize(d, f.Size*pangoScale)
			layoutSetFont(l, d)
			layoutSetText(l, s, -1)
			var ink, logical pangoRect
			layoutExtents(l, &ink, &logical)
			base := layoutBaseline(l)
			objectUnref(l)
			m := r.MeasureText(s, f)
			if m.Width != float64(logical.w)/pangoScale || m.Ascent != float64(base)/pangoScale || m.Ascent+m.Descent != float64(logical.h)/pangoScale {
				t.Errorf("%v %q: got %+v, pango logical %+v baseline %d", f, s, m, logical, base)
			}
			if s != "" && (ink.x < logical.x-pangoScale*3 || ink.x+ink.w > logical.x+logical.w+pangoScale*3) {
				t.Errorf("%v %q: ink %+v outside logical %+v", f, s, ink, logical)
			}
		}
	}
}

func TestRegisteredFontResolvesByPostScriptName(t *testing.T) {
	if os.Getenv("GFX_FONT_DIR") == "" {
		t.Skip("GFX_FONT_DIR is not set")
	}
	r := newRenderer(t, Options{})
	w := func(name string) float64 {
		return r.MeasureText("Hamburgefonstiv", gfx.Font{Name: name, Size: 40}).Width
	}
	reg, semi, bold, none := w("IBMPlexSans"), w("IBMPlexSans-SmBld"), w("IBMPlexSans-Bold"), w("NoSuchFont-Probe")
	if reg == none || !(reg < semi && semi < bold) {
		t.Errorf("plex widths: regular %v semibold %v bold %v, fallback %v", reg, semi, bold, none)
	}
}
