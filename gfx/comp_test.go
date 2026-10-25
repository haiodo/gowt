package gfx

import (
	"math"
	"sync"
	"testing"
	"time"
)

type fakeImg struct {
	w, h  float64
	items int // nodes drawn into it
}

func (f *fakeImg) Size() (float64, float64) { return f.w, f.h }

type drawRec struct {
	scale  float64
	origin Point
	items  int
}

type fakeRenderer struct {
	mu    sync.Mutex
	draws []drawRec
	delay time.Duration
}

func (r *fakeRenderer) MeasureText(s string, f Font) TextMetrics { return TextMetrics{} }
func (r *fakeRenderer) NewTarget(w, h int) Target                { return &fakeTarget{r: r, w: w, h: h} }

func (r *fakeRenderer) tileDraws() []drawRec {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]drawRec(nil), r.draws...)
}

type fakeTarget struct {
	r    *fakeRenderer
	w, h int
	cur  drawRec
}

func (t *fakeTarget) Begin(scale float64, origin Point, bg Color) {
	t.cur = drawRec{scale: scale, origin: origin}
}
func (t *fakeTarget) Draw(items []Item) {
	if t.r.delay > 0 {
		time.Sleep(t.r.delay)
	}
	t.cur.items = len(items)
	t.r.mu.Lock()
	t.r.draws = append(t.r.draws, t.cur)
	t.r.mu.Unlock()
}
func (t *fakeTarget) Size() (int, int)            { return t.w, t.h }
func (t *fakeTarget) Pixels() []byte              { return nil }
func (t *fakeTarget) Snapshot(w, h float64) Image { return &fakeImg{w, h, t.cur.items} }
func (t *fakeTarget) Close()                      {}

type fakeLayer struct {
	h       *fakeHost
	group   Group
	frame   Rect
	anchor  Point
	m       Affine
	opacity float64
	hidden  bool
	removed bool
	images  int
	img     *fakeImg
}

func (l *fakeLayer) SetImage(i Image) { l.images++; l.img, _ = i.(*fakeImg) }
func (l *fakeLayer) Set(f Rect, a Point, m Affine) {
	l.frame, l.anchor, l.m = f, a, m
}
func (l *fakeLayer) SetOpacity(a float64) { l.opacity = a }
func (l *fakeLayer) SetHidden(h bool)     { l.hidden = h }
func (l *fakeLayer) Remove()              { l.removed = true }

type fakeHost struct {
	layers  []*fakeLayer
	batches int
	open    bool
	// onCommit runs at the end of every transaction
	onCommit func()
}

func (h *fakeHost) NewLayer(g Group) Layer {
	l := &fakeLayer{h: h, group: g, opacity: 1}
	h.layers = append(h.layers, l)
	return l
}
func (h *fakeHost) Begin() { h.open = true }
func (h *fakeHost) Commit() {
	h.open = false
	h.batches++
	if h.onCommit != nil {
		h.onCommit()
	}
}

func (h *fakeHost) live(g Group) (out []*fakeLayer) {
	for _, l := range h.layers {
		if l.group == g && !l.removed {
			out = append(out, l)
		}
	}
	return
}

type rig struct {
	t   *testing.T
	s   *Scene
	r   *fakeRenderer
	h   *fakeHost
	c   *Compositor
	now time.Duration
	seq uint64
}

func newRig(t *testing.T, o Options) *rig {
	t.Helper()
	g := &rig{t: t, s: NewScene(), r: &fakeRenderer{}, h: &fakeHost{}}
	if o.Workers == 0 {
		o.Workers = 2
	}
	g.c = NewCompositor(g.s, g.r, g.h, o)
	g.c.SetViewport(800, 600, 2)
	t.Cleanup(g.c.Close)
	return g
}

func (g *rig) frame() {
	g.now += 16 * time.Millisecond
	g.seq++
	g.c.Frame(FrameInfo{Time: g.now, Interval: 16 * time.Millisecond, Seq: g.seq})
}

// settle runs frames until the compositor has nothing to do.
func (g *rig) settle() {
	g.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		g.frame()
		if !g.c.NeedsFrame() && g.c.Stats().TilesPending == 0 {
			// one more, in case a result was queued just before
			time.Sleep(2 * time.Millisecond)
			if !g.c.NeedsFrame() && g.c.Stats().TilesPending == 0 {
				return
			}
		}
		if time.Now().After(deadline) {
			g.t.Fatal("compositor did not settle")
		}
		time.Sleep(time.Millisecond)
	}
}

