//go:build darwin

package cg

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
)

// recTarget wraps a gfx.Target and records every drawn pass: scale, origin and the pixels.
type recTarget struct {
	gfx.Target
	r      *recRenderer
	scale  float64
	origin gfx.Point
}

func (t *recTarget) Begin(scale float64, origin gfx.Point, bg gfx.Color) {
	t.scale, t.origin = scale, origin
	t.Target.Begin(scale, origin, bg)
}

func (t *recTarget) Draw(items []gfx.Item) {
	t.Target.Draw(items)
	w, h := t.Target.Size()
	px := t.Target.Pixels()
	buf := make([]byte, len(px))
	copy(buf, px)
	t.r.mu.Lock()
	t.r.draws = append(t.r.draws, recDraw{scale: t.scale, origin: t.origin, w: w, h: h, px: buf})
	t.r.mu.Unlock()
}

type recDraw struct {
	scale  float64
	origin gfx.Point
	w, h   int
	px     []byte
}

// recRenderer records every target draw; wraps a real renderer.
type recRenderer struct {
	gfx.Renderer
	mu    sync.Mutex
	draws []recDraw
}

func (r *recRenderer) NewTarget(w, h int) gfx.Target {
	return &recTarget{Target: r.Renderer.NewTarget(w, h), r: r}
}

func (r *recRenderer) last() recDraw {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.draws[len(r.draws)-1]
}

// stubLayer is a gfx.Layer that remembers nothing but its removals.
type stubLayer struct{ removed bool }

func (l *stubLayer) SetImage(gfx.Image)                  {}
func (l *stubLayer) Set(gfx.Rect, gfx.Point, gfx.Affine) {}
func (l *stubLayer) SetOpacity(float64)                  {}
func (l *stubLayer) SetHidden(bool)                      {}
func (l *stubLayer) Remove()                             { l.removed = true }

type stubHost struct{}

func (stubHost) NewLayer(gfx.Group) gfx.Layer { return &stubLayer{} }
func (stubHost) Begin()                       {}
func (stubHost) Commit()                      {}

// directScene is a background with one shadowed card: the shadow exercises the blur the mode
// exists for. The camera origin is fractional, so an off-scale rasterization cannot sneak through
// whole-pixel luck.
func directScene() *gfx.Scene {
	s := gfx.NewScene()
	bg := gfx.NewNode(1)
	bgr := gfx.NewRecorder()
	bgr.FillRect(gfx.R(-1000, -1000, 4000, 3000), gfx.Color{R: 1, G: 1, B: 1, A: 1})
	card := gfx.NewNode(2)
	rec := gfx.NewRecorder()
	rec.FillShadowRRect(0, 0, 200, 120, 10, gfx.RGB(30, 120, 220), gfx.Shadow{DX: 0, DY: 6, Blur: 12, Color: gfx.Color{A: 0.4}})
	s.Update(func(tx *gfx.Tx) {
		tx.Add(nil, bg)
		tx.SetPicture(bg, bgr.Finish())
		tx.Add(nil, card)
		tx.SetPicture(card, rec.Finish())
		tx.Move(card, 300, 200)
	})
	s.Commit()
	return s
}

func directFrames(c *gfx.Compositor, n int) {
	for i := 0; i < n; i++ {
		c.Frame(gfx.FrameInfo{Time: time.Duration(i+1) * 16666667, Interval: 16666667, Seq: uint64(i + 1)})
	}
}

