//go:build windows

package d2d

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestFillShadowPath(t *testing.T) {
	img := render(t, 100, 60, 1, gfx.Color{}, func(r *gfx.Recorder) {
		r.FillShadowPath(gfx.EllipsePath(gfx.R(10, 10, 20, 20)), red, gfx.Shadow{DX: 30, Color: black})
	})
	if p := px(img, 20, 20); !near(p, [4]byte{255, 0, 0, 255}, 2) {
		t.Errorf("the shape: %v", p)
	}
	if p := px(img, 50, 20); !near(p, [4]byte{0, 0, 0, 255}, 4) {
		t.Errorf("the shadow of the ellipse: %v", p)
	}
	if p := px(img, 41, 11); p[3] != 0 {
		t.Errorf("the shadow must follow the ellipse, not its box: %v", p)
	}
}

func TestShadowBlur(t *testing.T) {
	img := render(t, 100, 100, 1, gfx.Color{}, func(r *gfx.Recorder) {
		r.FillShadowPath(gfx.EllipsePath(gfx.R(30, 30, 40, 40)), red, gfx.Shadow{Blur: 10, Color: black})
	})
	if p := px(img, 75, 50); p[3] == 0 || p[3] == 255 {
		t.Errorf("a blurred edge is partly covered: %v", p)
	}
}

func TestShadowFollowsTransform(t *testing.T) {
	// y up on screen: the local shadow offset DY=+20 must land above the shape
	img := render(t, 100, 100, 1, gfx.Color{}, func(r *gfx.Recorder) {
		r.Translate(0, 100)
		r.Scale(1, -1)
		r.FillShadowPath(gfx.RectPath(gfx.R(10, 10, 20, 20)), red, gfx.Shadow{DY: 20, Color: black})
	})
	if p := px(img, 20, 55); !near(p, [4]byte{0, 0, 0, 255}, 4) { // local y 45 is device y 55
		t.Errorf("mirrored shadow: %v", p)
	}
	if p := px(img, 20, 95); p[3] != 0 {
		t.Errorf("the shadow went the wrong way: %v", p)
	}
	img = render(t, 100, 100, 1, gfx.Color{}, func(r *gfx.Recorder) {
		r.Translate(0, 100)
		r.Scale(1, -1)
		r.FillShadowRRect(10, 10, 20, 20, 0, red, gfx.Shadow{DY: 20, Color: black})
	})
	if p := px(img, 20, 55); p[3] != 255 {
		t.Errorf("mirrored box shadow: %v", p)
	}
}

func TestStrokeShadowPath(t *testing.T) {
	var line gfx.Path
	line.MoveTo(10, 10)
	line.LineTo(60, 10)
	img := render(t, 100, 60, 1, gfx.Color{}, func(r *gfx.Recorder) {
		r.StrokeShadowPath(&line, gfx.StrokeStyle{Width: 4}, red, gfx.Shadow{DY: 20, Color: black})
		var low gfx.Path
		low.MoveTo(10, 50)
		low.LineTo(60, 50)
		r.StrokePath(&low, gfx.StrokeStyle{Width: 4}, red)
	})
	if p := px(img, 30, 10); !near(p, [4]byte{255, 0, 0, 255}, 2) {
		t.Errorf("the line: %v", p)
	}
	if p := px(img, 30, 30); !near(p, [4]byte{0, 0, 0, 255}, 4) {
		t.Errorf("the shadow of the line: %v", p)
	}
	if p := px(img, 30, 50); !near(p, [4]byte{255, 0, 0, 255}, 2) {
		t.Errorf("the plain line: %v", p)
	}
	if p := px(img, 30, 56); p[3] != 0 {
		t.Errorf("the plain stroke carries a shadow: %v", p)
	}
}

func TestTextShadow(t *testing.T) {
	alpha := func(img *gfx.Raster, y0, y1 int) (sum int) {
		for y := y0; y < y1; y++ {
			for x := 0; x < img.W; x++ {
				sum += int(px(img, x, y)[3])
			}
		}
		return
	}
	f := gfx.Font{Family: "Arial", Size: 20}
	with := render(t, 100, 80, 1, gfx.Color{}, func(r *gfx.Recorder) {
		r.TextShadow("HHH", 5, 25, f, red, gfx.Shadow{DY: 30, Color: black})
	})
	if alpha(with, 40, 60) == 0 {
		t.Error("no shadow under the text")
	}
	if alpha(with, 0, 30) == 0 {
		t.Error("no text")
	}
	plain := render(t, 100, 80, 1, gfx.Color{}, func(r *gfx.Recorder) { r.Text("HHH", 5, 25, f, red) })
	if a := alpha(plain, 40, 60); a != 0 {
		t.Errorf("plain text has ink below it: %d", a)
	}
}
