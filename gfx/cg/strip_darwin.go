//go:build darwin

package cg

import (
	"runtime"
	"sync/atomic"

	"github.com/haiodo/gowt/gfx"
)

// stripSet is one viewport bitmap drawn as parallel horizontal strips: the full-size bitmap
// context owns the memory, and a band context views each strip's rows, so the strips rasterize
// concurrently into disjoint memory. The band Surfaces are never drawn into from two
// goroutines at once and are not used after Close.
type stripSet struct {
	r      *Renderer
	w, h   int     // pixels of the whole bitmap
	full   uintptr // the context that owns the bitmap memory
	data   uintptr // CGBitmapContextGetData(full)
	bands  []*Surface
	bottom []int        // absolute bottom row of each strip in the shared bitmap
	imgs   atomic.Int32 // shared images over the bitmap still alive; Close waits for them
	dead   bool
}

// NewStripSet implements gfx.StripRenderer. Strip i covers rows [i*hpx/n, (i+1)*hpx/n).
func (r *Renderer) NewStripSet(n, wpx, hpx int) gfx.StripSet {
	if n < 1 {
		n = 1
	}
	full := cgBitmapContextCreate(0, uintptr(wpx), uintptr(hpx), 8, uintptr(wpx*4), r.cs, bitmapInfoBGRA)
	if full == 0 {
		panic("gfx/cg: CGBitmapContextCreate failed")
	}
	s := &stripSet{r: r, w: wpx, h: hpx, full: full, data: cgBitmapContextGetData(full)}
	for i := 0; i < n; i++ {
		y0, y1 := i*hpx/n, (i+1)*hpx/n
		ctx := cgBitmapContextCreate(s.data+uintptr(y0*wpx*4), uintptr(wpx), uintptr(y1-y0), 8, uintptr(wpx*4), r.cs, bitmapInfoBGRA)
		if ctx == 0 {
			panic("gfx/cg: CGBitmapContextCreate failed")
		}
		cgFontSmooth(ctx, false)
		cgInterpolate(ctx, 1)
		band := &Surface{r: r, ctx: ctx, W: wpx, H: y1 - y0}
		cgSaveGState(ctx) // the base state Begin returns to
		s.bands = append(s.bands, band)
		s.bottom = append(s.bottom, y1)
	}
	return s
}

func (s *stripSet) Target(i int) gfx.Target { return bandTarget{target{s.bands[i]}, s.bottom[i]} }

// bandTarget draws a strip with the absolute-bottom translate: BeginStrip's translate by the
// strip's absolute bottom row keeps every device coordinate exactly the single-target value.
type bandTarget struct {
	target
	bottom int
}

func (b bandTarget) Begin(scale float64, origin gfx.Point, bg gfx.Color) {
	b.Surface.BeginStrip(scale, origin.X, origin.Y, b.bottom, bg)
}

func (s *stripSet) Strips() int { return len(s.bands) }

// Image wraps the whole bitmap without copying; the image keeps the bitmap memory alive until
// it is garbage-collected.
func (s *stripSet) Image(wpt, hpt float64) gfx.Image {
	prov := cgDataProviderCreateWithData(0, s.data, uintptr(s.w*s.h*4), 0)
	ref := cgImageCreate(uintptr(s.w), uintptr(s.h), 8, 32, uintptr(s.w*4), s.r.cs, bitmapInfoBGRA, prov)
	cgDataProviderRelease(prov)
	s.imgs.Add(1)
	im := &Image{ref, wpt, hpt}
	runtime.AddCleanup(im, func(s *stripSet) { s.imageGone() }, s)
	return im
}

// imageGone drops a shared-image reference; the last one frees the bitmap a Close waited for.
func (s *stripSet) imageGone() {
	if s.imgs.Add(-1) == 0 && s.dead {
		s.free()
	}
}

func (s *stripSet) Close() {
	if s.dead {
		return
	}
	s.dead = true
	if s.imgs.Load() == 0 {
		s.free()
	}
}

func (s *stripSet) free() {
	for _, b := range s.bands {
		b.Close()
	}
	s.bands = nil
	if s.full != 0 {
		cgContextRelease(s.full)
		s.full = 0
	}
}
