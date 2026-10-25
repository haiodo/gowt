//go:build darwin

package cg

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func rectPath(x, y, w, h float64) *gfx.Path {
	p := &gfx.Path{}
	p.Rect(x, y, w, h)
	return p
}

func near(a, b byte, tol int) bool { return int(a)-int(b) <= tol && int(b)-int(a) <= tol }

func drawPic(t *testing.T, w, h int, rec func(r *gfx.Recorder)) *Surface {
	t.Helper()
	r := NewRenderer(Options{})
	s := r.NewSurface(w, h)
	t.Cleanup(s.Close)
	rc := gfx.NewRecorder()
	rec(rc)
	s.Begin(1, 0, 0, gfx.Color{})
	s.Replay(rc.Finish(), 0, 0)
	return s
}

func TestClipAndTransform(t *testing.T) {
	red := gfx.Color{R: 1, A: 1}
	s := drawPic(t, 100, 100, func(r *gfx.Recorder) {
		r.Save()
		r.ClipRect(gfx.R(0, 0, 50, 50))
		r.FillPath(rectPath(0, 0, 100, 100), red)
		r.Restore()
		// a general transform (rotation by 90 degrees about a point) goes through CGContextConcatCTM
		r.Save()
		r.Transform(gfx.TranslateBy(80, 60).Mul(gfx.RotateBy(1.5707963267948966)))
		r.FillPath(rectPath(0, 0, 20, 10), gfx.Color{G: 1, A: 1}) // covers x 70..80, y 60..80
		r.Restore()
		r.Save()
		var tri gfx.Path
		tri.MoveTo(60, 0)
		tri.LineTo(100, 0)
		tri.LineTo(100, 40)
		tri.Close()
		r.ClipPath(&tri)
		r.FillPath(rectPath(0, 0, 100, 100), gfx.Color{B: 1, A: 1})
		r.Restore()
	})
	if p := pixel(s, 25, 25); p != [4]byte{0, 0, 255, 255} {
		t.Errorf("inside the clip rect: %v", p)
	}
	if p := pixel(s, 75, 25); p[3] != 255 || p[0] != 255 { // blue inside the triangle
		if p[3] != 0 {
			t.Errorf("inside the triangle: %v", p)
		}
	}
	if p := pixel(s, 55, 55); p != [4]byte{} {
		t.Errorf("outside every clip must stay clear: %v", p)
	}
	if p := pixel(s, 75, 70); p != [4]byte{0, 255, 0, 255} {
		t.Errorf("rotated rect: %v", p)
	}
	if p := pixel(s, 95, 70); p != [4]byte{} {
		t.Errorf("beside the rotated rect: %v", p)
	}
	if p := pixel(s, 70, 5); p[3] != 0 { // inside the triangle at x=70, y<=10
		if p[0] != 255 {
			t.Errorf("triangle pixel %v", p)
		}
	}
	if p := pixel(s, 65, 30); p != [4]byte{} {
		t.Errorf("below the triangle hypotenuse: %v", p)
	}
}

func TestGradients(t *testing.T) {
	s := drawPic(t, 100, 100, func(r *gfx.Recorder) {
		r.FillGradient(rectPath(0, 0, 100, 50), gfx.NewLinearGradient(gfx.Point{}, gfx.Point{X: 100},
			gfx.Stop{Offset: 0, Color: gfx.Color{R: 1, A: 1}}, gfx.Stop{Offset: 1, Color: gfx.Color{B: 1, A: 1}}))
		r.FillGradient(rectPath(0, 50, 100, 50), gfx.NewRadialGradient(gfx.Point{X: 50, Y: 75}, 0, gfx.Point{X: 50, Y: 75}, 25,
			gfx.Stop{Offset: 0, Color: gfx.Color{G: 1, A: 1}}, gfx.Stop{Offset: 1, Color: gfx.Color{A: 1}}))
	})
	left, right := pixel(s, 2, 25), pixel(s, 97, 25)
	if left[2] < 240 || left[0] > 15 || right[0] < 240 || right[2] > 15 {
		t.Errorf("linear gradient ends: %v %v (BGRA)", left, right)
	}
	mid := pixel(s, 50, 25)
	if !near(mid[0], 128, 12) || !near(mid[2], 128, 12) {
		t.Errorf("linear midpoint %v", mid)
	}
	c, edge := pixel(s, 50, 75), pixel(s, 50, 98)
	if c[1] < 240 || edge[1] > 20 {
		t.Errorf("radial centre %v edge %v", c, edge)
	}
	// stroke with a gradient only covers the stroke
	s2 := drawPic(t, 100, 100, func(r *gfx.Recorder) {
		ln := &gfx.Path{}
		ln.MoveTo(0, 50)
		ln.LineTo(100, 50)
		r.StrokeGradient(ln, gfx.StrokeStyle{Width: 10}, gfx.NewLinearGradient(gfx.Point{}, gfx.Point{X: 100},
			gfx.Stop{Offset: 0, Color: gfx.Color{R: 1, A: 1}}, gfx.Stop{Offset: 1, Color: gfx.Color{B: 1, A: 1}}))
	})
	if pixel(s2, 10, 50)[2] < 200 || pixel(s2, 10, 20) != [4]byte{} {
		t.Errorf("gradient stroke: on %v off %v", pixel(s2, 10, 50), pixel(s2, 10, 20))
	}
}

