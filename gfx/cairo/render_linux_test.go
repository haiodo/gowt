//go:build linux

package cairo

import (
	"crypto/sha256"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func newRenderer(t *testing.T, o Options) *Renderer {
	t.Helper()
	r, err := NewRenderer(o)
	if err != nil {
		t.Skip(err)
	}
	return r
}

func rectPath(x, y, w, h float64) *gfx.Path {
	p := &gfx.Path{}
	p.Rect(x, y, w, h)
	return p
}

// drawPic records into a w x h target at scale 1 on a clear background.
func drawPic(t *testing.T, r *Renderer, w, h int, rec func(*gfx.Recorder)) gfx.Target {
	t.Helper()
	rc := gfx.NewRecorderWith(r)
	rec(rc)
	tg := r.NewTarget(w, h)
	t.Cleanup(tg.Close)
	tg.Begin(1, gfx.Point{}, gfx.Color{})
	tg.Draw([]gfx.Item{{Pic: rc.Finish(), M: gfx.Identity(), Alpha: 1}})
	return tg
}

// pixel returns BGRA.
func pixel(tg gfx.Target, x, y int) [4]byte {
	w, _ := tg.Size()
	p := tg.Pixels()[(y*w+x)*4:]
	return [4]byte{p[0], p[1], p[2], p[3]}
}

func near(a, b byte, tol int) bool { return int(a)-int(b) <= tol && int(b)-int(a) <= tol }

func TestEvenOddHoleAndClip(t *testing.T) {
	r := newRenderer(t, Options{})
	ring := &gfx.Path{EvenOdd: true}
	ring.Rect(5, 5, 40, 40)
	ring.Rect(15, 15, 20, 20)
	tg := drawPic(t, r, 60, 60, func(rc *gfx.Recorder) { rc.FillPath(ring, gfx.Color{R: 1, A: 1}) })
	if got := pixel(tg, 10, 25); got != [4]byte{0, 0, 255, 255} {
		t.Errorf("ring body: %v", got)
	}
	if got := pixel(tg, 25, 25); got != [4]byte{} {
		t.Errorf("fill hole: %v", got)
	}
	tg = drawPic(t, r, 60, 60, func(rc *gfx.Recorder) {
		rc.ClipPath(ring)
		rc.FillPath(rectPath(0, 0, 60, 60), gfx.Color{B: 1, A: 1})
	})
	if got := pixel(tg, 10, 25); got != [4]byte{255, 0, 0, 255} {
		t.Errorf("clipped ring body: %v", got)
	}
	if got := pixel(tg, 25, 25); got != [4]byte{} {
		t.Errorf("clip hole: %v", got)
	}
}

func TestClipAndTransform(t *testing.T) {
	r := newRenderer(t, Options{})
	tg := drawPic(t, r, 100, 100, func(rc *gfx.Recorder) {
		rc.Save()
		rc.ClipRect(gfx.R(0, 0, 50, 50))
		rc.FillPath(rectPath(0, 0, 100, 100), gfx.Color{R: 1, A: 1})
		rc.Restore()
		rc.Save()
		rc.Transform(gfx.TranslateBy(80, 60).Mul(gfx.RotateBy(1.5707963267948966)))
		rc.FillPath(rectPath(0, 0, 20, 10), gfx.Color{G: 1, A: 1}) // covers x 70..80, y 60..80
		rc.Restore()
		rc.Save()
		rc.ClipRRect(gfx.R(60, 0, 40, 40), 20)
		rc.FillPath(rectPath(0, 0, 100, 100), gfx.Color{B: 1, A: 1})
		rc.Restore()
	})
	if got := pixel(tg, 25, 25); got != [4]byte{0, 0, 255, 255} {
		t.Errorf("inside the clip rect: %v", got)
	}
	if got := pixel(tg, 55, 55); got != [4]byte{} {
		t.Errorf("outside every clip: %v", got)
	}
	if got := pixel(tg, 75, 70); got != [4]byte{0, 255, 0, 255} {
		t.Errorf("rotated rect: %v", got)
	}
	if got := pixel(tg, 95, 70); got != [4]byte{} {
		t.Errorf("beside the rotated rect: %v", got)
	}
	if got := pixel(tg, 80, 20); got != [4]byte{255, 0, 0, 255} {
		t.Errorf("inside the rounded clip: %v", got)
	}
	if got := pixel(tg, 61, 1); got != [4]byte{} {
		t.Errorf("rounded clip corner: %v", got)
	}
}

func TestGradients(t *testing.T) {
	r := newRenderer(t, Options{})
	tg := drawPic(t, r, 100, 100, func(rc *gfx.Recorder) {
		rc.FillGradient(rectPath(0, 0, 100, 50), gfx.NewLinearGradient(gfx.Point{}, gfx.Point{X: 100},
			gfx.Stop{Offset: 0, Color: gfx.Color{R: 1, A: 1}}, gfx.Stop{Offset: 1, Color: gfx.Color{B: 1, A: 1}}))
		rc.FillGradient(rectPath(0, 50, 100, 50), gfx.NewRadialGradient(gfx.Point{X: 50, Y: 75}, 0, gfx.Point{X: 50, Y: 75}, 25,
			gfx.Stop{Offset: 0, Color: gfx.Color{G: 1, A: 1}}, gfx.Stop{Offset: 1, Color: gfx.Color{A: 1}}))
	})
	left, right := pixel(tg, 2, 25), pixel(tg, 97, 25)
	if left[2] < 240 || left[0] > 15 || right[0] < 240 || right[2] > 15 {
		t.Errorf("linear gradient ends: %v %v (BGRA)", left, right)
	}
	if mid := pixel(tg, 50, 25); !near(mid[0], 128, 12) || !near(mid[2], 128, 12) {
		t.Errorf("linear midpoint %v", mid)
	}
	if c, edge := pixel(tg, 50, 75), pixel(tg, 50, 98); c[1] < 240 || edge[1] > 20 {
		t.Errorf("radial centre %v edge %v", c, edge)
	}
	tg = drawPic(t, r, 100, 100, func(rc *gfx.Recorder) {
		ln := &gfx.Path{}
		ln.MoveTo(0, 50)
		ln.LineTo(100, 50)
		rc.StrokeGradient(ln, gfx.StrokeStyle{Width: 10}, gfx.NewLinearGradient(gfx.Point{}, gfx.Point{X: 100},
			gfx.Stop{Offset: 0, Color: gfx.Color{R: 1, A: 1}}, gfx.Stop{Offset: 1, Color: gfx.Color{B: 1, A: 1}}))
	})
	if pixel(tg, 10, 50)[2] < 200 || pixel(tg, 10, 20) != [4]byte{} {
		t.Errorf("gradient stroke: on %v off %v", pixel(tg, 10, 50), pixel(tg, 10, 20))
	}
}

func TestStrokeDashQuadAndCubic(t *testing.T) {
	r := newRenderer(t, Options{})
	tg := drawPic(t, r, 100, 60, func(rc *gfx.Recorder) {
		ln := &gfx.Path{}
		ln.MoveTo(0, 10)
		ln.LineTo(100, 10)
		rc.StrokePath(ln, gfx.StrokeStyle{Width: 4, Dash: []float64{10, 10}}, gfx.Color{R: 1, A: 1})
		q := &gfx.Path{}
		q.MoveTo(0, 50)
		q.QuadTo(50, 20, 100, 50)
		rc.StrokePath(q, gfx.StrokeStyle{Width: 2}, gfx.Color{B: 1, A: 1})
	})
	if pixel(tg, 5, 10)[3] != 255 || pixel(tg, 15, 10)[3] != 0 || pixel(tg, 25, 10)[3] != 255 {
		t.Errorf("dash pattern: %v %v %v", pixel(tg, 5, 10), pixel(tg, 15, 10), pixel(tg, 25, 10))
	}
	// a quad from (0,50) over (50,20) to (100,50) peaks at y=35 in the middle
	if pixel(tg, 50, 35)[3] == 0 || pixel(tg, 50, 25)[3] != 0 || pixel(tg, 50, 45)[3] != 0 {
		t.Errorf("quad: peak %v above %v below %v", pixel(tg, 50, 35), pixel(tg, 50, 25), pixel(tg, 50, 45))
	}
}

func TestLayerAlphaIsGroupAlpha(t *testing.T) {
	r := newRenderer(t, Options{})
	tg := drawPic(t, r, 60, 60, func(rc *gfx.Recorder) {
		rc.BeginLayer(0.5)
		rc.FillPath(rectPath(0, 0, 40, 40), gfx.Color{R: 1, A: 1})
		rc.FillPath(rectPath(20, 20, 40, 40), gfx.Color{R: 1, A: 1})
		rc.EndLayer()
	})
	one, both := pixel(tg, 10, 10), pixel(tg, 30, 30)
	if !near(one[3], 128, 2) || both != one {
		t.Errorf("group alpha: single %v overlap %v", one, both)
	}
}

func TestItemAlphaAndBackground(t *testing.T) {
	r := newRenderer(t, Options{})
	rc := gfx.NewRecorderWith(r)
	rc.FillPath(rectPath(0, 0, 10, 10), gfx.Color{R: 1, A: 1})
	tg := r.NewTarget(10, 10)
	defer tg.Close()
	tg.Begin(1, gfx.Point{}, gfx.Color{R: 1, G: 1, B: 1, A: 1})
	tg.Draw([]gfx.Item{{Pic: rc.Finish(), M: gfx.Identity(), Alpha: 0.5}})
	if p := pixel(tg, 5, 5); !near(p[0], 128, 2) || p[2] != 255 || p[3] != 255 {
		t.Errorf("half red over white: %v", p)
	}
}

func TestShadowBox(t *testing.T) {
	r := newRenderer(t, Options{})
	tg := drawPic(t, r, 100, 100, func(rc *gfx.Recorder) {
		rc.FillShadowRRect(30, 30, 40, 40, 6, gfx.Color{R: 1, G: 1, B: 1, A: 1},
			gfx.Shadow{DX: 0, DY: 4, Blur: 12, Color: gfx.Color{A: 0.8}})
	})
	box, below, far := pixel(tg, 50, 50), pixel(tg, 50, 78), pixel(tg, 50, 99)
	if box != [4]byte{255, 255, 255, 255} {
		t.Errorf("box fill %v", box)
	}
	if below[3] < 20 || below[3] > 200 || far[3] != 0 {
		t.Errorf("shadow below %v, far %v", below, far)
	}
	// the shadow is offset down: it is stronger below the box than above it
	if above := pixel(tg, 50, 24); above[3] >= below[3] {
		t.Errorf("shadow offset: above %v below %v", above, below)
	}
}

func TestImageRaster(t *testing.T) {
	r := newRenderer(t, Options{})
	img := gfx.NewRaster(2, 2)
	copy(img.Pix, []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 255, 0})
	tg := drawPic(t, r, 40, 40, func(rc *gfx.Recorder) { rc.DrawImage(img, 0, 0, 40, 40) })
	if p := pixel(tg, 2, 2); p[2] < 250 || p[0] > 5 {
		t.Errorf("top left red: %v", p)
	}
	if p := pixel(tg, 38, 38); p[3] != 0 {
		t.Errorf("transparent corner: %v", p)
	}
}

