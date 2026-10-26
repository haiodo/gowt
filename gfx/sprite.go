package gfx

import (
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Space says how a Sprite's position and picture relate to the camera.
type Space uint8

const (
	// SpaceScene: the picture is in scene units and zooms with the scene; Pos is a scene point.
	SpaceScene Space = iota
	// SpacePinned: Pos is a scene point, the picture is in points and keeps its size: handles, labels.
	SpacePinned
	// SpaceScreen: Pos is a view point, the picture is in points.
	SpaceScreen
)

type spriteState struct {
	pos      Point
	opacity  float64
	rotation float64 // radians, about Pos
	scale    float64 // about Pos
}

// Sprite is a picture on the compositor's overlay, moved by the compositor alone: it is drawn once
// and then only placed, so a cursor flying over a busy scene costs no redrawing. The picture's
// origin (0, 0) lies at Pos. Its methods are safe from any goroutine; they take effect on the next
// Frame, and animations count their time from it.
type Sprite struct {
	c     *Compositor
	space Space

	mu       sync.Mutex
	st       spriteState
	pic      *Picture
	picDirty bool
	anims    []*anim
	removed  bool

	// the goroutine that runs Frame
	layer  Layer
	rs     float64 // device pixels per picture unit the image was drawn at
	wpx    int
	hpx    int
	bounds Rect
}

// NewSprite puts pic on the overlay at pos, above the sprites made before it.
func (c *Compositor) NewSprite(pic *Picture, space Space, pos Point) *Sprite {
	s := &Sprite{c: c, space: space, pic: pic, picDirty: true, st: spriteState{pos: pos, opacity: 1, scale: 1}}
	c.smu.Lock()
	c.pendingSprites = append(c.pendingSprites, s)
	c.smu.Unlock()
	c.spriteDirty.Store(true)
	return s
}

// SetPicture replaces what the sprite shows; it is drawn again once.
func (s *Sprite) SetPicture(p *Picture) {
	s.mu.Lock()
	s.pic, s.picDirty = p, true
	s.mu.Unlock()
	s.c.spriteDirty.Store(true)
}

// SetPos puts the sprite at p at once and stops any move or flight under way.
func (s *Sprite) SetPos(p Point) {
	s.mu.Lock()
	s.st.pos = p
	s.cancel(propPos, propAlong)
	s.mu.Unlock()
	s.c.spriteDirty.Store(true)
}

// SetOpacity sets the opacity at once and stops a fade under way.
func (s *Sprite) SetOpacity(a float64) {
	s.mu.Lock()
	s.st.opacity = a
	s.cancel(propOpacity)
	s.mu.Unlock()
	s.c.spriteDirty.Store(true)
}

// SetRotation sets the angle in radians at once and stops a rotation under way.
func (s *Sprite) SetRotation(rad float64) {
	s.mu.Lock()
	s.st.rotation = rad
	s.cancel(propRotation)
	s.mu.Unlock()
	s.c.spriteDirty.Store(true)
}

// SetScale sets the scale at once and stops a scaling under way.
func (s *Sprite) SetScale(f float64) {
	s.mu.Lock()
	s.st.scale = f
	s.cancel(propScale)
	s.mu.Unlock()
	s.c.spriteDirty.Store(true)
}

// Pos is the position as of the last Frame.
func (s *Sprite) Pos() Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st.pos
}

// Remove takes the sprite off the overlay.
func (s *Sprite) Remove() {
	s.mu.Lock()
	s.removed = true
	s.mu.Unlock()
	s.c.spriteDirty.Store(true)
}

func (s *Sprite) cancel(props ...animProp) {
	w := s.anims[:0]
	for _, a := range s.anims {
		keep := true
		for _, p := range props {
			keep = keep && a.prop != p
		}
		if keep {
			w = append(w, a)
		}
	}
	s.anims = w
}

func (s *Sprite) start(a *anim, props ...animProp) {
	s.mu.Lock()
	s.cancel(props...)
	s.anims = append(s.anims, a)
	s.mu.Unlock()
	s.c.spriteDirty.Store(true)
}

