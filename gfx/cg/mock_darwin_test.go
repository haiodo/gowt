//go:build darwin

package cg

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/pen"
	"github.com/haiodo/gowt/gfx/synth"
)

// Headless benchmarks of a canvas of mockup screens. MOCK_BENCH=1 enables the long report test;
// PEN_FILE names a .pen file whose screens are used as a second source (skipped without it).

// mockCanvas is N screens on a grid, from the synthetic generator or from a .pen file.
type mockCanvas struct {
	scene    *gfx.Scene
	roots    []*gfx.Node
	cols     int
	rows     int
	nodes    int
	texts    int
	measures int
	build    time.Duration
	heap     int64 // bytes the build left on the Go heap
}

func canvasSynth(n int) *mockCanvas {
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	t0 := time.Now()
	m := synth.NewMock(n, 1)
	el := time.Since(t0)
	runtime.GC()
	runtime.ReadMemStats(&m1)
	c := &mockCanvas{scene: m.Scene, cols: m.Cols, rows: m.Rows, build: el, heap: int64(m1.HeapAlloc) - int64(m0.HeapAlloc), nodes: m.Scene.Len()}
	for _, s := range m.Screens {
		c.roots = append(c.roots, s.Root)
		c.texts += s.Texts
	}
	return c
}

func canvasPen(tb testing.TB, r *Renderer, path string, n int) *mockCanvas {
	d, err := pen.Open(path, pen.Options{Measurer: r})
	if err != nil {
		tb.Fatal(err)
	}
	list := d.Screens()
	if len(list) == 0 {
		tb.Fatal("no screens in the file")
	}
	cols, rows := synth.Grid(n, synth.ScreenW, synth.ScreenH)
	px, py := synth.ScreenW+synth.ScreenGap, synth.ScreenH+synth.ScreenGap
	c := &mockCanvas{cols: cols, rows: rows}
	c.scene = gfx.NewScene()
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	t0 := time.Now()
	c.scene.Update(func(tx *gfx.Tx) {
		for i := 0; i < n; i++ {
			sc, err := d.Build(tx, nil, list[i%len(list)].Index)
			if err != nil {
				tb.Fatal(err)
			}
			tx.Move(sc.Root, float64(i%cols)*px, float64(i/cols)*py)
			c.roots = append(c.roots, sc.Root)
			c.nodes += sc.Nodes
			c.texts += sc.Texts
			c.measures += sc.Measures
		}
	})
	c.scene.Commit()
	c.build = time.Since(t0)
	runtime.GC()
	runtime.ReadMemStats(&m1)
	c.heap = int64(m1.HeapAlloc) - int64(m0.HeapAlloc)
	return c
}

// screenAt returns the scene origin of the screen i.
func (c *mockCanvas) screenAt(i int) gfx.Point {
	return gfx.Point{X: float64(i%c.cols) * (synth.ScreenW + synth.ScreenGap), Y: float64(i/c.cols) * (synth.ScreenH + synth.ScreenGap)}
}

// camera centres the view of 1440 x 900 points on the middle of screen i.
func (c *mockCanvas) camera(i int, zoom float64) gfx.Camera {
	o := c.screenAt(i)
	return gfx.Camera{Origin: gfx.Point{X: o.X + synth.ScreenW/2 - 720/zoom, Y: o.Y + synth.ScreenH/2 - 450/zoom}, Zoom: zoom}
}

func (c *mockCanvas) mid() int { return len(c.roots)/2 + c.cols/2 }

type headless struct {
	c    *gfx.Compositor
	host *Host
	fi   gfx.FrameInfo
}

func newHeadless(c *mockCanvas, r *Renderer) *headless {
	initHost()
	root := send0(send0(clsLayer, sels.alloc), sels.init)
	h := &headless{host: NewHost(root, 2)}
	h.c = gfx.NewCompositor(c.scene, r, h.host, gfx.Options{})
	h.c.SetViewport(1440, 900, 2)
	return h
}

func (h *headless) close() { h.c.Close(); h.host.Close() }

// frame runs one compositor frame as if 8 ms (120 Hz) had passed.
func (h *headless) frame() float64 {
	h.fi.Time += 8 * time.Millisecond
	h.fi.Seq++
	h.c.Frame(h.fi)
	return float64(h.c.Stats().FrameTime) / 1e6
}

// settle runs frames until every tile is drawn and nothing animates, and returns the wall time.
func (h *headless) settle(tb testing.TB) time.Duration {
	t0 := time.Now()
	for {
		h.frame()
		s := h.c.Stats()
		if s.Rastered > 0 && s.TilesPending == 0 && !h.c.NeedsFrame() {
			return time.Since(t0)
		}
		if time.Since(t0) > 30*time.Second {
			tb.Fatal("compositor did not settle in 30 s")
		}
		time.Sleep(200 * time.Microsecond)
	}
}

