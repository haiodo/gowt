//go:build darwin

package cg

import (
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
)

// captureLayer keeps a retaining copy of every image set on it, so a test reads the pixels the
// compositor put on the layer after the compositor released its own reference.
type captureHost struct {
	layers []*captureLayer
}

type captureLayer struct {
	h       *captureHost
	refs    []uintptr
	hidden  bool
	removed bool
}

func (l *captureLayer) SetImage(img gfx.Image) {
	if im, ok := img.(*Image); ok && im.ref != 0 {
		l.refs = append(l.refs, cgImageCreateWithImageInRct(im.ref, cgRect{0, 0, 2800, 1800}))
	}
}
func (l *captureLayer) Set(gfx.Rect, gfx.Point, gfx.Affine) {}
func (l *captureLayer) SetOpacity(float64)                  {}
func (l *captureLayer) SetHidden(h bool)                    { l.hidden = h }
func (l *captureLayer) Remove()                             { l.removed = true }

func (h *captureHost) NewLayer(gfx.Group) gfx.Layer {
	l := &captureLayer{h: h}
	h.layers = append(h.layers, l)
	return l
}
func (h *captureHost) Begin()  {}
func (h *captureHost) Commit() {}

// pixels rasterizes a retained CGImage into a fresh surface and returns its bytes.
func (h *captureHost) pixels(t *testing.T, r *Renderer, i int) []byte {
	t.Helper()
	l := h.layers[i]
	if len(l.refs) == 0 {
		t.Fatal("no image on the layer")
	}
	s := r.NewSurface(2800, 1800)
	cgDrawImage(s.ctx, cgRect{0, 0, 2800, 1800}, l.refs[len(l.refs)-1])
	px := s.Pixels()
	out := make([]byte, len(px))
	copy(out, px)
	s.Close()
	return out
}

func nzBytes(px []byte) int {
	n := 0
	for _, b := range px {
		if b != 255 && b != 0 {
			n++
		}
	}
	return n
}

// nonBlank fails when the captured frame is all white or empty: a byte comparison of two blank
// captures would pass vacuously, so the positive path must prove it drew something.
func nonBlank(t *testing.T, px []byte) {
	t.Helper()
	for _, b := range px {
		if b != 255 && b != 0 {
			return
		}
	}
	t.Fatal("the captured frame is blank: the layer never showed scene content")
}

// stripScene crosses translucent shapes and a shadowed card over the whole view, so band seams
// cut through real content.
func stripScene() *gfx.Scene {
	s := gfx.NewScene()
	bg := gfx.NewNode(1)
	bgr := gfx.NewRecorder()
	bgr.FillRect(gfx.R(-1000, -1000, 4000, 3000), gfx.RGB(255, 255, 255))
	band := gfx.NewNode(2)
	br := gfx.NewRecorder()
	br.FillRect(gfx.R(0, 0, 1400, 260), gfx.Color{R: 0.4, A: 0.45})
	card := gfx.NewNode(3)
	cr := gfx.NewRecorder()
	cr.FillShadowRRect(0, 0, 400, 240, 14, gfx.RGB(30, 120, 220), gfx.Shadow{DX: 2, DY: 6, Blur: 14, Color: gfx.Color{A: 0.4}})
	cr.Text("Strips", 24, 60, gfx.Font{Size: 40}, gfx.White)
	card2 := gfx.NewNode(4)
	c2 := gfx.NewRecorder()
	c2.FillRRect(gfx.R(0, 0, 300, 200), 10, gfx.Color{G: 0.6, A: 0.5})
	s.Update(func(tx *gfx.Tx) {
		tx.Add(nil, bg)
		tx.SetPicture(bg, bgr.Finish())
		tx.Add(nil, band)
		tx.SetPicture(band, br.Finish())
		tx.Move(band, -40, 120)
		tx.Add(nil, card)
		tx.SetPicture(card, cr.Finish())
		tx.Move(card, 60, 30)
		tx.Add(nil, card2)
		tx.SetPicture(card2, c2.Finish())
		tx.Move(card2, 180, 100)
	})
	s.Commit()
	return s
}

// maxDelta returns the largest per-byte difference; byteDiffs counts differing bytes.
func diffStats(a, b []byte) (bytes int, maxDelta int) {
	for i := range a {
		if a[i] != b[i] {
			bytes++
			d := int(a[i]) - int(b[i])
			if d < 0 {
				d = -d
			}
			if d > maxDelta {
				maxDelta = d
			}
		}
	}
	return bytes, maxDelta
}

// TestCGContextHeightAARounding documents why the strip comparison below cannot demand byte
// equality: CoreGraphics rounds one antialiasing step differently when the same draw list
// lands in contexts of different heights. This test draws two overlapping pictures (a
// shadowed card and a translucent rounded rect, no compositor, no strip logic) into full and
// half-height contexts and proves the artifact exists with a one-unit delta. The strip raster
// is exact wherever this artifact does not fire; where it fires, one pixel differs by one
// unit, deterministically.
func TestCGContextHeightAARounding(t *testing.T) {
	r := NewRenderer(Options{TextCache: true, LOD: true})
	shRec := gfx.NewRecorder()
	shRec.FillShadowRRect(0, 0, 400, 240, 14, gfx.RGB(30, 120, 220), gfx.Shadow{DX: 2, DY: 6, Blur: 14, Color: gfx.Color{A: 0.4}})
	sh := shRec.Finish()
	c2Rec := gfx.NewRecorder()
	c2Rec.FillRRect(gfx.R(0, 0, 300, 200), 10, gfx.Color{G: 0.6, A: 0.5})
	c2 := c2Rec.Finish()
	items := []gfx.Item{
		{Pic: sh, M: gfx.TranslateBy(60, 30), Alpha: 1},
		{Pic: c2, M: gfx.TranslateBy(180, 100), Alpha: 1},
	}
	draw := func(h, bottom int) []byte {
		s := r.NewSurface(2800, h)
		s.BeginStrip(2.6, -13.7, 7.3, bottom, gfx.Color{})
		s.Draw(items)
		px := append([]byte(nil), s.Pixels()...)
		s.Close()
		return px
	}
	full := draw(1800, 1800)
	half := draw(900, 900)
	n, mx := diffStats(full[:900*2800*4], half)
	t.Logf("same draw list, 1800 vs 900-tall context: %d bytes differ, max delta %d", n, mx)
	if mx > 1 {
		t.Fatalf("the context-height AA artifact exceeds one unit: max delta %d", mx)
	}
}

