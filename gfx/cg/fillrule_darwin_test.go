//go:build darwin

package cg

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestReplayEvenOddHole(t *testing.T) {
	r := NewRenderer(Options{})
	s := r.NewSurface(60, 60)
	defer s.Close()
	p := &gfx.Path{EvenOdd: true}
	p.Rect(5, 5, 40, 40)
	p.Rect(15, 15, 20, 20)
	q := &gfx.Path{}
	q.Rect(0, 0, 60, 60)

	rec := gfx.NewRecorder()
	rec.FillPath(p, gfx.Color{R: 1, A: 1})
	s.Begin(1, 0, 0, gfx.Color{})
	s.Replay(rec.Finish(), 0, 0)
	if got := pixel(s, 10, 25); got != [4]byte{0, 0, 255, 255} {
		t.Errorf("ring body: BGRA %v", got)
	}
	if got := pixel(s, 25, 25); got != [4]byte{} {
		t.Errorf("fill hole: BGRA %v", got)
	}

	rec = gfx.NewRecorder()
	rec.ClipPath(p)
	rec.FillPath(q, gfx.Color{B: 1, A: 1})
	s.Begin(1, 0, 0, gfx.Color{})
	s.Replay(rec.Finish(), 0, 0)
	if got := pixel(s, 10, 25); got != [4]byte{255, 0, 0, 255} {
		t.Errorf("clipped ring body: BGRA %v", got)
	}
	if got := pixel(s, 25, 25); got != [4]byte{} {
		t.Errorf("clip hole: BGRA %v", got)
	}
}
