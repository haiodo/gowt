//go:build darwin && gfxdemo

// Command demo is the windowed measurement of the gfx compositor on the synthetic 10k document:
// first frame, pan with a flying cursor, zoom gesture, drag of 1 node with its links and of 100
// nodes. Frame intervals come from the display link clock.
//
// It opens a window. Run it only with the tag, on a machine where that is allowed:
//
//	go run -tags gfxdemo ./gfx/view/demo
//
// It closes the window by itself after about 14 seconds, prints the numbers to stdout and exits
// with status 1 on any error or if it is still running after 40 seconds.
package main

import (
	"fmt"
	"math"
	"os"
	"slices"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/synth"
	"github.com/haiodo/gowt/gfx/view"
)

type phase struct {
	name      string
	dur       time.Duration
	intervals []float64 // ms between processed frames
	cpu       []float64 // ms the compositor took in Frame
	model     []float64 // ms spent in the scene update of the drag step
	rastered  uint64
	t0        time.Duration
	started   bool
}

func main() {
	time.AfterFunc(40*time.Second, func() { fmt.Println("error: timeout"); os.Exit(1) })
	doc := synth.New(10000, 1)
	var v *view.View
	err := g.Run(func(app *g.App) {
		app.OnError(func(err error) { fmt.Println("error:", err); os.Exit(1) })
		w := app.Window("gfx compositor", view.New(doc.Scene, func(vv *view.View) { v = vv }, g.Grow()))
		w.SetSize(1440, 900)
		w.Show()
		app.After(200*time.Millisecond, func() { run(w, v, doc) })
	})
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func run(w *g.Window, v *view.View, doc *synth.Doc) {
	c := v.Compositor()
	start := gfx.Point{X: float64(doc.Cols)*synth.CellW/2 - 720, Y: float64(doc.Rows)*synth.CellH/2 - 450}
	c.SetCamera(gfx.Camera{Origin: start, Zoom: 1})

	phases := []*phase{
		{name: "first frame", dur: 1200 * time.Millisecond},
		{name: "pan 300 pt/s + cursor flight", dur: 3 * time.Second},
		{name: "zoom gesture 1 -> 0.25 -> 1", dur: 3 * time.Second},
		{name: "settle", dur: 600 * time.Millisecond},
		{name: "drag 1 node, 10 links", dur: 2 * time.Second},
		{name: "drag 100 nodes", dur: 2 * time.Second},
	}
	mid := doc.Boxes[len(doc.Boxes)/2+doc.Cols/2]
	for k, off := range []int{2, 3, -2, -3, doc.Cols + 1, -doc.Cols - 1} {
		doc.Link(mid, doc.Boxes[len(doc.Boxes)/2+doc.Cols/2+off], k%2 == 0)
	}
	var block []*synth.Box
	var orig []gfx.Point
	var lifted []*gfx.Node
	b0 := doc.Cols*40 + 40
	for r := 0; r < 10; r++ {
		for q := 0; q < 10; q++ {
			bx := doc.Boxes[b0+r*doc.Cols+q]
			block = append(block, bx)
			orig = append(orig, gfx.Point{X: bx.X, Y: bx.Y})
		}
	}
	// the block sits elsewhere than the viewport: move the camera there for the last phase
	var cursor *gfx.Sprite
	firstDone := time.Duration(0)
	var clockStart time.Duration
	var last time.Duration
	var cur *phase
	idx := -1
	var base [1]gfx.Point
	var prevCPU time.Duration
	var end time.Duration
	finished := false

	c.OnFrame(func(fi gfx.FrameInfo) {
		if finished {
			return
		}
		if clockStart == 0 {
			clockStart = fi.Time
		}
		t := fi.Time - clockStart
		if cur != nil {
			if cur.started {
				cur.intervals = append(cur.intervals, float64(fi.Time-last)/1e6)
				cur.cpu = append(cur.cpu, float64(prevCPU)/1e6)
			}
			cur.started = true
		}
		last = fi.Time
		prevCPU = c.Stats().FrameTime

		// advance the phase
		if cur == nil || t-cur.t0 >= cur.dur {
			if cur != nil {
				cur.rastered = c.Stats().Rastered - cur.rastered
				switch cur.name[:4] {
				case "drag":
					c.Land(lifted...)
					lifted = lifted[:0]
				}
			}
			idx++
			if idx >= len(phases) {
				finished = true
				end = t
				report(phases, firstDone, c, v, end)
				w.Close()
				return
			}
			cur = phases[idx]
			cur.t0, cur.started = t, false
			cur.rastered = c.Stats().Rastered
			switch idx {
			case 1:
				base[0] = c.Camera().Origin
				cursor = c.NewSprite(cursorPic(), gfx.SpacePinned, gfx.Point{})
				path := &gfx.Path{}
				o := c.Camera().Origin
				path.MoveTo(o.X+100, o.Y+100)
				path.CubicTo(o.X+500, o.Y+50, o.X+900, o.Y+800, o.X+1300, o.Y+700)
				cursor.FlyAlong(path, 3*time.Second, gfx.EaseInOut, true, nil)
			case 2:
				base[0] = c.Camera().Origin
			case 4:
				lifted = append(lifted, mid.Node)
				for _, cn := range mid.Conns {
					lifted = append(lifted, cn.Node)
				}
				c.Lift(lifted...)
				base[0] = gfx.Point{X: mid.X, Y: mid.Y}
			case 5:
				// bring the block into view and lift it with its links
				c.SetCamera(gfx.Camera{Origin: gfx.Point{X: block[0].X - 200, Y: block[0].Y - 100}, Zoom: 1})
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
			}
		}
		el := (t - cur.t0).Seconds()
		switch idx {
		case 0:
			if firstDone == 0 && c.Stats().Rastered > 0 && c.Stats().TilesPending == 0 {
				firstDone = t
			}
		case 1:
			c.SetCamera(gfx.Camera{Origin: gfx.Point{X: base[0].X + 300*el, Y: base[0].Y}, Zoom: 1})
		case 2:
			z := math.Exp(math.Log(0.25) * math.Pow(math.Sin(math.Pi*el/3), 2))
			// zoom about the view centre
			cp := gfx.Point{X: 720, Y: 450}
			sp := gfx.Point{X: base[0].X + cp.X, Y: base[0].Y + cp.Y}
			c.SetCamera(gfx.Camera{Origin: gfx.Point{X: sp.X - cp.X/z, Y: sp.Y - cp.Y/z}, Zoom: z})
		case 4:
			t1 := time.Now()
			dx, dy := 150*math.Sin(el*3), 80*math.Cos(el*3)
			doc.Scene.Update(func(tx *gfx.Tx) { doc.Move(tx, mid, base[0].X+dx, base[0].Y+dy) })
			cur.model = append(cur.model, float64(time.Since(t1))/1e6)
		case 5:
			t1 := time.Now()
			dx, dy := 150*math.Sin(el*3), 80*math.Cos(el*3)
			doc.Scene.Update(func(tx *gfx.Tx) {
				for i, bx := range block {
					doc.Move(tx, bx, orig[i].X+dx, orig[i].Y+dy)
				}
			})
			cur.model = append(cur.model, float64(time.Since(t1))/1e6)
		}
	})
}

func cursorPic() *gfx.Picture {
	r := gfx.NewRecorder()
	var p gfx.Path
	p.MoveTo(0, 0)
	p.LineTo(14, 5)
	p.LineTo(6, 8)
	p.LineTo(5, 16)
	p.Close()
	r.FillPath(&p, gfx.Color{R: 0.9, G: 0.1, B: 0.2, A: 1})
	r.StrokePath(&p, gfx.StrokeStyle{Width: 1.5, Join: gfx.JoinRound}, gfx.Color{R: 1, G: 1, B: 1, A: 1})
	return r.Finish()
}

func pct(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := slices.Clone(v)
	slices.Sort(s)
	return s[int(p*float64(len(s)-1))]
}

func report(ph []*phase, first time.Duration, c *gfx.Compositor, v *view.View, total time.Duration) {
	fmt.Printf("gfx compositor demo, 10k boxes, %d tile workers\n", v.Compositor().Workers())
	fmt.Printf("first frame: all tiles of the view drawn after %v\n", first.Round(time.Millisecond))
	for _, p := range ph[1:] {
		over := 0
		for _, x := range p.intervals {
			if x > 12 {
				over++
			}
		}
		fmt.Printf("%-32s frames %4d  interval ms p50 %5.2f p95 %5.2f p99 %5.2f max %6.2f  (%d over 12 ms)  frame cpu ms p50 %5.2f p99 %5.2f max %5.2f  tiles %d",
			p.name, len(p.intervals), pct(p.intervals, .5), pct(p.intervals, .95), pct(p.intervals, .99), pct(p.intervals, 1), over,
			pct(p.cpu, .5), pct(p.cpu, .99), pct(p.cpu, 1), p.rastered)
		if len(p.model) > 0 {
			fmt.Printf("  model ms p50 %5.2f max %5.2f", pct(p.model, .5), pct(p.model, 1))
		}
		fmt.Println()
	}
	s := c.Stats()
	fmt.Printf("tiles drawn %d, worker time %v\n", s.Rastered, s.RasterTime.Round(time.Millisecond))
}