func opts() Options { return Options{TextCache: true, ShadowSprite: true, LOD: true} }

// tiles draws the 512 px tiles that cover the view at the zoom, one goroutine, as a compositor
// worker does; it returns the mean time per tile and the mean nodes per tile.
func tileRun(r *Renderer, c *mockCanvas, zoom float64, reps int) (ms, nodes float64) {
	cam := c.camera(c.mid(), zoom)
	scale := zoom * 2
	tw := 512 / scale
	tg := r.NewTarget(512, 512)
	defer tg.Close()
	var items gfx.Items
	x0, y0 := cam.Origin.X, cam.Origin.Y
	nx, ny := int(1440/zoom/tw)+1, int(900/zoom/tw)+1
	count, total := 0, 0
	t0 := time.Now()
	for rep := 0; rep < reps; rep++ {
		for j := 0; j < ny; j++ {
			for i := 0; i < nx; i++ {
				o := gfx.Point{X: x0 + float64(i)*tw, Y: y0 + float64(j)*tw}
				tg.Begin(scale, o, gfx.Color{R: 1, G: 1, B: 1, A: 1})
				c.scene.Collect(gfx.R(o.X, o.Y, tw, tw), &items)
				tg.Draw(items.List)
				count++
				total += len(items.List)
			}
		}
	}
	return float64(time.Since(t0)) / 1e6 / float64(count), float64(total) / float64(count)
}

func pctl(v []float64, p float64) float64 { return pct(v, p) }

func (c *mockCanvas) report(t *testing.T, r *Renderer, label string) {
	n := len(c.roots)
	t.Logf("[%s N=%d] build %.1f ms (%.2f ms/screen), heap %.2f MB (%.0f KB/screen), nodes %d (%.0f/screen), texts %d (%.0f/screen), measures %d",
		label, n, float64(c.build)/1e6, float64(c.build)/1e6/float64(n), float64(c.heap)/1e6, float64(c.heap)/1e3/float64(n),
		c.nodes, float64(c.nodes)/float64(n), c.texts, float64(c.texts)/float64(n), c.measures)

	for _, z := range []float64{1, 0.5, 0.1} {
		tileRun(r, c, z, 1) // warm the font and line caches
		ms, nodes := tileRun(r, c, z, 3)
		t.Logf("[%s N=%d] tile 512px at zoom %.1f (dpr 2): %.2f ms/tile, %.0f items/tile", label, n, z, ms, nodes)
	}

	// first frame at zoom 1 and when the whole canvas is in view
	for _, z := range []float64{1, 0.25} {
		h := newHeadless(c, r)
		h.c.SetCamera(c.camera(c.mid(), z))
		el := h.settle(t)
		s := h.c.Stats()
		t.Logf("[%s N=%d] first frame at zoom %.2f: %.0f ms, %d tiles, tile worker time %.0f ms", label, n, z, float64(el)/1e6, s.Rastered, float64(s.RasterTime)/1e6)
		h.close()
	}

	cam := c.camera(c.mid(), 1)
	hits, t0 := 0, time.Now()
	for i := 0; i < 2000; i++ {
		p := gfx.Point{X: cam.Origin.X + float64(i*37%1440), Y: cam.Origin.Y + float64(i*53%900)}
		if c.scene.HitTest(p, 4) != nil {
			hits++
		}
	}
	t.Logf("[%s N=%d] hit test: %.1f us per call, %d of 2000 hit", label, n, float64(time.Since(t0))/1e3/2000, hits)
	c.navReport(t, r, label)
	for _, z := range []float64{1, 0.5} {
		c.dragReport(t, r, label, z)
	}
}

