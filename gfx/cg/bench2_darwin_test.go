//go:build darwin

package cg

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/synth"
)

// tileGrid renders the viewport as 512 px tiles, handing tiles to workers goroutines that
// each own a surface. keep, when set, receives every finished tile.
func tileGrid(r *Renderer, d *synth.Doc, o gfx.Point, dev, px float64, workers int, surfs []*Surface, keep func(i int, s *Surface)) int {
	nx := (int(vpW*px) + tilePx - 1) / tilePx
	ny := (int(vpH*px) + tilePx - 1) / tilePx
	var next atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(s *Surface) {
			defer wg.Done()
			var buf []*gfx.Node
			for {
				i := int(next.Add(1) - 1)
				if i >= nx*ny {
					return
				}
				to := gfx.Point{X: o.X + float64(i%nx*tilePx)/dev, Y: o.Y + float64(i/nx*tilePx)/dev}
				buf, _ = drawRegion(s, d, to, dev, buf)
				if keep != nil {
					keep(i, s)
				}
			}
		}(surfs[w])
	}
	wg.Wait()
	return nx * ny
}

func newSurfs(r *Renderer, n int) []*Surface {
	out := make([]*Surface, n)
	for i := range out {
		out[i] = r.NewSurface(tilePx, tilePx)
	}
	return out
}

func TestParallelTilesMatchSerial(t *testing.T) {
	d := doc(1000)
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true})
	o := gfx.Point{X: 500, Y: 300}
	collect := func(workers int) [][]byte {
		out := make([][]byte, 24)
		var mu sync.Mutex
		surfs := newSurfs(r, workers)
		n := tileGrid(r, d, o, 2, 2, workers, surfs, func(i int, s *Surface) {
			mu.Lock()
			out[i] = append([]byte(nil), s.Pixels()...)
			mu.Unlock()
		})
		if n != 24 {
			t.Fatalf("%d tiles", n)
		}
		return out
	}
	serial, par := collect(1), collect(8)
	for i := range serial {
		if !bytes.Equal(serial[i], par[i]) {
			t.Fatalf("tile %d differs between 1 and 8 goroutines", i)
		}
	}
}

// BenchmarkTilesParallel renders all tiles of one 2x viewport; compare workers against 1 for scaling.
func BenchmarkTilesParallel(b *testing.B) {
	d := doc(10000)
	for _, vn := range []string{"mid", "zoomin"} {
		var v view
		for _, c := range views {
			if c.name == vn {
				v = c
			}
		}
		zoom := v.zoom(d)
		o := v.origin(d, zoom)
		for _, sprite := range []bool{true, false} {
			for _, w := range []int{1, 2, 4, 8, 12, 16} {
				b.Run(fmt.Sprintf("%s/2x/sprite=%v/workers=%d", vn, sprite, w), func(b *testing.B) {
					r := NewRenderer(Options{TextCache: true, ShadowSprite: sprite})
					surfs := newSurfs(r, w)
					tileGrid(r, d, o, zoom*2, 2, w, surfs, nil)
					b.ResetTimer()
					tiles := 0
					for i := 0; i < b.N; i++ {
						tiles = tileGrid(r, d, o, zoom*2, 2, w, surfs, nil)
					}
					b.ReportMetric(float64(tiles), "tiles")
				})
			}
		}
	}
	// the same with live blur shadows and uncached text: contention in CoreText shows here first
	b.Run("zoomin/2x/naive/workers=1", func(b *testing.B) { naiveTiles(b, d, 1) })
	b.Run("zoomin/2x/naive/workers=8", func(b *testing.B) { naiveTiles(b, d, 8) })
}

func naiveTiles(b *testing.B, d *synth.Doc, w int) {
	r := NewRenderer(Options{})
	surfs := newSurfs(r, w)
	o := views[2].origin(d, 1)
	tileGrid(r, d, o, 2, 2, w, surfs, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tileGrid(r, d, o, 2, 2, w, surfs, nil)
	}
}

