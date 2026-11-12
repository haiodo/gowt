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
	Workers int           // tiles of this compositor drawn at once; the rasterizers come from a pool shared by all compositors; NumCPU-2, at most 6, at least 1
	Margin  int           // tiles kept around the view; 1
	Settle  time.Duration // zoom stands still this long before tiles are redrawn at the new scale; 120 ms
	// ExactZoom draws tiles at exactly the zoom after a gesture instead of the next power of two
	// above it. Sharper at in-between zooms, but no tile is reused when the zoom comes back.
	ExactZoom bool
	// Direct draws the visible part of the scene in one pass per frame or damage instead of tiles:
	// no tile pool, no tile reuse, no snap to powers of two - every draw is at exactly zoom*dpr.
	// The view is one image covering the viewport, drawn again on every camera move, resize and
	// scene damage. Everything else - camera, HUD, sprites, lifts, input, QA - works as in the
	// tile mode. The viewport in device pixels must fit the renderer's target limit (32767 per
	// side), which a window on one screen always does.
	Direct bool
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
	HUDDraws     int           // times the HUD image was drawn by the last Frame
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

// tilePool is the process-wide pool of tile rasterizer goroutines, shared by every Compositor:
// windows do not multiply the workers. Started lazily on the first queued tile; a worker releases
// its tile-sized target before it waits.
var tilePool = struct {
	mu      sync.Mutex
	cond    *sync.Cond
	comps   map[*Compositor]struct{}
	started bool
	n       int
}{comps: map[*Compositor]struct{}{}}

func init() { tilePool.cond = sync.NewCond(&tilePool.mu) }

// poolDefault is the pool size: the same formula the per-compositor default used.
func poolDefault() int { return min(max(runtime.NumCPU()-2, 1), 6) }

// poolWorkers reports how many worker goroutines the pool has started; for tests.
func poolWorkers() int {
	tilePool.mu.Lock()
	defer tilePool.mu.Unlock()
	return tilePool.n
}

// dropQueued releases every queued tile without drawing: the queued flags go back and the pending
// count sinks, like the gen -1 results a live compositor gets from drain.
func (c *Compositor) dropQueued() {
	for _, j := range c.queue {
		c.pending.Add(-1)
		j.t.queued = false
	}
	c.queue = nil
}

// enqueue hands tiles to the shared pool.
func (c *Compositor) enqueue(add []job) {
	tilePool.mu.Lock()
	c.mu.Lock()
	if c.closed {
		// a frame raced the Close: give the queued flags back and drop the jobs
		c.dropQueuedWith(add)
		c.mu.Unlock()
		tilePool.mu.Unlock()
		return
	}
	c.queue = append(c.queue, add...)
	c.pending.Add(int32(len(add)))
	c.mu.Unlock()
	if !tilePool.started {
		tilePool.started = true
		tilePool.n = poolDefault()
		for i := 0; i < tilePool.n; i++ {
			go tileWorker()
		}
	}
	tilePool.comps[c] = struct{}{}
	tilePool.cond.Broadcast()
	tilePool.mu.Unlock()
}

// dropQueuedWith releases jobs that never reached the queue.
func (c *Compositor) dropQueuedWith(add []job) {
	for _, j := range add {
		j.t.queued = false
	}
}