func TestLayerAlphaIsGroupAlpha(t *testing.T) {
	s := drawPic(t, 60, 60, func(r *gfx.Recorder) {
		r.BeginLayer(0.5)
		r.FillPath(rectPath(0, 0, 40, 40), gfx.Color{R: 1, A: 1})
		r.FillPath(rectPath(20, 20, 40, 40), gfx.Color{R: 1, A: 1})
		r.EndLayer()
	})
	// the overlap of two opaque shapes inside one layer is as opaque as the rest of the layer
	a, b := pixel(s, 10, 10), pixel(s, 30, 30)
	if !near(a[3], 128, 3) || !near(b[3], 128, 3) {
		t.Errorf("layer alpha: single %v overlap %v", a, b)
	}
}

func TestImageRaster(t *testing.T) {
	img := gfx.NewRaster(2, 1)
	copy(img.Pix, []byte{255, 0, 0, 255, 0, 0, 255, 255}) // red, blue
	s := drawPic(t, 40, 20, func(r *gfx.Recorder) { r.DrawImage(img, 0, 0, 40, 20) })
	if l, rr := pixel(s, 5, 10), pixel(s, 35, 10); l[2] < 240 || rr[0] < 240 {
		t.Errorf("image halves: %v %v (BGRA)", l, rr)
	}
}

func TestDrawItemsAlphaAndClip(t *testing.T) {
	r := NewRenderer(Options{})
	s := r.NewSurface(100, 100)
	defer s.Close()
	rec := gfx.NewRecorder()
	rec.FillPath(rectPath(0, 0, 50, 50), gfx.Color{R: 1, A: 1})
	pic := rec.Finish()
	s.Begin(2, 10, 10, gfx.Color{})
	s.Draw([]gfx.Item{{
		Pic: pic, M: gfx.TranslateBy(10, 10), Alpha: 0.5,
		Clips: []gfx.Clip{{M: gfx.Identity(), Kind: gfx.ClipRect, Rect: gfx.R(10, 10, 20, 20)}},
	}})
	// scene (10,10) is pixel (0,0); the clip keeps scene 10..30, i.e. 40 pixels
	if p := pixel(s, 20, 20); !near(p[3], 128, 3) || p[2] < 120 {
		t.Errorf("inside: %v", p)
	}
	if p := pixel(s, 60, 60); p != [4]byte{} {
		t.Errorf("clipped away: %v", p)
	}
}

func TestMeasureTextAndRenderSubtree(t *testing.T) {
	r := NewRenderer(Options{TextCache: true})
	small := r.MeasureText("Hello", gfx.Font{Name: "Helvetica Neue", Size: 10})
	big := r.MeasureText("Hello", gfx.Font{Name: "Helvetica Neue", Size: 20})
	if small.Width < 15 || big.Width < small.Width*1.8 || small.Ascent <= 0 || small.Descent <= 0 {
		t.Errorf("metrics %+v %+v", small, big)
	}
	rec := gfx.NewRecorderWith(r)
	rec.Text("Hello", 0, 20, gfx.Font{Name: "Helvetica Neue", Size: 20}, gfx.Color{A: 1})
	pic := rec.Finish()
	if b := pic.Bounds; b.W() < big.Width || b.Min.Y > 20-big.Ascent+0.01 || b.Max.Y < 20+big.Descent-0.01 {
		t.Errorf("text bounds %v for metrics %+v", b, big)
	}

	sc := gfx.NewScene(gfx.R(-100, -100, 1000, 1000), 128)
	frame, kid := gfx.NewNode(1), gfx.NewNode(2)
	sc.Update(func(tx *gfx.Tx) {
		tx.Add(nil, frame)
		tx.Move(frame, 20, 20)
		tx.Add(frame, kid)
		tx.Move(kid, 10, 10)
		box := rectPath(0, 0, 30, 30)
		br := gfx.NewRecorder()
		br.FillPath(box, gfx.Color{G: 1, A: 1})
		tx.SetPicture(kid, br.Finish())
	})
	ras, err := gfx.RenderSubtree(r, sc, frame, gfx.Rect{}, 2, gfx.Color{R: 1, G: 1, B: 1, A: 1})
	if err != nil {
		t.Fatal(err)
	}
	if ras.W != 60 || ras.H != 60 {
		t.Fatalf("raster %dx%d", ras.W, ras.H)
	}
	data, err := ras.PNG()
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if rr, g, b, _ := im.At(30, 30).RGBA(); rr>>8 > 5 || g>>8 < 250 || b>>8 > 5 {
		t.Errorf("subtree pixel %v %v %v", rr>>8, g>>8, b>>8)
	}
}