// navReport pans at 600 points per second and runs a zoom gesture 1 -> 0.1 -> 1, at 8 ms per frame,
// and reports the compositor's own time per frame and how many frames missed their 8 ms.
func (c *mockCanvas) navReport(t *testing.T, r *Renderer, label string) {
	n := len(c.roots)
	h := newHeadless(c, r)
	defer h.close()
	run := func(name string, frames int, cam func(i int) gfx.Camera) {
		var cpu, gap []float64
		tiles0 := h.c.Stats().Rastered
		next := time.Now()
		last := next
		for i := 0; i < frames; i++ {
			h.c.SetCamera(cam(i))
			cpu = append(cpu, h.frame())
			now := time.Now()
			gap = append(gap, float64(now.Sub(last))/1e6)
			last = now
			next = next.Add(8 * time.Millisecond)
			time.Sleep(time.Until(next))
		}
		over := 0
		for _, g := range gap[1:] {
			if g > 12 {
				over++
			}
		}
		t.Logf("[%s N=%d] %s: frame cpu ms p50 %.2f p99 %.2f max %.2f; interval p99 %.1f max %.1f, %d of %d over 12 ms; tiles drawn %d",
			label, n, name, pctl(cpu, .5), pctl(cpu, .99), pctl(cpu, 1), pctl(gap[1:], .99), pctl(gap[1:], 1), over, frames-1, h.c.Stats().Rastered-tiles0)
	}
	base := c.camera(c.mid(), 1)
	h.c.SetCamera(base)
	h.settle(t)
	run("pan 600 pt/s at zoom 1", 250, func(i int) gfx.Camera {
		return gfx.Camera{Origin: gfx.Point{X: base.Origin.X + 600*float64(i)*0.008, Y: base.Origin.Y}, Zoom: 1}
	})
	h.settle(t)
	half := c.camera(c.mid(), 0.5)
	h.c.SetCamera(half)
	h.settle(t)
	run("pan 600 pt/s at zoom 0.5", 250, func(i int) gfx.Camera {
		return gfx.Camera{Origin: gfx.Point{X: half.Origin.X + 600/0.5*float64(i)*0.008, Y: half.Origin.Y}, Zoom: 0.5}
	})
	h.settle(t)
	cen := gfx.Point{X: base.Origin.X + 720, Y: base.Origin.Y + 450}
	run("zoom gesture 1 -> 0.1 -> 1", 250, func(i int) gfx.Camera {
		z := math.Exp(math.Log(0.1) * math.Pow(math.Sin(math.Pi*float64(i)/250), 2))
		return gfx.Camera{Origin: gfx.Point{X: cen.X - 720/z, Y: cen.Y - 450/z}, Zoom: z}
	})
	h.settle(t)
}

func (c *mockCanvas) dragReport(t *testing.T, r *Renderer, label string, zoom float64) {
	n := len(c.roots)
	h := newHeadless(c, r)
	defer h.close()
	h.c.SetCamera(c.camera(c.mid(), zoom))
	h.settle(t)

	steps := func(name string, node *gfx.Node, base gfx.Point) {
		t0 := time.Now()
		h.c.Lift(node)
		lift := float64(time.Since(t0)) / 1e6
		var frames, upd []float64
		draws := 0
		for i := 0; i < 90; i++ {
			dx, dy := 120*float64(i%30)/30, 40*float64(i%30)/30
			tu := time.Now()
			c.scene.Update(func(tx *gfx.Tx) { tx.Move(node, base.X+dx, base.Y+dy) })
			upd = append(upd, float64(time.Since(tu))/1e6)
			frames = append(frames, h.frame())
			draws += h.c.Stats().SpriteDraws
			if i == 4 {
				time.Sleep(50 * time.Millisecond) // let the tiles under the lifted node finish
			}
		}
		h.c.Land(node)
		h.settle(t)
		c.scene.Update(func(tx *gfx.Tx) { tx.Move(node, base.X, base.Y) })
		h.settle(t)
		t.Logf("[%s N=%d zoom %.1f] drag %s (%d nodes): Lift %.2f ms; first frames %.1f %.1f %.1f ms; steady frame ms p50 %.2f p99 %.2f max %.2f; Update ms p50 %.2f max %.2f; sprite redraws %d of 90",
			label, n, zoom, name, subtreeLen(node), lift, frames[0], frames[1], frames[2],
			pctl(frames[10:], .5), pctl(frames[10:], .99), pctl(frames[10:], 1), pctl(upd, .5), pctl(upd, 1), draws)
	}
	root := c.roots[c.mid()]
	steps("whole screen", root, gfx.Point{X: root.Transform()[4], Y: root.Transform()[5]})
	if el := pickElement(root); el != nil {
		steps("element", el, gfx.Point{X: el.Transform()[4], Y: el.Transform()[5]})
	}
}

func subtreeLen(n *gfx.Node) int {
	k := 1
	for _, c := range n.Children() {
		k += subtreeLen(c)
	}
	return k
}

