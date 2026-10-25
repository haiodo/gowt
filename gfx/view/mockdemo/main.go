//go:build (darwin || linux || windows) && gfxmock

// Command mockdemo is the windowed measurement of the gfx compositor on a canvas of UI mockup
// screens: N screens of 1920 x 1080 on a grid, from the synthetic generator or from a .pen file.
// It runs timed phases (first frame, pan, zoom gesture, drag of a whole screen, drag of an element
// inside a screen), prints frame statistics, and with GFXDEMO_STAY=1 stays open for use.
//
// It opens a window. Run it only on a machine where that is allowed:
//
//	go run -tags gfxmock ./gfx/view/mockdemo -n 50 -src synth
//	GFXDEMO_STAY=1 go run -tags gfxmock ./gfx/view/mockdemo -n 200 -src /path/to/design.pen
//
// Without GFXDEMO_STAY it closes the window itself and exits with status 1 on any error or after
// 60 seconds.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"slices"
	"time"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/offscreen"
	"github.com/haiodo/gowt/gfx/pen"
	"github.com/haiodo/gowt/gfx/synth"
	"github.com/haiodo/gowt/gfx/view"
)

// canvas is the screens on their grid, whatever made them.
type canvas struct {
	scene *gfx.Scene
	roots []*gfx.Node
	cols  int
	rows  int
	nodes int
}

func (c *canvas) pitch() (float64, float64) {
	return synth.ScreenW + synth.ScreenGap, synth.ScreenH + synth.ScreenGap
}

// centre is the scene point in the middle of screen i.
func (c *canvas) centre(i int) gfx.Point {
	px, py := c.pitch()
	return gfx.Point{X: float64(i%c.cols)*px + synth.ScreenW/2, Y: float64(i/c.cols)*py + synth.ScreenH/2}
}

func build(src string, n int) (*canvas, error) {
	if src == "synth" {
		m := synth.NewMock(n, 1)
		c := &canvas{scene: m.Scene, cols: m.Cols, rows: m.Rows, nodes: m.Scene.Len()}
		for _, s := range m.Screens {
			c.roots = append(c.roots, s.Root)
		}
		return c, nil
	}
	r, err := offscreen.New(offscreen.Options{LOD: true})
	if err != nil {
		return nil, err
	}
	d, err := pen.Open(src, pen.Options{Measurer: r})
	if err != nil {
		return nil, err
	}
	list := d.Screens()
	if len(list) == 0 {
		return nil, fmt.Errorf("%s has no screens", src)
	}
	cols, rows := synth.Grid(n, synth.ScreenW, synth.ScreenH)
	c := &canvas{cols: cols, rows: rows}
	px, py := c.pitch()
	c.scene = gfx.NewScene(gfx.R(-synth.ScreenGap, -synth.ScreenGap, float64(cols)*px+synth.ScreenGap, float64(rows)*py+synth.ScreenGap), 256)
	var berr error
	c.scene.Update(func(tx *gfx.Tx) {
		for i := 0; i < n && berr == nil; i++ {
			var sc *pen.Screen
			if sc, berr = d.Build(tx, nil, list[i%len(list)].Index); berr == nil {
				tx.Move(sc.Root, float64(i%cols)*px, float64(i/cols)*py)
				c.roots = append(c.roots, sc.Root)
				c.nodes += sc.Nodes
			}
		}
	})
	if berr != nil {
		return nil, berr
	}
	c.scene.Commit()
	if d.FontFallback() {
		fmt.Println("note: IBM Plex Sans is not installed, text uses Helvetica Neue")
	}
	for _, s := range d.Unsupported() {
		fmt.Println("note: not supported:", s)
	}
	return c, nil
}

type phase struct {
	name      string
	dur       time.Duration
	intervals []float64 // ms between processed frames
	cpu       []float64 // ms the compositor took in Frame
	paint     []float64 // ms the last widget paint took (Linux, CPU compositing), empty on macOS
	model     []float64 // ms of the scene update of the drag step
	rastered  uint64
	t0        time.Duration
	started   bool
}

// A self-contained build (tag gfxmockpen, embed_pen.go) starts on its embedded design and stays open.
var (
	defaultSrc  = "synth"
	defaultN    = 50
	defaultStay = false
)