// BenchmarkCompositeTiles is the CPU fallback compositor: 24 finished tiles drawn into a 2x viewport.
func BenchmarkCompositeTiles(b *testing.B) {
	d := doc(10000)
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true})
	surfs := newSurfs(r, 1)
	imgs := make([]*Image, 24)
	tileGrid(r, d, views[2].origin(d, 1), 2, 2, 1, surfs, func(i int, s *Surface) { imgs[i] = s.Snapshot(tilePx/2, tilePx/2) })
	dst := r.NewSurface(int(vpW*2), int(vpH*2))
	defer dst.Close()
	b.Run("aligned", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for k, im := range imgs {
				dst.Composite(im, float64(k%6*tilePx), float64(k/6*tilePx), tilePx, tilePx)
			}
		}
	})
	b.Run("panned-offset", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for k, im := range imgs {
				dst.Composite(im, float64(k%6*tilePx)-37.5, float64(k/6*tilePx)-11.25, tilePx, tilePx)
			}
		}
	})
	b.Run("integer-offset", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for k, im := range imgs {
				dst.Composite(im, float64(k%6*tilePx-37), float64(k/6*tilePx-11), tilePx, tilePx)
			}
		}
	})
	b.Run("snapshot-512", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			surfs[0].Snapshot(tilePx/2, tilePx/2).Release()
		}
	})
}

var textFont = gfx.Font{Name: "Helvetica Neue", Size: 13}

func textPic(s string) *gfx.Picture {
	rec := gfx.NewRecorder()
	rec.Text(s, 4, 16, textFont, gfx.Color{A: 1})
	return rec.Finish()
}

func BenchmarkText(b *testing.B) {
	const label = "Primary action label"
	pic := textPic(label)
	for _, cached := range []bool{false, true} {
		b.Run(fmt.Sprintf("draw/cached=%v", cached), func(b *testing.B) {
			r := NewRenderer(Options{TextCache: cached})
			s := r.NewSurface(400, 40)
			defer s.Close()
			s.Begin(2, 0, 0, white)
			s.Replay(pic, 0, 0)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Replay(pic, 0, 0)
			}
		})
	}
	b.Run("layout-only", func(b *testing.B) {
		r := NewRenderer(Options{})
		r.font(textFont)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r.Layout(label, textFont).Release()
		}
	})
	b.Run("font-lookup-uncached", func(b *testing.B) {
		NewRenderer(Options{}) // binds the libraries
		for i := 0; i < b.N; i++ {
			name := cfString(textFont.Name)
			f := ctFontCreateWithName(name, textFont.Size, 0)
			cfRelease(f)
			cfRelease(name)
		}
	})
	b.Run("measure", func(b *testing.B) {
		r := NewRenderer(Options{})
		l := r.Layout(label, textFont)
		defer l.Release()
		for i := 0; i < b.N; i++ {
			l.Width()
		}
	})
	// a call into CoreGraphics that does nothing visible: the floor of one purego round trip
	b.Run("call-floor", func(b *testing.B) {
		r := NewRenderer(Options{})
		s := r.NewSurface(8, 8)
		defer s.Close()
		for i := 0; i < b.N; i++ {
			cgLineWidth(s.ctx, 1)
		}
	})
}

// BenchmarkShadow compares a live blur shadow with the cached 9-slice sprite on one box.
func BenchmarkShadow(b *testing.B) {
	for _, blur := range []float64{4, 8, 16, 32} {
		rec := gfx.NewRecorder()
		rec.FillShadowRRect(40, 40, 140, 70, 8, white, gfx.Shadow{DY: 3, Blur: blur, Color: gfx.Color{A: 0.3}})
		pic := rec.Finish()
		for _, px := range []float64{1, 2} {
			for _, sprite := range []bool{false, true} {
				name := "blur"
				if sprite {
					name = "sprite"
				}
				b.Run(fmt.Sprintf("r=%v/%dx/%s", blur, int(px), name), func(b *testing.B) {
					r := NewRenderer(Options{ShadowSprite: sprite})
					s := r.NewSurface(int(260*px), int(190*px))
					defer s.Close()
					s.Begin(px, 0, 0, white)
					s.Replay(pic, 0, 0)
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						s.Replay(pic, 0, 0)
					}
				})
			}
		}
	}
	// a shadow-only box and the plain fill of the same box, to isolate what the shadow adds
	rec := gfx.NewRecorder()
	p := &gfx.Path{}
	p.RoundRect(40, 40, 140, 70, 8)
	rec.FillPath(p, white)
	pic := rec.Finish()
	b.Run("no-shadow-fill", func(b *testing.B) {
		r := NewRenderer(Options{})
		s := r.NewSurface(520, 380)
		defer s.Close()
		s.Begin(2, 0, 0, white)
		for i := 0; i < b.N; i++ {
			s.Replay(pic, 0, 0)
		}
	})
}

