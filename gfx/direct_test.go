package gfx

import "testing"

// TestDirectNoTilesNoSnap is the core of the Direct mode: no tile sets, no tile pool jobs, one
// layer of the whole viewport, every draw at exactly zoom*dpr - never a power of two.
func TestDirectNoTilesNoSnap(t *testing.T) {
	g := newRig(t, Options{Direct: true})
	g.addBackground()
	g.addBox(1, 100, 100, 50, 50)
	g.c.SetCamera(Camera{Zoom: 1.3})
	g.settle()
	if g.c.cur != nil || g.c.prev != nil {
		t.Fatal("Direct mode created a tile set")
	}
	layers := g.h.live(GroupTiles)
	if len(layers) != 1 {
		t.Fatalf("%d viewport layers, want 1", len(layers))
	}
	draws := g.r.tileDraws()
	if len(draws) == 0 {
		t.Fatal("nothing drawn")
	}
	want := 1.3 * 2 // zoom * dpr, not snapped to 4
	for _, d := range draws {
		if d.scale != want {
			t.Fatalf("drawn at scale %v, want exactly %v (no snap)", d.scale, want)
		}
		if d.origin != (Point{X: 0, Y: 0}) { // the rig's camera origin is the zero point
			t.Errorf("draw origin %v, want the view rect origin", d.origin)
		}
	}
	if n := g.c.Stats().TilesPending; n != 0 {
		t.Errorf("%d tiles pending", n)
	}
	if n := g.c.Workers(); n != 2 {
		t.Errorf("Workers %d", n)
	}
}

// TestDirectIdleDrawsNothing: an idle Direct frame reuses the image, like a settled tile view.
func TestDirectIdleDrawsNothing(t *testing.T) {
	g := newRig(t, Options{Direct: true})
	g.addBackground()
	n := g.addBox(1, 100, 100, 50, 50)
	g.settle()
	before := len(g.r.tileDraws())
	g.settle() // still nothing to do
	if got := len(g.r.tileDraws()); got != before {
		t.Fatalf("idle frames drew %d times", got-before)
	}
	g.s.Update(func(tx *Tx) { tx.Move(n, 200, 200) })
	g.settle()
	if got := len(g.r.tileDraws()) - before; got != 1 {
		t.Fatalf("one in-view damage: %d redraws", got)
	}
	// damage fully outside the view redraws nothing
	g.s.Update(func(tx *Tx) { tx.Move(n, 100000, 100000) })
	g.settle()
	if got := len(g.r.tileDraws()) - before; got != 2 {
		t.Fatalf("out-of-view damage: %d redraws total, want 2", got)
	}
}

// TestDirectCameraAndResize: every camera move and viewport change redraws once, at the exact
// scale of the moment.
func TestDirectCameraAndResize(t *testing.T) {
	g := newRig(t, Options{Direct: true})
	g.addBackground()
	g.settle()
	before := len(g.r.tileDraws())
	for _, z := range []float64{0.5, 1, 1.3, 2, 3, 4} {
		g.c.SetCamera(Camera{Zoom: z})
		g.settle()
		draws := g.r.tileDraws()
		d := draws[len(draws)-1]
		if want := z * 2; d.scale != want {
			t.Fatalf("zoom %v: drawn at %v, want %v", z, d.scale, want)
		}
	}
	g.c.SetViewport(1024, 768, 2) // resize: a redraw even though the camera did not move
	g.settle()
	if got := len(g.r.tileDraws()) - before; got != 7 {
		t.Fatalf("%d redraws for 6 zooms and a resize, want 7", got)
	}
}

// TestDirectLiftLand: lifts and landings move the node between the base image and a sprite layer
// without a tile in sight; the sprite is gone the frame the base shows the node again.
func TestDirectLiftLand(t *testing.T) {
	g := newRig(t, Options{Direct: true})
	g.addBackground()
	n := g.addBox(1, 100, 100, 50, 50)
	g.settle()
	g.c.Lift(n)
	g.settle()
	active := g.h.live(GroupActive)
	if len(active) != 1 {
		t.Fatalf("%d active layers after Lift", len(active))
	}
	if active[0].hidden {
		t.Error("the lift sprite is still hidden after the base redraw")
	}
	draws := g.r.tileDraws()
	last := draws[len(draws)-1]
	if last.items != 1 { // background only: the lifted box left the base image
		t.Errorf("base image has %d items after Lift, want 1", last.items)
	}
	g.c.Land(n)
	g.settle()
	if live := g.h.live(GroupActive); len(live) != 0 {
		t.Fatalf("%d active layers after Land", len(live))
	}
	draws = g.r.tileDraws()
	last = draws[len(draws)-1]
	if last.items != 2 { // background and the landed box
		t.Errorf("base image has %d items after Land, want 2", last.items)
	}
	if g.c.NeedsFrame() {
		t.Error("NeedsFrame still true after the landing settled")
	}
}

