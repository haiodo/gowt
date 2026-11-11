package gfx

import (
	"math"
	"testing"
	"time"
)

// TestSnapScaleExactZoom pins the zoom-scale rule: by default tiles are drawn at the next power of
// two of zoom*dpr, with ExactZoom at exactly zoom*dpr.
func TestSnapScaleExactZoom(t *testing.T) {
	zooms := []float64{0.07, 0.5, 1, 1.3, 2, 3, 4, 7.7, 64}
	for _, exact := range []bool{false, true} {
		for _, z := range zooms {
			g := newRig(t, Options{ExactZoom: exact})
			g.c.SetCamera(Camera{Zoom: z})
			got := g.c.snapScale()
			var want float64
			if exact {
				want = z * 2 // the rig's dpr
			} else {
				want = math.Exp2(math.Ceil(math.Log2(z*2) - 0.01))
			}
			if got != want {
				t.Errorf("ExactZoom=%v zoom=%v: snapScale=%v, want %v", exact, z, got, want)
			}
		}
	}
}

// TestOptionsFilled checks the defaults Options() reports for a zero Options and that the set
// fields survive NewCompositor unchanged.
func TestOptionsFilled(t *testing.T) {
	def := newRig(t, Options{}).c.Options()
	if def.TilePx != 512 || def.Workers != 2 || def.Margin != 1 || def.Settle != 120*time.Millisecond || def.ExactZoom {
		t.Errorf("default Options() = %+v", def)
	}
	set := Options{TilePx: 256, Workers: 3, Margin: 2, Settle: 40 * time.Millisecond, ExactZoom: true}
	got := newRig(t, set).c.Options()
	if got != set {
		t.Errorf("Options() = %+v, want %+v", got, set)
	}
}