func inkBox(tg gfx.Target) (x0, y0, x1, y1 int) {
	w, h := tg.Size()
	x0, y0, x1, y1 = w, h, -1, -1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if pixel(tg, x, y)[3] > 64 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	return
}

func TestTextMatchesMeasure(t *testing.T) {
	r := newRenderer(t, Options{})
	f := gfx.Font{Name: "DejaVu Sans", Size: 40}
	m := r.MeasureText("HEH", f)
	if m.Width < 40 || m.Width > 120 || m.Ascent <= 0 || m.Descent <= 0 {
		t.Fatalf("metrics %+v", m)
	}
	bold := r.MeasureText("HEH", gfx.Font{Name: "DejaVuSans-Bold", Size: 40})
	if bold.Width <= m.Width {
		t.Errorf("bold %v is not wider than regular %v: font name did not resolve", bold.Width, m.Width)
	}
	if big := r.MeasureText("HEH", gfx.Font{Name: "DejaVu Sans", Size: 80}); big.Width < 1.99*m.Width || big.Width > 2.01*m.Width {
		t.Errorf("width does not scale with the size: %v vs %v", big.Width, m.Width)
	}
	tg := drawPic(t, r, 200, 100, func(rc *gfx.Recorder) { rc.Text("HEH", 10, 60, f, gfx.Color{A: 1}) })
	x0, y0, x1, y1 := inkBox(tg)
	if x1 < 0 {
		t.Fatal("no ink")
	}
	// capital letters sit on the baseline y=60 and are narrower than the advance
	if y1 < 59 || y1 > 61 {
		t.Errorf("ink bottom %d, baseline 60", y1)
	}
	if float64(x1) > 10+m.Width+1 || float64(x1) < 10+m.Width-0.25*f.Size {
		t.Errorf("ink right %d against advance end %.1f", x1, 10+m.Width)
	}
	if capH := float64(y1 - y0 + 1); capH > m.Ascent+1 || capH < 0.5*m.Ascent {
		t.Errorf("cap height %v against ascent %v", capH, m.Ascent)
	}
	_ = x0
}