// MoveTo glides to p over d along ease; done, if set, runs on the Frame goroutine when it arrives.
func (s *Sprite) MoveTo(p Point, d time.Duration, ease Easing, done func()) {
	s.start(&anim{prop: propPos, to: p, dur: d, ease: orLinear(ease), done: done}, propPos, propAlong)
}

// FadeTo animates the opacity to a over d. done, when not nil, runs on the frame that finishes it.
func (s *Sprite) FadeTo(a float64, d time.Duration, ease Easing, done func()) {
	s.start(&anim{prop: propOpacity, to: Point{X: a}, dur: d, ease: orLinear(ease), done: done}, propOpacity)
}

// RotateTo animates the angle to rad over d.
func (s *Sprite) RotateTo(rad float64, d time.Duration, ease Easing, done func()) {
	s.start(&anim{prop: propRotation, to: Point{X: rad}, dur: d, ease: orLinear(ease), done: done}, propRotation)
}

// ScaleTo animates the scale to f over d.
func (s *Sprite) ScaleTo(f float64, d time.Duration, ease Easing, done func()) {
	s.start(&anim{prop: propScale, to: Point{X: f}, dur: d, ease: orLinear(ease), done: done}, propScale)
}

// FlyAlong moves the sprite along path (in the coordinates of Pos) over d. Easing shapes the
// progress along the arc length. With orient, the sprite is rotated to the path direction.
func (s *Sprite) FlyAlong(path *Path, d time.Duration, ease Easing, orient bool, done func()) {
	m := path.Measure(0.05)
	s.start(&anim{prop: propAlong, path: m, dur: d, ease: orLinear(ease), orient: orient, done: done}, propPos, propAlong, propRotation)
}

func orLinear(e Easing) Easing {
	if e == nil {
		return Linear
	}
	return e
}

func (s *Sprite) animating() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.anims) > 0
}

func (s *Sprite) drop() {
	if s.layer != nil {
		s.layer.Remove()
		s.layer = nil
	}
}

// stepSprites advances animations to the frame time and places sprites and lifted nodes.
func (c *Compositor) stepSprites(fi FrameInfo, camMoved bool) {
	c.smu.Lock()
	c.sprites = append(c.sprites, c.pendingSprites...)
	c.pendingSprites = nil
	c.smu.Unlock()

	var done []func()
	live := c.sprites[:0]
	for _, s := range c.sprites {
		s.mu.Lock()
		w := s.anims[:0]
		for _, a := range s.anims {
			if a.step(fi.Time, &s.st) {
				if a.done != nil {
					done = append(done, a.done)
				}
			} else {
				w = append(w, a)
			}
		}
		s.anims = w
		st, pic, dirty, removed := s.st, s.pic, s.picDirty, s.removed
		s.picDirty = false
		s.mu.Unlock()
		if removed {
			s.drop()
			continue
		}
		live = append(live, s)
		c.placeSprite(s, st, pic, dirty)
	}
	c.sprites = live
	c.stepLifts()
	for _, f := range done {
		f()
	}
}

func (c *Compositor) contentScale(sp Space) float64 {
	if sp == SpaceScene {
		return c.cam.Zoom
	}
	return 1
}

