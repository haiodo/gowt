//go:build darwin && gfxspike

// Command spike is the windowed half of the G-0 measurement: CALayer tiles filled from
// CGImages rasterized by gfx/cg, a pan animation and a flying agent cursor, with frame
// jitter recorded while tiles are being re-rasterized in the background.
//
// It opens a window. Build and run it only with the tag:
//
//	go run -tags gfxspike ./gfx/cg/spike
//
// It quits by itself after about 10 seconds and prints the numbers to stdout.
package main

import (
	"fmt"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/purego"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
	"github.com/haiodo/gowt/gfx/synth"
	"github.com/haiodo/gowt/internal/cocoa"
)

const (
	scale  = 2.0 // backing scale of the Retina screen this is meant for
	tilePx = 512
	tile   = tilePx / scale // points
	cols   = 9              // 1440 pt viewport plus a tile of margin on each side
	rows   = 6
	tick   = 8 * time.Millisecond
)

type pt struct{ X, Y float64 }
type rect struct{ X, Y, W, H float64 }

var (
	msgRect  func(recv, sel uintptr, r rect)
	msgPoint func(recv, sel uintptr, p pt)
	msgFloat func(recv, sel uintptr, f float64)
)

func sel(name string) uintptr { return uintptr(cocoa.OSSel_registerName(name)) }
func cls(name string) uintptr { return uintptr(cocoa.OSObjc_lookUpClass(name)) }
func send(recv uintptr, name string, args ...uintptr) uintptr {
	a := make([]int64, len(args))
	for i, v := range args {
		a[i] = int64(v)
	}
	return uintptr(cocoa.Msg(int64(recv), name, a...))
}

func newLayer() uintptr { return send(send(cls("CALayer"), "alloc"), "init") }

func main() {
	purego.RegisterLibFunc(&msgRect, purego.RTLD_DEFAULT, "objc_msgSend")
	purego.RegisterLibFunc(&msgPoint, purego.RTLD_DEFAULT, "objc_msgSend")
	purego.RegisterLibFunc(&msgFloat, purego.RTLD_DEFAULT, "objc_msgSend")

	doc := synth.New(10000, 1)
	r := cg.NewRenderer(cg.Options{TextCache: true, ShadowSprite: true, LOD: true})
	// the viewport starts at the middle of the document; the tile grid is one tile larger on every side
	origin := gfx.Point{X: float64(doc.Cols)*synth.CellW/2 - 720 - tile, Y: float64(doc.Rows)*synth.CellH/2 - 450 - tile}

	var cv *g.CanvasView
	err := g.Run(func(app *g.App) {
		w := app.Window("gfx spike", g.Canvas(nil, g.Grow()), g.Ref(&cv))
		w.SetSize(1440, 900)
		w.Show()
		app.After(300*time.Millisecond, func() { run(app, w, cv, doc, r, origin) })
	})
	if err != nil {
		fmt.Println("error:", err)
	}
}

