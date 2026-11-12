//go:build darwin

package cg

import (
	"fmt"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
)

// directBenchScene mirrors the profile of the tenniarb.tenn "Inspector concept" document the
// task measures on: 1036 nodes over about 2000x2000 points, most of them plain boxes and labels,
// only a few with shadows. (The synth document gives every box a shadow, which is a much heavier
// fill-rate load and not comparable.) The tenniarb stand itself cannot run from this repo; this
// is the gowt-side equivalent.
func directBenchScene() *gfx.Scene {
	s := gfx.NewScene()
	const cols, rows = 40, 26 // 1040 nodes
	for i := 0; i < cols*rows; i++ {
		x, y := float64(i%cols)*52, float64(i/cols)*76
		rec := gfx.NewRecorder()
		rec.FillRRect(gfx.R(0, 0, 44, 28), 4, gfx.RGB(240, 240, 245))
		rec.Text(fmt.Sprintf("Item %d", i), 4, 18, gfx.Font{Size: 11}, gfx.RGB(20, 20, 24))
		if i == 517 { // the three shadowed cards, like the real document
			rec.FillShadowRRect(0, 0, 44, 28, 6, gfx.RGB(30, 120, 220), gfx.Shadow{DX: 0, DY: 3, Blur: 10, Color: gfx.Color{A: 0.35}})
		}
		n := gfx.NewNode(i)
		pic := rec.Finish()
		s.Update(func(tx *gfx.Tx) {
			tx.Add(nil, n)
			tx.SetPicture(n, pic)
			tx.Move(n, x, y)
		})
	}
	s.Commit()
	return s
}

// runFrames drives n frames with a small pan each frame, so every frame does a full repaint in
// the Direct mode; returns the frame CPU times.
const (
	benchCols = 40
	benchRows = 26
)

func runFrames(c *gfx.Compositor, n int, pan float64) []time.Duration {
	var cpus []time.Duration
	for i := 0; i < n; i++ {
		cam := c.Camera()
		cam.Origin.X += pan
		c.SetCamera(cam)
		c.Frame(gfx.FrameInfo{Time: time.Duration(i+1) * 16666667, Interval: 16666667, Seq: uint64(i + 1)})
		cpus = append(cpus, c.Stats().FrameTime)
	}
	return cpus
}

func pctlDur(d []time.Duration, p float64) time.Duration {
	if len(d) == 0 {
		return 0
	}
	// insert sort is fine for a bench helper
	for i := 1; i < len(d); i++ {
		for j := i; j > 0 && d[j] < d[j-1]; j-- {
			d[j], d[j-1] = d[j-1], d[j]
		}
	}
	return d[int(p*float64(len(d)-1))]
}

// BenchmarkDirectFrame measures a full repaint per frame at the zooms of the task's acceptance
// (1x-8x) and the same for the tile mode, on the ~1k-node scene. Run with:
//
//	go test ./gfx/cg -run '^$' -bench DirectFrame -benchtime 1x
func BenchmarkDirectFrame(b *testing.B) {
	initHost()
	root := send0(send0(clsLayer, sels.alloc), sels.init)
	host := NewHost(root, 2)
	defer host.Close()
	zooms := []float64{1, 1.3, 2, 3, 4, 8}
	for _, direct := range []bool{true, false} {
		b.Run(map[bool]string{true: "Direct", false: "Tile"}[direct], func(b *testing.B) {
			for _, z := range zooms {
				b.Run(fmt.Sprintf("%gx", z), func(b *testing.B) {
					scene := directBenchScene()
					r := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true})
					o := gfx.Options{}
					if direct {
						o.Direct = true
					}
					c := gfx.NewCompositor(scene, r, host, o)
					c.SetViewport(1400, 900, 2)
					start := gfx.Point{X: float64(benchCols)*52/2 - 700, Y: float64(benchRows)*76/2 - 450}
					c.SetCamera(gfx.Camera{Origin: start, Zoom: z})
					for i := 0; i < 90 && (c.NeedsFrame() || c.Stats().TilesPending > 0); i++ {
						c.Frame(gfx.FrameInfo{Time: time.Duration(i+1) * 16666667, Interval: 16666667, Seq: uint64(i + 1)})
						time.Sleep(2 * time.Millisecond)
					}
					b.ResetTimer()
					cpus := runFrames(c, b.N, 2)
					b.StopTimer()
					b.ReportMetric(float64(pctlDur(cpus, .5))/1e6, "frame-cpu-ms-p50")
					b.ReportMetric(float64(pctlDur(cpus, .99))/1e6, "frame-cpu-ms-p99")
					c.Close()
				})
			}
		})
	}
}
