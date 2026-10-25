package offscreen

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestNewRendersPNG(t *testing.T) {
	r, err := New(Options{})
	if err != nil {
		t.Skip(err)
	}
	rc := gfx.NewRecorderWith(r)
	p := &gfx.Path{}
	p.Rect(0, 0, 30, 20)
	rc.FillPath(p, gfx.Color{G: 1, A: 1})
	rc.Text("ok", 2, 15, gfx.Font{Name: "Helvetica", Size: 12}, gfx.Color{A: 1})
	ras, err := gfx.RenderPicture(r, rc.Finish(), 2, gfx.Color{R: 1, G: 1, B: 1, A: 1})
	if err != nil {
		t.Fatal(err)
	}
	data, err := ras.PNG()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 60 || b.Dy() != 40 {
		t.Errorf("size %v, want 60x40", b)
	}
}