func main() {
	n := flag.Int("n", defaultN, "number of screens")
	src := flag.String("src", defaultSrc, `"synth", or the path of a .pen file`)
	fonts := flag.String("fonts", "", "directory with IBM Plex Sans .otf or .ttf files, registered for this process only")
	flag.Parse()
	if *fonts != "" {
		if _, err := offscreen.RegisterFonts(*fonts); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
	}
	if *n < 1 {
		fmt.Println("error: -n must be at least 1")
		os.Exit(1)
	}
	stay := defaultStay || os.Getenv("GFXDEMO_STAY") != ""
	if !stay {
		time.AfterFunc(60*time.Second, func() { fmt.Println("error: timeout"); os.Exit(1) })
	}
	t0 := time.Now()
	c, err := build(*src, *n)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	fmt.Printf("built %d screens, %d nodes in %v\n", len(c.roots), c.nodes, time.Since(t0).Round(time.Millisecond))
	var v *view.View
	err = g.Run(func(app *g.App) {
		app.OnError(func(err error) { fmt.Println("error:", err); os.Exit(1) })
		w := app.Window("gfx mockups", view.New(c.scene, func(vv *view.View) { v = vv }, g.Grow()))
		w.SetSize(1440, 900)
		w.Show()
		app.After(200*time.Millisecond, func() { run(w, v, c, *n, *src, stay) })
	})
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func camAt(p gfx.Point, zoom float64) gfx.Camera {
	return gfx.Camera{Origin: gfx.Point{X: p.X - 720/zoom, Y: p.Y - 450/zoom}, Zoom: zoom}
}

func subtreeLen(n *gfx.Node) int {
	k := 1
	for _, c := range n.Children() {
		k += subtreeLen(c)
	}
	return k
}

// element picks a frame inside the screen with a few nodes below it, to drag on its own.
func element(root *gfx.Node) *gfx.Node {
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

func run(w *g.Window, v *view.View, c *canvas, n int, src string, stay bool) {
	comp := v.Compositor()
	mi := len(c.roots)/2 + c.cols/2
	mi = min(mi, len(c.roots)-1)
	home := c.centre(mi)
	comp.SetCamera(camAt(home, 1))
	screen := c.roots[mi]
	elem := element(screen)
	if elem == nil {
		fmt.Println("error: no element to drag")
		os.Exit(1)
	}
	phases := []*phase{
		{name: "first frame", dur: 1200 * time.Millisecond},
		{name: "pan 600 pt/s + cursor flight", dur: 3 * time.Second},
		{name: "zoom gesture 1 -> 0.1 -> 1", dur: 3 * time.Second},
		{name: "settle", dur: 800 * time.Millisecond},
		{name: "drag whole screen", dur: 2 * time.Second},
		{name: "drag element in screen", dur: 2 * time.Second},
	}
	var cursor *gfx.Sprite
	var firstDone, clockStart, last, prevCPU, prevPaint time.Duration
	var cur *phase
	idx := -1
	var base gfx.Point
	var dragged *gfx.Node
	finished := false

	comp.OnFrame(func(fi gfx.FrameInfo) {
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
				cur.paint = append(cur.paint, float64(prevPaint)/1e6)
			}
			cur.started = true
		}
		last = fi.Time
		prevCPU = comp.Stats().FrameTime
		prevPaint = v.PaintTime()

		if cur == nil || t-cur.t0 >= cur.dur {
			if cur != nil {
				cur.rastered = comp.Stats().Rastered - cur.rastered
				if dragged != nil {
					comp.Land(dragged)
					dragged = nil
				}
			}
			idx++
			if idx >= len(phases) {
				finished = true
				report(phases, firstDone, comp, n, src)
				if !stay {
					w.Close()
				} else {
					cursor.Remove()
					interactive(v, c)
					comp.OnFrame(nil)
				}
				return
			}
			cur = phases[idx]
			fmt.Println("phase:", cur.name)
			cur.t0, cur.started = t, false
			cur.rastered = comp.Stats().Rastered
			switch idx {
			case 1:
				base = comp.Camera().Origin
				cursor = comp.NewSprite(cursorPic(), gfx.SpacePinned, gfx.Point{})
				path := &gfx.Path{}
				o := base
				path.MoveTo(o.X+100, o.Y+100)
				path.CubicTo(o.X+500, o.Y+50, o.X+900, o.Y+800, o.X+1300, o.Y+700)
				cursor.FlyAlong(path, 3*time.Second, gfx.EaseInOut, true, nil)
			case 4:
				dragged = screen
				base = gfx.Point{X: screen.Transform()[4], Y: screen.Transform()[5]}
				comp.Lift(dragged)
			case 5:
				dragged = elem
				base = gfx.Point{X: elem.Transform()[4], Y: elem.Transform()[5]}
				comp.Lift(dragged)
			}
		}
		el := (t - cur.t0).Seconds()
		switch idx {
		case 0:
			if firstDone == 0 && comp.Stats().Rastered > 0 && comp.Stats().TilesPending == 0 {
				firstDone = t
			}
		case 1:
			comp.SetCamera(gfx.Camera{Origin: gfx.Point{X: base.X + 600*el, Y: base.Y}, Zoom: 1})
		case 2:
			z := math.Exp(math.Log(0.1) * math.Pow(math.Sin(math.Pi*el/3), 2))
			comp.SetCamera(camAt(home, z))
		case 4, 5:
			t1 := time.Now()
			dx, dy := 150*math.Sin(el*3), 80*math.Cos(el*3)
			c.scene.Update(func(tx *gfx.Tx) { tx.Move(dragged, base.X+dx, base.Y+dy) })
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

func report(ph []*phase, first time.Duration, c *gfx.Compositor, n int, src string) {
	fmt.Printf("gfx mockup demo, %d screens from %s, %d tile workers\n", n, src, c.Workers())
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
		if m := pct(p.paint, 1); m > 0 {
			fmt.Printf("  paint ms p50 %5.2f p99 %5.2f max %5.2f", pct(p.paint, .5), pct(p.paint, .99), m)
		}
		if len(p.model) > 0 {
			fmt.Printf("  model ms p50 %5.2f max %5.2f", pct(p.model, .5), pct(p.model, 1))
		}
		fmt.Println()
	}
	s := c.Stats()
	fmt.Printf("tiles drawn %d, worker time %v\n", s.Rastered, s.RasterTime.Round(time.Millisecond))
}
