package gfx

import (
	"math"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Options tune a Compositor; the zero value gives the defaults noted on each field.
type Options struct {
	TilePx  int           // tile side in device pixels; 512
	Workers int           // tile rasterizer goroutines; NumCPU-2, at most 6, at least 1
	Margin  int           // tiles kept around the view; 1
	Settle  time.Duration // zoom stands still this long before tiles are redrawn at the new scale; 120 ms
	// ExactZoom draws tiles at exactly the zoom after a gesture instead of the next power of two
	// above it. Sharper at in-between zooms, but no tile is reused when the zoom comes back.
	ExactZoom bool
}

func (o *Options) fill() {
	if o.TilePx <= 0 {
		o.TilePx = 512
	}
	if o.Workers <= 0 {
		o.Workers = min(max(runtime.NumCPU()-2, 1), 6)
	}
	if o.Margin <= 0 {
		o.Margin = 1
	}
	if o.Settle <= 0 {
		o.Settle = 120 * time.Millisecond
	}
}

// Camera maps the scene to the view: Origin is the scene point at the view's top left, Zoom is
// view points per scene unit.
type Camera struct {
	Origin Point
	Zoom   float64
}

// Stats counts what the compositor did; Frame fills the per-frame fields.
type Stats struct {
	FrameTime    time.Duration // CPU time of the last Frame
	TilesApplied int           // tile images put on layers by the last Frame
	TilesPending int           // tiles queued or being drawn
	Rastered     uint64        // tiles drawn since the start
	RasterTime   time.Duration // total worker time spent drawing tiles
	SpriteDraws  int           // sprites drawn by the last Frame on the calling goroutine
}

type tileKey [2]int32

type tile struct {
	key     tileKey
	layer   Layer
	has     bool // an image is on the layer
	dirty   bool // the picture is out of date: redraw
	queued  bool
	gen     atomic.Int32 // bumps when the scene changes under the tile
	evicted atomic.Bool
	rect    Rect  // in scene units
	pend    Image // a finished image held back while a lift or landing is under way, see drain
}

// drop takes the tile out for good.
func (t *tile) drop() {
	t.evicted.Store(true)
	if t.layer != nil {
		t.layer.Remove()
	}
	if t.pend != nil {
		Release(t.pend)
		t.pend = nil
	}
}

type tileSet struct {
	scale float64 // device pixels per scene unit
	dpr   float64
	tw    float64 // tile side in scene units
	tiles map[tileKey]*tile
}

func (s *tileSet) rect(k tileKey) Rect {
	return R(float64(k[0])*s.tw, float64(k[1])*s.tw, s.tw, s.tw)
}

type job struct {
	t *tile
	s *tileSet
}

type result struct {
	t   *tile
	gen int32
	img Image
}

type camSnap struct {
	center Point
	hw, hh float64
}

type lift struct {
	node    *Node
	layer   Layer
	scale   float64
	rb      Rect // scene area drawn into the image
	clipped bool // rb is less than the node's bounds
	w0      Affine
	rev     uint32
	deep    uint32
	has     bool
	wpx     int
	hpx     int
	landing bool
	// hold: the sprite is drawn but hidden, and the node stays in the tiles, until the tiles under it
	// are redrawn without it; then both change in one host transaction, so the node is on screen once
	hold  bool
	hrect Rect // the area whose tiles change when the node leaves or returns
	// pending: the node is still in the tiles, scene.Lift has not run: the first sprite is being drawn
	// on a worker. drawing: a worker draws an image for this lift, the old one (if any) stays placed.
	pending, drawing bool
	dead             bool // removed from c.lifts: a finished image is dropped
}

// scratch is a target and a collect buffer for drawing a sprite image; index 0 is the Frame
// goroutine's, the others belong to the goroutines of a parallel batch.
type scratch struct {
	t     Target
	items Items
}

// liftJob is one lifted node whose sprite image must be drawn again.
type liftJob struct {
	l         *lift
	area      Rect
	wpx, hpx  int
	w         Affine
	rev, deep uint32
	want      float64
	clipped   bool
	img       Image
}

// Compositor shows a Scene on a Host: static tiles drawn by a pool of workers, an active layer of
// lifted nodes (sprites that follow their node), and an overlay of free sprites. Everything but
// NeedsFrame and the Sprite methods is for the goroutine that runs Frame.
type Compositor struct {
	scene *Scene
	r     Renderer
	host  Host
	o     Options

	vw, vh, dpr float64
	cam         Camera
	camDirty    atomic.Bool
	// spriteDirty is set by every change to a Sprite, which Frame has not seen yet: it makes NeedsFrame true,
	// so a sprite moved or animated while the scene is idle still gets its frame.
	spriteDirty atomic.Bool
	ticking     atomic.Bool // OnFrame is set: frames run on every display tick

	cur, prev  *tileSet
	zoomT      time.Duration
	zooming    bool
	lastZoom   float64
	lastDPR    float64
	layoutDrty bool
	settled    bool // no tile under a lift in transition waits for a redraw, see drain
	lastRange  [4]int32

	lifts          []*lift
	sprites        []*Sprite
	smu            sync.Mutex
	pendingSprites []*Sprite
	scr            []scratch
	jobs           []liftJob
	liftReady      []liftJob // images drawn on workers, guarded by mu
	onFrame        func(FrameInfo)
	fi             FrameInfo
	busy           bool

	mu      sync.Mutex
	cond    *sync.Cond
	queue   []job
	ready   []result
	closed  bool
	readyN  atomic.Bool
	liftN   atomic.Bool // liftReady has something
	snap    atomic.Pointer[camSnap]
	pending atomic.Int32
	wg      sync.WaitGroup

	rastered atomic.Uint64
	rasterNs atomic.Int64
	stats    Stats
}

// NewCompositor returns a compositor that draws s with r onto h. Most apps get one from view.New.
func NewCompositor(s *Scene, r Renderer, h Host, o Options) *Compositor {
	o.fill()
	c := &Compositor{scene: s, r: r, host: h, o: o, dpr: 1, vw: 800, vh: 600, cam: Camera{Zoom: 1}, lastZoom: -1, scr: make([]scratch, 1)}
	c.cond = sync.NewCond(&c.mu)
	c.snap.Store(&camSnap{})
	c.camDirty.Store(true)
	for i := 0; i < o.Workers; i++ {
		c.wg.Add(1)
		go c.worker()
	}
	return c
}

// Close stops the workers and removes every layer.
func (c *Compositor) Close() {
	c.mu.Lock()
	c.closed = true
	c.cond.Broadcast()
	c.mu.Unlock()
	c.wg.Wait()
	c.host.Begin()
	for _, set := range []*tileSet{c.cur, c.prev} {
		c.dropSet(set)
	}
	c.cur, c.prev = nil, nil
	for _, l := range c.lifts {
		l.layer.Remove()
	}
	c.lifts = nil
	for _, sp := range c.sprites {
		sp.drop()
	}
	c.sprites = nil
	c.host.Commit()
	c.mu.Lock()
	for _, r := range c.ready {
		Release(r.img)
	}
	c.ready = nil
	for _, j := range c.liftReady {
		Release(j.img)
	}
	c.liftReady = nil
	c.mu.Unlock()
	for i := range c.scr {
		if c.scr[i].t != nil {
			c.scr[i].t.Close()
		}
	}
}

// SetViewport sets the view size in points and the device pixels per point.
func (c *Compositor) SetViewport(w, h, dpr float64) {
	if w != c.vw || h != c.vh || dpr != c.dpr {
		c.vw, c.vh, c.dpr = w, h, dpr
		c.camDirty.Store(true)
	}
}

// SetCamera moves and zooms the view; Zoom is view points per scene unit.
func (c *Compositor) SetCamera(cam Camera) {
	if cam.Zoom <= 0 {
		return
	}
	if cam != c.cam {
		c.cam = cam
		c.camDirty.Store(true)
	}
}

// Workers is the number of tile rasterizer goroutines.
func (c *Compositor) Workers() int { return c.o.Workers }

// Camera returns the current camera.
func (c *Compositor) Camera() Camera { return c.cam }

// SceneAt maps a view point to a scene point.
func (c *Compositor) SceneAt(p Point) Point {
	return Point{c.cam.Origin.X + p.X/c.cam.Zoom, c.cam.Origin.Y + p.Y/c.cam.Zoom}
}

// ViewAt maps a scene point to a view point.
func (c *Compositor) ViewAt(p Point) Point {
	return Point{(p.X - c.cam.Origin.X) * c.cam.Zoom, (p.Y - c.cam.Origin.Y) * c.cam.Zoom}
}

// ZoomAt multiplies the zoom by f keeping the scene point under the view point p where it is.
func (c *Compositor) ZoomAt(f float64, p Point) {
	sp := c.SceneAt(p)
	z := c.cam.Zoom * f
	c.SetCamera(Camera{Origin: Point{sp.X - p.X/z, sp.Y - p.Y/z}, Zoom: z})
}

// OnFrame sets f to run at the start of every Frame with the frame clock, before the scene's
// transactions are applied: the place to advance model animations. While f is set, NeedsFrame is
// true and the display link runs a frame on every tick; OnFrame(nil) lets an idle scene stop ticking.
func (c *Compositor) OnFrame(f func(FrameInfo)) {
	c.onFrame = f
	c.ticking.Store(f != nil)
}

// FrameTime is the clock of the Frame being or last run.
func (c *Compositor) FrameTime() FrameInfo { return c.fi }

// Stats returns the counters of the last frame and the totals so far.
func (c *Compositor) Stats() Stats {
	s := c.stats
	s.Rastered = c.rastered.Load()
	s.RasterTime = time.Duration(c.rasterNs.Load())
	s.TilesPending = int(c.pending.Load())
	return s
}

// NeedsFrame reports whether a Frame would change anything: safe from any goroutine, cheap enough
// for a display-link callback to call on every tick.
func (c *Compositor) NeedsFrame() bool {
	return c.ticking.Load() || c.camDirty.Load() || c.spriteDirty.Load() || c.readyN.Load() || c.liftN.Load() || c.busy || c.scene.Pending()
}

func (c *Compositor) viewRect() Rect {
	return R(c.cam.Origin.X, c.cam.Origin.Y, c.vw/c.cam.Zoom, c.vh/c.cam.Zoom)
}

func (c *Compositor) snapScale() float64 {
	s := c.cam.Zoom * c.dpr
	if c.o.ExactZoom {
		return s
	}
	return math.Exp2(math.Ceil(math.Log2(s) - 0.01))
}

func (c *Compositor) newSet() *tileSet {
	s := c.snapScale()
	return &tileSet{scale: s, dpr: c.dpr, tw: float64(c.o.TilePx) / s, tiles: map[tileKey]*tile{}}
}

func (c *Compositor) dropSet(s *tileSet) {
	if s == nil {
		return
	}
	for k, t := range s.tiles {
		t.drop()
		delete(s.tiles, k)
	}
}

// Frame brings the host up to date for the frame described by fi. It runs on the UI goroutine.
func (c *Compositor) Frame(fi FrameInfo) {
	t0 := time.Now()
	c.fi = fi
	c.stats.TilesApplied, c.stats.SpriteDraws = 0, 0
	if c.onFrame != nil {
		c.onFrame(fi)
	}
	damage := c.scene.Commit()
	camMoved := c.camDirty.Swap(false)
	c.spriteDirty.Store(false)
	c.host.Begin()
	if camMoved {
		v := c.viewRect().Center()
		c.snap.Store(&camSnap{v, c.vw / c.cam.Zoom / 2, c.vh / c.cam.Zoom / 2})
		c.layoutDrty = true
	}
	c.handleZoom(fi)
	if len(damage) > 0 {
		c.invalidate(damage)
	}
	c.applyLifts()
	c.ensureTiles(camMoved || len(damage) > 0)
	c.drain()
	c.stepSprites(fi, camMoved)
	if c.layoutDrty {
		c.layout()
		c.layoutDrty = false
	}
	c.host.Commit()
	c.busy = c.zooming || len(c.lifts) > 0 && c.anyLanding() || c.animating()
	c.stats.FrameTime = time.Since(t0)
}

func (c *Compositor) handleZoom(fi FrameInfo) {
	if c.cam.Zoom != c.lastZoom || c.dpr != c.lastDPR {
		c.lastZoom, c.lastDPR = c.cam.Zoom, c.dpr
		c.zoomT, c.zooming = fi.Time, true
	}
	if c.cur == nil {
		c.cur, c.zooming = c.newSet(), false
		return
	}
	if c.zooming && fi.Time-c.zoomT >= c.o.Settle {
		c.zooming = false
		if want := c.snapScale(); want != c.cur.scale || c.dpr != c.cur.dpr {
			c.dropSet(c.prev)
			c.prev, c.cur = c.cur, c.newSet()
			c.layoutDrty = true
		}
	}
}

// invalidate marks the tiles under the damaged scene areas for redraw.
func (c *Compositor) invalidate(damage []Rect) {
	for _, set := range []*tileSet{c.cur, c.prev} {
		if set == nil {
			continue
		}
		for _, d := range damage {
			x0, y0, x1, y1 := set.span(d)
			for y := y0; y <= y1; y++ {
				for x := x0; x <= x1; x++ {
					if t := set.tiles[tileKey{x, y}]; t != nil {
						t.gen.Add(1)
						t.dirty = true
					}
				}
			}
		}
	}
}

func (s *tileSet) span(r Rect) (x0, y0, x1, y1 int32) {
	return int32(math.Floor(r.Min.X / s.tw)), int32(math.Floor(r.Min.Y / s.tw)), int32(math.Floor(r.Max.X / s.tw)), int32(math.Floor(r.Max.Y / s.tw))
}

// ensureTiles creates and queues the tiles of the current set that the view and its margin need,
// and drops the ones far outside.
func (c *Compositor) ensureTiles(moved bool) {
	set := c.cur
	x0, y0, x1, y1 := set.span(c.viewRect())
	m := int32(c.o.Margin)
	x0, y0, x1, y1 = x0-m, y0-m, x1+m, y1+m
	rng := [4]int32{x0, y0, x1, y1}
	if moved || rng != c.lastRange {
		c.lastRange = rng
		for k, t := range set.tiles {
			if k[0] < x0-1 || k[0] > x1+1 || k[1] < y0-1 || k[1] > y1+1 {
				t.drop()
				delete(set.tiles, k)
			}
		}
	}
	// far from the scale of the tiles, new ones are not worth drawing in the middle of a gesture:
	// the next set is about to replace them
	ratio := c.cam.Zoom * c.dpr / set.scale
	skipNew := c.zooming && (ratio < 0.5 || ratio > 2)
	var add []job
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			k := tileKey{x, y}
			t := set.tiles[k]
			if t == nil {
				if skipNew {
					continue
				}
				t = &tile{key: k, dirty: true, rect: set.rect(k)}
				set.tiles[k] = t
			}
			if t.dirty && !t.queued {
				t.queued = true
				add = append(add, job{t, set})
			}
		}
	}
	if len(add) > 0 {
		c.pending.Add(int32(len(add)))
		c.mu.Lock()
		c.queue = append(c.queue, add...)
		c.cond.Broadcast()
		c.mu.Unlock()
	}
}

