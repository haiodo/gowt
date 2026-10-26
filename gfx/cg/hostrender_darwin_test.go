//go:build darwin

package cg

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

// Render draws the layers from their images: placed, scaled, with the opacity, nothing elsewhere.
func TestHostRender(t *testing.T) {
	initHost()
	root := send0(send0(clsLayer, sels.alloc), sels.init)
	h := NewHost(root, 1)
	defer h.Close()
	r := NewRenderer(Options{})
	s := r.NewSurface(30, 40)
	defer s.Close()
	s.Begin(1, 0, 0, gfx.Color{R: 1, A: 1})
	h.Begin()
	l := h.NewLayer(gfx.GroupTiles)
	l.SetImage(s.Snapshot(30, 40))
	l.Set(gfx.R(10, 20, 30, 40), gfx.Point{}, gfx.Identity())
	l.SetOpacity(0.5)
	h.Commit()
	img := h.Render(100, 100, 2)
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 200 {
		t.Fatalf("size %v", img.Bounds())
	}
	if c := img.NRGBAAt(2*25, 2*40); c.R != 255 || c.G != 0 || c.A < 120 || c.A > 135 {
		t.Errorf("inside the layer: %+v", c)
	}
	for _, p := range [][2]int{{2 * 5, 2 * 40}, {2 * 25, 2 * 10}, {2 * 45, 2 * 40}, {2 * 25, 2 * 65}} {
		if c := img.NRGBAAt(p[0], p[1]); c.A != 0 {
			t.Errorf("outside the layer at %v: %+v", p, c)
		}
	}
	l.Remove()
	if c := h.Render(100, 100, 1).NRGBAAt(25, 40); c.A != 0 {
		t.Errorf("after Remove: %+v", c)
	}
}