func (g *rig) addBox(id int, x, y, w, h float64) *Node {
	return add(g.s, nil, id, x, y, w, h)
}

func TestFirstFrameFillsViewAndMargin(t *testing.T) {
	g := newRig(t, Options{})
	g.addBox(1, 100, 100, 50, 50)
	g.c.SetCamera(Camera{Zoom: 1})
	g.settle()
	// scale 2 (dpr 2, zoom 1): a tile is 256 scene units; the 800x600 view spans 4x3 tiles, +1 ring
	tiles := g.h.live(GroupTiles)
	if len(tiles) != 6*5 {
		t.Fatalf("%d tile layers, want 30", len(tiles))
	}
	for _, d := range g.r.tileDraws() {
		if d.scale != 2 {
			t.Fatalf("tile drawn at scale %v", d.scale)
		}
	}
	if g.h.open {
		t.Error("host batch left open")
	}
}

func TestDamageRedrawsOnlyTouchedTiles(t *testing.T) {
	g := newRig(t, Options{})
	n := g.addBox(1, 100, 100, 50, 50)
	g.settle()
	before := len(g.r.tileDraws())
	g.s.Update(func(tx *Tx) { tx.Move(n, 600, 100) }) // 100..150 -> 600..650: tile x 0 -> tile x 2
	g.settle()
	got := g.r.tileDraws()[before:]
	if len(got) != 2 {
		t.Fatalf("%d tiles redrawn, want 2 (old and new bounds): %+v", len(got), got)
	}
	origins := map[Point]bool{}
	for _, d := range got {
		origins[d.origin] = true
	}
	if !origins[Point{0, 0}] || !origins[Point{512, 0}] {
		t.Errorf("redrawn tiles %v", origins)
	}
	// the old image stays on its layer while the tile is redrawn: no layer was removed or recreated
	if len(g.h.live(GroupTiles)) != 30 {
		t.Errorf("%d tile layers", len(g.h.live(GroupTiles)))
	}
}

func TestPanIsWholePixelsAndEvicts(t *testing.T) {
	g := newRig(t, Options{})
	g.addBox(1, 100, 100, 50, 50)
	g.settle()
	for i, ox := range []float64{13.3, 100.77, 255.5} {
		g.c.SetCamera(Camera{Origin: Point{ox, ox / 3}, Zoom: 1})
		g.settle()
		for _, l := range g.h.live(GroupTiles) {
			for _, v := range []float64{l.frame.Min.X, l.frame.Min.Y, l.frame.Max.X, l.frame.Max.Y} {
				if p := v * 2; math.Abs(p-math.Round(p)) > 1e-9 {
					t.Fatalf("step %d: tile edge %v is not a whole device pixel", i, v)
				}
			}
			if w := l.frame.W(); math.Abs(w-256) > 1 {
				t.Fatalf("tile width %v", w)
			}
		}
	}
	if n := len(g.h.live(GroupTiles)); n > 8*7 {
		t.Errorf("%d tile layers kept after panning", n)
	}
}

func TestZoomGestureScalesThenRedraws(t *testing.T) {
	g := newRig(t, Options{Settle: 100 * time.Millisecond})
	g.addBox(1, 100, 100, 50, 50)
	g.settle()
	n0 := len(g.r.tileDraws())
	// a gesture: zoom changes on every frame for 10 frames
	for i := 0; i < 10; i++ {
		g.c.ZoomAt(1.05, Point{400, 300})
		g.frame()
	}
	time.Sleep(20 * time.Millisecond)
	g.frame()
	for _, d := range g.r.tileDraws()[n0:] {
		if d.scale != 2 {
			t.Fatalf("a tile was drawn at scale %v during the gesture", d.scale)
		}
	}
	z := g.c.Camera().Zoom
	for _, l := range g.h.live(GroupTiles) {
		if w := l.frame.W(); math.Abs(w-256*z) > 1 {
			t.Fatalf("tile width %v during the gesture, want about %v", w, 256*z)
		}
	}
	g.settle()
	want := math.Exp2(math.Ceil(math.Log2(z*2) - 0.01))
	sawNew := false
	for _, d := range g.r.tileDraws()[n0:] {
		sawNew = sawNew || d.scale == want
	}
	if !sawNew {
		t.Errorf("no tile redrawn at scale %v after the gesture (zoom %v)", want, z)
	}
	// the old tiles are gone once the new ones cover the view
	for _, l := range g.h.live(GroupTiles) {
		if w := l.frame.W(); math.Abs(w-512/want*z*2/2*1) > 2 && math.Abs(w-512*z/want) > 2 {
			t.Fatalf("tile width %v after settling", w)
		}
	}
}

