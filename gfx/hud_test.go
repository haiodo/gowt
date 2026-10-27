package gfx

import "testing"

func hudLayer(t *testing.T, g *rig) *fakeLayer {
	t.Helper()
	ls := g.h.live(GroupHUD)
	if len(ls) != 1 {
		t.Fatalf("%d HUD layers, want 1", len(ls))
	}
	return ls[0]
}

func TestHUDDrawnOnChangeOnly(t *testing.T) {
	g := newRig(t, Options{})
	g.c.SetCamera(Camera{Zoom: 1})
	hud := g.c.HUD()
	n := add(hud, nil, 1, 10, 10, 100, 20)
	g.settle()
	l := hudLayer(t, g)
	if l.frame != R(10, 10, 100, 20) || l.hidden {
		t.Fatalf("frame %v hidden %v", l.frame, l.hidden)
	}
	if l.images != 1 {
		t.Fatalf("%d images, want 1", l.images)
	}

	// camera moves and zooms do not touch it
	g.c.SetCamera(Camera{Origin: Point{50, 50}, Zoom: 2})
	g.settle()
	if l.images != 1 || l.frame != R(10, 10, 100, 20) {
		t.Errorf("the camera moved the HUD: %d images, frame %v", l.images, l.frame)
	}

	// a change draws it once, at the new place
	hud.Update(func(tx *Tx) { tx.Move(n, 30, 40) })
	g.settle()
	if l.images != 2 || l.frame != R(30, 40, 100, 20) {
		t.Errorf("after a move: %d images, frame %v", l.images, l.frame)
	}

	// the view size and the backing scale draw it again
	g.c.SetViewport(1000, 600, 2)
	g.settle()
	if l.images != 3 {
		t.Errorf("after a resize: %d images, want 3", l.images)
	}
	g.c.SetViewport(1000, 600, 3)
	g.settle()
	if l.images != 4 {
		t.Errorf("after a scale change: %d images, want 4", l.images)
	}

	// content outside the view is cut to the view; nothing inside hides the layer
	hud.Update(func(tx *Tx) { tx.Move(n, 950, 10) })
	g.settle()
	if l.frame != R(950, 10, 50, 20) {
		t.Errorf("clipped to the view: %v", l.frame)
	}
	hud.Update(func(tx *Tx) { tx.Move(n, 2000, 10) })
	g.settle()
	if !l.hidden {
		t.Error("a HUD outside the view must be hidden")
	}
}

func TestOnViewport(t *testing.T) {
	g := newRig(t, Options{})
	hud := g.c.HUD()
	n := add(hud, nil, 1, 0, 0, 40, 20)
	var got [][3]float64
	g.c.OnViewport(func(w, h, dpr float64) {
		got = append(got, [3]float64{w, h, dpr})
		hud.Update(func(tx *Tx) { tx.Move(n, w-40, 0) }) // pinned to the right edge
	})
	g.settle()
	if len(got) != 1 || got[0] != [3]float64{800, 600, 2} {
		t.Fatalf("calls %v", got)
	}
	l := hudLayer(t, g)
	if l.frame != R(760, 0, 40, 20) {
		t.Errorf("frame %v, want the right edge", l.frame)
	}
	g.c.SetViewport(500, 600, 2)
	g.settle()
	if len(got) != 2 || l.frame != R(460, 0, 40, 20) {
		t.Errorf("after a resize: calls %v, frame %v", got, l.frame)
	}
	g.frame()
	if len(got) != 2 {
		t.Error("OnViewport ran without a change")
	}
}

func TestHUDHit(t *testing.T) {
	g := newRig(t, Options{})
	if g.c.HUDHit(Point{5, 5}, 0) != nil {
		t.Fatal("a hit without a HUD")
	}
	hud := g.c.HUD()
	n := add(hud, nil, 1, 10, 10, 100, 20)
	g.settle()
	if g.c.HUDHit(Point{20, 20}, 0) != n {
		t.Error("no hit on the HUD node")
	}
	if g.c.HUDHit(Point{200, 20}, 0) != nil {
		t.Error("a hit beside it")
	}
	hud.Update(func(tx *Tx) { tx.SetShape(n, nil, HitNone) })
	g.settle()
	if g.c.HUDHit(Point{20, 20}, 0) != nil {
		t.Error("HitNone must pass the pointer on to the scene")
	}
}
