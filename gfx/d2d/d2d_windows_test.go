//go:build windows

package d2d

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/haiodo/gowt/gfx"
)

var (
	white = gfx.Color{R: 1, G: 1, B: 1, A: 1}
	red   = gfx.Color{R: 1, A: 1}
	green = gfx.Color{G: 1, A: 1}
	blue  = gfx.Color{B: 1, A: 1}
	black = gfx.Color{A: 1}
)

var shared *Renderer

func TestMain(m *testing.M) {
	r, err := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true, FontDir: os.Getenv("GFX_FONT_DIR")})
	if err != nil {
		panic(err)
	}
	shared = r
	code := m.Run()
	r.Close()
	os.Exit(code)
}

func rectPath(x, y, w, h float64) *gfx.Path {
	p := &gfx.Path{}
	p.Rect(x, y, w, h)
	return p
}

func record(f func(r *gfx.Recorder)) *gfx.Picture {
	rc := gfx.NewRecorderWith(shared)
	f(rc)
	return rc.Finish()
}

// render draws the picture at scale into a w x h pixel raster with scene origin (0, 0).
func render(t testing.TB, w, h int, scale float64, bg gfx.Color, f func(r *gfx.Recorder)) *gfx.Raster {
	t.Helper()
	tg := shared.NewTarget(w, h)
	defer tg.Close()
	tg.Begin(scale, gfx.Point{}, bg)
	tg.Draw([]gfx.Item{{Pic: record(f), M: gfx.Identity(), Alpha: 1}})
	return gfx.FromBGRA(tg.Pixels(), w, h)
}

func px(r *gfx.Raster, x, y int) [4]byte {
	i := (y*r.W + x) * 4
	return [4]byte{r.Pix[i], r.Pix[i+1], r.Pix[i+2], r.Pix[i+3]}
}

func near(a [4]byte, b [4]byte, tol int) bool {
	for i := range a {
		if d := int(a[i]) - int(b[i]); d > tol || d < -tol {
			return false
		}
	}
	return true
}

func want(t *testing.T, what string, got, exp [4]byte, tol int) {
	t.Helper()
	if !near(got, exp, tol) {
		t.Errorf("%s: got %v, want %v (+-%d)", what, got, exp, tol)
	}
}

func TestDevice(t *testing.T) {
	t.Logf("software (WARP) device: %v", shared.Software())
	if !shared.Software() {
		t.Error("the default renderer must use WARP")
	}
}

func TestFillAndBackground(t *testing.T) {
	r := render(t, 40, 30, 1, white, func(r *gfx.Recorder) { r.FillPath(rectPath(10, 10, 20, 10), red) })
	want(t, "inside", px(r, 15, 15), [4]byte{255, 0, 0, 255}, 0)
	want(t, "outside", px(r, 2, 2), [4]byte{255, 255, 255, 255}, 0)
	want(t, "left edge exact", px(r, 10, 15), [4]byte{255, 0, 0, 255}, 0)
	want(t, "left of edge", px(r, 9, 15), [4]byte{255, 255, 255, 255}, 0)
	// a transparent background stays transparent
	r = render(t, 20, 20, 1, gfx.Color{}, func(r *gfx.Recorder) { r.FillPath(rectPath(0, 0, 10, 10), green) })
	want(t, "transparent bg", px(r, 15, 15), [4]byte{}, 0)
	want(t, "fill on transparent", px(r, 5, 5), [4]byte{0, 255, 0, 255}, 0)
}

func TestScaleAndOrigin(t *testing.T) {
	tg := shared.NewTarget(40, 40)
	defer tg.Close()
	tg.Begin(2, gfx.Point{X: 100, Y: 50}, white)
	tg.Draw([]gfx.Item{{Pic: record(func(r *gfx.Recorder) { r.FillPath(rectPath(105, 55, 5, 5), blue) }), M: gfx.Identity(), Alpha: 1}})
	r := gfx.FromBGRA(tg.Pixels(), 40, 40)
	want(t, "inside", px(r, 12, 12), [4]byte{0, 0, 255, 255}, 0)
	want(t, "before", px(r, 9, 12), [4]byte{255, 255, 255, 255}, 0)
	want(t, "after", px(r, 20, 12), [4]byte{255, 255, 255, 255}, 0)
}

