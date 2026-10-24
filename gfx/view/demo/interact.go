//go:build darwin && gfxdemo

package main

import (
	"fmt"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
	"github.com/haiodo/gowt/gfx/synth"
	"github.com/haiodo/gowt/gfx/view"
	"github.com/haiodo/gowt/swt"
)

const (
	modeNone = iota
	modePan
	modeDrag
)

// dropMargin keeps a dropped box's grab point this far inside the view, view points.
const dropMargin = 8

// clampToView moves a view point into the view, inset by margin.
func clampToView(p gfx.Point, w, h, margin float64) gfx.Point {
	return gfx.Point{X: min(max(p.X, margin), w-margin), Y: min(max(p.Y, margin), h-margin)}
}

// interactive is the GFXDEMO_STAY mode after the timed phases: the view's own scroll and pinch
// navigation, plus dragging a box with its connectors and dragging the empty space.
func interactive(v *view.View, doc *synth.Doc) {
	c := v.Compositor()
	var (
		mode   int
		box    *synth.Box
		lifted []*gfx.Node
		grab   gfx.Point // box origin minus the pointer, in scene points
		last   gfx.Point
		shape  = view.CursorArrow
	)
	cursor := func(s view.Cursor) {
		if s != shape {
			shape = s
			v.SetCursor(s)
		}
	}
	land := func() {
		if mode == modeDrag {
			c.Land(lifted...)
			lifted = lifted[:0]
		}
		mode = modeNone
	}
	v.OnPointer(func(p *view.Pointer) {
		if cg.Tracing {
			cg.Tracef("demo pointer kind=%d mode=%d drag=%v aborted=%v", p.Kind, mode, p.Drag, p.Aborted)
		}
		switch p.Kind {
		case view.PointerDown:
			if p.Button != 1 {
				return
			}
			last = p.Pos
			if b := boxOf(doc, p.Node); b != nil {
				mode, box = modeDrag, b
				grab = gfx.Point{X: b.X - p.Scene.X, Y: b.Y - p.Scene.Y}
				lifted = append(lifted[:0], b.Node)
				for _, cn := range b.Conns {
					lifted = append(lifted, cn.Node)
				}
				c.Lift(lifted...)
				v.AutoScroll(view.AutoScrollOptions{})
				cursor(view.CursorGrabbing)
				return
			}
			mode = modePan
			cursor(view.CursorGrabbing)
		case view.PointerMove:
			switch {
			case mode == modeDrag && p.Drag:
				x, y := p.Scene.X+grab.X, p.Scene.Y+grab.Y
				doc.Scene.Update(func(tx *gfx.Tx) { doc.Move(tx, box, x, y) })
			case mode == modePan && p.Drag:
				v.Pan(p.Pos.X-last.X, p.Pos.Y-last.Y)
				last = p.Pos
			case !p.Drag && boxOf(doc, p.Node) != nil:
				cursor(view.CursorHand)
			case !p.Drag:
				cursor(view.CursorGrab)
			}
		case view.PointerUp, view.PointerCancel:
			if p.Kind == view.PointerUp && p.Outside && mode == modeDrag {
				// the demo's drop rule, as in Figma and Freeform: the box lands at the nearest
				// visible point, never offscreen
				w, h := v.Canvas().Size()
				q := c.SceneAt(clampToView(p.Pos, float64(w), float64(h), dropMargin))
				x, y := q.X+grab.X, q.Y+grab.Y
				doc.Scene.Update(func(tx *gfx.Tx) { doc.Move(tx, box, x, y) })
			}
			if p.Aborted && mode == modeDrag {
				x, y := p.Origin.X+grab.X, p.Origin.Y+grab.Y
				doc.Scene.Update(func(tx *gfx.Tx) { doc.Move(tx, box, x, y) })
			}
			land()
			cursor(view.CursorGrab)
		}
	})
	v.OnKey(func(k *view.Key) {
		if !k.Down {
			return
		}
		w, h := v.Canvas().Size()
		mid := gfx.Point{X: float64(w) / 2, Y: float64(h) / 2}
		switch {
		case k.Char == '0':
			c.ZoomAt(1/c.Camera().Zoom, mid)
		case k.Char == '=' || k.Char == '+':
			c.ZoomAt(1.25, mid)
		case k.Char == '-':
			c.ZoomAt(0.8, mid)
		case k.Code == int(swt.ARROW_LEFT):
			v.Pan(100, 0)
		case k.Code == int(swt.ARROW_RIGHT):
			v.Pan(-100, 0)
		case k.Code == int(swt.ARROW_UP):
			v.Pan(0, 100)
		case k.Code == int(swt.ARROW_DOWN):
			v.Pan(0, -100)
		}
	})
	v.SetCursor(view.CursorGrab)
	fmt.Println("interactive: two-finger scroll = pan, pinch = zoom, hold Cmd and two-finger scroll up/down = zoom, two-finger double tap = zoom 1, drag empty space = pan, drag a box = move it, Escape while dragging = put it back, keys 0 = - + arrows after a click")
}

// boxOf maps a hit node to its box: boxes carry their index as the node id, connectors do not.
func boxOf(doc *synth.Doc, n *gfx.Node) *synth.Box {
	if n != nil && n.ID() < len(doc.Boxes) && doc.Boxes[n.ID()].Node == n {
		return doc.Boxes[n.ID()]
	}
	return nil
}
