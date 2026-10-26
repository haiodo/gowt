package gfx

import "math"

type hudKey struct{ w, h, dpr float64 }

// hudState is what the Frame goroutine keeps for the HUD.
type hudState struct {
	layer    Layer
	drawn    hudKey // the viewport the image was drawn for
	notified hudKey // the viewport OnViewport last heard
	items    Items
}

// HUD returns the heads-up scene, made on the first call: a scene in view points (y down, the
// origin at the top left of the view) that is drawn over everything, tiles, lifted nodes and
// sprites, and does not move or zoom with the camera. Use it for indicators at the canvas edge,
// rulers, a minimap and text overlays. It is an ordinary Scene: nodes, pictures, transforms, clips,
// opacity and Update from any goroutine.
//
// The compositor draws the scene into one image that covers the bounds of its content inside the
// view, and draws it again only when the scene changes or the view size or backing scale does,
// so a HUD costs nothing in a frame that changes nothing in it. Do not give a node over the whole
// view a picture to catch the pointer: set the node's shape with HitNone or a real shape, since
// HitBounds takes the pointer wherever the node's bounds are, see HUDHit.
func (c *Compositor) HUD() *Scene {
	if s := c.hud.Load(); s != nil {
		return s
	}
	c.hud.CompareAndSwap(nil, NewScene())
	return c.hud.Load()
}

// HUDHit returns the topmost HUD node under the view point p, or nil. tol is the stroke tolerance
// and bounds slack in points. It sees the HUD as of the last Frame, and it comes before the scene
// in view.Pointer: when it returns a node the pointer is over the HUD.
func (c *Compositor) HUDHit(p Point, tol float64) *Node {
	if s := c.hud.Load(); s != nil {
		return s.HitTest(p, tol)
	}
	return nil
}

// Viewport returns the view size in points and the device pixels per point, as of the last
// SetViewport.
func (c *Compositor) Viewport() (w, h, dpr float64) { return c.vw, c.vh, c.dpr }

// OnViewport sets f to run on the Frame goroutine, at the start of the HUD step, with the view size
// in points and the backing scale: once at the first Frame after it is set and again whenever either
// changes. Place the HUD nodes there (an indicator on the right edge moves with the width); the
// changes it makes are drawn in the same frame.
func (c *Compositor) OnViewport(f func(w, h, dpr float64)) {
	c.onViewport = f
	c.hudState.notified = hudKey{}
	c.spriteDirty.Store(true)
}

func (c *Compositor) hudPending() bool {
	h := c.hud.Load()
	return h != nil && h.Pending()
}

// stepHUD brings the HUD layer up to date.
func (c *Compositor) stepHUD() {
	st := &c.hudState
	key := hudKey{c.vw, c.vh, c.dpr}
	if c.onViewport != nil && key != st.notified {
		st.notified = key
		c.onViewport(c.vw, c.vh, c.dpr)
	}
	h := c.hud.Load()
	if h == nil {
		return
	}
	damage := h.Commit()
	if len(damage) == 0 && key == st.drawn && st.layer != nil {
		return
	}
	st.drawn = key
	var b Rect
	h.Read(func() { b = h.Root().SubtreeBounds() })
	b = b.Intersect(R(0, 0, c.vw, c.vh))
	if noArea(b) || b.W() <= 0 || b.H() <= 0 {
		if st.layer != nil {
			st.layer.SetHidden(true)
		}
		return
	}
	d := c.dpr
	x0, y0 := math.Floor(b.Min.X*d)/d, math.Floor(b.Min.Y*d)/d
	wpx, hpx := int(math.Ceil((b.Max.X-x0)*d)), int(math.Ceil((b.Max.Y-y0)*d))
	area := R(x0, y0, float64(wpx)/d, float64(hpx)/d)
	h.Collect(area, &st.items)
	img := c.drawArea(area.Min, d, wpx, hpx, st.items.List)
	if st.layer == nil {
		st.layer = c.host.NewLayer(GroupHUD)
	}
	st.layer.SetImage(img)
	Release(img)
	st.layer.Set(area, Point{}, Identity())
	st.layer.SetHidden(false)
	c.stats.HUDDraws++
}
