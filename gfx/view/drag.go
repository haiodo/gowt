package view

import "github.com/haiodo/gowt/gfx"

// dropMargin keeps a node dropped outside the view this far inside it, in view points.
const dropMargin = 8

// lifter is the part of gfx.Compositor a drag uses.
type lifter interface {
	Lift(nodes ...*gfx.Node)
	Land(nodes ...*gfx.Node)
}

// drag moves nodes with the pointer from the press that began it to the release.
type drag struct {
	lift  lifter
	scene *gfx.Scene
	nodes []*gfx.Node
	start []gfx.Point // translation of each node at the press, in its parent's space
	grab  gfx.Point   // scene point of the press
}

func (r *router) beginDrag(l lifter, s *gfx.Scene, p *Pointer, nodes []*gfx.Node) {
	if r.drag != nil {
		r.endDrag()
	}
	d := &drag{lift: l, scene: s, nodes: nodes, grab: p.Scene}
	for _, n := range nodes {
		t := n.Transform()
		d.start = append(d.start, gfx.Point{X: t[4], Y: t[5]})
	}
	l.Lift(nodes...)
	r.drag = d
}

func (r *router) endDrag() {
	if r.drag != nil {
		r.drag.lift.Land(r.drag.nodes...)
		r.drag = nil
	}
}

// moveTo puts the nodes where the pointer at scene point q takes them.
func (d *drag) moveTo(q gfx.Point) {
	dx, dy := q.X-d.grab.X, q.Y-d.grab.Y
	d.scene.Update(func(tx *gfx.Tx) {
		for i, n := range d.nodes {
			v := gfx.Point{X: dx, Y: dy}
			if par := n.Parent(); par != nil {
				if inv, ok := par.World().Invert(); ok {
					o := inv.Apply(gfx.Point{})
					w := inv.Apply(v)
					v = gfx.Point{X: w.X - o.X, Y: w.Y - o.Y}
				}
			}
			tx.Move(n, d.start[i].X+v.X, d.start[i].Y+v.Y)
		}
	})
}

// stepDrag runs before the app's pointer callback, so the callback sees the nodes already moved.
func (r *router) stepDrag(p *Pointer) {
	d := r.drag
	if d == nil {
		return
	}
	switch p.Kind {
	case PointerMove:
		if p.Drag {
			d.moveTo(p.Scene)
		}
	case PointerUp:
		q := p.Scene
		if p.Outside && r.size != nil {
			w, h := r.size()
			c := gfx.Point{X: min(max(p.Pos.X, dropMargin), w-dropMargin), Y: min(max(p.Pos.Y, dropMargin), h-dropMargin)}
			q = r.cam.SceneAt(c)
		}
		d.moveTo(q)
		r.endDrag()
	case PointerCancel:
		if p.Aborted {
			d.moveTo(d.grab)
		}
		r.endDrag()
	}
}
