package gfx

import (
	"errors"
	"fmt"
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

// ScreenScaler is implemented by the Targets of the backends that draw Shadow.Screen: it sets how many
// device pixels one screen point of such a shadow is. The Compositor sets the display's backing scale
// before every tile; a Target that was not told uses 1.
type ScreenScaler interface{ SetScreenScale(devicePerPoint float64) }

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

// maxRender is the largest side of a rendered image in device pixels.
const maxRender = 1 << 15

// pixelSize is the device size of an area of scene units at scale, or an error that names the cause.
func pixelSize(area Rect, scale float64) (w, h int, err error) {
	fw, fh := math.Ceil(area.W()*scale), math.Ceil(area.H()*scale)
	if !(scale > 0) || !(fw >= 1 && fw <= maxRender && fh >= 1 && fh <= maxRender) {
		return 0, 0, fmt.Errorf("gfx: cannot render %.0fx%.0f scene units at scale %v: each side must come to 1..%d pixels", area.W(), area.H(), scale, maxRender)
	}
	return int(fw), int(fh), nil
}

// RenderSubtree draws the subtree of root, with the clips of its ancestors, into a Raster; a nil
// root draws the whole scene. area is the scene rectangle to draw, or the zero Rect for the bounds
// of the subtree; scale is device pixels per scene unit. bg is the background and is transparent
// when it is the zero Color; use White for an opaque page. It commits the scene's pending
// transactions first and needs no window.
func RenderSubtree(r Renderer, s *Scene, root *Node, area Rect, scale float64, bg Color) (*Raster, error) {
	s.Commit()
	if root == nil {
		root = s.Root()
	}
	if noArea(area) {
		s.Read(func() { area = root.SubtreeBounds() })
	}
	if noArea(area) {
		return nil, fmt.Errorf("gfx: nothing to render: node %d and its subtree draw nothing (no picture, or clipped away)", root.ID())
	}
	w, h, err := pixelSize(area, scale)
	if err != nil {
		return nil, err
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
		return nil, errors.New("gfx: nothing to render: the picture draws nothing")
	}
	w, h, err := pixelSize(p.Bounds, scale)
	if err != nil {
		return nil, err
	}
	t := r.NewTarget(w, h)
	defer t.Close()
	t.Begin(scale, p.Bounds.Min, bg)
	t.Draw([]Item{{Pic: p, M: Identity(), Alpha: 1}})
	return FromBGRA(t.Pixels(), w, h), nil
}
