package view

import (
	"math"
	"testing"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

// fakeCam has the compositor's camera arithmetic without a renderer.
type fakeCam struct{ c gfx.Camera }

func (f *fakeCam) Camera() gfx.Camera     { return f.c }
func (f *fakeCam) SetCamera(c gfx.Camera) { f.c = c }
func (f *fakeCam) SceneAt(p gfx.Point) gfx.Point {
	return gfx.Point{X: f.c.Origin.X + p.X/f.c.Zoom, Y: f.c.Origin.Y + p.Y/f.c.Zoom}
}
func (f *fakeCam) ZoomAt(k float64, p gfx.Point) {
	sp := f.SceneAt(p)
	z := f.c.Zoom * k
	f.c = gfx.Camera{Origin: gfx.Point{X: sp.X - p.X/z, Y: sp.Y - p.Y/z}, Zoom: z}
}

// rig is a router over a real scene with one 20 x 20 node at (100, 50).
type rig struct {
	r    *router
	cam  *fakeCam
	node *gfx.Node
	got  []*Pointer
}

func newRig(t *testing.T) *rig {
	t.Helper()
	s := gfx.NewScene(gfx.R(-1000, -1000, 4000, 4000), 128)
	n := gfx.NewNode(1)
	p := &gfx.Path{}
	p.Rect(0, 0, 20, 20)
	rec := gfx.NewRecorder()
	rec.FillPath(p, gfx.Color{A: 1})
	s.Update(func(tx *gfx.Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, rec.Finish())
		tx.SetShape(n, p, gfx.HitFill)
		tx.Move(n, 100, 50)
	})
	s.Commit()
	cam := &fakeCam{gfx.Camera{Origin: gfx.Point{X: 100, Y: 50}, Zoom: 2}}
	rg := &rig{cam: cam, node: n}
	rg.r = newRouter(cam, func(p gfx.Point) *gfx.Node { return s.HitTest(p, 0) })
	rg.r.onPointer = func(p *Pointer) { rg.got = append(rg.got, p) }
	return rg
}

func pt(x, y float64) gfx.Point { return gfx.Point{X: x, Y: y} }

func TestPointerSceneAndNode(t *testing.T) {
	rg := newRig(t)
	rg.r.pointer(PointerDown, pt(20, 10), 1, 2, g.Mods{Shift: true})
	p := rg.got[0]
	if p.Scene != pt(110, 55) || p.Node != rg.node || p.Count != 2 || !p.Mods.Shift || p.Target != rg.node {
		t.Fatalf("%+v", p)
	}
	rg.r.pointer(PointerUp, pt(20, 10), 1, 0, g.Mods{})
	rg.r.pointer(PointerMove, pt(200, 200), 0, 0, g.Mods{})
	if p := rg.got[2]; p.Node != nil || p.Drag || p.Target != nil {
		t.Fatalf("hover on empty space: %+v", p)
	}
}

func TestPointerCapture(t *testing.T) {
	rg := newRig(t)
	rg.r.pointer(PointerDown, pt(20, 10), 1, 1, g.Mods{})
	rg.r.pointer(PointerMove, pt(-500, 900), 0, 0, g.Mods{}) // far outside the view
	m := rg.got[1]
	if !m.Drag || m.Button != 1 || m.Target != rg.node || m.Origin != pt(110, 55) || m.Node != nil {
		t.Fatalf("drag move: %+v", m)
	}
	rg.r.pointer(PointerDown, pt(-500, 900), 3, 1, g.Mods{}) // a second button keeps the first press as the target
	if p := rg.got[2]; p.Target != rg.node {
		t.Fatalf("second press lost the target: %+v", p)
	}
	rg.r.pointer(PointerUp, pt(-500, 900), 1, 0, g.Mods{})
	rg.r.pointer(PointerMove, pt(-400, 900), 0, 0, g.Mods{})
	if p := rg.got[4]; !p.Drag || p.Button != 3 {
		t.Fatalf("button 3 still holds the capture: %+v", p)
	}
	rg.r.pointer(PointerUp, pt(-400, 900), 3, 0, g.Mods{})
	if p := rg.got[5]; !p.Drag {
		t.Fatalf("the last release belongs to the drag: %+v", p)
	}
	rg.r.pointer(PointerMove, pt(-300, 900), 0, 0, g.Mods{})
	if p := rg.got[6]; p.Drag || p.Target != nil {
		t.Fatalf("capture must end with the last release: %+v", p)
	}
}

func TestPointerCancel(t *testing.T) {
	rg := newRig(t)
	rg.r.pointer(PointerDown, pt(20, 10), 1, 1, g.Mods{})
	rg.r.pointer(PointerCancel, pt(30, 10), 0, 0, g.Mods{})
	if p := rg.got[1]; !p.Drag || p.Kind != PointerCancel || p.Target != rg.node {
		t.Fatalf("%+v", p)
	}
	rg.r.pointer(PointerMove, pt(40, 10), 0, 0, g.Mods{})
	if rg.got[2].Drag {
		t.Fatal("a cancel must release the capture")
	}
}