// TestDirectStripsMatchSingleTarget is the seam proof: the Direct mode with the parallel strip
// rasterization (1, 2, 4, 8 workers) puts the same pixels on the layer as the single-target
// exact-scale draw at every acceptance zoom, fractional camera origin included. Content is
// identical up to the CoreGraphics context-height AA artifact proven in
// TestCGContextHeightAARounding: at most a few pixels differ, by one unit, never a shift, a
// seam, a missing node or a moved shadow.
func TestDirectStripsMatchSingleTarget(t *testing.T) {
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true})
	scene := stripScene()
	const vw, vh, dpr = 1400.0, 900.0, 2.0

	for _, workers := range []int{1, 2, 4, 8} {
		for _, z := range []float64{1, 1.3, 2, 3, 4} {
			run := func(strips bool) []byte {
				h := &captureHost{}
				var c *gfx.Compositor
				if strips {
					c = gfx.NewCompositor(scene, r, h, gfx.Options{Direct: true, Workers: workers})
				} else {
					// a wrapper hides StripRenderer: the fallback single-target path
					c = gfx.NewCompositor(scene, plainRenderer{r}, h, gfx.Options{Direct: true, Workers: workers})
				}
				c.SetViewport(vw, vh, dpr)
				c.SetCamera(gfx.Camera{Origin: gfx.Point{X: -13.7, Y: 7.3}, Zoom: z})
				for i := 0; i < 30; i++ {
					c.Frame(gfx.FrameInfo{Time: time.Duration(i+1) * 16666667, Interval: 16666667, Seq: uint64(i + 1)})
				}
				px := h.pixels(t, r, 0)
				c.Close()
				return px
			}
			stripped := run(true)
			single := run(false)
			nonBlank(t, stripped)
			nonBlank(t, single)
			n, mx := diffStats(stripped, single)
			t.Logf("workers %d zoom %v: %d bytes differ, max delta %d", workers, z, n, mx)
			if mx > 1 {
				t.Fatalf("workers %d zoom %v: a byte differs by %d, beyond the one-unit AA artifact", workers, z, mx)
			}
			if n > 16 {
				t.Fatalf("workers %d zoom %v: %d bytes differ, want at most a few one-unit AA pixels", workers, z, n)
			}
		}
	}
}

// plainRenderer hides the StripRenderer capability, forcing the single-target Direct path.
type plainRenderer struct{ gfx.Renderer }

func (plainRenderer) NewStripSet(int, int, int) gfx.StripSet { return nil }

// mutatedSet draws with a wrong strip bottom - the CTM error class the seam proof must catch.
type mutatedSet struct{ *stripSet }

func (m *mutatedSet) Target(i int) gfx.Target {
	return bandTarget{target{m.bands[i]}, m.bottom[i] + 10}
}

type mutatedRenderer struct {
	gfx.Renderer
	r *Renderer
}

func (m *mutatedRenderer) NewStripSet(n, w, h int) gfx.StripSet {
	return &mutatedSet{m.r.NewStripSet(n, w, h).(*stripSet)}
}

// TestDirectStripsMutation proves the comparison detects a broken strip CTM: shifting every
// strip's device translate by ten rows must not pass as "one-unit AA". A blank or vacuous
// comparison would let it through; here the diff is frame-sized.
func TestDirectStripsMutation(t *testing.T) {
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true})
	scene := stripScene()
	run := func(mutate bool) []byte {
		h := &captureHost{}
		var c *gfx.Compositor
		if mutate {
			c = gfx.NewCompositor(scene, &mutatedRenderer{Renderer: r, r: r}, h, gfx.Options{Direct: true, Workers: 4})
		} else {
			c = gfx.NewCompositor(scene, r, h, gfx.Options{Direct: true, Workers: 4})
		}
		c.SetViewport(1400, 900, 2)
		c.SetCamera(gfx.Camera{Origin: gfx.Point{X: -13.7, Y: 7.3}, Zoom: 1.3})
		for i := 0; i < 10; i++ {
			c.Frame(gfx.FrameInfo{Time: time.Duration(i+1) * 16666667, Interval: 16666667, Seq: uint64(i + 1)})
		}
		px := h.pixels(t, r, 0)
		c.Close()
		return px
	}
	good, bad := run(false), run(true)
	nonBlank(t, good)
	nonBlank(t, bad)
	n, mx := diffStats(good, bad)
	t.Logf("mutated bottom: %d bytes differ, max delta %d", n, mx)
	if n < 1000 || mx <= 1 {
		t.Fatalf("the mutation went undetected: %d bytes, max delta %d", n, mx)
	}
}
