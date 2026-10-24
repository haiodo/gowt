//go:build darwin && gfxmock

package main

import (
	"fmt"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/view"
	"github.com/haiodo/gowt/swt"
)

const (
	modeNone = iota
	modePan
	modeDrag
)

// interactive is the GFXDEMO_STAY mode: the view's own scroll and pinch navigation, a drag of an
// element (press on it), a drag of a whole screen (press on its background, or hold Option), and
// a drag of the empty canvas.
func interactive(v *view.View, c *canvas) {
	comp := v.Compositor()
	var (
		mode  int
		moved *gfx.Node
		grab  gfx.Point // node origin minus the pointer, in the parent's coordinates
		last  gfx.Point
		shape = view.CursorArrow
	)
	cursor := func(s view.Cursor) {
		if s != shape {
			shape = s
			v.SetCursor(s)
		}
	}
	// screenOf is the ancestor of n that sits directly under the scene root.
	screenOf := func(n *gfx.Node) *gfx.Node {
		for n != nil && n.Parent() != nil && n.Parent() != c.scene.Root() {
			n = n.Parent()
		}
		if n != nil && n.Parent() == c.scene.Root() {
			return n
		}
		return nil
	}
	land := func() {
		if mode == modeDrag {
			comp.Land(moved)
			moved = nil
		}
		mode = modeNone
	}
	v.OnPointer(func(p *view.Pointer) {
		switch p.Kind {
		case view.PointerDown:
			if p.Button != 1 {
				return
			}
			last = p.Pos
			sc := screenOf(p.Node)
			if sc == nil {
				mode = modePan
				cursor(view.CursorGrabbing)
				return
			}
			target := p.Node
			if target == sc || p.Mods.Alt {
				target = sc
			}
			mode, moved = modeDrag, target
			// the parent's world transform is a translation here: screens and elements are never rotated
			pw := target.Parent().World()
			t := target.Transform()
			grab = gfx.Point{X: t[4] - (p.Scene.X - pw[4]), Y: t[5] - (p.Scene.Y - pw[5])}
			comp.Lift(target)
			cursor(view.CursorGrabbing)
		case view.PointerMove:
			switch {
			case mode == modeDrag && p.Drag:
				pw := moved.Parent().World()
				x, y := p.Scene.X-pw[4]+grab.X, p.Scene.Y-pw[5]+grab.Y
				c.scene.Update(func(tx *gfx.Tx) { tx.Move(moved, x, y) })
			case mode == modePan && p.Drag:
				v.Pan(p.Pos.X-last.X, p.Pos.Y-last.Y)
				last = p.Pos
			case !p.Drag && screenOf(p.Node) != nil:
				cursor(view.CursorHand)
			case !p.Drag:
				cursor(view.CursorGrab)
			}
		case view.PointerUp, view.PointerCancel:
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
			comp.ZoomAt(1/comp.Camera().Zoom, mid)
		case k.Char == '=' || k.Char == '+':
			comp.ZoomAt(1.25, mid)
		case k.Char == '-':
			comp.ZoomAt(0.8, mid)
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
	stats := &scrollStats{}
	v.OnFrameReport(stats.add)
	fmt.Println("interactive: two-finger scroll = pan, pinch = zoom, Cmd + scroll = zoom, two-finger double tap = zoom 1, drag empty space = pan, drag an element = move it, drag a screen background or hold Option = move the whole screen, keys 0 = - + arrows after a click")
}