func TestClipAndTransform(t *testing.T) {
	s := render(t, 100, 100, 1, gfx.Color{}, func(r *gfx.Recorder) {
		r.Save()
		r.ClipRect(gfx.R(0, 0, 50, 50))
		r.FillPath(rectPath(0, 0, 100, 100), red)
		r.Restore()
		r.Save()
		r.Transform(gfx.TranslateBy(80, 60).Mul(gfx.RotateBy(math.Pi / 2)))
		r.FillPath(rectPath(0, 0, 20, 10), green) // covers x 70..80, y 60..80
		r.Restore()
		r.Save()
		var tri gfx.Path
		tri.MoveTo(60, 0)
		tri.LineTo(100, 0)
		tri.LineTo(100, 40)
		tri.Close()
		r.ClipPath(&tri)
		r.FillPath(rectPath(0, 0, 100, 100), blue)
		r.Restore()
	})
	want(t, "inside the clip rect", px(s, 25, 25), [4]byte{255, 0, 0, 255}, 0)
	want(t, "inside the triangle", px(s, 90, 10), [4]byte{0, 0, 255, 255}, 0)
	want(t, "outside every clip", px(s, 55, 55), [4]byte{}, 0)
	want(t, "rotated rect", px(s, 75, 70), [4]byte{0, 255, 0, 255}, 0)
	want(t, "beside the rotated rect", px(s, 95, 70), [4]byte{}, 0)
	want(t, "below the hypotenuse", px(s, 65, 30), [4]byte{}, 0)
}

func TestRotatedRectClip(t *testing.T) {
	s := render(t, 100, 100, 1, gfx.Color{}, func(r *gfx.Recorder) {
		r.Save()
		r.Translate(50, 50)
		r.Rotate(math.Pi / 4)
		r.ClipRect(gfx.R(-20, -20, 40, 40)) // a diamond with corners 28 away from the center
		r.Translate(-50, -50)
		r.FillPath(rectPath(0, 0, 100, 100), red)
		r.Restore()
	})
	want(t, "center", px(s, 50, 50), [4]byte{255, 0, 0, 255}, 0)
	want(t, "diamond tip", px(s, 50, 26), [4]byte{255, 0, 0, 255}, 0)
	want(t, "square corner is clipped", px(s, 33, 33), [4]byte{}, 0)
}

func TestEvenOdd(t *testing.T) {
	for _, eo := range []bool{false, true} {
		s := render(t, 60, 60, 1, white, func(r *gfx.Recorder) {
			p := &gfx.Path{EvenOdd: eo}
			p.Rect(0, 0, 60, 60)
			p.Rect(20, 20, 20, 20) // same winding direction as the outer one
			r.FillPath(p, red)
		})
		hole := px(s, 30, 30)
		if eo {
			want(t, "even-odd hole", hole, [4]byte{255, 255, 255, 255}, 0)
		} else {
			want(t, "winding fills the hole", hole, [4]byte{255, 0, 0, 255}, 0)
		}
		want(t, "ring", px(s, 10, 10), [4]byte{255, 0, 0, 255}, 0)
	}
}

func TestGradients(t *testing.T) {
	s := render(t, 100, 40, 1, white, func(r *gfx.Recorder) {
		r.FillGradient(rectPath(0, 0, 100, 20), gfx.NewLinearGradient(gfx.Point{}, gfx.Point{X: 100},
			gfx.Stop{Offset: 0, Color: black}, gfx.Stop{Offset: 1, Color: white}))
		r.FillGradient(rectPath(0, 20, 100, 20), gfx.NewRadialGradient(gfx.Point{X: 50, Y: 30}, 0, gfx.Point{X: 50, Y: 30}, 20,
			gfx.Stop{Offset: 0, Color: red}, gfx.Stop{Offset: 1, Color: blue}))
	})
	want(t, "linear start", px(s, 1, 10), [4]byte{3, 3, 3, 255}, 3)
	want(t, "linear mid", px(s, 50, 10), [4]byte{128, 128, 128, 255}, 4)
	want(t, "linear end", px(s, 98, 10), [4]byte{250, 250, 250, 255}, 5)
	want(t, "radial center", px(s, 50, 30), [4]byte{255, 0, 0, 255}, 15)
	want(t, "radial beyond radius", px(s, 5, 30), [4]byte{0, 0, 255, 255}, 2)
	if a, b := px(s, 20, 10), px(s, 70, 10); a[0] >= b[0] {
		t.Errorf("gradient not increasing: %v %v", a, b)
	}
}

