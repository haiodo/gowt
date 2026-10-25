//go:build darwin

package cg

import (
	"fmt"
	"sync"
	"testing"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/synth"
)

// Viewport of the budget scenarios, in points.
const vpW, vpH = 1440.0, 900.0

var (
	docMu sync.Mutex
	docs  = map[int]*synth.Doc{}
)

func doc(n int) *synth.Doc {
	docMu.Lock()
	defer docMu.Unlock()
	if d := docs[n]; d != nil {
		return d
	}
	d := synth.New(n, 1)
	docs[n] = d
	return d
}

var configs = []struct {
	name string
	o    Options
}{
	{"naive", Options{}},
	{"cached", Options{TextCache: true, ShadowSprite: true}},
	{"cached+lod", Options{TextCache: true, ShadowSprite: true, LOD: true}},
}

// view is a viewport position: zoom is points per world unit, the center is in world units.
type view struct {
	name string
	zoom func(d *synth.Doc) float64
}

var views = []view{
	{"zoomout", func(d *synth.Doc) float64 {
		w := float64(d.Cols) * synth.CellW
		h := float64(d.Rows) * synth.CellH
		return min(vpW/w, vpH/h)
	}},
	{"mid", func(*synth.Doc) float64 { return 0.35 }},
	{"zoomin", func(*synth.Doc) float64 { return 1 }},
}

// origin returns the world position of the top left corner of the viewport.
func (v view) origin(d *synth.Doc, zoom float64) gfx.Point {
	cx := float64(d.Cols) * synth.CellW / 2
	cy := float64(d.Rows) * synth.CellH / 2
	return gfx.Point{X: cx - vpW/zoom/2, Y: cy - vpH/zoom/2}
}

var white = gfx.Color{R: 1, G: 1, B: 1, A: 1}

// drawRegion renders the world rect at devScale device pixels per world unit.
func drawRegion(s *Surface, d *synth.Doc, o gfx.Point, devScale float64, buf []*gfx.Node) ([]*gfx.Node, int) {
	s.Begin(devScale, o.X, o.Y, white)
	r := gfx.Rect{Min: o, Max: gfx.Point{X: o.X + float64(s.W)/devScale, Y: o.Y + float64(s.H)/devScale}}
	buf = d.Scene.Query(r, buf[:0])
	for _, n := range buf {
		s.Replay(n.Picture(), n.World()[4], n.World()[5])
	}
	return buf, len(buf)
}

func BenchmarkViewport(b *testing.B) {
	for _, n := range []int{1000, 10000, 50000} {
		d := doc(n)
		for _, v := range views {
			zoom := v.zoom(d)
			o := v.origin(d, zoom)
			for _, px := range []float64{1, 2} {
				for _, cfg := range configs {
					b.Run(fmt.Sprintf("%d/%s/%dx/%s", n, v.name, int(px), cfg.name), func(b *testing.B) {
						r := NewRenderer(cfg.o)
						s := r.NewSurface(int(vpW*px), int(vpH*px))
						defer s.Close()
						var buf []*gfx.Node
						cnt := 0
						buf, cnt = drawRegion(s, d, o, zoom*px, buf) // warm caches
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							buf, cnt = drawRegion(s, d, o, zoom*px, buf)
						}
						b.ReportMetric(float64(cnt), "nodes")
					})
				}
			}
		}
	}
}

const tilePx = 512

// BenchmarkTile is one 512x512 device pixel tile at the middle of the 10k document.
func BenchmarkTile(b *testing.B) {
	d := doc(10000)
	for _, v := range views {
		zoom := v.zoom(d)
		o := v.origin(d, zoom)
		for _, px := range []float64{1, 2} {
			for _, cfg := range configs[1:] {
				b.Run(fmt.Sprintf("%s/%dx/%s", v.name, int(px), cfg.name), func(b *testing.B) {
					r := NewRenderer(cfg.o)
					s := r.NewSurface(tilePx, tilePx)
					defer s.Close()
					var buf []*gfx.Node
					cnt := 0
					buf, cnt = drawRegion(s, d, o, zoom*px, buf)
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						buf, cnt = drawRegion(s, d, o, zoom*px, buf)
					}
					b.ReportMetric(float64(cnt), "nodes")
				})
			}
		}
	}
}
