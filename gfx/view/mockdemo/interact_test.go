//go:build darwin && gfxmock

package main

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestClampToView(t *testing.T) {
	for _, c := range []struct{ in, want gfx.Point }{
		{gfx.Point{X: -300, Y: 150}, gfx.Point{X: 8, Y: 150}},
		{gfx.Point{X: 900, Y: 5000}, gfx.Point{X: 392, Y: 292}},
		{gfx.Point{X: 100, Y: 100}, gfx.Point{X: 100, Y: 100}},
	} {
		if got := clampToView(c.in, 400, 300, dropMargin); got != c.want {
			t.Errorf("%v: got %v want %v", c.in, got, c.want)
		}
	}
}