func (c *Compositor) placeSprite(s *Sprite, st spriteState, pic *Picture, dirty bool) {
	if pic == nil || pic.Bounds.Empty() {
		if s.layer != nil {
			s.layer.SetHidden(true)
		}
		return
	}
	cs := c.contentScale(s.space)
	want := cs * c.dpr
	if s.layer == nil || dirty || (s.space == SpaceScene && !c.zooming && math.Abs(s.rs-want) > 1e-9*want) {
		b := pic.Bounds
		s.bounds = b
		// rs is device pixels per picture unit; a scene picture's units are scaled by the zoom on screen
		s.rs = want
		s.wpx, s.hpx = int(math.Ceil(b.W()*s.rs))+1, int(math.Ceil(b.H()*s.rs))+1
		img := c.drawArea(b.Min, s.rs, s.wpx, s.hpx, []Item{{Pic: pic, M: Identity(), Alpha: 1}})
		if s.layer == nil {
			s.layer = c.host.NewLayer(GroupOverlay)
		}
		s.layer.SetImage(img)
		Release(img)
		c.stats.SpriteDraws++
	}
	a := st.pos
	if s.space != SpaceScreen {
		a = c.ViewAt(a)
	}
	b := s.bounds
	min := Point{a.X + cs*b.Min.X, a.Y + cs*b.Min.Y}
	k := cs / s.rs
	size := Point{float64(s.wpx) * k, float64(s.hpx) * k}
	if st.rotation == 0 && st.scale == 1 {
		min = Point{c.px(min.X), c.px(min.Y)}
	}
	m := RotateBy(st.rotation).Mul(ScaleBy(st.scale, st.scale))
	s.layer.Set(Rect{min, Point{min.X + size.X, min.Y + size.Y}}, Point{a.X - min.X, a.Y - min.Y}, m)
	s.layer.SetOpacity(st.opacity)
	s.layer.SetHidden(false)
}

// drawArea draws items into a fresh image of wpx x hpx pixels showing the scene area from origin at
// the given scale.
func (c *Compositor) drawArea(origin Point, scale float64, wpx, hpx int, items []Item) Image {
	return c.drawOn(&c.scr[0], origin, scale, wpx, hpx, items)
}

func (c *Compositor) drawOn(s *scratch, origin Point, scale float64, wpx, hpx int, items []Item) Image {
	if s.t != nil {
		if w, h := s.t.Size(); w != wpx || h != hpx {
			s.t.Close()
			s.t = nil
		}
	}
	if s.t == nil {
		s.t = c.r.NewTarget(wpx, hpx)
	}
	s.t.Begin(scale, origin, Color{})
	s.t.Draw(items)
	return s.t.Snapshot(float64(wpx)/c.dpr, float64(hpx)/c.dpr)
}

// drawJobs draws the images of jobs on up to Workers goroutines and returns when all are done:
// the sprites follow the node in the same frame, with no lag to hide.
func (c *Compositor) drawJobs(jobs []liftJob) {
	if len(jobs) == 0 {
		return
	}
	// waking a goroutine costs about as much as drawing a few connectors
	n := min((len(jobs)+3)/4, c.o.Workers)
	for len(c.scr) < max(n, 1) {
		c.scr = append(c.scr, scratch{})
	}
	var next atomic.Int32
	left := atomic.Int32{}
	left.Store(int32(len(jobs)))
	done := make(chan struct{})
	run := func(s *scratch) {
		for {
			i := int(next.Add(1)) - 1
			if i >= len(jobs) {
				return
			}
			j := &jobs[i]
			c.scene.CollectSubtree(j.l.node, j.area, &s.items)
			j.img = c.drawOn(s, j.area.Min, j.want, j.wpx, j.hpx, s.items.List)
			if left.Add(-1) == 0 {
				close(done)
			}
		}
	}
	for i := 1; i < n; i++ {
		go run(&c.scr[i])
	}
	run(&c.scr[0])
	// wait for the jobs, not for the goroutines: one that woke up late finds nothing left to take
	<-done
}

func (c *Compositor) stepLifts() {
	live := c.lifts[:0]
	for _, l := range c.lifts {
		var attached bool
		c.scene.Read(func() { attached = l.node.scene == c.scene })
		if !attached || (l.landing && c.settled) {
			l.dead = true
			l.layer.Remove()
			continue
		}
		live = append(live, l)
	}
	c.lifts = live
	c.refreshLifts(live)
}

// dropLift removes l from the active layer.
func (c *Compositor) dropLift(l *lift) {
	l.layer.Remove()
	c.lifts = slices.DeleteFunc(c.lifts, func(x *lift) bool { return x == l })
}