// next takes the most urgent job: tiles in view first, then the nearest to the view's center.
func (c *Compositor) next() (job, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		if c.closed {
			return job{}, false
		}
		// drop what was evicted while it waited
		w := c.queue[:0]
		for _, j := range c.queue {
			if j.t.evicted.Load() {
				c.pending.Add(-1)
				c.pushLocked(result{t: j.t, gen: -1})
				continue
			}
			w = append(w, j)
		}
		c.queue = w
		if len(c.queue) > 0 {
			break
		}
		c.cond.Wait()
	}
	ctr := c.snap.Load().center
	best, bestD := 0, math.Inf(1)
	for i, j := range c.queue {
		r := j.s.rect(j.t.key)
		d := r.Center().Dist(ctr)
		if !r.Intersects(c.viewRectAtomic()) {
			d += 1e12
		}
		if d < bestD {
			best, bestD = i, d
		}
	}
	j := c.queue[best]
	c.queue = append(c.queue[:best], c.queue[best+1:]...)
	return j, true
}

// pushLocked adds a result (gen -1 means "give the queued flag back") for the next Frame.
func (c *Compositor) pushLocked(r result) {
	c.ready = append(c.ready, r)
	c.readyN.Store(true)
}

// viewRectAtomic is the view as of the last camera change, for the workers: a stale read only
// misorders a job.
func (c *Compositor) viewRectAtomic() Rect {
	s := c.snap.Load()
	return R(s.center.X-s.hw, s.center.Y-s.hh, 2*s.hw, 2*s.hh)
}

