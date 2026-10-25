package synth

import (
	"fmt"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func BenchmarkHitTest(b *testing.B) {
	for _, n := range []int{1000, 10000, 50000} {
		d := New(n, 1)
		mid := d.Boxes[len(d.Boxes)/2]
		boxPt := mid.Node.Bounds().Center()
		cm := mid.Conns[0].Shape.Measure(0.05)
		connPt, _ := cm.At(cm.Length() / 2)
		miss := gfx.Point{X: -50, Y: -50}
		for _, c := range []struct {
			name string
			p    gfx.Point
		}{{"box", boxPt}, {"connector", connPt}, {"miss", miss}} {
			b.Run(fmt.Sprintf("%d/%s", n, c.name), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					d.Scene.HitTest(c.p, 4)
				}
			})
		}
	}
}

// BenchmarkIndexUpdate moves boxes through Scene.Update (remove from and insert into the grid).
func BenchmarkIndexUpdate(b *testing.B) {
	for _, n := range []int{10000, 50000} {
		d := New(n, 1)
		for _, count := range []int{1, 100} {
			b.Run(fmt.Sprintf("%d/boxes=%d", n, count), func(b *testing.B) {
				sel := d.Boxes[n/2 : n/2+count]
				for i := 0; i < b.N; i++ {
					dx := float64(i % 50)
					d.Scene.Update(func(tx *gfx.Tx) {
						for _, bx := range sel {
							tx.Move(bx.Node, bx.X+dx, bx.Y)
						}
					})
					d.Scene.Commit()
				}
			})
		}
	}
}

// BenchmarkMoveBox is the whole model-side cost of one drag step: index update plus
// rebuilding the box's connector pictures (curve, measure, arrowhead, split, record).
func BenchmarkMoveBox(b *testing.B) {
	d := New(10000, 1)
	bx := d.Boxes[5000]
	for len(bx.Conns) < 10 {
		d.Link(bx, d.Boxes[5000+len(bx.Conns)], len(bx.Conns)%2 == 0)
	}
	x, y := bx.X, bx.Y
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.MoveBox(bx, x+float64(i%40), y)
	}
}

func BenchmarkQuery(b *testing.B) {
	for _, n := range []int{10000, 50000} {
		d := New(n, 1)
		cx, cy := float64(d.Cols)*CellW/2, float64(d.Rows)*CellH/2
		for _, c := range []struct {
			name string
			r    gfx.Rect
		}{
			{"tile-512pt", gfx.R(cx, cy, 512, 512)},
			{"viewport", gfx.R(cx, cy, 1440, 900)},
			{"all", gfx.R(-100, -100, 1e7, 1e7)},
		} {
			b.Run(fmt.Sprintf("%d/%s", n, c.name), func(b *testing.B) {
				var buf []*gfx.Node
				for i := 0; i < b.N; i++ {
					buf = d.Scene.Query(c.r, buf[:0])
				}
				b.ReportMetric(float64(len(buf)), "nodes")
			})
		}
	}
}

func BenchmarkBuild(b *testing.B) {
	for _, n := range []int{10000, 50000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				New(n, 1)
			}
		})
	}
}
