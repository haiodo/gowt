package view

import (
	"math"
	"testing"
	"time"

	g "github.com/haiodo/gowt"
)

const frame = 16 * time.Millisecond

// autoRig is a rig over a 400 x 300 view with a fake frame clock and a drag in progress.
func autoRig(t *testing.T) *rig {
	rg := newRig(t)
	rg.cam.c.Zoom = 1
	rg.r.size = func() (float64, float64) { return 400, 300 }
	rg.r.pointer(PointerDown, pt(200, 150), 1, 1, g.Mods{})
	rg.r.startAuto(AutoScrollOptions{})
	rg.got = nil
	return rg
}

func (rg *rig) frames(n int) {
	for i := 0; i < n; i++ {
		rg.tick += frame
		rg.r.tick(rg.tick)
	}
}

func TestAutoScrollEvenAndGrowsWithDistance(t *testing.T) {
	var steps []float64
	for _, x := range []float64{-20, -100, -300} { // past the left edge, increasingly
		rg := autoRig(t)
		rg.r.pointer(PointerMove, pt(x, 150), 0, 0, g.Mods{})
		rg.frames(1) // the first frame only starts the clock
		var prev float64 = rg.cam.c.Origin.X
		var d []float64
		for i := 0; i < 5; i++ {
			rg.frames(1)
			d = append(d, prev-rg.cam.c.Origin.X)
			prev = rg.cam.c.Origin.X
		}
		for _, v := range d {
			if v <= 0 || math.Abs(v-d[0]) > 1e-9 {
				t.Fatalf("x=%v: steps not even: %v", x, d)
			}
		}
		if rg.cam.c.Origin.Y != 50 {
			t.Fatalf("y moved: %v", rg.cam.c.Origin)
		}
		steps = append(steps, d[0])
	}
	if !(steps[0] < steps[1] && steps[1] <= steps[2]) {
		t.Fatalf("speed does not grow: %v", steps)
	}
	if cap := 1000 * frame.Seconds(); steps[2] > cap+1e-9 {
		t.Fatalf("step %v over the cap %v", steps[2], cap)
	}
}

func TestAutoScrollInsideBandAndCentre(t *testing.T) {
	rg := autoRig(t)
	rg.r.pointer(PointerMove, pt(200, 150), 0, 0, g.Mods{})
	rg.frames(5)
	if rg.cam.c.Origin != pt(100, 50) {
		t.Fatalf("scrolled in the middle: %v", rg.cam.c.Origin)
	}
	rg.r.pointer(PointerMove, pt(390, 150), 0, 0, g.Mods{}) // 10 pt from the right edge: in the band
	rg.frames(3)
	if rg.cam.c.Origin.X <= 100 {
		t.Fatalf("no scroll in the band: %v", rg.cam.c.Origin)
	}
}

// The app moves a node to p.Scene on every move; the synthetic moves keep it under the pointer.
func TestAutoScrollNodeFollowsScene(t *testing.T) {
	rg := autoRig(t)
	var scene []float64
	rg.r.onPointer = func(p *Pointer) {
		if p.Kind == PointerMove {
			scene = append(scene, p.Scene.X)
			if !p.Drag || !p.Outside {
				t.Errorf("synthetic move %+v", p)
			}
		}
	}
	rg.r.pointer(PointerMove, pt(-50, 150), 0, 0, g.Mods{})
	rg.frames(6) // the pointer stands still
	if len(scene) < 5 {
		t.Fatalf("moves: %v", scene)
	}
	for i := 1; i < len(scene); i++ {
		if scene[i] >= scene[i-1] {
			t.Fatalf("scene point under the pointer does not move left: %v", scene)
		}
	}
	if want := rg.cam.SceneAt(pt(-50, 150)).X; scene[len(scene)-1] != want {
		t.Fatalf("scene %v want %v", scene[len(scene)-1], want)
	}
}

func TestAutoScrollStops(t *testing.T) {
	for name, end := range map[string]func(*rig){
		"release": func(rg *rig) { rg.r.pointer(PointerUp, pt(-50, 150), 1, 0, g.Mods{}) },
		"escape":  func(rg *rig) { rg.r.abort() },
		"stop":    nil,
	} {
		rg := autoRig(t)
		var stop func()
		rg.r.pointer(PointerMove, pt(-50, 150), 0, 0, g.Mods{})
		if end == nil {
			stop = func() { rg.r.stopAuto() }
			end = func(*rig) { stop() }
		}
		rg.frames(3)
		x := rg.cam.c.Origin.X
		end(rg)
		if name == "escape" {
			last := rg.got[len(rg.got)-1]
			if last.Kind != PointerCancel || !last.Aborted || !last.Outside {
				t.Fatalf("escape: %+v", last)
			}
		}
		rg.frames(5)
		if rg.cam.c.Origin.X != x || rg.r.autoWake.Load() {
			t.Fatalf("%s: still scrolling", name)
		}
	}
}

func TestOutsideCoordinates(t *testing.T) {
	rg := autoRig(t)
	rg.r.pointer(PointerMove, pt(-13.5, 4000), 0, 0, g.Mods{})
	p := rg.got[len(rg.got)-1]
	if !p.Outside || p.Pos != pt(-13.5, 4000) || p.Scene != rg.cam.SceneAt(pt(-13.5, 4000)) {
		t.Fatalf("%+v", p)
	}
	rg.r.pointer(PointerMove, pt(399, 299), 0, 0, g.Mods{})
	if rg.got[len(rg.got)-1].Outside {
		t.Fatal("inside reported outside")
	}
}