func (c *Compositor) worker() {
	defer c.wg.Done()
	var t Target
	var items Items
	defer func() {
		if t != nil {
			t.Close()
		}
	}()
	for {
		j, ok := c.next()
		if !ok {
			return
		}
		if t == nil {
			t = c.r.NewTarget(c.o.TilePx, c.o.TilePx)
		}
		gen := j.t.gen.Load()
		start := time.Now()
		rect := j.s.rect(j.t.key)
		c.scene.Collect(rect, &items)
		t.Begin(j.s.scale, rect.Min, Color{})
		t.Draw(items.List)
		img := t.Snapshot(float64(c.o.TilePx)/j.s.dpr, float64(c.o.TilePx)/j.s.dpr)
		c.rasterNs.Add(int64(time.Since(start)))
		c.rastered.Add(1)
		c.pending.Add(-1)
		c.mu.Lock()
		c.pushLocked(result{j.t, gen, img})
		c.mu.Unlock()
	}
}

// pendingTileBudget is the most tile images a frame applies while a sprite is drawn: the host
// commit costs about 0.15 ms per tile.
const pendingTileBudget = 8

// tileApplyTime caps the time a frame spends applying finished tiles (a whole new set cost 40-76 ms).
// Tiles in view go first; the previous set stays under them until the view is covered.
const tileApplyTime = 4 * time.Millisecond

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// drain puts the finished tile images on their layers. While a lift or a landing is under way, the
// images of tiles under it are held back until every such tile is redrawn, and then applied in the
// same transaction in which the sprites appear or go: a node is never seen in its sprite and in a
// tile at once, and never in neither.
func (c *Compositor) drain() {
	if c.readyN.Load() {
		c.mu.Lock()
		res := c.ready
		c.ready = nil
		c.readyN.Store(false)
		c.mu.Unlock()
		var trs []Rect
		for _, l := range c.lifts {
			if l.hold || l.landing {
				trs = append(trs, l.hrect)
			}
		}
		// while a first sprite is drawn the node moves in the tiles, which redraw every frame: apply
		// only what fits the budget, the rest waits for the next frame
		budget := len(res)
		if slices.ContainsFunc(c.lifts, func(l *lift) bool { return l.pending }) {
			budget = pendingTileBudget
		}
		view, applyStart := c.viewRect(), time.Now()
		slices.SortStableFunc(res, func(a, b result) int {
			return boolInt(b.t.rect.Intersects(view)) - boolInt(a.t.rect.Intersects(view))
		})
		for i, r := range res {
			if i >= budget || i > 0 && time.Since(applyStart) > tileApplyTime {
				c.mu.Lock()
				c.ready = append(res[i:len(res):len(res)], c.ready...)
				c.readyN.Store(true)
				c.mu.Unlock()
				break
			}
			t := r.t
			t.queued = false
			if r.gen < 0 {
				continue
			}
			if t.evicted.Load() {
				Release(r.img)
				continue
			}
			t.dirty = r.gen != t.gen.Load()
			if slices.ContainsFunc(trs, t.rect.Intersects) {
				if t.pend != nil {
					Release(t.pend)
				}
				t.pend = r.img
				continue
			}
			c.show(t, r.img)
		}
	}
	c.requeue()
	c.settled = true
	for _, l := range c.lifts {
		if (l.hold || l.landing) && !c.clean(l.hrect) {
			c.settled = false
			break
		}
	}
	if c.settled {
		for _, set := range []*tileSet{c.prev, c.cur} {
			if set == nil {
				continue
			}
			for _, t := range set.tiles {
				if t.pend != nil {
					c.show(t, t.pend)
					t.pend = nil
				}
			}
		}
	}
	if c.prev != nil && c.covered() {
		c.dropSet(c.prev)
		c.prev = nil
	}
}

