//go:build darwin || linux || windows

package view

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestCovered(t *testing.T) {
	v := &View{overlays: []*Overlay{{b: [4]int{10, 20, 100, 30}}, {b: [4]int{-1, -1, -1, -1}}}}
	for _, c := range []struct {
		p    gfx.Point
		want bool
	}{
		{gfx.Point{X: 10, Y: 20}, true},
		{gfx.Point{X: 109.5, Y: 49.5}, true},
		{gfx.Point{X: 110, Y: 30}, false},
		{gfx.Point{X: 50, Y: 50}, false},
		{gfx.Point{X: 9, Y: 25}, false},
		{gfx.Point{X: -1, Y: -1}, false}, // an overlay not placed yet covers nothing
	} {
		if got := v.covered(c.p); got != c.want {
			t.Errorf("covered(%v) = %v, want %v", c.p, got, c.want)
		}
	}
}
