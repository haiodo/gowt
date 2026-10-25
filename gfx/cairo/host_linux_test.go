//go:build linux

package cairo

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

// solid snapshots a w x h target filled with c.
func solid(r *Renderer, w, h int, c gfx.Color) gfx.Image {
	rc := gfx.NewRecorderWith(r)
	rc.FillPath(rectPath(0, 0, float64(w), float64(h)), c)
	tg := r.NewTarget(w, h)
	defer tg.Close()
	tg.Begin(1, gfx.Point{}, gfx.Color{})
	tg.Draw([]gfx.Item{{Pic: rc.Finish(), M: gfx.Identity(), Alpha: 1}})
	return tg.Snapshot(float64(w), float64(h))
}

func TestHostCompositesInGroupOrder(t *testing.T) {
	r := newRenderer(t, Options{})
	dirty := 0
	h, err := NewHost(func() { dirty++ })
	if err != nil {
		t.Skip(err)
	}
	defer h.Close()
	red, blue := solid(r, 20, 20, gfx.Color{R: 1, A: 1}), solid(r, 20, 20, gfx.Color{B: 1, A: 1})
	defer gfx.Release(red)
	defer gfx.Release(blue)

	// the overlay layer is made first and still lies above the tile made after it
	over := h.NewLayer(gfx.GroupOverlay)
	tile := h.NewLayer(gfx.GroupTiles)
	h.Begin()
	over.SetImage(blue)
	over.Set(gfx.R(10, 10, 20, 20), gfx.Point{}, gfx.Identity())
	tile.SetImage(red)
	tile.Set(gfx.R(0, 0, 20, 20), gfx.Point{}, gfx.Identity())
	h.Commit()
	if dirty != 1 {
		t.Errorf("OnDirty ran %d times for one commit", dirty)
	}
	h.Commit()
	if dirty != 1 {
		t.Errorf("an unchanged commit ran OnDirty: %d", dirty)
	}

	tg := r.NewTarget(40, 40).(*Target)
	defer tg.Close()
	h.Paint(tg.cr)
	if p := pixel(tg, 5, 5); p != [4]byte{0, 0, 255, 255} {
		t.Errorf("tile only: %v", p)
	}
	if p := pixel(tg, 15, 15); p != [4]byte{255, 0, 0, 255} {
		t.Errorf("overlay above tile: %v", p)
	}
	if p := pixel(tg, 35, 5); p != [4]byte{} {
		t.Errorf("outside both: %v", p)
	}

	tile.SetOpacity(0.5)
	over.SetHidden(true)
	tile.Set(gfx.R(0, 0, 20, 20), gfx.Point{X: 10, Y: 10}, gfx.TranslateBy(20, 0)) // moved 20 right
	h.Commit()
	tg.Begin(1, gfx.Point{}, gfx.Color{})
	h.Paint(tg.cr)
	if p := pixel(tg, 25, 5); !near(p[2], 128, 2) || !near(p[3], 128, 2) {
		t.Errorf("half transparent moved tile: %v", p)
	}
	if p := pixel(tg, 5, 5); p != [4]byte{} {
		t.Errorf("tile left its old place: %v", p)
	}
	if p := pixel(tg, 15, 15); p != [4]byte{} {
		t.Errorf("hidden layer drew: %v", p)
	}
	over.Remove()
	tile.Remove()
}