// show puts a finished image on the layer of t and releases it.
func (c *Compositor) show(t *tile, img Image) {
	if t.layer == nil {
		t.layer = c.host.NewLayer(GroupTiles)
	}
	t.layer.SetImage(img)
	Release(img)
	t.has = true
	c.stats.TilesApplied++
	c.layoutDrty = true
}

// requeue puts tiles that changed while they were being drawn back in line.
func (c *Compositor) requeue() {
	var add []job
	for _, t := range c.cur.tiles {
		if t.dirty && !t.queued {
			t.queued = true
			add = append(add, job{t, c.cur})
		}
	}
	if len(add) > 0 {
		c.pending.Add(int32(len(add)))
		c.mu.Lock()
		c.queue = append(c.queue, add...)
		c.cond.Broadcast()
		c.mu.Unlock()
	}
}

// covered reports whether every tile in view has an image at the current scale.
func (c *Compositor) covered() bool {
	x0, y0, x1, y1 := c.cur.span(c.viewRect())
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if t := c.cur.tiles[tileKey{x, y}]; t == nil || !t.has {
				return false
			}
		}
	}
	return true
}

// clean reports whether no tile touching r in the current set waits for a redraw.
func (c *Compositor) clean(r Rect) bool {
	x0, y0, x1, y1 := c.cur.span(r)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if t := c.cur.tiles[tileKey{x, y}]; t != nil && (t.dirty || t.queued) {
				return false
			}
		}
	}
	return true
}

