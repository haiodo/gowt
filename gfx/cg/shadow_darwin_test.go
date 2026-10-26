//go:build darwin

package cg

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

var (
	shRed   = gfx.Color{R: 1, A: 1}
	shBlack = gfx.Color{A: 1}
)

func TestFillShadowPath(t *testing.T) {
	sh := gfx.Shadow{DX: 30, Color: shBlack}
	s := drawPic(t, 100, 60, func(r *gfx.Recorder) {
		r.FillShadowPath(gfx.EllipsePath(gfx.R(10, 10, 20, 20)), shRed, sh)
	})
	if p := pixel(s, 20, 20); p != [4]byte{0, 0, 255, 255} {
		t.Errorf("the shape: %v", p)
	}
	if p := pixel(s, 50, 20); p != [4]byte{0, 0, 0, 255} {
		t.Errorf("the shadow of the ellipse: %v", p)
	}
	if p := pixel(s, 41, 11); p[3] != 0 {
		t.Errorf("the shadow must follow the ellipse, not its box: %v", p)
	}
}

func TestShadowBlur(t *testing.T) {
	s := drawPic(t, 100, 100, func(r *gfx.Recorder) {
		r.FillShadowPath(gfx.EllipsePath(gfx.R(30, 30, 40, 40)), shRed, gfx.Shadow{Blur: 10, Color: shBlack})
	})
	if p := pixel(s, 75, 50); p[3] == 0 || p[3] == 255 {
		t.Errorf("a blurred edge is partly covered: %v", p)
	}
}

func TestShadowFollowsTransform(t *testing.T) {
	// y up on screen: the local shadow offset DY=+20 must land above the shape
	s := drawPic(t, 100, 100, func(r *gfx.Recorder) {
		r.Translate(0, 100)
		r.Scale(1, -1)
		r.FillShadowPath(gfx.RectPath(gfx.R(10, 10, 20, 20)), shRed, gfx.Shadow{DY: 20, Color: shBlack})
	})
	if p := pixel(s, 20, 55); p != [4]byte{0, 0, 0, 255} { // local y 45 is device y 55
		t.Errorf("mirrored shadow: %v", p)
	}
	if p := pixel(s, 20, 95); p[3] != 0 {
		t.Errorf("the shadow went the wrong way: %v", p)
	}
	s = drawPic(t, 100, 100, func(r *gfx.Recorder) {
		r.Translate(0, 100)
		r.Scale(1, -1)
		r.FillShadowRRect(10, 10, 20, 20, 0, shRed, gfx.Shadow{DY: 20, Color: shBlack})
	})
	if p := pixel(s, 20, 55); p[3] != 255 {
		t.Errorf("mirrored box shadow: %v", p)
	}
}

func TestStrokeShadowPath(t *testing.T) {
	var line gfx.Path
	line.MoveTo(10, 10)
	line.LineTo(60, 10)
	s := drawPic(t, 100, 60, func(r *gfx.Recorder) {
		r.StrokeShadowPath(&line, gfx.StrokeStyle{Width: 4}, shRed, gfx.Shadow{DY: 20, Color: shBlack})
	})
	if p := pixel(s, 30, 10); p != [4]byte{0, 0, 255, 255} {
		t.Errorf("the line: %v", p)
	}
	if p := pixel(s, 30, 30); p != [4]byte{0, 0, 0, 255} {
		t.Errorf("the shadow of the line: %v", p)
	}
	// a plain stroke after the shadowed one must not carry the shadow on
	s = drawPic(t, 100, 60, func(r *gfx.Recorder) {
		r.StrokeShadowPath(&line, gfx.StrokeStyle{Width: 4}, shRed, gfx.Shadow{DY: 20, Color: shBlack})
		var low gfx.Path
		low.MoveTo(10, 35)
		low.LineTo(60, 35)
		r.StrokePath(&low, gfx.StrokeStyle{Width: 4}, shRed)
	})
	if p := pixel(s, 30, 35); p[3] == 0 {
		t.Fatalf("no plain line: %v", p)
	}
	if p := pixel(s, 30, 55); p[3] != 0 {
		t.Errorf("the plain stroke carries a shadow: %v", p)
	}
}

func TestTextShadow(t *testing.T) {
	alpha := func(s *Surface, y0, y1 int) (sum int) {
		for y := y0; y < y1; y++ {
			for x := 0; x < 100; x++ {
				sum += int(pixel(s, x, y)[3])
			}
		}
		return
	}
	f := gfx.Font{Family: "Helvetica", Size: 20}
	with := drawPic(t, 100, 80, func(r *gfx.Recorder) {
		r.TextShadow("HHH", 5, 25, f, shRed, gfx.Shadow{DY: 30, Color: shBlack})
	})
	if alpha(with, 40, 60) == 0 {
		t.Error("no shadow under the text")
	}
	if alpha(with, 0, 30) == 0 {
		t.Error("no text")
	}
	plain := drawPic(t, 100, 80, func(r *gfx.Recorder) { r.Text("HHH", 5, 25, f, shRed) })
	if a := alpha(plain, 40, 60); a != 0 {
		t.Errorf("plain text has ink below it: %d", a)
	}
}