// TestDirectExactZoomCombination: ExactZoom is redundant under Direct (every draw is exact
// already) and the two do not fight.
func TestDirectExactZoomCombination(t *testing.T) {
	g := newRig(t, Options{Direct: true, ExactZoom: true})
	g.addBackground()
	g.addBox(1, 100, 100, 50, 50)
	g.c.SetCamera(Camera{Zoom: 1.3})
	g.settle()
	draws := g.r.tileDraws()
	if want := 1.3 * 2; draws[len(draws)-1].scale != want {
		t.Fatalf("scale %v, want %v", draws[len(draws)-1].scale, want)
	}
	if got := g.c.snapScale(); got != 1.3*2 {
		t.Errorf("snapScale %v, want zoom*dpr", got)
	}
}

// TestDirectLiftHoldNoDoubleDraw is the reviewer's case: a translucent node lifted in the Direct
// mode must never be in the base image and in the sprite within one host commit. The hold now
// hides the sprite until the base is redrawn without the node, as in the tile mode.
func TestDirectLiftHoldNoDoubleDraw(t *testing.T) {
	g := newRig(t, Options{Direct: true})
	g.addBackground()
	n := NewNode(3)
	rec := NewRecorder()
	rec.FillRect(R(0, 0, 50, 50), Color{R: 1, A: 1})
	g.s.Update(func(tx *Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
		tx.Move(n, 100, 100)
		tx.SetOpacity(n, 0.5)
	})
	g.settle()
	base := g.h.live(GroupTiles)
	if len(base) != 1 || base[0].img == nil || base[0].img.items != 2 {
		t.Fatalf("base before Lift: %+v", base)
	}

	g.c.Lift(n)
	// the commit Lift just made: the base still holds the old image with the node, so the sprite
	// must be hidden, or the half-transparent node would draw twice in one commit
	active := g.h.live(GroupActive)
	if len(active) != 1 {
		t.Fatalf("%d active layers after Lift", len(active))
	}
	if !active[0].hidden {
		t.Error("the lift sprite is not held: the node is in the base and in the sprite in one commit")
	}
	if base[0].img.items != 2 {
		t.Errorf("base image changed during Lift: %d items", base[0].img.items)
	}

	g.frame() // the base redraws without the node, the hold releases in the same commit
	if base[0].img.items != 1 {
		t.Errorf("base after the redraw: %d items, want the background only", base[0].img.items)
	}
	if active[0].hidden {
		t.Error("the sprite is still hidden after the base redraw")
	}

	g.c.Land(n)
	g.frame()
	if live := g.h.live(GroupActive); len(live) != 0 {
		t.Fatalf("%d active layers after Land", len(live))
	}
	if base[0].img.items != 2 {
		t.Errorf("base after Land: %d items, want the node back once", base[0].img.items)
	}
}

// TestDirectBlankView: an empty view has no layer and no draw; content arriving makes one.
func TestDirectBlankView(t *testing.T) {
	g := newRig(t, Options{Direct: true})
	g.settle()
	if len(g.h.live(GroupTiles)) != 0 || len(g.r.tileDraws()) != 0 {
		t.Fatal("an empty view drew something")
	}
	n := NewNode(1)
	rec := NewRecorder()
	rec.FillRect(R(0, 0, 50, 50), Color{R: 1, A: 1})
	g.s.Update(func(tx *Tx) { tx.Add(nil, n); tx.SetPicture(n, rec.Finish()); tx.Move(n, 100, 100) })
	g.settle()
	if len(g.h.live(GroupTiles)) != 1 {
		t.Fatal("no viewport layer for a box in view")
	}
	// move it away: the layer hides, still no blank draw
	g.s.Update(func(tx *Tx) { tx.Move(n, 100000, 100000) })
	g.settle()
	layers := g.h.live(GroupTiles)
	if len(layers) != 1 || !layers[0].hidden {
		t.Errorf("layer after the box left: %+v", layers)
	}
}

// TestDirectHUD: the HUD draws over the base image in Direct mode as in the tile mode.
func TestDirectHUD(t *testing.T) {
	g := newRig(t, Options{Direct: true})
	g.addBackground()
	g.settle()
	hud := g.c.HUD()
	rec := NewRecorder()
	rec.FillRect(R(0, 0, 40, 20), Color{R: 1, A: 1})
	badge := NewNode(1)
	hud.Update(func(tx *Tx) { tx.Add(nil, badge); tx.SetPicture(badge, rec.Finish()) })
	g.settle()
	layers := g.h.live(GroupHUD)
	if len(layers) != 1 || layers[0].img == nil {
		t.Fatalf("HUD layers: %+v", layers)
	}
	if hudDraws := g.c.Stats().HUDDraws; hudDraws != 1 {
		t.Errorf("HUDDraws %d, want 1", hudDraws)
	}
}

// --- parallel strip rasterization ---

type stripRec struct {
	origin Point
	scale  float64
	items  int
}