func TestStroke(t *testing.T) {
	s := render(t, 100, 40, 1, white, func(r *gfx.Recorder) {
		var p gfx.Path
		p.MoveTo(10, 10)
		p.LineTo(90, 10)
		r.StrokePath(&p, gfx.StrokeStyle{Width: 4}, black)
		var q gfx.Path
		q.MoveTo(10, 30)
		q.LineTo(90, 30)
		r.StrokePath(&q, gfx.StrokeStyle{Width: 4, Dash: []float64{10, 10}}, black)
	})
	want(t, "solid", px(s, 50, 10), [4]byte{0, 0, 0, 255}, 0)
	want(t, "solid edge", px(s, 50, 7), [4]byte{255, 255, 255, 255}, 0)
	want(t, "dash on", px(s, 15, 30), [4]byte{0, 0, 0, 255}, 0)
	want(t, "dash off", px(s, 25, 30), [4]byte{255, 255, 255, 255}, 0)
	want(t, "dash on again", px(s, 35, 30), [4]byte{0, 0, 0, 255}, 0)
}

func TestStrokeCapsAndJoins(t *testing.T) {
	s := render(t, 100, 60, 1, white, func(r *gfx.Recorder) {
		var p gfx.Path
		p.MoveTo(20, 15)
		p.LineTo(80, 15)
		r.StrokePath(&p, gfx.StrokeStyle{Width: 10, Cap: gfx.CapSquare}, black)
		var q gfx.Path
		q.MoveTo(20, 45)
		q.LineTo(80, 45)
		r.StrokePath(&q, gfx.StrokeStyle{Width: 10, Cap: gfx.CapButt}, black)
	})
	want(t, "square cap reaches past the end", px(s, 16, 15), [4]byte{0, 0, 0, 255}, 0)
	want(t, "butt cap stops at the end", px(s, 16, 45), [4]byte{255, 255, 255, 255}, 0)
}

func TestAlphaLayer(t *testing.T) {
	s := render(t, 60, 40, 1, white, func(r *gfx.Recorder) {
		r.BeginLayer(0.5)
		r.FillPath(rectPath(0, 0, 40, 40), red)
		r.FillPath(rectPath(20, 0, 40, 40), red) // overlaps: a layer composites once
		r.EndLayer()
	})
	one := px(s, 10, 10)
	want(t, "half red on white", one, [4]byte{255, 128, 128, 255}, 2)
	if px(s, 30, 10) != one {
		t.Errorf("overlap differs inside one layer: %v vs %v", px(s, 30, 10), one)
	}
	// per-item alpha
	tg := shared.NewTarget(20, 20)
	defer tg.Close()
	tg.Begin(1, gfx.Point{}, white)
	tg.Draw([]gfx.Item{{Pic: record(func(r *gfx.Recorder) { r.FillPath(rectPath(0, 0, 20, 20), blue) }), M: gfx.Identity(), Alpha: 0.25}})
	r := gfx.FromBGRA(tg.Pixels(), 20, 20)
	want(t, "item alpha", px(r, 5, 5), [4]byte{191, 191, 255, 255}, 2)
}

func TestItemClips(t *testing.T) {
	tg := shared.NewTarget(60, 60)
	defer tg.Close()
	tg.Begin(1, gfx.Point{}, white)
	pic := record(func(r *gfx.Recorder) { r.FillPath(rectPath(0, 0, 60, 60), red) })
	var p gfx.Path
	p.Ellipse(30, 30, 20, 20)
	tg.Draw([]gfx.Item{
		{Pic: pic, M: gfx.Identity(), Alpha: 1, Clips: []gfx.Clip{{M: gfx.Identity(), Kind: gfx.ClipRect, Rect: gfx.R(0, 0, 40, 60)}, {M: gfx.Identity(), Kind: gfx.ClipPath, Path: &p}}},
	})
	r := gfx.FromBGRA(tg.Pixels(), 60, 60)
	want(t, "inside both", px(r, 30, 30), [4]byte{255, 0, 0, 255}, 0)
	want(t, "inside the circle, outside the rect", px(r, 45, 30), [4]byte{255, 255, 255, 255}, 0)
	want(t, "outside the circle", px(r, 12, 12), [4]byte{255, 255, 255, 255}, 0)
}

