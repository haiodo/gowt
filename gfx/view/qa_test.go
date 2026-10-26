//go:build darwin || linux || windows

package view

import (
	"math"
	"strings"
	"testing"

	"github.com/haiodo/gowt/internal/qaview"
)

// The QA entry moves the camera before it returns, through the router and its handlers.
func TestQAInputScrollAndPinch(t *testing.T) {
	rg := newRig(t)
	h := qaHook{&View{r: rg.r}}
	if err := h.Scroll(0, 0, -30, 20, true, "changed", "", qaview.Mods{}); err != nil {
		t.Fatal(err)
	}
	if o := rg.cam.c.Origin; o != pt(115, 40) {
		t.Fatalf("scroll without frames: origin %v, want 115,40", o)
	}
	if rg.r.acc.wake.Load() {
		t.Error("the scroll accumulator still wants frames")
	}

	var seen *Scroll
	rg.r.onScroll = func(s *Scroll) bool { seen = s; return true }
	if err := h.Scroll(20, 10, 1, 2, true, "began", "", qaview.Mods{Shift: true}); err != nil {
		t.Fatal(err)
	}
	if seen == nil || seen.Phase != PhaseBegan || !seen.Mods.Shift || seen.Scene != pt(125, 45) {
		t.Fatalf("handler got %+v", seen)
	}

	at := pt(300, 200)
	before := rg.cam.SceneAt(at)
	if err := h.Magnify(300, 200, 0.5, "changed", qaview.Mods{}); err != nil {
		t.Fatal(err)
	}
	if z := rg.cam.c.Zoom; math.Abs(z-3) > 1e-9 || math.Abs(rg.cam.SceneAt(at).X-before.X) > 1e-9 {
		t.Fatalf("pinch: camera %+v", rg.cam.c)
	}

	err := h.Scroll(0, 0, 0, 0, true, "sideways", "", qaview.Mods{})
	if err == nil || !strings.Contains(err.Error(), "sideways") {
		t.Fatalf("bad phase: %v", err)
	}
}

// The QA pointer goes through the router: the press sets the drag target, the release ends it.
func TestQAInputPointer(t *testing.T) {
	rg := newRig(t)
	h := qaHook{&View{r: rg.r}}
	for _, ev := range []struct {
		kind string
		x, y float64
	}{{"down", 20, 10}, {"move", 40, 20}, {"up", 40, 20}} {
		if err := h.Pointer(ev.kind, ev.x, ev.y, 1, 1, qaview.Mods{Alt: true}); err != nil {
			t.Fatal(err)
		}
	}
	if len(rg.got) != 3 || rg.got[0].Kind != PointerDown || rg.got[0].Node != rg.node || !rg.got[0].Mods.Alt ||
		rg.got[1].Kind != PointerMove || !rg.got[1].Drag || rg.got[1].Button != 1 ||
		rg.got[2].Kind != PointerUp || rg.got[2].Scene != pt(120, 60) {
		t.Fatalf("events %+v", rg.got)
	}
	if err := h.Pointer("tap", 0, 0, 1, 1, qaview.Mods{}); err == nil || !strings.Contains(err.Error(), "tap") {
		t.Fatalf("bad kind: %v", err)
	}
}
