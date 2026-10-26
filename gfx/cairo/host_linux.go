//go:build linux

package cairo

import "github.com/haiodo/gowt/gfx"

var _ gfx.Host = (*Host)(nil)

// Host is a gfx.Host that composites its layers on the CPU into a cairo context. The owner calls
// Paint from its draw handler when OnDirty says one is due; all calls are from one goroutine.
type Host struct {
	groups [4][]*layer
	dirty  bool
	// OnDirty runs at Commit when a layer changed since the last Commit.
	OnDirty func()
}

// NewHost returns a Host with no layers. onDirty may be nil.
func NewHost(onDirty func()) (*Host, error) {
	if err := bind(); err != nil {
		return nil, err
	}
	return &Host{OnDirty: onDirty}, nil
}

// Close releases every layer image.
func (h *Host) Close() {
	for g := range h.groups {
		for _, l := range h.groups[g] {
			l.release()
		}
		h.groups[g] = nil
	}
}

// Begin starts the layer changes of one frame.
func (h *Host) Begin() {}

// Commit queues a redraw of the widget.
func (h *Host) Commit() {
	if h.dirty {
		h.dirty = false
		if h.OnDirty != nil {
			h.OnDirty()
		}
	}
}

// NewLayer adds a layer in group g.
func (h *Host) NewLayer(g gfx.Group) gfx.Layer {
	l := &layer{h: h, g: g, op: 1, m: gfx.Identity()}
	h.groups[g] = append(h.groups[g], l)
	return l
}

// Paint draws the layers into cr, whose user space is the view in points, y down.
func (h *Host) Paint(cr uintptr) {
	for _, g := range h.groups {
		for _, l := range g {
			l.paint(cr)
		}
	}
}

type layer struct {
	h      *Host
	g      gfx.Group
	img    *Image
	frame  gfx.Rect
	anchor gfx.Point
	m      gfx.Affine
	op     float64
	hidden bool
}

func (l *layer) release() {
	if l.img != nil {
		l.img.Release()
		l.img = nil
	}
}

func (l *layer) SetImage(img gfx.Image) {
	i, ok := img.(*Image)
	if !ok || i.surf == 0 {
		return
	}
	l.release()
	l.img = &Image{surf: surfaceReference(i.surf), w: i.w, h: i.h, wpx: i.wpx, hpx: i.hpx}
	l.h.dirty = true
}

func (l *layer) Set(f gfx.Rect, anchor gfx.Point, m gfx.Affine) {
	if f == l.frame && anchor == l.anchor && m == l.m {
		return
	}
	l.frame, l.anchor, l.m = f, anchor, m
	l.h.dirty = true
}

func (l *layer) SetOpacity(a float64) {
	if a != l.op {
		l.op = a
		l.h.dirty = true
	}
}

func (l *layer) SetHidden(hid bool) {
	if hid != l.hidden {
		l.hidden = hid
		l.h.dirty = true
	}
}

func (l *layer) Remove() {
	gs := l.h.groups[l.g]
	for i, o := range gs {
		if o == l {
			l.h.groups[l.g] = append(gs[:i:i], gs[i+1:]...)
			break
		}
	}
	l.release()
	l.h.dirty = true
}

func (l *layer) paint(cr uintptr) {
	if l.hidden || l.img == nil || l.op <= 0 || l.frame.W() <= 0 || l.frame.H() <= 0 {
		return
	}
	save(cr)
	translate(cr, l.frame.Min.X+l.anchor.X, l.frame.Min.Y+l.anchor.Y)
	plain := l.m == gfx.Identity()
	if !plain {
		cm := matrix{l.m[0], l.m[1], l.m[2], l.m[3], l.m[4], l.m[5]}
		transform(cr, &cm)
	}
	translate(cr, -l.anchor.X, -l.anchor.Y)
	sx, sy := l.frame.W()/float64(l.img.wpx), l.frame.H()/float64(l.img.hpx)
	scale(cr, sx, sy)
	setSourceSurface(cr, l.img.surf, 0, 0)
	if plain && sx == 1 && sy == 1 {
		patternSetFilter(getSource(cr), filterFast)
	} else {
		patternSetFilter(getSource(cr), filterBilinear)
	}
	rectangle(cr, 0, 0, float64(l.img.wpx), float64(l.img.hpx))
	clip(cr)
	if l.op >= 1 {
		paint(cr)
	} else {
		paintWithAlpha(cr, l.op)
	}
	restore(cr)
}