type fakeStripTarget struct{ rec *stripRec }

func (t fakeStripTarget) Begin(scale float64, origin Point, _ Color) {
	t.rec.origin, t.rec.scale = origin, scale
}
func (t fakeStripTarget) Draw(items []Item)           { t.rec.items = len(items) }
func (t fakeStripTarget) Size() (int, int)            { return 100, 50 }
func (t fakeStripTarget) Pixels() []byte              { return nil }
func (t fakeStripTarget) Snapshot(w, h float64) Image { return &fakeImg{w, h, 0} }
func (t fakeStripTarget) Close()                      {}

type fakeStripSet struct {
	recs   []stripRec
	closed bool
}

func newFakeStripSet(n int) *fakeStripSet { return &fakeStripSet{recs: make([]stripRec, n)} }

func (f *fakeStripSet) Target(i int) Target      { return fakeStripTarget{&f.recs[i]} }
func (f *fakeStripSet) Strips() int              { return len(f.recs) }
func (f *fakeStripSet) Image(w, h float64) Image { return &fakeImg{w, h, 0} }
func (f *fakeStripSet) Close()                   { f.closed = true }

// fakeStripRenderer adds the strip capability to the fake renderer and remembers every set.
type fakeStripRenderer struct {
	fakeRenderer
	sets []*fakeStripSet
}

func (r *fakeStripRenderer) NewStripSet(n, _, _ int) StripSet {
	s := newFakeStripSet(n)
	r.sets = append(r.sets, s)
	return s
}

// TestDirectStripsInvariants pins the strip partition: the strips cover the viewport in order
// with the origins that map scene points to the same device pixels, a node is collected for
// every strip it crosses, the sets rotate over three buffers, and Close frees them.
func TestDirectStripsInvariants(t *testing.T) {
	fr := &fakeStripRenderer{}
	g := &rig{t: t, s: NewScene(), r: &fr.fakeRenderer, h: &fakeHost{}}
	g.c = NewCompositor(g.s, fr, g.h, Options{Direct: true, Workers: 4})
	g.c.SetViewport(800, 600, 2) // 1600x1200 device, 4 strips of 300
	t.Cleanup(g.c.Close)
	g.addBackground()
	g.addBox(1, 100, 100, 50, 50) // crosses the strip at y 300/2=450? box 100..150 pt, strips at pt 450 boundary
	g.c.SetCamera(Camera{Zoom: 1})
	g.settle()
	if len(fr.sets) != 1 {
		t.Fatalf("%d strip sets, want 1", len(fr.sets))
	}
	set := fr.sets[0]
	if set.Strips() != 4 {
		t.Fatalf("%d strips, want 4", set.Strips())
	}
	for i, rec := range set.recs {
		if rec.origin != (Point{}) {
			t.Errorf("strip %d origin %v, want the view origin (0,0)", i, rec.origin)
		}
		if rec.scale != 2 {
			t.Errorf("strip %d scale %v, want 2", i, rec.scale)
		}
		if rec.items == 0 {
			t.Errorf("strip %d drew nothing", i)
		}
	}
	// the box at 100..150 pt lies in the third strip only; the background crosses all four:
	// every strip drew, and the total reaches every item of the single collect
	var total int
	for _, rec := range set.recs {
		total += rec.items
	}
	var all Items
	g.s.Collect(g.c.viewRect(), &all)
	if total < len(all.List) {
		t.Errorf("strips drew %d items in total, the whole view has %d", total, len(all.List))
	}

	// three redrawn frames more (a move damages the scene): the rotation reuses the first
	// set, which must not be closed
	n := g.addBox(2, 300, 100, 10, 10)
	for i := 0; i < 3; i++ {
		g.s.Update(func(tx *Tx) { tx.Move(n, 301+float64(i), 100) })
		g.frame()
	}
	if len(fr.sets) != 3 {
		t.Fatalf("%d strip sets after 4 redraws, want 3 (the rotation reuses)", len(fr.sets))
	}
	if fr.sets[0].closed {
		t.Error("the rotated-out set is closed while the compositor lives")
	}
	g.s.Update(func(tx *Tx) { tx.Move(n, 306, 100) })
	g.frame()
	if len(fr.sets) != 3 {
		t.Fatalf("%d strip sets after a 5th redraw, want the same 3", len(fr.sets))
	}
	// a resize replaces the sets: the rotation slot is closed when reused at a new size
	g.c.SetViewport(400, 300, 2) // 800x600 device
	for i := 0; i < 4; i++ {
		g.s.Update(func(tx *Tx) { tx.Move(n, 300, 101+float64(i)) })
		g.frame()
	}
	if !fr.sets[0].closed {
		t.Error("the old-size set is not closed after the rotation passed it")
	}
	g.c.Close()
	for _, s := range fr.sets {
		if !s.closed {
			t.Error("a strip set leaked past Close")
		}
	}
}