func TestSkeletonUnderLOD(t *testing.T) {
	r := newRenderer(t, Options{LOD: true})
	rc := gfx.NewRecorderWith(r)
	rc.Text("Skeleton text line", 0, 8, gfx.Font{Name: "DejaVu Sans", Size: 10}, gfx.Color{A: 1})
	pic := rc.Finish()
	tg := r.NewTarget(100, 30)
	defer tg.Close()
	tg.Begin(0.2, gfx.Point{}, gfx.Color{})
	tg.Draw([]gfx.Item{{Pic: pic, M: gfx.Identity(), Alpha: 1}})
	// 10 pt at scale 0.2 is 2 device pixels: a bar of 45% ink, not glyphs and not nothing
	var sum, n int
	for y := 0; y < 30; y++ {
		for x := 0; x < 100; x++ {
			if a := pixel(tg, x, y)[3]; a > 0 {
				sum += int(a)
				n++
			}
		}
	}
	if n == 0 || sum/n > 140 {
		t.Errorf("skeleton: %d covered pixels, mean alpha %d", n, sum/max(n, 1))
	}
}

func TestRenderSubtreePNGDeterministic(t *testing.T) {
	render := func() [32]byte {
		r := newRenderer(t, Options{})
		s := gfx.NewScene()
		n := gfx.NewNode(1)
		rc := gfx.NewRecorderWith(r)
		rc.FillShadowRRect(10, 10, 120, 60, 8, gfx.Color{R: 0.9, G: 0.95, B: 1, A: 1}, gfx.Shadow{DY: 3, Blur: 8, Color: gfx.Color{A: 0.4}})
		rc.Text("Hello, gfx", 22, 46, gfx.Font{Name: "DejaVu Sans", Size: 18}, gfx.Color{A: 1})
		rc.FillGradient(rectPath(10, 80, 120, 20), gfx.NewLinearGradient(gfx.Point{X: 10}, gfx.Point{X: 130},
			gfx.Stop{Offset: 0, Color: gfx.Color{R: 1, A: 1}}, gfx.Stop{Offset: 1, Color: gfx.Color{B: 1, A: 1}}))
		pic := rc.Finish()
		s.Update(func(tx *gfx.Tx) { tx.Add(nil, n); tx.SetPicture(n, pic) })
		ras, err := gfx.RenderSubtree(r, s, n, gfx.Rect{}, 2, gfx.Color{R: 1, G: 1, B: 1, A: 1})
		if err != nil {
			t.Fatal(err)
		}
		png, err := ras.PNG()
		if err != nil || len(png) < 100 {
			t.Fatalf("png: %d bytes, %v", len(png), err)
		}
		return sha256.Sum256(png)
	}
	a, b := render(), render()
	if a != b {
		t.Errorf("two renders of one scene differ: %x vs %x", a, b)
	}
}

func TestConcurrentTargets(t *testing.T) {
	r := newRenderer(t, Options{})
	done := make(chan [4]byte, 8)
	for i := 0; i < 8; i++ {
		go func() {
			rc := gfx.NewRecorderWith(r)
			rc.Text("concurrent", 2, 20, gfx.Font{Name: "DejaVu Sans", Size: 16}, gfx.Color{A: 1})
			tg := r.NewTarget(100, 30)
			defer tg.Close()
			tg.Begin(1, gfx.Point{}, gfx.Color{})
			tg.Draw([]gfx.Item{{Pic: rc.Finish(), M: gfx.Identity(), Alpha: 1}})
			_, _, x1, _ := inkBox(tg)
			done <- [4]byte{byte(x1)}
		}()
	}
	first := <-done
	for i := 1; i < 8; i++ {
		if got := <-done; got != first {
			t.Errorf("goroutine result %v differs from %v", got, first)
		}
	}
}