func TestVisibleTilesFirst(t *testing.T) {
	g := newRig(t, Options{Workers: 1})
	g.r.delay = 2 * time.Millisecond
	g.addBox(1, 100, 100, 50, 50)
	g.c.SetCamera(Camera{Zoom: 1})
	g.settle()
	view := R(0, 0, 800, 600)
	seenMargin := false
	for _, d := range g.r.tileDraws() {
		inView := R(d.origin.X, d.origin.Y, 256, 256).Intersects(view)
		if !inView {
			seenMargin = true
		} else if seenMargin {
			t.Fatalf("a tile in view (%v) was drawn after a margin tile", d.origin)
		}
	}
}

func TestLiftKeepsTilesQuiet(t *testing.T) {
	g := newRig(t, Options{})
	n := g.addBox(1, 100, 100, 50, 50)
	g.addBox(2, 600, 400, 50, 50)
	g.settle()
	g.c.Lift(n)
	g.settle()
	// lifting redraws the tiles under the node once, without it
	n0 := len(g.r.tileDraws())
	act := g.h.live(GroupActive)
	if len(act) != 1 || act[0].frame.Min != (Point{100, 100}) || act[0].frame.W() < 50 {
		t.Fatalf("sprite %+v", act)
	}
	for i := 1; i <= 20; i++ {
		g.s.Update(func(tx *Tx) { tx.Move(n, 100+float64(i)*10, 100+float64(i)*5) })
		g.frame()
	}
	g.settle()
	if got := len(g.r.tileDraws()); got != n0 {
		t.Errorf("%d tiles redrawn while a lifted node moved", got-n0)
	}
	if f := act[0].frame; f.Min != (Point{300, 200}) {
		t.Errorf("sprite at %v, want (300, 200)", f.Min)
	}
	if act[0].images != 1 {
		t.Errorf("the sprite was drawn %d times, a plain move must not redraw it", act[0].images)
	}
	g.c.Land(n)
	g.settle()
	if len(g.h.live(GroupActive)) != 0 {
		t.Error("sprite still there after the tiles were redrawn")
	}
	if len(g.r.tileDraws()) == n0 {
		t.Error("landing redrew no tile")
	}
}

// A node moved in the frame it is lifted must not show at the new place while a tile still holds
// its old pixels: the sprite is drawn by Lift itself and waits where the node was.
func TestLiftHoldsSpriteUntilTilesAreClean(t *testing.T) {
	g := newRig(t, Options{})
	n := g.addBox(1, 100, 100, 50, 50)
	g.settle()
	g.c.Lift(n)
	act := g.h.live(GroupActive)
	if len(act) != 1 || act[0].images != 1 || act[0].frame.Min != (Point{100, 100}) {
		t.Fatalf("sprite right after Lift: %+v", act)
	}
	g.s.Update(func(tx *Tx) { tx.Move(n, 300, 200) })
	g.c.Frame(FrameInfo{Time: g.now + time.Millisecond, Interval: time.Millisecond, Seq: g.seq + 1})
	if g.c.clean(R(100, 100, 50, 50)) && act[0].frame.Min != (Point{300, 200}) {
		t.Errorf("tiles are clean but the sprite is held at %v", act[0].frame.Min)
	}
	if !g.c.clean(R(100, 100, 50, 50)) && act[0].frame.Min != (Point{100, 100}) {
		t.Errorf("sprite at %v while a tile under the lifted node is stale", act[0].frame.Min)
	}
	g.settle()
	if act[0].frame.Min != (Point{300, 200}) || act[0].images != 1 {
		t.Errorf("after the tiles: sprite at %v, drawn %d times", act[0].frame.Min, act[0].images)
	}
}