// TestDirectExactScalePixels pins the acceptance of the task: in Direct mode the visible area is
// drawn in one pass at exactly zoom*dpr, pixel-identical to rendering that area at that scale
// with no compositor at all; in the tile mode the same frame is drawn at a power of two instead.
func TestDirectExactScalePixels(t *testing.T) {
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true})
	scene := directScene()
	const vw, vh, dpr = 1400.0, 900.0, 2.0
	zooms := []float64{1, 1.3, 2, 3, 4}

	for _, direct := range []bool{true, false} {
		for _, z := range zooms {
			rr := &recRenderer{Renderer: r}
			o := gfx.Options{}
			if direct {
				o.Direct = true
			}
			c := gfx.NewCompositor(scene, rr, stubHost{}, o)
			c.SetViewport(vw, vh, dpr)
			c.SetCamera(gfx.Camera{Origin: gfx.Point{X: -13, Y: 7}, Zoom: z})
			for i := 0; i < 900 && (c.NeedsFrame() || c.Stats().TilesPending > 0); i++ {
				directFrames(c, 1)
				time.Sleep(2 * time.Millisecond)
			}
			if len(rr.draws) == 0 {
				t.Fatalf("direct=%v zoom %v: nothing drawn", direct, z)
			}
			d := rr.last()
			if direct {
				if want := z * dpr; d.scale != want {
					t.Fatalf("direct zoom %v: drawn at %v, want exactly %v", z, d.scale, want)
				}
				view := gfx.R(-13, 7, vw/z, vh/z)
				ref, err := gfx.RenderSubtree(r, scene, nil, view, z*dpr, gfx.Color{})
				if err != nil {
					t.Fatal(err)
				}
				rpx := ref.Pix
				got := unpremultiplyBGRA(d.px)
				if d.w != int(vw*dpr) || d.h != int(vh*dpr) {
					t.Fatalf("direct zoom %v: %dx%d, want %dx%d", z, d.w, d.h, int(vw*dpr), int(vh*dpr))
				}
				if !bytes.Equal(got, rpx) {
					t.Fatalf("direct zoom %v: %d bytes differ from the exact-scale reference", z, countDiff(got, rpx))
				}
			} else {
				snap := mathExp2Ceil(z * dpr)
				if d.scale != snap {
					t.Fatalf("tile zoom %v: drawn at %v, want the snapped %v", z, d.scale, snap)
				}
				if z != 1 && z != 2 && z != 4 && d.scale == z*dpr {
					t.Fatalf("tile zoom %v: not snapped", z)
				}
			}
			c.Close()
		}
	}
}

// unpremultiplyBGRA converts the premultiplied BGRA of Target.Pixels to the straight RGBA of
// gfx.Raster, the same conversion gfx.FromBGRA does.
func unpremultiplyBGRA(pix []byte) []byte {
	out := make([]byte, len(pix))
	for i := 0; i+3 < len(pix); i += 4 {
		b, g, r, a := pix[i], pix[i+1], pix[i+2], pix[i+3]
		if a != 0 && a != 255 {
			r = byte(min(int(r)*255/int(a), 255))
			g = byte(min(int(g)*255/int(a), 255))
			b = byte(min(int(b)*255/int(a), 255))
		}
		out[i], out[i+1], out[i+2], out[i+3] = r, g, b, a
	}
	return out
}

func countDiff(a, b []byte) int {
	n := 0
	for i := range a {
		if i < len(b) && a[i] != b[i] {
			n++
		}
	}
	return n
}

func mathExp2Ceil(x float64) float64 {
	e := 1.0
	for e < x/1.0001 {
		e *= 2
	}
	return e
}

// TestFillShadowPathSpritePixels: a FillShadowPath whose path is a rounded rectangle takes the
// same cached-sprite path as FillShadowRRect - the two render byte-identically.
func TestFillShadowPathSpritePixels(t *testing.T) {
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true})
	build := func(rec func(*gfx.Recorder)) *gfx.Picture {
		recr := gfx.NewRecorder()
		rec(recr)
		return recr.Finish()
	}
	shadow := gfx.Shadow{DX: 1.5, DY: 3, Blur: 6, Color: gfx.Color{A: 0.35}}
	draw := func(pic *gfx.Picture) []byte {
		tg := r.NewTarget(300, 200)
		defer tg.Close()
		tg.Begin(2.6, gfx.Point{X: -3.3, Y: 7.7}, gfx.Color{})
		tg.Draw([]gfx.Item{{Pic: pic, M: gfx.Identity(), Alpha: 1}})
		return tg.Pixels()
	}
	pathPic := build(func(rec *gfx.Recorder) {
		p := &gfx.Path{}
		p.RoundRect(10, 8, 200, 120, 12)
		rec.FillShadowPath(p, gfx.RGB(30, 120, 220), shadow)
	})
	boxPic := build(func(rec *gfx.Recorder) {
		rec.FillShadowRRect(10, 8, 200, 120, 12, gfx.RGB(30, 120, 220), shadow)
	})
	a, b := draw(pathPic), draw(boxPic)
	if !bytes.Equal(a, b) {
		t.Fatalf("FillShadowPath(rrect) and FillShadowRRect differ in %d bytes", countDiff(a, b))
	}
}