func TestScrollPansByPointsAndLines(t *testing.T) {
	rg := newRig(t)
	rg.r.scroll(pt(0, 0), -30, 20, true, PhaseChanged, PhaseNone, g.Mods{})
	if o := rg.cam.c.Origin; o != pt(115, 40) {
		t.Fatalf("precise delta moves by points over zoom: %v", o)
	}
	rg.r.scroll(pt(0, 0), 0, 1, false, PhaseNone, PhaseNone, g.Mods{})
	if o := rg.cam.c.Origin; o != pt(115, 40-lineStep/2) {
		t.Fatalf("a wheel line moves by lineStep: %v", o)
	}
}

func TestScrollHandlerConsumes(t *testing.T) {
	rg := newRig(t)
	var seen *Scroll
	rg.r.onScroll = func(s *Scroll) bool { seen = s; return true }
	rg.r.scroll(pt(20, 10), 5, 6, true, PhaseBegan, PhaseNone, g.Mods{})
	if seen == nil || seen.Scene != pt(110, 55) || seen.Phase != PhaseBegan || rg.cam.c.Origin != pt(100, 50) {
		t.Fatalf("%+v camera %v", seen, rg.cam.c)
	}
}

func TestScrollWithCmdZoomsAboutPointer(t *testing.T) {
	rg := newRig(t)
	at := pt(40, 30)
	before := rg.cam.SceneAt(at)
	rg.r.scroll(at, 0, 40, true, PhaseChanged, PhaseNone, g.Mods{Cmd: true})
	if rg.cam.c.Zoom <= 2 || rg.cam.SceneAt(at) != before {
		t.Fatalf("camera %+v, scene point under pointer %v want %v", rg.cam.c, rg.cam.SceneAt(at), before)
	}
}

func TestMomentumStopsOnPressAndTouch(t *testing.T) {
	rg := newRig(t)
	coast := func() { rg.r.scroll(pt(0, 0), 0, 10, true, PhaseNone, PhaseChanged, g.Mods{}) }
	rg.r.scroll(pt(0, 0), 0, 10, true, PhaseBegan, PhaseNone, g.Mods{})
	y := rg.cam.c.Origin.Y
	coast()
	if rg.cam.c.Origin.Y == y {
		t.Fatal("momentum after a finger scroll must move the camera")
	}
	rg.r.pointer(PointerDown, pt(0, 0), 1, 1, g.Mods{})
	y = rg.cam.c.Origin.Y
	coast()
	if rg.cam.c.Origin.Y != y {
		t.Fatal("a press stops the coasting")
	}
	rg.r.scroll(pt(0, 0), 0, 0, true, PhaseBegan, PhaseNone, g.Mods{})
	rg.r.scroll(pt(0, 0), 0, 0, true, PhaseMayBegin, PhaseNone, g.Mods{})
	coast()
	if rg.cam.c.Origin.Y != y {
		t.Fatal("fingers resting on the trackpad stop the coasting")
	}
}

func TestMagnifyKeepsPointAndClamps(t *testing.T) {
	rg := newRig(t)
	at := pt(300, 200)
	before := rg.cam.SceneAt(at)
	rg.r.magnify(at, 0.5, PhaseChanged, g.Mods{})
	if z := rg.cam.c.Zoom; math.Abs(z-3) > 1e-9 {
		t.Fatalf("zoom %v", z)
	}
	if got := rg.cam.SceneAt(at); math.Abs(got.X-before.X) > 1e-9 || math.Abs(got.Y-before.Y) > 1e-9 {
		t.Fatalf("scene point moved: %v -> %v", before, got)
	}
	rg.r.magnify(at, 1000, PhaseChanged, g.Mods{})
	if rg.cam.c.Zoom != rg.r.maxZ {
		t.Fatalf("zoom %v beyond the limit %v", rg.cam.c.Zoom, rg.r.maxZ)
	}
}

func TestSmartZoomToggles(t *testing.T) {
	rg := newRig(t)
	rg.r.smartZoom(pt(10, 10), g.Mods{})
	if rg.cam.c.Zoom != 1 {
		t.Fatalf("first smart zoom goes to 1: %v", rg.cam.c.Zoom)
	}
	rg.r.smartZoom(pt(10, 10), g.Mods{})
	if math.Abs(rg.cam.c.Zoom-2) > 1e-9 {
		t.Fatalf("second goes back: %v", rg.cam.c.Zoom)
	}
}

func TestPhaseOf(t *testing.T) {
	for mask, want := range map[uint64]Phase{0: PhaseNone, 1: PhaseBegan, 2: PhaseChanged, 4: PhaseChanged, 8: PhaseEnded, 16: PhaseCancelled, 32: PhaseMayBegin} {
		if got := phaseOf(mask); got != want {
			t.Errorf("mask %d: %d want %d", mask, got, want)
		}
	}
}