// take pops the most urgent queued tile of any compositor: tiles in view first, then the nearest
// to the view's center, waiting when there is no work anywhere. release runs before the wait, so
// an idle worker holds no tile target. Locks are taken pool-then-compositor everywhere.
func take(release func()) (*Compositor, job) {
	tilePool.mu.Lock()
	defer tilePool.mu.Unlock()
	for {
		for c := range tilePool.comps {
			c.mu.Lock()
			if c.closed {
				c.dropQueued()
				c.mu.Unlock()
				delete(tilePool.comps, c)
				continue
			}
			if c.inflight >= c.o.Workers {
				// its workers are busy: leave the queue for the wake from a finished tile
				c.mu.Unlock()
				continue
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
			if len(c.queue) == 0 {
				c.mu.Unlock()
				delete(tilePool.comps, c)
				continue
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
			c.inflight++
			c.wg.Add(1) // Close waits for the result push
			c.mu.Unlock()
			return c, j
		}
		release()
		tilePool.cond.Wait()
	}
}

// tileWorker draws tiles for every compositor. The target belongs to one renderer: kept while the
// jobs come from a single compositor, released when the source changes and before waiting.
func tileWorker() {
	var t Target
	var items Items
	var last *Compositor
	defer func() {
		if t != nil {
			t.Close()
		}
	}()
	for {
		c, j := take(func() {
			if t != nil {
				t.Close()
				t = nil
			}
		})
		if last != c && t != nil {
			t.Close()
			t = nil
		}
		last = c
		gen := j.t.gen.Load()
		start := time.Now()
		rect := j.s.rect(j.t.key)
		c.scene.Collect(rect, &items)
		var img Image = blankTile{}
		if len(items.List) > 0 {
			if t == nil {
				t = c.r.NewTarget(c.o.TilePx, c.o.TilePx)
			}
			if ss, ok := t.(ScreenScaler); ok {
				ss.SetScreenScale(j.s.dpr)
			}
			t.Begin(j.s.scale, rect.Min, Color{})
			t.Draw(items.List)
			img = t.Snapshot(float64(c.o.TilePx)/j.s.dpr, float64(c.o.TilePx)/j.s.dpr)
		}
		c.rasterNs.Add(int64(time.Since(start)))
		c.rastered.Add(1)
		c.pending.Add(-1)
		c.mu.Lock()
		c.pushLocked(result{j.t, gen, img})
		c.inflight--
		c.mu.Unlock()
		c.wg.Done()
		// a freed slot may unblock a capped compositor; pool-then-compositor order kept: no locks held
		tilePool.mu.Lock()
		tilePool.cond.Broadcast()
		tilePool.mu.Unlock()
	}
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
	hud         atomic.Pointer[Scene]
	hudState    hudState
	ticking     atomic.Bool // OnFrame is set: frames run on every display tick

	cur, prev *tileSet
	// direct is the Direct mode state: one image of the viewport, redrawn whole on a camera move,
	// a resize or scene damage; damage and lift rects wait in direct.damage for the next Frame.
	direct struct {
		layer  Layer
		damage []Rect
		drawn  bool
		items  Items
		// rot is the rotating viewport target of the Direct mode; when the snapshot shares the
		// target memory, a target is not drawn into again for two frames. sets is the same
		// rotation for the parallel-strip rasterization; only one of them is in use.
		rot    []Target
		sets   []StripSet
		setPx  [][2]int // pixel size of each set in sets
		rotN   int
		rotSet bool // the first target decided whether the snapshot shares memory
		bitems []Items
	}
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
	onViewport     func(w, h, dpr float64)
	fi             FrameInfo
	busy           bool

	mu     sync.Mutex
	queue  []job
	ready  []result
	closed bool
	// inflight is how many queued tiles workers hold right now; take never gives a compositor
	// more than o.Workers at once, the per-compositor concurrency the dedicated workers had.
	inflight int
	readyN   atomic.Bool
	liftN    atomic.Bool // liftReady has something
	snap     atomic.Pointer[camSnap]
	pending  atomic.Int32
	wg       sync.WaitGroup

	rastered atomic.Uint64
	rasterNs atomic.Int64
	stats    Stats
}

// NewCompositor returns a compositor that draws s with r onto h. Most apps get one from view.New.
func NewCompositor(s *Scene, r Renderer, h Host, o Options) *Compositor {
	o.fill()
	c := &Compositor{scene: s, r: r, host: h, o: o, dpr: 1, vw: 800, vh: 600, cam: Camera{Zoom: 1}, lastZoom: -1, scr: make([]scratch, 1)}
	c.snap.Store(&camSnap{})
	c.camDirty.Store(true)
	return c
}

// Options returns the effective options of the compositor: those it was created with, defaults
// filled in.
func (c *Compositor) Options() Options { return c.o }

// Close stops the workers and removes every layer.
func (c *Compositor) Close() {
	// pool-then-compositor, the order take and enqueue use: no worker hands a job out of a
	// closing compositor, and Close's Wait below covers the jobs already handed out.
	tilePool.mu.Lock()
	c.mu.Lock()
	c.closed = true
	c.dropQueued()
	c.mu.Unlock()
	delete(tilePool.comps, c)
	tilePool.mu.Unlock()
	c.wg.Wait()
	c.host.Begin()
	for _, set := range []*tileSet{c.cur, c.prev} {
		c.dropSet(set)
	}
	c.cur, c.prev = nil, nil
	for _, t := range c.direct.rot {
		if t != nil {
			t.Close()
		}
	}
	for _, set := range c.direct.sets {
		if set != nil {
			set.Close()
		}
	}
	c.direct.rot = nil
	c.direct.sets = nil
	c.direct.setPx = nil
	c.direct.bitems = nil
	c.direct.rotN = 0
	c.direct.rotSet = false
	if c.direct.layer != nil {
		c.direct.layer.Remove()
		c.direct.layer = nil
	}
	c.direct.drawn = false
	for _, l := range c.lifts {
		l.layer.Remove()
	}
	c.lifts = nil
	for _, sp := range c.sprites {
		sp.drop()
	}
	c.sprites = nil
	if c.hudState.layer != nil {
		c.hudState.layer.Remove()
		c.hudState.layer = nil
	}
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

// Workers limits how many of this compositor's tiles are drawn at once and caps the per-frame
// sprite redraw batch; the rasterizers come from a pool shared by every compositor.
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
	return c.ticking.Load() || c.camDirty.Load() || c.spriteDirty.Load() || c.readyN.Load() || c.liftN.Load() || c.busy || c.scene.Pending() || c.hudPending()
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
	c.stats.TilesApplied, c.stats.SpriteDraws, c.stats.HUDDraws = 0, 0, 0
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
		if !c.o.Direct {
			c.layoutDrty = true
		}
	}
	if !c.o.Direct {
		c.handleZoom(fi)
	}
	if len(damage) > 0 {
		c.invalidate(damage)
	}
	c.applyLifts()
	if c.o.Direct {
		c.stepDirect(camMoved)
	} else {
		c.ensureTiles(camMoved || len(damage) > 0)
		c.drain()
	}
	c.stepSprites(fi, camMoved)
	c.stepHUD()
	if c.layoutDrty {
		c.layout()
		c.layoutDrty = false
	}
	c.host.Commit()
	if c.o.Direct {
		c.busy = len(c.lifts) > 0 && c.anyLanding() || c.animating()
	} else {
		c.busy = c.zooming || len(c.lifts) > 0 && c.anyLanding() || c.animating()
	}
	c.stats.FrameTime = time.Since(t0)
}

// stepDirect brings the single viewport image up to date. The redraw is whole: one Collect of the
// view rect and one draw at exactly zoom*dpr. Lifts and landings are synchronous here, so the
// base image and the sprite layer always change in one host transaction: no hold-back queue of
// the tile mode is needed. After a redraw every hold is released, so the frame is settled.
func (c *Compositor) stepDirect(camMoved bool) {
	view := c.viewRect()
	need := camMoved || !c.direct.drawn
	if !need {
		for _, d := range c.direct.damage {
			if d.Intersects(view) {
				need = true
				break
			}
		}
	}
	for _, l := range c.lifts {
		if (l.hold || l.landing) && l.hrect.Intersects(view) {
			need = true
		}
	}
	c.settled = true
	if need {
		c.drawDirect(view)
		c.direct.damage = c.direct.damage[:0]
	}
	for _, l := range c.lifts {
		if l.hold && (need || !l.hrect.Intersects(view)) {
			l.hold = false
			l.layer.SetHidden(false)
		}
	}
}

// sharedSnapshotter is implemented by Targets whose snapshots share the target memory instead
// of copying it; the Direct mode rotates such targets so one is not drawn into again until the
// display has dropped the previous frame's image.
type sharedSnapshotter interface {
	SnapshotSharedImage(wpt, hpt float64) Image
}

// directTarget returns the viewport target to draw this frame into: a fresh position of a
// three-target rotation when the snapshot shares the target memory, the reused single target
// otherwise. The kind of target is decided once, on the first frame.
func (c *Compositor) directTarget(wpx, hpx int) Target {
	if !c.direct.rotSet {
		c.direct.rotSet = true
		t := c.r.NewTarget(wpx, hpx)
		if _, ok := t.(sharedSnapshotter); ok {
			c.direct.rot = []Target{t, nil, nil}
			c.direct.rotN = 1
		} else {
			c.scr[0].t = t
		}
		return t
	}
	if c.direct.rot == nil {
		t := c.scr[0].t
		if t != nil {
			if w, h := t.Size(); w != wpx || h != hpx {
				t.Close()
				t = nil
			}
		}
		if t == nil {
			t = c.r.NewTarget(wpx, hpx)
		}
		c.scr[0].t = t
		return t
	}
	t := c.direct.rot[c.direct.rotN]
	if t != nil {
		if w, h := t.Size(); w != wpx || h != hpx {
			t.Close()
			t = nil
		}
	}
	if t == nil {
		t = c.r.NewTarget(wpx, hpx)
	}
	c.direct.rot[c.direct.rotN] = t
	c.direct.rotN = (c.direct.rotN + 1) % len(c.direct.rot)
	return t
}

// directStripSet returns the rotated strip set for the parallel rasterization, or false when
// the renderer has no strip support. The rotation matches directTarget: a set is drawn into
// again only two frames later.
func (c *Compositor) directStripSet(wpx, hpx int) (StripSet, bool) {
	sr, ok := c.r.(StripRenderer)
	if !ok {
		return nil, false
	}
	n := min(max(c.o.Workers, 1), 8, max(1, hpx/32))
	if !c.direct.rotSet {
		c.direct.rotSet = true
		set := sr.NewStripSet(n, wpx, hpx)
		c.direct.sets = []StripSet{set, nil, nil}
		c.direct.setPx = [][2]int{{wpx, hpx}, {}, {}}
		c.direct.rotN = 1
		return set, true
	}
	i := c.direct.rotN
	set := c.direct.sets[i]
	if set != nil && c.direct.setPx[i] != [2]int{wpx, hpx} {
		set.Close()
		set = nil
	}
	if set == nil {
		set = sr.NewStripSet(n, wpx, hpx)
		c.direct.setPx[i] = [2]int{wpx, hpx}
	}
	c.direct.sets[i] = set
	c.direct.rotN = (c.direct.rotN + 1) % len(c.direct.sets)
	return set, true
}

// drawStrips rasterizes the viewport in the set's parallel strips: one Collect per strip on
// this goroutine, then the strip draws on worker goroutines into consecutive device-pixel
// rows of one bitmap. A scene point maps to the same device pixel as in the single-target
// draw (the strip origin shifts the CTM by the row offset), so the strips meet exactly and a
// node crossing a seam is drawn by both, each half clipped to its rows by the bitmap bounds.
func (c *Compositor) drawStrips(set StripSet, view Rect, scale float64, wpx, hpx int) Image {
	n := set.Strips()
	for len(c.direct.bitems) < n {
		c.direct.bitems = append(c.direct.bitems, Items{})
	}
	bitems := c.direct.bitems[:n]
	eps := 1e-3 / scale
	type band struct {
		origin Point
		rect   Rect
	}
	bands := make([]band, n)
	for i := 0; i < n; i++ {
		y0, y1 := i*hpx/n, (i+1)*hpx/n
		// every strip uses the view origin: the strip targets translate by the strip's
		// absolute bottom row (BeginStrip), which keeps every device coordinate exactly the
		// single-target value, so the strips meet byte-exactly. The collect rect covers the
		// strip's rows; what sticks out past them is clipped by the strip bitmap's bounds.
		bands[i] = band{view.Min, R(view.Min.X-eps, view.Min.Y+float64(y0)/scale-eps, view.W()+2*eps, float64(y1-y0)/scale+2*eps)}
		c.scene.Collect(bands[i].rect, &bitems[i])
	}
	var wg sync.WaitGroup
	draw := func(i int) {
		t := set.Target(i)
		if ss, ok := t.(ScreenScaler); ok {
			ss.SetScreenScale(c.dpr)
		}
		t.Begin(scale, bands[i].origin, Color{})
		t.Draw(bitems[i].List)
	}
	for i := 1; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			draw(i)
		}()
	}
	draw(0)
	wg.Wait()
	return set.Image(float64(wpx)/c.dpr, float64(hpx)/c.dpr)
}

