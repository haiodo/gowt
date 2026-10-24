//go:build darwin

package cg

import (
	"math"
	"testing"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/synth"
)

func pixel(s *Surface, x, y int) [4]byte {
	p := s.Pixels()
	i := (y*s.W + x) * 4
	return [4]byte{p[i], p[i+1], p[i+2], p[i+3]} // B G R A
}

func TestReplayFillStrokeText(t *testing.T) {
	r := NewRenderer(Options{TextCache: true})
	s := r.NewSurface(200, 100)
	defer s.Close()
	rec := gfx.NewRecorder()
	box := &gfx.Path{}
	box.Rect(10, 10, 80, 40)
	rec.FillPath(box, gfx.Color{R: 1, A: 1})
	line := &gfx.Path{}
	line.MoveTo(100, 70)
	line.LineTo(180, 70)
	rec.StrokePath(line, gfx.StrokeStyle{Width: 4}, gfx.Color{B: 1, A: 1})
	rec.Text("Hello", 100, 30, gfx.Font{Name: "Helvetica Neue", Size: 20}, gfx.Color{G: 1, A: 1})
	pic := rec.Finish()

	s.Begin(1, 0, 0, gfx.Color{})
	s.Replay(pic, 0, 0)
	if got := pixel(s, 50, 30); got != [4]byte{0, 0, 255, 255} {
		t.Errorf("red fill (y down): BGRA %v", got)
	}
	if got := pixel(s, 140, 70); got != [4]byte{255, 0, 0, 255} {
		t.Errorf("blue stroke: BGRA %v", got)
	}
	if got := pixel(s, 5, 5); got != [4]byte{} {
		t.Errorf("background must stay transparent: %v", got)
	}
	green := 0
	for y := 10; y < 35; y++ {
		for x := 100; x < 160; x++ {
			if pixel(s, x, y)[1] > 128 {
				green++
			}
		}
	}
	if green < 50 {
		t.Errorf("text drew %d green pixels above its baseline", green)
	}
	if w := r.Layout("Hello", gfx.Font{Name: "Helvetica Neue", Size: 20}); w.Width() < 30 || w.Width() > 70 {
		t.Errorf("text width %v", w.Width())
	}
}

// The sprite shadow must stay close to the live blur shadow.
func TestShadowSpriteMatchesBlur(t *testing.T) {
	rec := gfx.NewRecorder()
	rec.FillShadowRRect(40, 40, 120, 60, 8, gfx.Color{R: 1, G: 1, B: 1, A: 1}, gfx.Shadow{DY: 3, Blur: 8, Color: gfx.Color{A: 0.4}})
	pic := rec.Finish()
	for _, scale := range []float64{1, 2} {
		var out [2][]byte
		for i, sprite := range []bool{false, true} {
			r := NewRenderer(Options{ShadowSprite: sprite})
			s := r.NewSurface(int(200*scale), int(140*scale))
			s.Begin(scale, 0, 0, gfx.Color{})
			s.Replay(pic, 0, 0)
			out[i] = append([]byte(nil), s.Pixels()...)
			s.Close()
		}
		if a := out[0][(int(100*scale)*int(200*scale)+int(100*scale))*4+3]; a < 20 {
			t.Fatalf("scale %v: no shadow below the box (alpha %d)", scale, a)
		}
		var maxd, sum int
		for i := range out[0] {
			d := int(math.Abs(float64(out[0][i]) - float64(out[1][i])))
			maxd = max(maxd, d)
			sum += d
		}
		t.Logf("scale %v: max channel difference %d, mean %.3f", scale, maxd, float64(sum)/float64(len(out[0])))
		if maxd > 24 {
			t.Errorf("scale %v: sprite shadow differs from blur by %d", scale, maxd)
		}
	}
}

func TestSceneRender(t *testing.T) {
	d := synth.New(300, 3)
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true})
	s := r.NewSurface(1000, 700)
	defer s.Close()
	s.Begin(1, 0, 0, gfx.Color{R: 1, G: 1, B: 1, A: 1})
	for _, n := range d.Scene.Query(gfx.R(0, 0, 1000, 700), nil) {
		s.Replay(n.Pic, n.X, n.Y)
	}
	nonWhite := 0
	for i := 0; i < len(s.Pixels()); i += 4 {
		if s.Pixels()[i] != 255 {
			nonWhite++
		}
	}
	if nonWhite < 20000 {
		t.Fatalf("scene drew only %d non-white pixels", nonWhite)
	}
}
