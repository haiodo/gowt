package synth

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestQueryMatchesBruteForce(t *testing.T) {
	d := New(1000, 1)
	for _, r := range []gfx.Rect{gfx.R(0, 0, 500, 400), gfx.R(1000, 800, 1, 1), gfx.R(-50, -50, 100000, 100000), gfx.R(2000, 300, 700, 90)} {
		got := map[int]bool{}
		for _, n := range d.Scene.Query(r, nil) {
			if got[n.ID] {
				t.Fatalf("node %d reported twice", n.ID)
			}
			got[n.ID] = true
		}
		want := 0
		for _, n := range d.Scene.Nodes() {
			if n.Bounds().Intersects(r) {
				want++
				if !got[n.ID] {
					t.Fatalf("node %d missing for %+v", n.ID, r)
				}
			}
		}
		if want != len(got) {
			t.Fatalf("rect %+v: %d nodes, brute force %d", r, len(got), want)
		}
	}
}

func TestHitTestAndMove(t *testing.T) {
	d := New(200, 2)
	b := d.Boxes[45]
	c := b.Node.Bounds().Center()
	if n := d.Scene.HitTest(c, 3); n != b.Node {
		t.Fatalf("hit %v, want box 45", n)
	}
	conn := b.Conns[0]
	m := conn.Node.Shape.Measure(0.05)
	mid, _ := m.At(m.Length() / 2)
	if n := d.Scene.HitTest(mid, 3); n != conn.Node {
		t.Fatalf("connector midpoint hit %v", n)
	}

	dirty := d.MoveBox(b, b.Node.X+500, b.Node.Y+300)
	if dirty.W() < 500 {
		t.Fatalf("dirty rect %+v too small", dirty)
	}
	if n := d.Scene.HitTest(c, 3); n == b.Node {
		t.Fatal("box still hit at its old place")
	}
	nc := b.Node.Bounds().Center()
	if n := d.Scene.HitTest(nc, 3); n != b.Node {
		t.Fatalf("box not hit at its new place: %v", n)
	}
	m = conn.Node.Shape.Measure(0.05)
	mid, _ = m.At(m.Length() / 2)
	found := false
	for _, n := range d.Scene.Query(gfx.Rect{Min: mid, Max: mid}.Expand(1), nil) {
		found = found || n == conn.Node
	}
	if !found {
		t.Fatal("moved connector not found at its new midpoint")
	}
}