// drawDirect redraws the viewport image at exactly zoom*dpr and puts it on the layer.
func (c *Compositor) drawDirect(view Rect) {
	scale := c.cam.Zoom * c.dpr
	wpx, hpx := int(math.Ceil(c.vw*c.dpr)), int(math.Ceil(c.vh*c.dpr))
	if wpx < 1 || hpx < 1 {
		return
	}
	c.scene.Collect(view, &c.direct.items)
	if len(c.direct.items.List) == 0 {
		if c.direct.layer != nil {
			c.direct.layer.SetHidden(true)
		}
		c.direct.drawn = true
		return
	}
	t0 := time.Now()
	var img Image
	if set, ok := c.directStripSet(wpx, hpx); ok && set != nil {
		img = c.drawStrips(set, view, scale, wpx, hpx)
	} else {
		t := c.directTarget(wpx, hpx)
		if ss, ok := t.(ScreenScaler); ok {
			ss.SetScreenScale(c.dpr)
		}
		t.Begin(scale, view.Min, Color{})
		t.Draw(c.direct.items.List)
		if sh, ok := t.(sharedSnapshotter); ok {
			img = sh.SnapshotSharedImage(float64(wpx)/c.dpr, float64(hpx)/c.dpr)
		} else {
			img = t.Snapshot(float64(wpx)/c.dpr, float64(hpx)/c.dpr)
		}
	}
	if c.direct.layer == nil {
		c.direct.layer = c.host.NewLayer(GroupTiles)
	}
	c.direct.layer.SetImage(img)
	Release(img)
	c.direct.layer.Set(Rect{Point{0, 0}, Point{float64(wpx) / c.dpr, float64(hpx) / c.dpr}}, Point{}, Identity())
	c.direct.layer.SetHidden(false)
	c.direct.drawn = true
	c.stats.TilesApplied++
	c.rastered.Add(1)
	c.rasterNs.Add(int64(time.Since(t0)))
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

// invalidate marks what the next Frame must redraw: in the tile mode the tiles under the damaged
// scene areas, in the Direct mode the damage list itself.
func (c *Compositor) invalidate(damage []Rect) {
	if c.o.Direct {
		c.direct.damage = append(c.direct.damage, damage...)
		return
	}
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
		c.enqueue(add)
	}
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

// blankTile is the result of a tile with nothing in it: it gets no image and no layer. A clear
// image of a whole tile costs as much memory as a drawn one.
type blankTile struct{}

func (blankTile) Size() (float64, float64) { return 0, 0 }

// show puts a finished image on the layer of t and releases it.
func (c *Compositor) show(t *tile, img Image) {
	if _, ok := img.(blankTile); ok {
		if t.layer != nil {
			t.layer.Remove()
			t.layer = nil
		}
		t.has = true
		c.layoutDrty = true
		return
	}
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
		c.enqueue(add)
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
		// hold the sprite hidden until the static base is redrawn without the node, so the node
		// is not in the base and in the sprite in one commit; the base is the tiles or, in the
		// Direct mode, the viewport image redrawn from direct.damage
		if l.has && (c.cur != nil || c.o.Direct) {
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
