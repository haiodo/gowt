// Package fonttest holds the font checks every gfx backend must pass, so that cg, cairo and d2d are
// held to the same behavior. A backend test calls them with its own renderer.
package fonttest

import (
	"math"
	"os"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

const inkW, inkH = 200, 50

// ink draws s at (10, 32) and returns the bounds of the pixels it touched, in whole pixels.
func ink(t *testing.T, r gfx.Renderer, draw func(*gfx.Recorder)) (minX, maxX int, pix []byte) {
	t.Helper()
	rec := gfx.NewRecorderWith(r)
	draw(rec)
	tg := r.NewTarget(inkW, inkH)
	defer tg.Close()
	tg.Begin(1, gfx.Point{}, gfx.Color{})
	tg.Draw([]gfx.Item{{Pic: rec.Finish(), M: gfx.Identity(), Alpha: 1}})
	pix = append([]byte(nil), tg.Pixels()...)
	minX, maxX = inkW, -1
	for y := 0; y < inkH; y++ {
		for x := 0; x < inkW; x++ {
			if pix[(y*inkW+x)*4+3] > 40 {
				minX, maxX = min(minX, x), max(maxX, x)
			}
		}
	}
	return minX, maxX, pix
}

// Tracking checks that Font.Spacing widens the measured line by Spacing per rune and that what is
// drawn follows the measurement: glyphs of a tracked line stand where single glyphs placed by the
// measured advances stand.
func Tracking(t *testing.T, r gfx.Renderer) {
	const sp = 3.0
	for _, s := range []string{"Hello", "Жуков", "a"} {
		f := gfx.Font{Size: 20}
		base := r.MeasureText(s, f)
		tr := r.MeasureText(s, gfx.Font{Size: 20, Spacing: sp})
		n := float64(len([]rune(s)))
		if d := tr.Width - base.Width - n*sp; math.Abs(d) > 0.02 {
			t.Errorf("%q: tracked width %v, plain %v, want %v more", s, tr.Width, base.Width, n*sp)
		}
		if tr.Ascent != base.Ascent || tr.Descent != base.Descent {
			t.Errorf("%q: tracking changed the vertical metrics: %+v %+v", s, tr, base)
		}
	}

	// "HHHH" has no kerning, so every glyph can be placed by its own measured advance.
	const s = "HHHH"
	f := gfx.Font{Size: 20}
	adv := r.MeasureText("H", f).Width
	_, _, tracked := ink(t, r, func(rec *gfx.Recorder) {
		rec.Text(s, 10, 32, gfx.Font{Size: 20, Spacing: sp}, gfx.Color{A: 1})
	})
	_, _, single := ink(t, r, func(rec *gfx.Recorder) {
		for i := 0; i < len(s); i++ {
			rec.Text("H", 10+float64(i)*(adv+sp), 32, f, gfx.Color{A: 1})
		}
	})
	_, _, plain := ink(t, r, func(rec *gfx.Recorder) { rec.Text(s, 10, 32, f, gfx.Color{A: 1}) })
	var maxd, vsPlain int
	for i := range tracked {
		maxd = max(maxd, int(math.Abs(float64(tracked[i])-float64(single[i]))))
		vsPlain = max(vsPlain, int(math.Abs(float64(tracked[i])-float64(plain[i]))))
	}
	if vsPlain < 128 {
		t.Errorf("tracking did not move the glyphs: max difference from the plain line %d", vsPlain)
	}
	if maxd > 48 {
		t.Errorf("tracked line differs from glyphs placed by the measured advance: max channel difference %d", maxd)
	}
	// the drawn line must not run past the measured one
	_, right, _ := ink(t, r, func(rec *gfx.Recorder) {
		rec.Text(s, 10, 32, gfx.Font{Size: 20, Spacing: sp}, gfx.Color{A: 1})
	})
	if w := r.MeasureText(s, gfx.Font{Size: 20, Spacing: sp}).Width; float64(right) > 10+w {
		t.Errorf("ink reaches x=%d, past the measured end %v", right, 10+w)
	}
}

// Faces checks that Family, Weight and Italic reach the face the backend has under that name. The
// Plex checks need IBM Plex Sans in GFX_FONT_DIR and skip without it. sansFamily is a family the
// platform always has with an italic face.
func Faces(t *testing.T, r gfx.Renderer, sansFamily string) {
	t.Run("plex", func(t *testing.T) {
		if os.Getenv("GFX_FONT_DIR") == "" {
			t.Skip("GFX_FONT_DIR is not set")
		}
		w := func(f gfx.Font) float64 {
			f.Size = 40
			return r.MeasureText("Hamburgefonstiv", f).Width
		}
		fam := func(weight int) gfx.Font { return gfx.Font{Family: "IBM Plex Sans", Weight: weight} }
		for _, c := range []struct {
			weight int
			name   string
		}{{400, "IBMPlexSans"}, {500, "IBMPlexSans-Medm"}, {600, "IBMPlexSans-SmBld"}, {700, "IBMPlexSans-Bold"}, {300, "IBMPlexSans-Light"}} {
			if got, want := w(fam(c.weight)), w(gfx.Font{Name: c.name}); got != want {
				t.Errorf("Family Plex weight %d measures %v, %s measures %v", c.weight, got, c.name, want)
			}
		}
		if w(fam(0)) != w(fam(400)) {
			t.Error("weight 0 is not 400")
		}
		none := w(gfx.Font{Family: "NoSuchFont-Probe", Weight: 400})
		if reg := w(fam(400)); reg == none {
			t.Error("Plex measures as the fallback font")
		}
		if !(w(fam(400)) < w(fam(600)) && w(fam(600)) < w(fam(700))) {
			t.Errorf("widths do not grow with weight: %v %v %v", w(fam(400)), w(fam(600)), w(fam(700)))
		}
	})
	t.Run("fallback", func(t *testing.T) {
		// an unknown family is the default sans at the same weight
		for _, wt := range []int{400, 700} {
			a := r.MeasureText("Hamburgefonstiv", gfx.Font{Family: "NoSuchFont-Probe", Weight: wt, Size: 30})
			b := r.MeasureText("Hamburgefonstiv", gfx.Font{Family: "AnotherMissing", Weight: wt, Size: 30})
			if a != b {
				t.Errorf("weight %d: two missing families differ: %+v %+v", wt, a, b)
			}
		}
		_, _, regular := ink(t, r, func(rec *gfx.Recorder) {
			rec.Text("Hamburgefonstiv", 10, 32, gfx.Font{Family: "NoSuchFont-Probe", Weight: 400, Size: 20}, gfx.Color{A: 1})
		})
		_, _, bold := ink(t, r, func(rec *gfx.Recorder) {
			rec.Text("Hamburgefonstiv", 10, 32, gfx.Font{Family: "NoSuchFont-Probe", Weight: 700, Size: 20}, gfx.Color{A: 1})
		})
		if string(regular) == string(bold) {
			t.Error("bold fallback draws like regular")
		}
	})
	t.Run("weight and italic", func(t *testing.T) {
		draw := func(f gfx.Font) []byte {
			f.Family, f.Size = sansFamily, 20
			_, _, p := ink(t, r, func(rec *gfx.Recorder) { rec.Text("Hamburgefonstiv", 10, 32, f, gfx.Color{A: 1}) })
			return p
		}
		reg, bold, ital := draw(gfx.Font{Weight: 400}), draw(gfx.Font{Weight: 700}), draw(gfx.Font{Weight: 400, Italic: true})
		if string(reg) == string(bold) {
			t.Errorf("%s: weight 700 draws like 400", sansFamily)
		}
		if string(reg) == string(ital) {
			t.Errorf("%s: italic draws like upright", sansFamily)
		}
	})
}