// Every host transaction, Lift's own included, must show a lifted node exactly once: in a tile or in
// its sprite, never in both (a shadow would be blended twice) and never in neither.
func TestLiftShowsNodeOnceInEveryTransaction(t *testing.T) {
	g := newRig(t, Options{})
	n := g.addBox(1, 100, 100, 50, 50)
	g.settle()
	g.r.delay = 3 * time.Millisecond // tiles lag behind the frames
	commits := 0
	g.h.onCommit = func() {
		commits++
		seen := 0
		for _, l := range g.h.layers {
			if l.removed || l.hidden || l.img == nil {
				continue
			}
			if l.group == GroupActive || (l.group == GroupTiles && l.img.items > 0) {
				seen++
			}
		}
		if seen != 1 {
			for _, l := range g.h.layers {
				if !l.removed && l.img != nil {
					t.Logf("  layer group %d hidden %v items %d frame %v", l.group, l.hidden, l.img.items, l.frame)
				}
			}
			t.Errorf("transaction %d shows the node %d times", commits, seen)
		}
	}
	g.c.Lift(n)
	for i := 1; i <= 30; i++ {
		g.s.Update(func(tx *Tx) { tx.Move(n, 100+float64(i)*10, 100) })
		g.frame()
		time.Sleep(time.Millisecond)
	}
	g.settle()
	g.c.Land(n)
	for i := 0; i < 10; i++ {
		g.frame()
		time.Sleep(time.Millisecond)
	}
	g.settle()
	if len(g.h.live(GroupActive)) != 0 {
		t.Error("sprite still there after landing")
	}
	g.h.onCommit = nil // Close removes every layer
}

func TestLiftedLiveNodeRedrawsEachFrame(t *testing.T) {
	g := newRig(t, Options{})
	conn := g.addBox(1, 100, 100, 50, 50)
	g.settle()
	g.c.Lift(conn)
	g.settle() // a lifted sprite is held in place until the tiles are redrawn without its node
	l := g.h.live(GroupActive)[0]
	base := l.images
	for i := 0; i < 5; i++ {
		w := 50 + float64(i+1)*10
		g.s.Update(func(tx *Tx) { tx.SetPicture(conn, boxPic(w, 50)) })
		g.frame()
	}
	if l.images != base+5 {
		t.Errorf("sprite drawn %d times in 5 frames of a changing picture", l.images-base)
	}
}

func TestSpriteAnimation(t *testing.T) {
	g := newRig(t, Options{})
	g.settle()
	sp := g.c.NewSprite(boxPic(10, 10), SpaceScreen, Point{0, 0})
	g.frame()
	l := g.h.live(GroupOverlay)[0]
	sp.MoveTo(Point{100, 200}, 160*time.Millisecond, Linear, nil)
	sp.FadeTo(0.5, 160*time.Millisecond, Linear, nil)
	var finished bool
	sp.RotateTo(1, 160*time.Millisecond, EaseInOut, func() { finished = true })
	g.frame() // starts the clock
	for i := 0; i < 5; i++ {
		g.frame()
	}
	// 5 frames of 16 ms = 80 ms of 160: half way
	if p := sp.Pos(); math.Abs(p.X-50) > 1e-6 || math.Abs(p.Y-100) > 1e-6 {
		t.Errorf("position at half time %v", p)
	}
	if math.Abs(l.opacity-0.75) > 1e-6 {
		t.Errorf("opacity at half time %v", l.opacity)
	}
	if finished {
		t.Error("done ran early")
	}
	for i := 0; i < 6; i++ {
		g.frame()
	}
	if p := sp.Pos(); p != (Point{100, 200}) || !finished || l.opacity != 0.5 {
		t.Errorf("end state %v finished=%v opacity=%v", p, finished, l.opacity)
	}
	if g.c.NeedsFrame() {
		t.Error("animation over but frames still wanted")
	}
}

