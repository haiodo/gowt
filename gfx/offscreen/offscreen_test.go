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

// A picture recorded before any renderer exists still gets the bounds of the text as drawn.
func TestRecordBeforeRendererMeasuresWithBackend(t *testing.T) {
	f := gfx.Font{Name: "Helvetica", Size: 20}
	rec := gfx.NewRecorder()
	rec.Text("Measure me, please", 0, 20, f, gfx.Black)
	pic := rec.Finish()
	r, err := Default()
	if err != nil {
		t.Skip(err)
	}
	if w := r.MeasureText("Measure me, please", f).Width; pic.Bounds.W() < w-1 || pic.Bounds.W() > w+6 {
		t.Errorf("picture is %v wide, the backend measures %v", pic.Bounds.W(), w)
	}
}

func TestPNGOfNilRootIsTheWholeScene(t *testing.T) {
	rec := gfx.NewRecorder()
	rec.FillRect(gfx.R(0, 0, 30, 20), gfx.Black)
	n := gfx.NewNode(0)
	s := gfx.NewScene()
	s.Update(func(tx *gfx.Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
	})
	data, err := PNG(s, nil, 1, gfx.White)
	if err != nil {
		t.Skip(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil || img.Bounds().Dx() != 30 || img.Bounds().Dy() != 20 {
		t.Fatalf("%v %v", img, err)
	}
}