func TestShadow(t *testing.T) {
	s := render(t, 120, 100, 1, white, func(r *gfx.Recorder) {
		r.FillShadowRRect(30, 20, 60, 40, 6, white, gfx.Shadow{DX: 0, DY: 8, Blur: 10, Color: gfx.Color{A: 0.5}})
		r.FillShadowRRect(5, 70, 12, 12, 2, white, gfx.Shadow{DX: 0, DY: 3, Blur: 6, Color: gfx.Color{A: 0.5}}) // small box
	})
	want(t, "box fill", px(s, 60, 40), [4]byte{255, 255, 255, 255}, 0)
	below := px(s, 60, 68)
	if below[0] > 245 || below[0] < 120 {
		t.Errorf("shadow just below the box: %v", below)
	}
	if far := px(s, 60, 95); far != [4]byte{255, 255, 255, 255} {
		t.Errorf("far below the shadow: %v", far)
	}
	if side, top := px(s, 60, 14), px(s, 60, 68); side[0] <= top[0] {
		t.Errorf("shadow must be heavier below (dy=8) than above: top %v bottom %v", side, top)
	}
	if small := px(s, 11, 87); small[0] > 250 {
		t.Errorf("small box has no shadow: %v", small)
	}
}

func TestImage(t *testing.T) {
	img := gfx.NewRaster(2, 2)
	copy(img.Pix, []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 255, 128})
	s := render(t, 40, 40, 1, white, func(r *gfx.Recorder) { r.DrawImage(img, 10, 10, 20, 20) })
	want(t, "outside", px(s, 5, 5), [4]byte{255, 255, 255, 255}, 0)
	// bilinear scaling smears the 2x2 texels at the centre: only the quadrant interiors are checked
	if p := px(s, 12, 12); p[0] < 200 || p[1] > 60 {
		t.Errorf("top left should be red: %v", p)
	}
	if p := px(s, 27, 12); p[1] < 200 || p[0] > 60 {
		t.Errorf("top right should be green: %v", p)
	}
	if p := px(s, 12, 27); p[2] < 200 || p[0] > 60 {
		t.Errorf("bottom left should be blue: %v", p)
	}
}

func inkCount(r *gfx.Raster, x0, y0, x1, y1 int) int {
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if p := px(r, x, y); p[0] < 128 {
				n++
			}
		}
	}
	return n
}

func TestTextDrawsAtTheMeasuredPlace(t *testing.T) {
	font := gfx.Font{Name: "Segoe UI", Size: 20}
	m := shared.MeasureText("Hamburg", font)
	s := render(t, 200, 60, 1, white, func(r *gfx.Recorder) { r.Text("Hamburg", 10, 40, font, black) })
	if inkCount(s, 10, 40-int(m.Ascent)-1, 10+int(m.Width)+2, 40+int(m.Descent)+1) < 100 {
		t.Error("no ink where the text should be")
	}
	if n := inkCount(s, 10+int(m.Width)+4, 0, 200, 60); n != 0 {
		t.Errorf("ink %d past the measured width %.1f", n, m.Width)
	}
	if n := inkCount(s, 0, 0, 200, 40-int(m.Ascent)-2); n != 0 {
		t.Errorf("ink %d above the ascent %.1f", n, m.Ascent)
	}
	// text is grayscale antialiased: no colour fringes
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			if p := px(s, x, y); p[0] != p[1] || p[1] != p[2] {
				t.Fatalf("coloured pixel %v at %d,%d: text must be grayscale", p, x, y)
			}
		}
	}
}

