//go:build linux

package cairo

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestHostHUDGroupIsOnTop(t *testing.T) {
	r := newRenderer(t, Options{})
	h, err := NewHost(func() {})
	if err != nil {
		t.Skip(err)
	}
	defer h.Close()
	red, blue := solid(r, 20, 20, gfx.Color{R: 1, A: 1}), solid(r, 20, 20, gfx.Color{B: 1, A: 1})
	defer gfx.Release(red)
	defer gfx.Release(blue)

	// the HUD layer is made first and still lies above the overlay made after it
	hud := h.NewLayer(gfx.GroupHUD)
	over := h.NewLayer(gfx.GroupOverlay)
	h.Begin()
	hud.SetImage(blue)
	hud.Set(gfx.R(10, 0, 20, 20), gfx.Point{}, gfx.Identity())
	over.SetImage(red)
	over.Set(gfx.R(0, 0, 20, 20), gfx.Point{}, gfx.Identity())
	h.Commit()
	tg := r.NewTarget(40, 20).(*Target)
	defer tg.Close()
	h.Paint(tg.cr)
	if p := pixel(tg, 15, 5); p != [4]byte{255, 0, 0, 255} {
		t.Errorf("HUD over overlay: %v", p)
	}
	if p := pixel(tg, 5, 5); p != [4]byte{0, 0, 255, 255} {
		t.Errorf("overlay only: %v", p)
	}
}