// spriteArea is the scene area a sprite of a node with bounds sb draws at this scale: the view, or
// more of the node beyond it when that fits maxSpritePx. wpx is -1 when the image is too large.
func (c *Compositor) spriteArea(sb Rect, want float64) (area Rect, wpx, hpx int) {
	view := c.viewRect().Expand(64 / c.cam.Zoom)
	wide := c.viewRect().Expand(max(c.vw, c.vh) / c.cam.Zoom / 4)
	area = sb.Intersect(view)
	if w := sb.Intersect(wide); sb != area && w.W()*w.H()*want*want < maxSpritePx {
		// draw beyond the view, so that a drag does not redraw the sprite at every step
		area = w
	}
	if area.Empty() {
		return area, 0, 0
	}
	wpx, hpx = int(math.Ceil(area.W()*want))+1, int(math.Ceil(area.H()*want))+1
	if wpx > 1<<14 || hpx > 1<<14 {
		wpx = -1
	}
	return area, wpx, hpx
}

// asyncSpritePx is the image size above which a sprite is drawn on a worker: at the speed of
// the CoreGraphics raster that is about what a frame can afford.
const asyncSpritePx = 1 << 18

// bigSprite reports whether the first image of l is too big to draw within a frame.
func (c *Compositor) bigSprite(l *lift) bool {
	var sb Rect
	c.scene.Read(func() { sb = l.node.SubtreeBounds() })
	_, wpx, hpx := c.spriteArea(sb, c.cam.Zoom*c.dpr)
	return wpx > 0 && wpx*hpx > asyncSpritePx
}

// drawAsync draws the image of l on a goroutine of its own and queues it for applyLifts. The area
// is chosen under the scene lock together with the transform, so that image and transform match
// however the node moves meanwhile.
func (c *Compositor) drawAsync(l *lift) {
	l.drawing = true
	want := c.cam.Zoom * c.dpr
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		j := liftJob{l: l, want: want}
		var items Items
		c.scene.Read(func() {
			sb := l.node.SubtreeBounds()
			j.w, j.rev, j.deep = l.node.world, l.node.rev, l.node.deep
			j.area, j.wpx, j.hpx = c.spriteArea(sb, want)
			if j.wpx > 0 {
				j.clipped = j.area != sb
				collectSubtree(l.node, j.area, &items)
			}
		})
		if j.wpx > 0 {
			t := c.r.NewTarget(j.wpx, j.hpx)
			t.Begin(want, j.area.Min, Color{})
			t.Draw(items.List)
			j.img = t.Snapshot(float64(j.wpx)/c.dpr, float64(j.hpx)/c.dpr)
			t.Close()
		}
		c.mu.Lock()
		c.liftReady = append(c.liftReady, j)
		c.liftN.Store(true)
		c.mu.Unlock()
	}()
}

// applyLifts shows the images the workers have finished. A first image also lifts its node, in the
// same host transaction: the node leaves the tiles once and the sprite stays hidden until the tiles
// under it are redrawn (hold), like for a lift drawn at once.
func (c *Compositor) applyLifts() {
	if !c.liftN.Load() {
		return
	}
	c.mu.Lock()
	done := c.liftReady
	c.liftReady = nil
	c.liftN.Store(false)
	c.mu.Unlock()
	for i := range done {
		j := &done[i]
		l := j.l
		l.drawing = false
		if l.dead || j.img == nil {
			Release(j.img)
			if l.pending && !l.dead {
				// nothing to draw in the view: show the node lifted without an image
				c.liftNow(l)
			}
			continue
		}
		l.layer.SetImage(j.img)
		Release(j.img)
		l.rb, l.clipped, l.w0, l.rev, l.deep, l.scale, l.has = j.area, j.clipped, j.w, j.rev, j.deep, j.want, true
		l.wpx, l.hpx = j.wpx, j.hpx
		c.stats.SpriteDraws++
		if l.pending {
			c.liftNow(l)
			continue
		}
		var w Affine
		c.scene.Read(func() { w = l.node.world })
		c.placeLift(l, w)
	}
}