func TestTextWeightAndFamily(t *testing.T) {
	if os.Getenv("GFX_FONT_DIR") == "" {
		t.Skip("GFX_FONT_DIR not set")
	}
	const probe = "Hamburgefonstiv 0123"
	w := func(name string) float64 { return shared.MeasureText(probe, gfx.Font{Name: name, Size: 100}).Width }
	reg, bold, light, semi := w("IBMPlexSans"), w("IBMPlexSans-Bold"), w("IBMPlexSans-Light"), w("IBMPlexSans-SmBld")
	nofont := w("NoSuchFont-Probe")
	t.Logf("regular %.1f bold %.1f light %.1f semibold %.1f fallback %.1f", reg, bold, light, semi, nofont)
	if reg == nofont {
		t.Error("IBMPlexSans measures like the fallback: the font was not registered")
	}
	if !(bold > reg && reg > light) || semi <= reg || semi >= bold {
		t.Errorf("weights do not order: light %.1f regular %.1f semibold %.1f bold %.1f", light, reg, semi, bold)
	}
	if w("IBM Plex Sans") != reg {
		t.Errorf("family name and PostScript name differ: %.1f %.1f", w("IBM Plex Sans"), reg)
	}
}

func TestLODSkeleton(t *testing.T) {
	font := gfx.Font{Name: "Segoe UI", Size: 14}
	// at scale 0.2 the text is 2.8 device pixels: a bar instead of glyphs
	s := render(t, 100, 30, 0.2, white, func(r *gfx.Recorder) { r.Text("Skeleton text line", 0, 14, font, black) })
	bar := 0
	for x := 0; x < s.W; x++ {
		for y := 0; y < s.H; y++ {
			if p := px(s, x, y); p[0] < 250 {
				bar++
				break
			}
		}
	}
	if bar < 5 {
		t.Errorf("skeleton bar missing: %d columns with ink", bar)
	}
}

