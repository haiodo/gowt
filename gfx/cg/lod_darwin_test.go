//go:build darwin

package cg

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/synth"
)

const lodDPR = 2.0

var lodArea = gfx.R(0, 0, 3000, 1300)

// ink is the mean darkness (0 white, 1 black) of an opaque raster.
func ink(r *gfx.Raster) float64 {
	var sum float64
	for i := 0; i+3 < len(r.Pix); i += 4 {
		sum += 255 - (0.114*float64(r.Pix[i]) + 0.587*float64(r.Pix[i+1]) + 0.299*float64(r.Pix[i+2]))
	}
	return sum / 255 / float64(r.W*r.H)
}

// shrink resamples r by the factor f<=1: bilinear taps at the output pixel centres, which is
// what a layer does with no mipmaps, or a box average over the source footprint, which is what
// mipmaps give.
func shrink(r *gfx.Raster, f float64, box bool) *gfx.Raster {
	w, h := max(1, int(float64(r.W)*f)), max(1, int(float64(r.H)*f))
	out := gfx.NewRaster(w, h)
	at := func(x, y, c int) float64 {
		x, y = min(max(x, 0), r.W-1), min(max(y, 0), r.H-1)
		return float64(r.Pix[(y*r.W+x)*4+c])
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			for c := 0; c < 4; c++ {
				var v float64
				if box {
					x0, x1 := int(float64(x)/f), max(int(float64(x+1)/f), int(float64(x)/f)+1)
					y0, y1 := int(float64(y)/f), max(int(float64(y+1)/f), int(float64(y)/f)+1)
					n := 0
					for yy := y0; yy < y1; yy++ {
						for xx := x0; xx < x1; xx++ {
							v += at(xx, yy, c)
							n++
						}
					}
					v /= float64(n)
				} else {
					sx, sy := (float64(x)+0.5)/f-0.5, (float64(y)+0.5)/f-0.5
					ix, iy := int(math.Floor(sx)), int(math.Floor(sy))
					fx, fy := sx-float64(ix), sy-float64(iy)
					v = at(ix, iy, c)*(1-fx)*(1-fy) + at(ix+1, iy, c)*fx*(1-fy) + at(ix, iy+1, c)*(1-fx)*fy + at(ix+1, iy+1, c)*fx*fy
				}
				out.Pix[(y*w+x)*4+c] = byte(v + 0.5)
			}
		}
	}
	return out
}

// savePNG writes r under $GFX_LOD_DIR, if it is set.
func savePNG(t *testing.T, r *gfx.Raster, name string) {
	dir := os.Getenv("GFX_LOD_DIR")
	if dir == "" {
		return
	}
	b, err := r.PNG()
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, name), b, 0o644)
}

func lodZooms() []float64 {
	var zs []float64
	for z := 0.5; z > 0.019; z /= 1.04 {
		zs = append(zs, z)
	}
	return zs
}

func lodRender(t *testing.T, r *Renderer, doc *synth.Doc, scale float64) *gfx.Raster {
	t.Helper()
	ras, err := gfx.RenderSubtree(r, doc.Scene, doc.Scene.Root(), lodArea, scale, white)
	if err != nil {
		t.Fatal(err)
	}
	return ras
}

// TestLODCoverageContinuous renders the synth scene through the real renderer over a sweep of
// zoom levels and requires the mean ink to change smoothly: no wash-out, no jump at the LOD switch.
func TestLODCoverageContinuous(t *testing.T) {
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true})
	doc := synth.New(10000, 1)
	zs := lodZooms()
	cov := make([]float64, len(zs))
	for i, z := range zs {
		ras := lodRender(t, r, doc, z*lodDPR)
		cov[i] = ink(ras)
		if i%6 == 0 {
			savePNG(t, ras, fmt.Sprintf("exact_z%.3f.png", z))
		}
	}
	worst, at := 1.0, 1
	for i := 1; i < len(zs); i++ {
		q := cov[i] / cov[i-1]
		t.Logf("zoom %.3f ink %.4f  step %.3f", zs[i], cov[i], q)
		if math.Abs(math.Log(q)) > math.Abs(math.Log(worst)) {
			worst, at = q, i
		}
	}
	t.Logf("worst step %.3f at zoom %.3f", worst, zs[at])
	// 4 % zoom steps: smooth ink changes by a few percent per step
	if worst > 1.12 || worst < 1/1.12 {
		t.Errorf("ink jumps by %.3f between zoom %.3f and %.3f", worst, zs[at-1], zs[at])
	}
	// the scene keeps its look: far out, the ink stays within 2x of the ink at the start of the sweep
	for i, z := range zs {
		if cov[i] < cov[0]/2 {
			t.Errorf("zoom %.3f washes out: ink %.4f against %.4f at zoom %.2f", z, cov[i], cov[0], zs[0])
			break
		}
	}
}

// TestLODTileGesture renders a tile level and shrinks it to a zoom the compositor has not re-rastered
// yet, the way a layer does during a pinch, with and without mipmaps.
func TestLODTileGesture(t *testing.T) {
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true, LOD: true})
	doc := synth.New(10000, 1)
	for _, level := range []float64{0.5, 1} {
		base := lodRender(t, r, doc, level)
		for _, k := range []float64{1, 1.5, 2, 3, 4, 6, 8} {
			f := 1 / k
			want := ink(lodRender(t, r, doc, level*f))
			lin, box := shrink(base, f, false), shrink(base, f, true)
			t.Logf("level %.2f shrink %.1fx: direct %.4f  bilinear %.4f (%.2f)  box %.4f (%.2f)", level, k, want, ink(lin), ink(lin)/want, ink(box), ink(box)/want)
			if k <= 2 && (ink(box)/want < 0.8 || ink(box)/want > 1.25) {
				t.Errorf("level %.2f shrunk %.1fx: ink %.2f of a render at that scale", level, k, ink(box)/want)
			}
			if k == 4 {
				savePNG(t, lin, fmt.Sprintf("gesture_l%.1f_bilinear.png", level))
				savePNG(t, box, fmt.Sprintf("gesture_l%.1f_box.png", level))
			}
		}
	}
}
