//go:build darwin

package cg

import (
	"fmt"
	"math"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/synth"
)

func pct(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := slices.Clone(v)
	slices.Sort(s)
	return s[int(p*float64(len(s)-1))]
}

// TestCompositorHeadless runs the compositor of the demo with the real CoreGraphics renderer, the
// CALayer host on a bare layer and a real display link, but no window: what it measures is our own
// work per frame, not the system compositor. GFX_PHASE_SECONDS sets the length of each phase
// (default 1); the numbers go to the test log.
func TestCompositorHeadless(t *testing.T) {
	if testing.Short() {
		t.Skip("takes several seconds")
	}
	secs := 1.0
	if s := os.Getenv("GFX_PHASE_SECONDS"); s != "" {
		fmt.Sscan(s, &secs)
	}
	phaseDur := time.Duration(secs * float64(time.Second))

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
	if err := dl.Start(); err != nil {
		t.Skip(err)
	}

	mid := doc.Boxes[len(doc.Boxes)/2+doc.Cols/2]
	mi := len(doc.Boxes)/2 + doc.Cols/2
	for k, off := range []int{2, 3, -2, -3, doc.Cols + 1, -doc.Cols - 1} {
		doc.Link(mid, doc.Boxes[mi+off], k%2 == 0)
	}
	var block []*synth.Box
	var orig []gfx.Point
	for q := 0; q < 100; q++ {
		bx := doc.Boxes[doc.Cols*40+40+q/10*doc.Cols+q%10]
		block = append(block, bx)
		orig = append(orig, gfx.Point{X: bx.X, Y: bx.Y})
	}

	type stat struct {
		name           string
		intervals, cpu []float64
		model          []float64
		setup          float64 // ms the setup took, a Lift in the drag phases
		tiles          uint64
	}
	run := func(name string, d time.Duration, setup func(), step func(el float64, st *stat), end func()) *stat {
		st := &stat{name: name}
		if setup != nil {
			ts := time.Now()
			setup()
			st.setup = float64(time.Since(ts)) / 1e6
		}
		t0 := time.Now()
		tiles0 := c.Stats().Rastered
		var last time.Duration
		var lastCPU time.Duration
		for time.Since(t0) < d {
			fi := <-frames
			if last != 0 {
				st.intervals = append(st.intervals, float64(fi.Time-last)/1e6)
				st.cpu = append(st.cpu, float64(lastCPU)/1e6)
			}
			last = fi.Time
			step(time.Since(t0).Seconds(), st)
			c.Frame(fi)
			lastCPU = c.Stats().FrameTime
		}
		if end != nil {
			end()
		}
		st.tiles = c.Stats().Rastered - tiles0
		return st
	}

	// first frame: from the first Frame until every tile in view has an image
	t0 := time.Now()
	for c.Stats().Rastered == 0 || c.Stats().TilesPending > 0 || c.NeedsFrame() {
		c.Frame(<-frames)
		if time.Since(t0) > 10*time.Second {
			t.Fatal("first frame did not finish")
		}
	}
	first := time.Since(t0)

	var cursor *gfx.Sprite
	var stats []*stat
	stats = append(stats, run("pan 300 pt/s + cursor", phaseDur, func() {
		cursor = c.NewSprite(gfx.NewRecorder().Finish(), gfx.SpacePinned, gfx.Point{})
		p := &gfx.Path{}
		p.MoveTo(start.X+100, start.Y+100)
		p.CubicTo(start.X+500, start.Y+50, start.X+900, start.Y+800, start.X+1300, start.Y+700)
		cursor.FlyAlong(p, phaseDur, gfx.EaseInOut, true, nil)
	}, func(el float64, _ *stat) {
		c.SetCamera(gfx.Camera{Origin: gfx.Point{X: start.X + 300*el, Y: start.Y}, Zoom: 1})
	}, func() { cursor.Remove() }))

	pan := c.Camera().Origin
	stats = append(stats, run("zoom 1 -> 0.25 -> 1", phaseDur, nil, func(el float64, _ *stat) {
		z := math.Exp(math.Log(0.25) * math.Pow(math.Sin(math.Pi*el/phaseDur.Seconds()), 2))
		cp := gfx.Point{X: 720, Y: 450}
		sp := gfx.Point{X: pan.X + cp.X, Y: pan.Y + cp.Y}
		c.SetCamera(gfx.Camera{Origin: gfx.Point{X: sp.X - cp.X/z, Y: sp.Y - cp.Y/z}, Zoom: z})
	}, nil))

	// settle at zoom 1 and bring the dragged box into view
	c.SetCamera(gfx.Camera{Origin: gfx.Point{X: mid.X - 600, Y: mid.Y - 400}, Zoom: 1})
	run("settle", 800*time.Millisecond, nil, func(float64, *stat) {}, nil)

	var lifted []*gfx.Node
	x0, y0 := mid.X, mid.Y
	stats = append(stats, run("drag 1 node, 10 links", phaseDur, func() {
		lifted = append(lifted, mid.Node)
		for _, cn := range mid.Conns {
			lifted = append(lifted, cn.Node)
		}
		c.Lift(lifted...)
	}, func(el float64, st *stat) {
		t1 := time.Now()
		dx, dy := 150*math.Sin(el*3), 80*math.Cos(el*3)
		doc.Scene.Update(func(tx *gfx.Tx) { doc.Move(tx, mid, x0+dx, y0+dy) })
		st.model = append(st.model, float64(time.Since(t1))/1e6)
	}, func() { c.Land(lifted...) }))
	run("settle", 500*time.Millisecond, nil, func(float64, *stat) {}, nil)

	c.SetCamera(gfx.Camera{Origin: gfx.Point{X: block[0].X - 200, Y: block[0].Y - 100}, Zoom: 1})
	run("settle", 800*time.Millisecond, nil, func(float64, *stat) {}, nil)
	lifted = lifted[:0]
	stats = append(stats, run("drag 100 nodes", phaseDur, func() {
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
		c.Lift(lifted...)
	}, func(el float64, st *stat) {
		t1 := time.Now()
		dx, dy := 150*math.Sin(el*3), 80*math.Cos(el*3)
		doc.Scene.Update(func(tx *gfx.Tx) {
			for i, bx := range block {
				doc.Move(tx, bx, orig[i].X+dx, orig[i].Y+dy)
			}
		})
		st.model = append(st.model, float64(time.Since(t1))/1e6)
	}, func() { c.Land(lifted...) }))

	t.Logf("%d tile workers; first frame (all tiles in view and ring drawn): %v", c.Workers(), first.Round(time.Millisecond))
	for _, s := range stats {
		over := 0
		for _, x := range s.intervals {
			if x > 12 {
				over++
			}
		}
		line := fmt.Sprintf("%-24s frames %4d interval ms p50 %5.2f p99 %5.2f max %6.2f (%d over 12 ms) | frame cpu ms p50 %5.2f p99 %5.2f max %5.2f | tiles %d",
			s.name, len(s.intervals), pct(s.intervals, .5), pct(s.intervals, .99), pct(s.intervals, 1), over, pct(s.cpu, .5), pct(s.cpu, .99), pct(s.cpu, 1), s.tiles)
		if len(s.model) > 0 {
			line += fmt.Sprintf(" | model ms p50 %5.2f max %5.2f | setup (Lift) ms %5.2f", pct(s.model, .5), pct(s.model, 1), s.setup)
		}
		t.Log(line)
		if len(s.intervals) < 5 {
			t.Errorf("%s: only %d frames", s.name, len(s.intervals))
		}
	}
}

// TestNewLayerOpacity: a fresh CALayer is opaque, so SetOpacity(0) on it must reach the layer.
func TestNewLayerOpacity(t *testing.T) {
	initHost()
	root := send0(send0(clsLayer, sels.alloc), sels.init)
	host := NewHost(root, 2)
	defer host.Close()
	l := host.NewLayer(gfx.GroupOverlay)
	l.SetOpacity(0)
	if op := msgF32Ret(l.(*layer).ref, sels.opacity); op != 0 {
		t.Errorf("opacity %v after SetOpacity(0), want 0", op)
	}
}
