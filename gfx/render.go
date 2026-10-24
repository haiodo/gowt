package gfx

import (
	"errors"
	"math"
)

// Clip is one clip of an Item: the shape in the coordinates of the transform M, which maps them
// to the target's scene space.
type Clip struct {
	M      Affine
	Kind   ClipKind
	Rect   Rect
	Radius float64
	Path   *Path
}

// Item is one picture to draw: through M, under the clips, composited at Alpha.
type Item struct {
	Pic   *Picture
	M     Affine
	Alpha float64
	Clips []Clip
}

// Target is an offscreen bitmap a Renderer draws into. One Target is used by one goroutine at a time.
type Target interface {
	// Begin clears the target to bg and maps the scene point origin to the top left pixel
	// at scale device pixels per scene unit.
	Begin(scale float64, origin Point, bg Color)
	// Draw paints the items in order.
	Draw(items []Item)
	Size() (wpx, hpx int)
	// Pixels is premultiplied BGRA, rows top to bottom; valid until Close.
	Pixels() []byte
	// Snapshot returns the current pixels as an Image of the given size in points, for a Host.
	Snapshot(wpt, hpt float64) Image
	Close()
}

// Renderer is a drawing backend: CoreGraphics on macOS. It is safe for concurrent use, its Targets
// are not shared.
type Renderer interface {
	TextMeasurer
	NewTarget(wpx, hpx int) Target
}

// Release frees an Image a backend made, if it has anything to free.
func Release(img Image) {
	if r, ok := img.(interface{ Release() }); ok {
		r.Release()
	}
}

// noArea is true for the zero Rect and for EmptyRect: callers pass either to mean "the default area".
func noArea(r Rect) bool { return r.Empty() || r == Rect{} }

// RenderSubtree draws the subtree of root, with the clips of its ancestors, into a Raster.
// area is the scene rectangle to draw, or an empty Rect for the subtree bounds; scale is device
// pixels per scene unit. It needs no window.
func RenderSubtree(r Renderer, s *Scene, root *Node, area Rect, scale float64, bg Color) (*Raster, error) {
	s.Commit()
	if noArea(area) {
		s.Read(func() { area = root.SubtreeBounds() })
	}
	if noArea(area) {
		return nil, errors.New("gfx: nothing to render")
	}
	w, h := int(math.Ceil(area.W()*scale)), int(math.Ceil(area.H()*scale))
	if w <= 0 || h <= 0 || w > 1<<15 || h > 1<<15 {
		return nil, errors.New("gfx: render size out of range")
	}
	t := r.NewTarget(w, h)
	defer t.Close()
	var b Items
	s.CollectSubtree(root, area, &b)
	t.Begin(scale, area.Min, bg)
	t.Draw(b.List)
	return FromBGRA(t.Pixels(), w, h), nil
}

// RenderPicture draws one picture at the given scale into a Raster of its bounds.
func RenderPicture(r Renderer, p *Picture, scale float64, bg Color) (*Raster, error) {
	if p.Bounds.Empty() {
		return nil, errors.New("gfx: nothing to render")
	}
	w, h := int(math.Ceil(p.Bounds.W()*scale)), int(math.Ceil(p.Bounds.H()*scale))
	if w <= 0 || h <= 0 || w > 1<<15 || h > 1<<15 {
		return nil, errors.New("gfx: render size out of range")
	}
	t := r.NewTarget(w, h)
	defer t.Close()
	t.Begin(scale, p.Bounds.Min, bg)
	t.Draw([]Item{{Pic: p, M: Identity(), Alpha: 1}})
	return FromBGRA(t.Pixels(), w, h), nil
}