// TestMeasureAgainstTheFontFile checks the DirectWrite width against the advances in the file: four
// digits of a tabular font have no kerning, so the width is four advances scaled to the size.
func TestMeasureAgainstTheFontFile(t *testing.T) {
	dir := os.Getenv("GFX_FONT_DIR")
	if dir == "" {
		t.Skip("GFX_FONT_DIR not set")
	}
	data, err := os.ReadFile(filepath.Join(dir, "IBMPlexSans-Regular.otf"))
	if err != nil {
		t.Skip(err)
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	var b sfnt.Buffer
	gi, err := f.GlyphIndex(&b, '0')
	if err != nil {
		t.Fatal(err)
	}
	const size = 64
	adv, err := f.GlyphAdvance(&b, gi, fixed.I(size), font.HintingNone)
	if err != nil {
		t.Fatal(err)
	}
	want := 4 * float64(adv) / 64
	got := shared.MeasureText("0000", gfx.Font{Name: "IBMPlexSans", Size: size}).Width
	if math.Abs(got-want) > 0.05 {
		t.Errorf("width of 0000 at %d: DirectWrite %.3f, font file %.3f", size, got, want)
	}
}

func TestMeasureMatchesDirectWrite(t *testing.T) {
	// widths scale with size and are additive for a string without kerning pairs
	f := func(n float64) gfx.Font { return gfx.Font{Name: "Segoe UI", Size: n} }
	a, b := shared.MeasureText("Wide text", f(10)), shared.MeasureText("Wide text", f(40))
	if math.Abs(b.Width/a.Width-4) > 0.05 {
		t.Errorf("width does not scale with size: %.2f vs %.2f", a.Width, b.Width)
	}
	if b.Ascent <= 0 || b.Descent <= 0 || math.Abs(b.Ascent/a.Ascent-4) > 0.05 {
		t.Errorf("ascent/descent: %+v %+v", a, b)
	}
	if e := shared.MeasureText("", f(12)); e.Width != 0 {
		t.Errorf("empty string: %+v", e)
	}
	sp := shared.MeasureText("a b", f(12))
	nosp := shared.MeasureText("ab", f(12))
	if sp.Width <= nosp.Width {
		t.Errorf("a space must add width: %v vs %v", sp.Width, nosp.Width)
	}
	tr := shared.MeasureText("ab  ", f(12)) // trailing spaces count, as the typographic advance of CoreText does
	if tr.Width <= nosp.Width {
		t.Errorf("trailing space must count: %v vs %v", tr.Width, nosp.Width)
	}
}

// galleryPic draws every op once; the hash test and the dumped PNG use it.
func galleryPic(r *gfx.Recorder) {
	r.FillPath(rectPath(0, 0, 400, 300), gfx.Color{R: 0.96, G: 0.96, B: 0.97, A: 1})
	r.FillShadowRRect(20, 20, 120, 80, 10, white, gfx.Shadow{DX: 0, DY: 6, Blur: 14, Color: gfx.Color{A: 0.35}})
	star := &gfx.Path{EvenOdd: true}
	for i := 0; i < 5; i++ {
		a := float64(i) * 4 * math.Pi / 5
		x, y := 210+40*math.Sin(a), 60-40*math.Cos(a)
		if i == 0 {
			star.MoveTo(x, y)
		} else {
			star.LineTo(x, y)
		}
	}
	star.Close()
	r.FillPath(star, gfx.Color{R: 0.9, G: 0.3, B: 0.2, A: 1})
	r.FillGradient(rectPath(280, 20, 100, 80), gfx.NewLinearGradient(gfx.Point{X: 280, Y: 20}, gfx.Point{X: 380, Y: 100},
		gfx.Stop{Offset: 0, Color: gfx.Color{R: 0.2, G: 0.4, B: 0.9, A: 1}}, gfx.Stop{Offset: 1, Color: gfx.Color{R: 0.9, G: 0.2, B: 0.6, A: 1}}))
	var circle gfx.Path
	circle.Ellipse(70, 170, 40, 40)
	r.FillGradient(&circle, gfx.NewRadialGradient(gfx.Point{X: 60, Y: 160}, 0, gfx.Point{X: 70, Y: 170}, 40,
		gfx.Stop{Offset: 0, Color: white}, gfx.Stop{Offset: 1, Color: gfx.Color{R: 0.1, G: 0.5, B: 0.3, A: 1}}))
	var wave gfx.Path
	wave.MoveTo(130, 190)
	wave.CubicTo(150, 120, 190, 240, 210, 150)
	r.StrokePath(&wave, gfx.StrokeStyle{Width: 5, Cap: gfx.CapRound, Join: gfx.JoinRound}, gfx.Color{R: 0.1, G: 0.1, B: 0.5, A: 1})
	var dash gfx.Path
	dash.MoveTo(130, 210)
	dash.LineTo(260, 210)
	r.StrokePath(&dash, gfx.StrokeStyle{Width: 3, Dash: []float64{8, 5}}, black)
	r.Save()
	r.ClipRRect(gfx.R(280, 120, 100, 70), 16)
	r.FillPath(rectPath(270, 110, 60, 60), gfx.Color{R: 0.2, G: 0.7, B: 0.9, A: 1})
	r.FillPath(rectPath(330, 150, 70, 60), gfx.Color{R: 0.95, G: 0.7, B: 0.1, A: 1})
	r.Restore()
	r.BeginLayer(0.5)
	r.FillPath(rectPath(20, 230, 60, 50), red)
	r.FillPath(rectPath(50, 245, 60, 50), blue)
	r.EndLayer()
	r.Save()
	r.Translate(300, 260)
	r.Rotate(-0.3)
	r.FillPath(rectPath(-30, -15, 60, 30), gfx.Color{R: 0.3, G: 0.3, B: 0.3, A: 1})
	r.Restore()
	img := gfx.NewRaster(4, 4)
	for i := range img.Pix {
		img.Pix[i] = byte(i * 16)
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	r.DrawImage(img, 140, 240, 40, 40)
	r.Text("Segoe UI 14 regular", 200, 240, gfx.Font{Name: "Segoe UI", Size: 14}, black)
	r.Text("IBM Plex Sans Bold 18", 200, 262, gfx.Font{Name: "IBMPlexSans-Bold", Size: 18}, black)
	r.Text("Helvetica Neue 12 (fallback)", 200, 282, gfx.Font{Name: "HelveticaNeue", Size: 12}, gfx.Color{R: 0.4, G: 0.4, B: 0.4, A: 1})
}

func renderGallery(t testing.TB) *gfx.Raster {
	return render(t, 800, 600, 2, white, galleryPic)
}

func TestGallery(t *testing.T) {
	r := renderGallery(t)
	png, err := r.PNG()
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("GFX_OUT"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "gallery.png"), png, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256(png)
	t.Logf("gallery png sha256 %s (%d bytes)", hex.EncodeToString(sum[:]), len(png))
}

func TestWICPNG(t *testing.T) {
	r := renderGallery(t)
	data, err := EncodePNG(r)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	got := gfx.RasterFrom(img)
	if got.W != r.W || got.H != r.H || !bytes.Equal(got.Pix, r.Pix) {
		t.Error("the WIC PNG decodes to other pixels than the raster")
	}
	again, err := EncodePNG(r)
	if err != nil || !bytes.Equal(again, data) {
		t.Errorf("WIC PNG is not deterministic: err %v", err)
	}
}

// TestHashAcrossRuns keeps the hash of the gallery PNG in the temp directory: the first run writes it,
// every later run on the machine must produce the same bytes.
func TestHashAcrossRuns(t *testing.T) {
	data, err := renderGallery(t).PNG()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	file := filepath.Join(os.TempDir(), "gfx-d2d-gallery.sha256")
	prev, err := os.ReadFile(file)
	if err != nil {
		t.Logf("first run: recording %s", got)
		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if string(prev) != got {
		t.Errorf("gallery hash changed between runs: was %s, now %s", prev, got)
	}
}

func TestDeterministic(t *testing.T) {
	a, b := renderGallery(t), renderGallery(t)
	pa, _ := a.PNG()
	pb, _ := b.PNG()
	if sha256.Sum256(pa) != sha256.Sum256(pb) {
		t.Error("two renders of one scene differ")
	}
	// a second renderer (a new device) draws the same pixels
	r2, err := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true, FontDir: os.Getenv("GFX_FONT_DIR")})
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	rc := gfx.NewRecorderWith(r2)
	galleryPic(rc)
	c, err := gfx.RenderPicture(r2, rc.Finish(), 1, white)
	if err != nil {
		t.Fatal(err)
	}
	d, err := gfx.RenderPicture(shared, record(galleryPic), 1, white)
	if err != nil {
		t.Fatal(err)
	}
	pc, _ := c.PNG()
	pd, _ := d.PNG()
	if sha256.Sum256(pc) != sha256.Sum256(pd) {
		t.Error("two devices draw different pixels")
	}
}

func TestConcurrentTargets(t *testing.T) {
	want, _ := renderGallery(t).PNG()
	ref := sha256.Sum256(want)
	errs := make(chan string, 8)
	for g := 0; g < 8; g++ {
		go func() {
			for i := 0; i < 4; i++ {
				p, _ := renderGallery(t).PNG()
				if sha256.Sum256(p) != ref {
					errs <- "concurrent render differs"
					return
				}
			}
			errs <- ""
		}()
	}
	for g := 0; g < 8; g++ {
		if e := <-errs; e != "" {
			t.Error(e)
		}
	}
}

func TestRenderSubtree(t *testing.T) {
	scene := gfx.NewScene()
	frame, box := gfx.NewNode(1), gfx.NewNode(2)
	scene.Update(func(tx *gfx.Tx) {
		tx.Add(nil, frame)
		tx.Add(frame, box)
		tx.SetPicture(frame, record(func(r *gfx.Recorder) { r.FillPath(rectPath(0, 0, 100, 60), blue) }))
		tx.SetPicture(box, record(func(r *gfx.Recorder) { r.FillPath(rectPath(0, 0, 20, 20), red) }))
		tx.Move(frame, 50, 50)
		tx.Move(box, 10, 10)
		tx.ClipRRect(frame, gfx.R(0, 0, 100, 60), 8)
	})
	ras, err := gfx.RenderSubtree(shared, scene, frame, gfx.Rect{}, 2, white)
	if err != nil {
		t.Fatal(err)
	}
	if ras.W != 200 || ras.H != 120 {
		t.Fatalf("size %dx%d", ras.W, ras.H)
	}
	want(t, "child over parent", px(ras, 40, 40), [4]byte{255, 0, 0, 255}, 0)
	want(t, "parent", px(ras, 100, 80), [4]byte{0, 0, 255, 255}, 0)
	want(t, "rounded corner clipped", px(ras, 0, 0), [4]byte{255, 255, 255, 255}, 0)
	if _, err := ras.PNG(); err != nil {
		t.Error(err)
	}
}

func BenchmarkGallery(b *testing.B) {
	pic := record(galleryPic)
	tg := shared.NewTarget(800, 600)
	defer tg.Close()
	items := []gfx.Item{{Pic: pic, M: gfx.Identity(), Alpha: 1}}
	for i := 0; i < b.N; i++ {
		tg.Begin(2, gfx.Point{}, white)
		tg.Draw(items)
	}
}