func run(app *g.App, w *g.Window, cv *g.CanvasView, doc *synth.Doc, r *cg.Renderer, origin gfx.Point) {
	view := uintptr(cv.Unwrap().View.Id)
	send(view, "setWantsLayer:", 1)
	root := send(view, "layer")
	container := newLayer()
	msgPoint(container, sel("setAnchorPoint:"), pt{0, 0})
	send(root, "addSublayer:", container)

	tiles := make([]uintptr, cols*rows)
	for i := range tiles {
		l := newLayer()
		msgRect(l, sel("setFrame:"), rect{float64(i%cols) * tile, float64(i/cols) * tile, tile, tile})
		msgFloat(l, sel("setContentsScale:"), scale)
		send(container, "addSublayer:", l)
		tiles[i] = l
	}

	// render renders the given tiles on workers goroutines and sets them as layer contents
	render := func(ids []int, workers int) {
		var next atomic.Int64
		var wg sync.WaitGroup
		imgs := make([]*cg.Image, len(ids))
		for k := 0; k < workers; k++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				s := r.NewSurface(tilePx, tilePx)
				defer s.Close()
				for {
					j := int(next.Add(1) - 1)
					if j >= len(ids) {
						return
					}
					i := ids[j]
					o := gfx.Point{X: origin.X + float64(i%cols)*tile, Y: origin.Y + float64(i/cols)*tile}
					s.Begin(scale, o.X, o.Y, gfx.Color{R: 1, G: 1, B: 1, A: 1})
					for _, n := range doc.Scene.Query(gfx.Rect{Min: o, Max: gfx.Point{X: o.X + tile, Y: o.Y + tile}}, nil) {
						s.Replay(n.Pic, n.X, n.Y)
					}
					imgs[j] = s.Snapshot(tile, tile)
				}
			}()
		}
		wg.Wait()
		app.Sync(func() {
			cls := cls("CATransaction")
			send(cls, "begin")
			send(cls, "setDisableActions:", 1)
			for j, i := range ids {
				send(tiles[i], "setContents:", imgs[j].Ref())
			}
			send(cls, "commit")
		})
	}

	all := make([]int, len(tiles))
	for i := range all {
		all[i] = i
	}
	t0 := time.Now()
	render(all, runtime.NumCPU())
	fmt.Printf("first frame: %d tiles of %dx%d px on %d goroutines: %v\n", len(all), tilePx, tilePx, runtime.NumCPU(), time.Since(t0))

	// the agent cursor flies along a curve over the document while the container pans
	cursor := newLayer()
	msgRect(cursor, sel("setFrame:"), rect{0, 0, 14, 14})
	send(cursor, "setBackgroundColor:", cg.ColorRef(gfx.Color{R: 1, A: 1}))
	send(root, "addSublayer:", cursor)
	path := &gfx.Path{}
	path.MoveTo(100, 100)
	path.CubicTo(500, 50, 900, 800, 1300, 700)
	meas := path.Measure(0.05)

	var (
		phase      atomic.Int32 // 0: animation only, 1: animation while tiles re-render
		start      = time.Now()
		last       = start
		mu         sync.Mutex
		gaps, cost [2][]float64 // milliseconds per phase
	)
	stop := app.Every(tick, func() {
		now := time.Now()
		el := now.Sub(start).Seconds()
		ph := phase.Load()
		x := -tile + 40*el // pan 40 pt/s keeps the grid margin inside the window for 10 s
		cls := cls("CATransaction")
		send(cls, "begin")
		send(cls, "setDisableActions:", 1)
		msgPoint(container, sel("setPosition:"), pt{x, -tile})
		c, _ := meas.At(meas.Length() * float64(int(el*1000)%3000) / 3000)
		msgPoint(cursor, sel("setPosition:"), pt{c.X, c.Y})
		send(cls, "commit")
		mu.Lock()
		gaps[ph] = append(gaps[ph], now.Sub(last).Seconds()*1000)
		cost[ph] = append(cost[ph], time.Since(now).Seconds()*1000)
		mu.Unlock()
		last = now
	})

	app.After(3*time.Second, func() {
		phase.Store(1)
		go func() {
			// 12 tiles every 250 ms on 4 goroutines, like the strip exposed by a fast pan
			for i := 0; i < 20; i++ {
				ids := make([]int, 0, 12)
				for k := 0; k < 12; k++ {
					ids = append(ids, (i*7+k)%len(tiles))
				}
				t := time.Now()
				render(ids, 4)
				if d := time.Since(t); d < 250*time.Millisecond {
					time.Sleep(250*time.Millisecond - d)
				}
			}
		}()
	})
	app.After(9*time.Second, func() {
		stop()
		mu.Lock()
		for ph, name := range []string{"animation only", "animation + background tile raster"} {
			fmt.Printf("%s: %d ticks of %v\n", name, len(gaps[ph]), tick)
			fmt.Printf("  interval ms: %s\n", pct(gaps[ph]))
			fmt.Printf("  layer update ms: %s\n", pct(cost[ph]))
		}
		mu.Unlock()
		w.Close()
	})
}

func pct(v []float64) string {
	if len(v) == 0 {
		return "no data"
	}
	s := slices.Clone(v)
	slices.Sort(s)
	at := func(p float64) float64 { return s[int(p*float64(len(s)-1))] }
	over := 0
	for _, x := range s {
		if x > 12 {
			over++
		}
	}
	return fmt.Sprintf("p50 %.2f p95 %.2f p99 %.2f max %.2f, %d over 12 ms", at(0.5), at(0.95), at(0.99), s[len(s)-1], over)
}