// px rounds a view coordinate in points to a whole device pixel.
func (c *Compositor) px(v float64) float64 { return math.Round(v*c.dpr) / c.dpr }

func (c *Compositor) layout() {
	for _, set := range []*tileSet{c.prev, c.cur} {
		if set == nil {
			continue
		}
		for _, t := range set.tiles {
			if t.layer == nil {
				continue
			}
			r := set.rect(t.key)
			a, b := c.ViewAt(r.Min), c.ViewAt(r.Max)
			// round both edges, so that neighbours meet exactly and no tile is resampled by a fraction
			f := Rect{Point{c.px(a.X), c.px(a.Y)}, Point{c.px(b.X), c.px(b.Y)}}
			t.layer.Set(f, Point{}, Identity())
		}
	}
}

// Lift moves nodes, with their subtrees, from the static tiles to the active layer: each becomes a
// sprite that follows its node without redrawing tiles. Nodes whose picture changes every frame, such
// as connectors, are redrawn into their sprite on each Frame. Call Land when the drag ends.
func (c *Compositor) Lift(nodes ...*Node) {
	var dmg []Rect
	c.host.Begin()
	fresh := make([]*lift, 0, len(nodes))
	for _, n := range nodes {
		l := &lift{node: n, layer: c.host.NewLayer(GroupActive)}
		c.lifts = append(c.lifts, l)
		// a sprite too big to draw within a frame is drawn on a worker first; the node stays in the
		// tiles until it is ready (applyLifts)
		if c.bigSprite(l) {
			l.pending = true
			c.drawAsync(l)
			continue
		}
		l.hrect = c.scene.liftNode(n)
		dmg = append(dmg, l.hrect)
		fresh = append(fresh, l)
	}
	c.invalidate(dmg)
	// draw now, before the frame applies the moves that follow the lift
	c.refreshLifts(fresh)
	for _, l := range fresh {
		if l.has && c.cur != nil {
			l.hold = true
			l.layer.SetHidden(true)
		}
	}
	c.host.Commit()
	c.busy = true
}

// Land returns the nodes to the static tiles. Their sprites stay until the tiles under them are redrawn.
func (c *Compositor) Land(nodes ...*Node) {
	var dmg []Rect
	for _, n := range nodes {
		for _, l := range c.lifts {
			if l.node == n && !l.landing {
				if l.pending {
					// never left the tiles: nothing to return
					l.dead = true
					c.dropLift(l)
					break
				}
				l.landing = true
				r := c.scene.landNode(n)
				l.hrect = l.hrect.Union(r)
				dmg = append(dmg, r)
				break
			}
		}
	}
	c.invalidate(dmg)
	c.busy = true
}

func (c *Compositor) anyLanding() bool {
	for _, l := range c.lifts {
		if l.landing || l.hold {
			return true
		}
	}
	return false
}

func (c *Compositor) animating() bool {
	for _, s := range c.sprites {
		if s.animating() {
			return true
		}
	}
	return false
}
