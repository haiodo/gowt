package view

import (
	"testing"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

type fakeLift struct{ lifted, landed int }

func (f *fakeLift) Lift(n ...*gfx.Node) { f.lifted += len(n) }
func (f *fakeLift) Land(n ...*gfx.Node) { f.landed += len(n) }

func dragRig(t *testing.T) (*router, *gfx.Scene, *gfx.Node, *fakeLift) {
	t.Helper()
	s := gfx.NewScene()
	n := gfx.NewNode(1)
	rec := gfx.NewRecorder()
	rec.FillRect(gfx.R(0, 0, 20, 20), gfx.Black)
	s.Update(func(tx *gfx.Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
		tx.Move(n, 100, 50)
	})
	s.Commit()
	cam := &fakeCam{gfx.Camera{Origin: gfx.Point{X: 100, Y: 50}, Zoom: 2}}
	r := newRouter(cam, func(p gfx.Point) *gfx.Node { return s.HitTest(p, 0) })
	r.size = func() (float64, float64) { return 200, 100 }
	lf := &fakeLift{}
	r.onPointer = func(p *Pointer) {
		if p.Kind == PointerDown {
			r.beginDrag(lf, s, p, []*gfx.Node{p.Node})
		}
	}
	return r, s, n, lf
}

func pos(s *gfx.Scene, n *gfx.Node) gfx.Point {
	s.Commit()
	t := n.Transform()
	return gfx.Point{X: t[4], Y: t[5]}
}

func TestDragMovesAndLands(t *testing.T) {
	r, s, n, lf := dragRig(t)
	r.pointer(PointerDown, pt(10, 10), 1, 1, g.Mods{})
	r.pointer(PointerMove, pt(30, 20), 0, 0, g.Mods{}) // 20 x 10 view points at zoom 2: 10 x 5 units
	if p := pos(s, n); p != pt(110, 55) {
		t.Fatalf("after move %v", p)
	}
	r.pointer(PointerUp, pt(30, 20), 1, 0, g.Mods{})
	if lf.lifted != 1 || lf.landed != 1 || r.drag != nil {
		t.Fatalf("lift %d land %d drag %v", lf.lifted, lf.landed, r.drag)
	}
}

func TestDragAbortRestores(t *testing.T) {
	r, s, n, lf := dragRig(t)
	r.pointer(PointerDown, pt(10, 10), 1, 1, g.Mods{})
	r.pointer(PointerMove, pt(30, 20), 0, 0, g.Mods{})
	r.abort()
	if p := pos(s, n); p != pt(100, 50) || lf.landed != 1 {
		t.Fatalf("after abort %v, landed %d", p, lf.landed)
	}
}

func TestDragDropOutsideIsClamped(t *testing.T) {
	r, s, n, _ := dragRig(t)
	r.pointer(PointerDown, pt(10, 10), 1, 1, g.Mods{})
	r.pointer(PointerUp, pt(500, 10), 1, 0, g.Mods{})
	// the view is 200 wide, so the drop point is 192 view points: 96 units right of the origin
	if p := pos(s, n); p.X != 100+(100+96-105) {
		t.Fatalf("drop at %v", p)
	}
}