func TestCursorFliesAlongCubicOverBusyTiles(t *testing.T) {
	g := newRig(t, Options{})
	g.addBox(1, 100, 100, 50, 50)
	g.settle()
	cur := g.c.NewSprite(boxPic(8, 8), SpacePinned, Point{})
	path := &Path{}
	path.MoveTo(100, 100)
	path.CubicTo(500, 50, 900, 800, 1300, 700)
	cur.FlyAlong(path, 320*time.Millisecond, Linear, true, nil)
	g.frame()
	var prev Point
	moved := 0
	for i := 0; i < 25; i++ {
		// scene edits every frame keep tiles being redrawn while the cursor flies
		g.s.Update(func(tx *Tx) { tx.Move(g.s.Node(1), 100+float64(i), 100) })
		g.frame()
		if p := cur.Pos(); p != prev {
			moved++
			prev = p
		}
	}
	if moved < 18 {
		t.Errorf("the cursor moved on %d of 25 frames", moved)
	}
	if end := cur.Pos(); end.Dist(Point{1300, 700}) > 1e-6 {
		t.Errorf("end %v", end)
	}
	l := g.h.live(GroupOverlay)[0]
	if l.m == Identity() {
		t.Error("orient: the sprite must be rotated along the path")
	}
}

func TestEasingEndpoints(t *testing.T) {
	for name, e := range map[string]Easing{"linear": Linear, "in": EaseIn, "out": EaseOut, "inout": EaseInOut, "bezier": CubicBezier(0.25, 0.1, 0.25, 1)} {
		if e(0) != 0 || math.Abs(e(1)-1) > 1e-9 {
			t.Errorf("%s: %v %v", name, e(0), e(1))
		}
		prev := -1.0
		for i := 0; i <= 20; i++ {
			v := e(float64(i) / 20)
			if v < prev-1e-9 {
				t.Errorf("%s not monotonic at %d", name, i)
			}
			prev = v
		}
	}
	if v := CubicBezier(0.25, 0.1, 0.25, 1)(0.5); v < 0.75 || v > 0.85 {
		t.Errorf("CSS ease at 0.5: %v", v)
	}
}

func TestZoomAtKeepsPoint(t *testing.T) {
	g := newRig(t, Options{})
	g.c.SetCamera(Camera{Origin: Point{100, 50}, Zoom: 2})
	p := Point{300, 200}
	before := g.c.SceneAt(p)
	g.c.ZoomAt(1.5, p)
	if after := g.c.SceneAt(p); after.Dist(before) > 1e-9 {
		t.Errorf("point moved %v -> %v", before, after)
	}
	if v := g.c.ViewAt(before); v.Dist(p) > 1e-9 {
		t.Errorf("ViewAt %v", v)
	}
}

// An OnFrame callback drives model animations, so it must keep frames coming on an idle scene.
func TestOnFrameKeepsTicking(t *testing.T) {
	g := newRig(t, Options{})
	g.settle()
	if g.c.NeedsFrame() {
		t.Fatal("idle compositor needs a frame")
	}
	n := 0
	g.c.OnFrame(func(FrameInfo) { n++ })
	if !g.c.NeedsFrame() {
		t.Fatal("NeedsFrame is false while OnFrame is set")
	}
	g.frame()
	if n != 1 {
		t.Fatalf("OnFrame ran %d times, want 1", n)
	}
	g.c.OnFrame(nil)
	if g.c.NeedsFrame() {
		t.Fatal("NeedsFrame stays true after OnFrame(nil)")
	}
}

// A change to a sprite on an idle scene must ask for a frame by itself: a crosshair moved or a pulse
// started from an event handler would otherwise wait for the next camera move.
func TestSpriteChangeWakesAnIdleScene(t *testing.T) {
	g := newRig(t, Options{})
	g.settle()
	sp := g.c.NewSprite(boxPic(10, 10), SpaceScreen, Point{})
	if !g.c.NeedsFrame() {
		t.Fatal("a new sprite does not ask for its first frame")
	}
	g.settle()
	for name, change := range map[string]func(){
		"SetPos":     func() { sp.SetPos(Point{5, 5}) },
		"SetOpacity": func() { sp.SetOpacity(0.5) },
		"SetPicture": func() { sp.SetPicture(boxPic(20, 20)) },
		"MoveTo":     func() { sp.MoveTo(Point{50, 50}, 64*time.Millisecond, Linear, nil) },
		"FadeTo":     func() { sp.FadeTo(0, 64*time.Millisecond, Linear, nil) },
		"Remove":     sp.Remove,
	} {
		if g.c.NeedsFrame() {
			t.Fatalf("%s: frames wanted before the change", name)
		}
		change()
		if !g.c.NeedsFrame() {
			t.Errorf("%s does not ask for a frame", name)
		}
		g.settle()
	}
}
