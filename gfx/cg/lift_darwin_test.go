//go:build darwin

package cg

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/synth"
)

// TestCompositorHeadlessLiftFrame is the drag of 100 nodes in the order of the windowed demo: the
// camera jumps and the nodes are lifted one frame later with no settle between. DRAG_SETTLE=1
// waits for the tiles first. Moving boxes rigidly must leave their inner connectors alone, so the
// sprites drawn per frame stay far below the 320 lifted nodes.
func TestCompositorHeadlessLiftFrame(t *testing.T) {
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true})
	doc := synth.New(10000, 1)
	initHost()
	root := send0(send0(clsLayer, sels.alloc), sels.init)
	host := NewHost(root, 2)
	defer host.Close()
	c := gfx.NewCompositor(doc.Scene, r, host, gfx.Options{})
	defer c.Close()
	c.SetViewport(1440, 900, 2)
	start := gfx.Point{X: float64(doc.Cols)*synth.CellW/2 - 720, Y: float64(doc.Rows)*synth.CellH/2 - 450}
	c.SetCamera(gfx.Camera{Origin: start, Zoom: 1})
	frames := make(chan gfx.FrameInfo, 4)
	dl, err := NewDisplayLink(func(fi gfx.FrameInfo) {
		select {
		case frames <- fi:
		default:
		}
	})
	if err != nil {
		t.Skip(err)
	}
	defer dl.Close()
	dl.Start()
	for c.Stats().Rastered == 0 || c.Stats().TilesPending > 0 || c.NeedsFrame() {
		c.Frame(<-frames)
	}
	var block []*synth.Box
	var orig []gfx.Point
	for q := 0; q < 100; q++ {
		bx := doc.Boxes[doc.Cols*40+40+q/10*doc.Cols+q%10]
		block = append(block, bx)
		orig = append(orig, gfx.Point{X: bx.X, Y: bx.Y})
	}
	t0 := time.Now()
	r0 := c.Stats().Rastered
	var lifted []*gfx.Node
	var cpus []float64
	draws := 0
	for i := 0; i < 240; i++ {
		fi := <-frames
		el := time.Since(t0).Seconds()
		if i == 0 {
			c.SetCamera(gfx.Camera{Origin: gfx.Point{X: block[0].X - 200, Y: block[0].Y - 100}, Zoom: 1})
		}
		if i == 1 && os.Getenv("DRAG_SETTLE") != "" {
			for c.Stats().TilesPending > 0 || c.NeedsFrame() {
				c.Frame(<-frames)
			}
			r0 = c.Stats().Rastered
		}
		if i == 1 {
			seen := map[*gfx.Node]bool{}
			for _, bx := range block {
				lifted = append(lifted, bx.Node)
				seen[bx.Node] = true
			}
			for _, bx := range block {
				for _, cn := range bx.Conns {
					if !seen[cn.Node] {
						seen[cn.Node] = true
						lifted = append(lifted, cn.Node)
					}
				}
			}
			t.Logf("lifted %d nodes", len(lifted))
			r0b := c.Stats().Rastered
			t.Logf("tiles before lift (camera only, frame 0 only): %d", r0b-r0)
			r0 = r0b
			tl := time.Now()
			c.Lift(lifted...)
			t.Logf("Lift call %.2f ms", float64(time.Since(tl))/1e6)
		}
		if i >= 2 {
			dx, dy := 150*math.Sin(el*3), 80*math.Cos(el*3)
			doc.Scene.Update(func(tx *gfx.Tx) {
				for i, bx := range block {
					doc.Move(tx, bx, orig[i].X+dx, orig[i].Y+dy)
				}
			})
		}
		c.Frame(fi)
		s := c.Stats()
		cpus = append(cpus, float64(s.FrameTime)/1e6)
		if i >= 20 {
			draws += s.SpriteDraws
		}
		if i > 20 && float64(s.FrameTime)/1e6 > 4 {
			t.Logf("slow frame %d: %.2f ms draws %d tiles %d", i, float64(s.FrameTime)/1e6, s.SpriteDraws, s.TilesApplied)
		}
		if i < 14 || i == 100 {
			t.Logf("frame %3d cpu %6.2f ms spriteDraws %3d tilesApplied %2d rastered(since lift) %d pending %d", i, float64(s.FrameTime)/1e6, s.SpriteDraws, s.TilesApplied, s.Rastered-r0, s.TilesPending)
		}
	}
	t.Logf("tiles since lift: %d; steady cpu p50 %.2f p99 %.2f max %.2f", c.Stats().Rastered-r0, pct(cpus[20:], .5), pct(cpus[20:], .99), pct(cpus[20:], 1))
	if per := draws / (len(cpus) - 20); per > 80 {
		t.Errorf("%d sprites drawn per frame while 100 boxes move together", per)
	}
}