// liftNow takes the node of a pending lift out of the tiles.
func (c *Compositor) liftNow(l *lift) {
	l.pending = false
	l.hrect = c.scene.liftNode(l.node)
	c.invalidate([]Rect{l.hrect})
	if l.has && c.cur != nil {
		l.hold = true
		l.layer.SetHidden(true)
	} else if l.has {
		var w Affine
		c.scene.Read(func() { w = l.node.world })
		c.placeLift(l, w)
	}
	c.busy = true
}

// refreshLifts draws again the sprites whose node changed, all at once on the worker goroutines,
// and places every sprite. Images too big for a frame are drawn in the background, and the old
// image stays where it is until the new one is ready.
func (c *Compositor) refreshLifts(lifts []*lift) {
	view := c.viewRect().Expand(64 / c.cam.Zoom)
	want := c.cam.Zoom * c.dpr
	jobs := c.jobs[:0]
	for _, l := range lifts {
		if l.pending {
			continue
		}
		if l.hold {
			if !c.settled {
				continue
			}
			l.hold = false
		}
		var w Affine
		var rev, deep uint32
		var sb Rect
		c.scene.Read(func() {
			w, sb = l.node.world, l.node.SubtreeBounds()
			rev, deep = l.node.Version()
		})
		if sb.Empty() {
			l.layer.SetHidden(true)
			continue
		}
		if l.drawing || (l.landing && l.has) {
			if l.has {
				c.placeLift(l, w)
			}
			continue
		}
		moved := w != l.w0
		// a sprite cut to the view stays good while the view is still inside what it drew
		stale := l.clipped && moved && !l.rb.Translate(Point{w[4] - l.w0[4], w[5] - l.w0[5]}).covers(view)
		if l.has && rev == l.rev && deep == l.deep && w.SameLinear(l.w0) && !stale &&
			(c.zooming || math.Abs(l.scale-want) <= 1e-9*want) {
			c.placeLift(l, w)
			continue
		}
		area, wpx, hpx := c.spriteArea(sb, want)
		if area.Empty() {
			l.layer.SetHidden(true)
			l.has = false
			continue
		}
		if wpx < 0 {
			continue
		}
		if wpx*hpx > asyncSpritePx {
			c.drawAsync(l)
			if l.has {
				c.placeLift(l, w)
			}
			continue
		}
		jobs = append(jobs, liftJob{l: l, area: area, wpx: wpx, hpx: hpx, want: want, w: w, rev: rev, deep: deep, clipped: area != sb})
	}
	c.drawJobs(jobs)
	for i := range jobs {
		j := &jobs[i]
		l := j.l
		l.layer.SetImage(j.img)
		Release(j.img)
		l.rb, l.clipped, l.w0, l.rev, l.deep, l.scale, l.has = j.area, j.clipped, j.w, j.rev, j.deep, j.want, true
		l.wpx, l.hpx = j.wpx, j.hpx
		c.stats.SpriteDraws++
		c.placeLift(l, j.w)
		jobs[i] = liftJob{}
	}
	c.jobs = jobs[:0]
}

// placeLift puts the sprite of l where its node, at world transform w, is.
func (c *Compositor) placeLift(l *lift, w Affine) {
	d := Point{w[4] - l.w0[4], w[5] - l.w0[5]}
	min := c.ViewAt(Point{l.rb.Min.X + d.X, l.rb.Min.Y + d.Y})
	// the image has whole pixels at the scale it was drawn at; its size follows the zoom
	k := c.cam.Zoom / l.scale * c.dpr
	min = Point{c.px(min.X), c.px(min.Y)}
	l.layer.Set(Rect{min, Point{min.X + float64(l.wpx)*k/c.dpr, min.Y + float64(l.hpx)*k/c.dpr}}, Point{}, Identity())
	l.layer.SetHidden(false)
}

// maxSpritePx is the most device pixels a sprite is widened to beyond the view.
const maxSpritePx = 16 << 20

func (r Rect) covers(o Rect) bool {
	return r.Min.X <= o.Min.X && r.Min.Y <= o.Min.Y && r.Max.X >= o.Max.X && r.Max.Y >= o.Max.Y
}
