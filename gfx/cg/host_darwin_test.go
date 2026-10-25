//go:build darwin

package cg

import (
	"math"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

// The host works on a bare CALayer: no view, no window.
func TestHostLayers(t *testing.T) {
	initHost()
	root := send0(send0(clsLayer, sels.alloc), sels.init)
	h := NewHost(root, 2)
	defer h.Close()

	r := NewRenderer(Options{})
	s := r.NewSurface(64, 64)
	defer s.Close()
	s.Begin(1, 0, 0, gfx.Color{R: 1, A: 1})
	img := s.Snapshot(32, 32)

	h.Begin()
	l := h.NewLayer(gfx.GroupTiles).(*layer)
	l.SetImage(img)
	l.Set(gfx.R(10, 20, 30, 40), gfx.Point{}, gfx.Identity())
	l.SetOpacity(0.5)
	h.Commit()
	if got := msgRectRet(l.ref, sels.frame); got != (cgRect{10, 20, 30, 40}) {
		t.Errorf("frame %v", got)
	}
	if send0(l.ref, sels.minFilter) != filterTrilinear {
		t.Error("tile layer does not minify trilinearly")
	}
	if send0(l.ref, sels.contents) == 0 {
		t.Error("no contents")
	}
	if o := msgF32Ret(l.ref, sels.opacity); o != 0.5 {
		t.Errorf("opacity %v", o)
	}

	// rotation by 90 degrees about the frame's centre: the frame stays, the transform is applied
	h.Begin()
	l.Set(gfx.R(10, 20, 30, 40), gfx.Point{X: 15, Y: 20}, gfx.RotateBy(math.Pi/2))
	h.Commit()
	m := msgAffRet(l.ref, sels.affineTransform)
	if math.Abs(m.A) > 1e-9 || math.Abs(m.B-1) > 1e-9 || math.Abs(m.C+1) > 1e-9 {
		t.Errorf("transform %+v", m)
	}
	if f := msgRectRet(l.ref, sels.frame); math.Abs(f.W-40) > 1e-6 || math.Abs(f.H-30) > 1e-6 {
		t.Errorf("transformed frame %+v", f)
	}

	// back to plain: the anchor returns to the corner, setFrame places exactly
	h.Begin()
	l.Set(gfx.R(5, 6, 30, 40), gfx.Point{}, gfx.Identity())
	h.Commit()
	if got := msgRectRet(l.ref, sels.frame); got != (cgRect{5, 6, 30, 40}) {
		t.Errorf("frame after reset %v", got)
	}
	l.Remove()
}
