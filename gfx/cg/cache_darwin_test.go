//go:build darwin

package cg

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
)

func imagePic(img gfx.Image, w, h float64) *gfx.Picture {
	rec := gfx.NewRecorder()
	rec.DrawImage(img, 0, 0, w, h)
	return rec.Finish()
}

func rssMB(t *testing.T) float64 {
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		t.Fatal(err)
	}
	kb, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	return kb / 1024
}

func heapMB() float64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.HeapAlloc) / (1 << 20)
}

// TestRasterImagesReleased: the CGImage of a Raster goes once the Raster is unreachable, and the
// cache does not keep the Raster reachable itself.
func TestRasterImagesReleased(t *testing.T) {
	r := NewRenderer(Options{})
	s := r.NewSurface(32, 32)
	defer s.Close()
	keep := gfx.NewRaster(16, 16)
	for i := 0; i < 200; i++ {
		s.Begin(1, 0, 0, white)
		s.Replay(imagePic(gfx.NewRaster(16, 16), 16, 16), 0, 0)
		s.Replay(imagePic(keep, 16, 16), 0, 0) // last, so the Surface's iterator holds only keep
	}
	deadline := time.Now().Add(5 * time.Second)
	for r.images.Len() > 1 && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(time.Millisecond)
	}
	if n := r.images.Len(); n != 1 {
		t.Fatalf("%d CGImages left for 1 live Raster", n)
	}
	if _, ok := r.images.Load(keep); !ok {
		t.Fatal("the CGImage of the live Raster was released")
	}
}

// TestTextCachesBounded draws and measures many more distinct strings than a cache generation holds.
func TestTextCachesBounded(t *testing.T) {
	r := NewRenderer(Options{TextCache: true})
	s := r.NewSurface(64, 32)
	defer s.Close()
	s.Begin(1, 0, 0, white)
	for i := 0; i < 3*maxLines; i++ {
		s.Replay(textPic(strconv.Itoa(i)), 0, 0)
	}
	for i := 0; i < 3*maxMetrics; i++ {
		r.MeasureText(strconv.Itoa(i), textFont)
	}
	if n := r.lines.Len(); n > 2*maxLines {
		t.Fatalf("%d CTLines cached, want at most %d", n, 2*maxLines)
	}
	if n := r.metrics.Len(); n > 2*maxMetrics {
		t.Fatalf("%d metrics cached, want at most %d", n, 2*maxMetrics)
	}
	l, _ := r.lines.Load(lineKey{r.font(textFont), strconv.Itoa(3*maxLines - 1), 0})
	if l == nil || l.ref == 0 {
		t.Fatal("the newest line is not cached")
	}
}

// TestCacheEvictionWhileDrawing draws on several goroutines through tiny caches under a busy GC.
// Only a smoke test: with the KeepAlive in text removed it still passed 10 runs.
func TestCacheEvictionWhileDrawing(t *testing.T) {
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true})
	r.lines.Max, r.sprites.Max = 8, 2
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
			}
		}
	}()
	defer close(stop)
	done := make(chan struct{})
	for g := 0; g < 4; g++ {
		go func() {
			defer func() { done <- struct{}{} }()
			s := r.NewSurface(64, 32)
			defer s.Close()
			for i := 0; i < 500; i++ {
				rec := gfx.NewRecorder()
				rec.FillShadowRRect(8, 4, 40, 20, 4, white, gfx.Shadow{Blur: float64(2 + (g+i)%5), Color: gfx.Color{A: 0.3}})
				rec.Text(strconv.Itoa((g*7+i)%40), 4, 16, textFont, gfx.Color{A: 1})
				s.Begin(1, 0, 0, white)
				s.Replay(rec.Finish(), 0, 0)
			}
		}()
	}
	for range 4 {
		<-done
	}
}

// TestTextCacheHitRate pans the 10k document right and back and compares the lines the bounded
// cache lays out with those an unbounded one does.
func TestTextCacheHitRate(t *testing.T) {
	if testing.Short() {
		t.Skip("lays out the 10k document")
	}
	const frames = 40
	d := doc(10000)
	for _, v := range views[1:] {
		zoom := v.zoom(d)
		o := v.origin(d, zoom)
		misses := func(max int) (int64, int) {
			r := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true})
			r.lines.Max = max
			s := r.NewSurface(int(vpW), int(vpH))
			defer s.Close()
			var buf []*gfx.Node
			draws := 0
			for f := 0; f < frames; f++ {
				dx := float64(min(f, frames-f)) * 100 / zoom
				var n int
				buf, n = drawRegion(s, d, gfx.Point{X: o.X + dx, Y: o.Y}, zoom, buf)
				draws += n
			}
			return r.lines.Misses(), draws
		}
		bounded, nodes := misses(maxLines)
		small, _ := misses(256)
		unbounded, _ := misses(1 << 30)
		t.Logf("%s, %d frames, %d node draws: lines laid out %d with a %d-line generation, %d with 256, %d unbounded",
			v.name, frames, nodes, bounded, maxLines, small, unbounded)
		if bounded > unbounded {
			t.Fatalf("%s: the bound costs %d extra layouts", v.name, bounded-unbounded)
		}
	}
}

// TestRasterChurnMemory is the loop of the Tenniarb report: every frame draws a new Raster and
// drops the old one. GFX_CACHE_MEASURE=1 runs it and logs heap and RSS.
func TestRasterChurnMemory(t *testing.T) {
	if os.Getenv("GFX_CACHE_MEASURE") == "" {
		t.Skip("set GFX_CACHE_MEASURE=1")
	}
	const frames, w, h = 50, 1600, 1000
	r := NewRenderer(Options{})
	s := r.NewSurface(400, 250)
	defer s.Close()
	h0, r0 := heapMB(), rssMB(t)
	for i := 0; i < frames; i++ {
		img := gfx.NewRaster(w, h)
		for j := range img.Pix {
			img.Pix[j] = byte(i + j)
		}
		s.Begin(0.25, 0, 0, white)
		s.Replay(imagePic(img, w, h), 0, 0)
	}
	h1, r1 := heapMB(), rssMB(t)
	t.Logf("%d frames of a %dx%d raster: heap %.0f -> %.0f MB, RSS %.0f -> %.0f MB", frames, w, h, h0, h1, r0, r1)
}

// BenchmarkRasterDraw is the cache lookup plus the draw of a small Raster converted earlier.
func BenchmarkRasterDraw(b *testing.B) {
	r := NewRenderer(Options{})
	s := r.NewSurface(16, 16)
	defer s.Close()
	pic := imagePic(gfx.NewRaster(4, 4), 4, 4)
	s.Begin(1, 0, 0, white)
	s.Replay(pic, 0, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Replay(pic, 0, 0)
	}
}
