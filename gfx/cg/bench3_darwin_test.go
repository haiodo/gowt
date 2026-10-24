//go:build darwin

package cg

import (
	"fmt"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

// BenchmarkBreakdown splits a zoomed-in viewport redraw (10k document, ~50 boxes and ~100
// connectors in view) by node class and op kind.
func BenchmarkBreakdown(b *testing.B) {
	d := doc(10000)
	o := views[2].origin(d, 1)
	only := func(kinds ...gfx.OpKind) uint32 {
		m := ^uint32(0)
		for _, k := range kinds {
			m &^= 1 << k
		}
		return m
	}
	cases := []struct {
		name  string
		boxes bool
		conns bool
		skip  uint32
	}{
		{"clear-only", false, false, 0},
		{"boxes+conns", true, true, 0},
		{"boxes", true, false, 0},
		{"conns", false, true, 0},
		{"boxes:shadowbox-op", true, false, only(gfx.OpShadowBox, gfx.OpSave, gfx.OpRestore, gfx.OpTranslate)},
		{"boxes:stroke-op", true, false, only(gfx.OpStroke)},
		{"boxes:fill-op", true, false, only(gfx.OpFill)},
		{"boxes:text-op", true, false, only(gfx.OpText)},
		{"conns:stroke-op", false, true, only(gfx.OpStroke)},
		{"conns:fill-op", false, true, only(gfx.OpFill)},
	}
	for _, px := range []float64{1, 2} {
		for _, c := range cases {
			b.Run(fmt.Sprintf("%dx/%s", int(px), c.name), func(b *testing.B) {
				r := NewRenderer(Options{TextCache: true, ShadowSprite: true})
				r.skip = c.skip
				s := r.NewSurface(int(vpW*px), int(vpH*px))
				defer s.Close()
				rect := gfx.Rect{Min: o, Max: gfx.Point{X: o.X + vpW, Y: o.Y + vpH}}
				var use []*gfx.Node
				for _, n := range d.Scene.Query(rect, nil) {
					isConn := n.ID >= 1<<20
					if isConn && c.conns || !isConn && c.boxes {
						use = append(use, n)
					}
				}
				run := func() {
					s.Begin(px, o.X, o.Y, white)
					for _, n := range use {
						s.Replay(n.Pic, n.X, n.Y)
					}
				}
				run()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					run()
				}
				b.ReportMetric(float64(len(use)), "nodes")
			})
		}
	}
}
