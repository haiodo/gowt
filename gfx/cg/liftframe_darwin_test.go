//go:build darwin

package cg

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
)

// demoElement is the element the windowed mockdemo drags: a frame two to four levels under the screen.
func demoElement(root *gfx.Node) *gfx.Node {
	var best *gfx.Node
	var walk func(n *gfx.Node, d int)
	walk = func(n *gfx.Node, d int) {
		if best == nil && d >= 2 && d <= 4 && n.Bounds().W() > 80 {
			if k := subtreeLen(n); k >= 4 && k <= 60 {
				best = n
				return
			}
		}
		for _, c := range n.Children() {
			walk(c, d+1)
		}
	}
	walk(root, 0)
	return best
}

// liftBound is generous: the stall it guards against was 130 to 170 ms.
const liftBound = 30.0

// liftFrames drags a whole screen and then, in the frame that lands it, an element, the sequence of
// the windowed demo. It fails when the lift call or any frame around it costs more than liftBound
// (the call alone more than 12 ms), or when the node does not reach its sprite.
func liftFrames(t *testing.T, c *mockCanvas) {
	r := NewRenderer(opts())
	h := newHeadless(c, r)
	defer h.close()
	h.c.SetCamera(c.camera(c.mid(), 1))
	h.settle(t)
	screen := c.roots[c.mid()]
	elem := demoElement(screen)
	if elem == nil {
		t.Fatal("no element to drag")
	}
	run := func(name string, node, prev *gfx.Node) {
		base := gfx.Point{X: node.Transform()[4], Y: node.Transform()[5]}
		t0 := time.Now()
		if prev != nil {
			h.c.Land(prev)
		}
		h.c.Lift(node)
		call := float64(time.Since(t0)) / 1e6
		worst, lifted := 0.0, -1
		for i := 0; i < 200; i++ {
			dx, dy := 150*math.Sin(float64(i)*0.024), 80*math.Cos(float64(i)*0.024)
			c.scene.Update(func(tx *gfx.Tx) { tx.Move(node, base.X+dx, base.Y+dy) })
			worst = max(worst, h.frame())
			if lifted < 0 && node.Lifted() {
				lifted = i
			}
			time.Sleep(8 * time.Millisecond)
		}
		t.Logf("%s (%d nodes): Lift call %.2f ms, worst frame %.2f ms, node lifted at frame %d", name, subtreeLen(node), call, worst, lifted)
		if call > 12 {
			t.Errorf("%s: Lift call %.1f ms", name, call)
		}
		if worst > liftBound {
			t.Errorf("%s: a frame took %.1f ms", name, worst)
		}
		if lifted < 0 {
			t.Errorf("%s: the node was never lifted", name)
		}
	}
	run("screen", screen, nil)
	run("element", elem, screen)
	h.c.Land(elem)
	h.settle(t)
}

func TestLiftFrameSynth(t *testing.T) {
	liftFrames(t, canvasSynth(50))
}

// TestPenLiftFrame is the same on the screens of the .pen file named by PEN_FILE.
func TestPenLiftFrame(t *testing.T) {
	path := os.Getenv("PEN_FILE")
	if path == "" {
		t.Skip("PEN_FILE not set")
	}
	liftFrames(t, canvasPen(t, NewRenderer(opts()), path, 200))
}