// pickElement finds a frame two levels under the screen with a few nodes below it.
func pickElement(root *gfx.Node) *gfx.Node {
	var best *gfx.Node
	var walk func(n *gfx.Node, d int)
	walk = func(n *gfx.Node, d int) {
		if d >= 2 && d <= 4 && best == nil {
			if k := subtreeLen(n); k >= 4 && k <= 40 && n.Bounds().W() > 80 {
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

func TestMockReport(t *testing.T) {
	if os.Getenv("MOCK_BENCH") == "" {
		t.Skip("MOCK_BENCH not set")
	}
	r := NewRenderer(opts())
	counts := []int{10, 50, 200}
	if s := os.Getenv("MOCK_N"); s != "" {
		counts = nil
		for _, f := range splitComma(s) {
			v, _ := strconv.Atoi(f)
			counts = append(counts, v)
		}
	}
	for _, n := range counts {
		canvasSynth(n).report(t, r, "synth")
	}
	if path := os.Getenv("PEN_FILE"); path != "" {
		for _, n := range counts {
			canvasPen(t, r, path, n).report(t, r, "pen")
		}
	}
}

func splitComma(s string) []string {
	var out []string
	cur := ""
	for _, ch := range s {
		if ch == ',' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(ch)
		}
	}
	return append(out, cur)
}

// TestMockSmoke keeps the harness honest in the regular test run: two synthetic screens settle, a
// whole screen can be lifted and moved, and an element can be picked.
func TestMockSmoke(t *testing.T) {
	r := NewRenderer(opts())
	c := canvasSynth(2)
	h := newHeadless(c, r)
	defer h.close()
	h.c.SetCamera(c.camera(0, 0.5))
	h.settle(t)
	if h.c.Stats().Rastered == 0 {
		t.Fatal("nothing drawn")
	}
	root := c.roots[0]
	if pickElement(root) == nil {
		t.Error("no element to drag")
	}
	h.c.Lift(root)
	c.scene.Update(func(tx *gfx.Tx) { tx.Move(root, 50, 20) })
	h.frame()
	h.c.Land(root)
	h.settle(t)
}

// BenchmarkMockTile is one 512 px tile in the middle of N screens at three zooms.
func BenchmarkMockTile(b *testing.B) {
	r := NewRenderer(opts())
	for _, n := range []int{10, 50, 200} {
		c := canvasSynth(n)
		for _, z := range []float64{1, 0.5, 0.1} {
			b.Run(fmt.Sprintf("synth/N=%d/zoom=%.1f", n, z), func(b *testing.B) {
				tileRun(r, c, z, 1)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					tileRun(r, c, z, 1)
				}
			})
		}
	}
}

// BenchmarkMockBuild is the cost of building N synthetic screens, and of the layout of a .pen
// screen when PEN_FILE is set.
func BenchmarkMockBuild(b *testing.B) {
	r := NewRenderer(opts())
	for _, n := range []int{10, 50} {
		b.Run(fmt.Sprintf("synth/N=%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				synth.NewMock(n, 1)
			}
		})
	}
	if path := os.Getenv("PEN_FILE"); path != "" {
		for _, n := range []int{10, 50} {
			b.Run(fmt.Sprintf("pen/N=%d", n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					canvasPen(b, r, path, n)
				}
			})
		}
	}
}

// BenchmarkTextLayout measures what the layout asks of CoreText: cold lines, then cached ones.
func BenchmarkTextLayout(b *testing.B) {
	var strs []string
	for i := 0; i < 2000; i++ {
		strs = append(strs, fmt.Sprintf("Sample label %d for measuring text %d", i, i*7))
	}
	f := gfx.Font{Name: "HelveticaNeue", Size: 13}
	b.Run("cold", func(b *testing.B) {
		r := NewRenderer(opts())
		for i := 0; i < b.N; i++ {
			r.MeasureText(strs[i%len(strs)]+strconv.Itoa(i), f)
		}
	})
	b.Run("cached", func(b *testing.B) {
		r := NewRenderer(opts())
		for _, s := range strs {
			r.MeasureText(s, f)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r.MeasureText(strs[i%len(strs)], f)
		}
	})
}

// TestPenRender draws the screens of the .pen file named by PEN_FILE into PNGs in PEN_OUT, for a
// look at the loader's fidelity. Skipped without both variables.
func TestPenRender(t *testing.T) {
	path, out := os.Getenv("PEN_FILE"), os.Getenv("PEN_OUT")
	if path == "" || out == "" {
		t.Skip("PEN_FILE and PEN_OUT not set")
	}
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true})
	d, err := pen.Open(path, pen.Options{Measurer: r})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("system font fallback: %v", d.FontFallback())
	for _, in := range d.Screens() {
		s := gfx.NewScene()
		var sc *pen.Screen
		s.Update(func(tx *gfx.Tx) { sc, err = d.Build(tx, nil, in.Index) })
		if err != nil {
			t.Fatal(err)
		}
		s.Commit()
		img, err := gfx.RenderSubtree(r, s, sc.Root, gfx.R(0, 0, in.W, in.H), 1, gfx.Color{R: 1, G: 1, B: 1, A: 1})
		if err != nil {
			t.Fatal(err)
		}
		png, err := img.PNG()
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(out, fmt.Sprintf("%s-%d.png", filepath.Base(path), in.Index))
		if err := os.WriteFile(name, png, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
