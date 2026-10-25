package gfx

import (
	"testing"
	"time"
)

// A dragged node released anywhere - in view, on the edge, in the margin ring, far outside the
// view - must land: its sprite goes, the node never shows twice on the way, and a tile shows it
// once the camera is panned to it.
func TestLandCompletesWherever(t *testing.T) {
	for _, tc := range []struct {
		name string
		x, y float64
		pan  bool // pan down and back first, so that tiles remain just outside the ring
	}{
		{"in view", 300, 200, false},
		{"across a tile edge in view", 240, 250, false},
		{"across the bottom edge", 300, 580, false},
		{"in the ring", 300, 800, false},
		{"outside the ring", 300, 1500, false},
		{"in the band beyond the ring", 300, 1100, true},
		{"far away", 5000, 6000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newRig(t, Options{})
			n := g.addBox(1, 100, 100, 60, 50)
			g.settle()
			if tc.pan {
				g.c.SetCamera(Camera{Origin: Point{0, 256}, Zoom: 1})
				g.settle()
				g.c.SetCamera(Camera{Zoom: 1})
				g.settle()
			}
			g.r.delay = 2 * time.Millisecond
			mustShow := tc.y <= 800 && tc.x <= 300
			g.h.onCommit = func() {
				tile, sprite := false, false
				for _, l := range g.h.layers {
					if l.removed || l.hidden || l.img == nil {
						continue
					}
					tile = tile || (l.group == GroupTiles && l.img.items > 0)
					sprite = sprite || l.group == GroupActive
				}
				if tile && sprite {
					t.Error("the node shows in a tile and in its sprite")
				}
				if mustShow && !tile && !sprite {
					t.Error("the node shows nowhere")
				}
			}
			g.c.Lift(n)
			for i := 1; i <= 10; i++ {
				f := float64(i) / 10
				g.s.Update(func(tx *Tx) { tx.Move(n, 100+(tc.x-100)*f, 100+(tc.y-100)*f) })
				g.frame()
				time.Sleep(time.Millisecond)
			}
			g.c.Land(n)
			g.settle()
			if a := g.h.live(GroupActive); len(a) != 0 {
				t.Fatalf("%d sprites left after landing", len(a))
			}
			g.h.onCommit = nil
			g.c.SetCamera(Camera{Origin: Point{tc.x - 300, tc.y - 200}, Zoom: 1})
			g.settle()
			shown := 0
			for _, l := range g.h.live(GroupTiles) {
				if l.img != nil && l.img.items > 0 {
					shown++
				}
			}
			if shown == 0 {
				t.Error("no tile shows the landed node after panning to it")
			}
		})
	}
}

// Escape moves the node back and lands it in the same step.
func TestLandAfterMoveBack(t *testing.T) {
	g := newRig(t, Options{})
	n := g.addBox(1, 100, 100, 60, 50)
	g.settle()
	g.r.delay = 2 * time.Millisecond
	g.h.onCommit = func() {
		tile, sprite := false, false
		for _, l := range g.h.layers {
			if l.removed || l.hidden || l.img == nil {
				continue
			}
			tile = tile || (l.group == GroupTiles && l.img.items > 0)
			sprite = sprite || l.group == GroupActive
		}
		if tile && sprite {
			t.Error("the node shows in a tile and in its sprite")
		}
		if !tile && !sprite {
			t.Error("the node shows nowhere")
		}
	}
	g.c.Lift(n)
	for i := 1; i <= 5; i++ {
		g.s.Update(func(tx *Tx) { tx.Move(n, 100+float64(i)*40, 100+float64(i)*80) })
		g.frame()
	}
	g.s.Update(func(tx *Tx) { tx.Move(n, 100, 100) })
	g.c.Land(n)
	g.settle()
	g.h.onCommit = nil
	if a := g.h.live(GroupActive); len(a) != 0 {
		t.Fatalf("%d sprites left", len(a))
	}
}