// BenchmarkDrag: the CPU work per frame while dragging; the compositor moves the finished sprites.
func BenchmarkDrag(b *testing.B) {
	d := doc(10000)
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true})
	mid := d.Boxes[len(d.Boxes)/2+d.Cols/2]

	b.Run("sprite-raster-2x", func(b *testing.B) {
		bb := mid.Node.Bounds()
		s := r.NewSurface(int(bb.W()*2)+1, int(bb.H()*2)+1)
		defer s.Close()
		for i := 0; i < b.N; i++ {
			s.Begin(2, bb.Min.X, bb.Min.Y, gfx.Color{})
			s.Replay(mid.Node.Picture(), mid.Node.World()[4], mid.Node.World()[5])
			s.Snapshot(bb.W(), bb.H()).Release()
		}
	})

	// 10 links on one box: the 4 it has plus 6 to near neighbours
	mi := len(d.Boxes)/2 + d.Cols/2
	for k, off := range []int{2, 3, -2, -3, d.Cols + 1, -d.Cols - 1} {
		d.Link(mid, d.Boxes[mi+off], k%2 == 0)
	}
	for _, mode := range []string{"viewport-layer", "dirty-layer"} {
		for _, px := range []float64{1, 2} {
			b.Run(fmt.Sprintf("1node-10links/%dx/%s", int(px), mode), func(b *testing.B) {
				s := r.NewSurface(int(vpW*px), int(vpH*px))
				defer s.Close()
				x0, y0 := mid.X, mid.Y
				for i := 0; i < b.N; i++ {
					dx := float64(i % 40)
					dirty := d.MoveBox(mid, x0+dx, y0+dx/2)
					drawActive(s, mid.Conns, dirty, px, mode == "dirty-layer", vpW, vpH)
				}
				d.MoveBox(mid, x0, y0)
			})
		}
	}

	// a 10x10 block of boxes: every incident connector is redrawn once per frame
	block := d.Cols*40 + 40
	var sel []*synth.Box
	set := map[*synth.Conn]bool{}
	var conns []*synth.Conn
	for r0 := 0; r0 < 10; r0++ {
		for c0 := 0; c0 < 10; c0++ {
			bx := d.Boxes[block+r0*d.Cols+c0]
			sel = append(sel, bx)
			for _, c := range bx.Conns {
				if !set[c] {
					set[c] = true
					conns = append(conns, c)
				}
			}
		}
	}
	for _, mode := range []string{"viewport-layer", "dirty-layer"} {
		for _, px := range []float64{1, 2} {
			b.Run(fmt.Sprintf("100nodes/%dx/%s", int(px), mode), func(b *testing.B) {
				s := r.NewSurface(int(vpW*px), int(vpH*px))
				defer s.Close()
				orig := make([]gfx.Point, len(sel))
				for i, bx := range sel {
					orig[i] = gfx.Point{X: bx.X, Y: bx.Y}
				}
				for i := 0; i < b.N; i++ {
					dx := float64(i % 40)
					dirty := gfx.EmptyRect()
					for k, bx := range sel {
						dirty = dirty.Union(d.MoveBox(bx, orig[k].X+dx, orig[k].Y+dx/2))
					}
					drawActive(s, conns, dirty, px, mode == "dirty-layer", vpW, vpH)
				}
				for k, bx := range sel {
					d.MoveBox(bx, orig[k].X, orig[k].Y)
				}
				b.ReportMetric(float64(len(conns)), "links")
			})
		}
	}
}

// drawActive redraws the active layer: all given connectors, either into a viewport-sized
// surface or only into the dirty rectangle (clamped to the viewport).
func drawActive(s *Surface, conns []*synth.Conn, dirty gfx.Rect, px float64, dirtyOnly bool, w, h float64) {
	o := dirty.Min
	if !dirtyOnly {
		// the viewport sits on the first connector's box; all of them are on screen
		o = gfx.Point{X: dirty.Center().X - w/2, Y: dirty.Center().Y - h/2}
	}
	if dirtyOnly {
		s.BeginClipped(px, o.X, o.Y, dirty)
	} else {
		s.Begin(px, o.X, o.Y, gfx.Color{})
	}
	for _, c := range conns {
		s.Replay(c.Node.Picture(), 0, 0)
	}
}
