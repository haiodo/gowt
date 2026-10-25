//go:build darwin

package cg

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

// TestSkeletonMatchesTextInk compares the ink of a text line with the ink of its skeleton bar at the
// size where the glyphs hand over to it: the switch must not change the weight of the block.
func TestSkeletonMatchesTextInk(t *testing.T) {
	plain := NewRenderer(Options{TextCache: true})
	lod := NewRenderer(Options{TextCache: true, LOD: true})
	for _, size := range []float64{11, 13} {
		for _, str := range []string{"Primary action label", "Button 1234"} {
			rec := gfx.NewRecorder()
			f := gfx.Font{Name: "Helvetica Neue", Size: size}
			rec.Text(str, 0, size, f, gfx.Color{R: 0.1, G: 0.1, B: 0.12, A: 1})
			pic := rec.Finish()
			for _, px := range []float64{3, 4, 5, 6, 8} {
				a, _ := gfx.RenderPicture(plain, pic, px/size, white)
				b, _ := gfx.RenderPicture(lod, pic, px/size, white)
				ia, ib := ink(a), ink(b)
				t.Logf("size %2.0f %-22q px %.0f: glyphs %.4f  lod %.4f  ratio %.2f", size, str, px, ia, ib, ib/ia)
				if q := ib / ia; q < 0.85 || q > 1.15 {
					t.Errorf("size %.0f %q at %.0f px: skeleton ink is %.2f of the glyph ink", size, str, px, q)
				}
			}
		}
	}
}

// A picture with a Lod fades from the substitute to the full picture over the band above LodBelow.
func TestLodCrossFade(t *testing.T) {
	r := NewRenderer(Options{LOD: true})
	box := &gfx.Path{}
	box.Rect(0, 0, 100, 100)
	full, sub := gfx.NewRecorder(), gfx.NewRecorder()
	full.FillPath(box, gfx.Color{R: 1, A: 1})
	sub.FillPath(box, gfx.Color{B: 1, A: 1})
	pic := full.Finish()
	pic.Lod, pic.LodBelow = sub.Finish(), 50
	red := func(scale float64) (rd, bl byte) {
		s := r.NewSurface(100, 100)
		defer s.Close()
		s.Begin(scale, 0, 0, white)
		s.Replay(pic, 0, 0)
		p := pixel(s, 20, 20)
		return p[2], p[0]
	}
	if rd, bl := red(0.45); rd != 0 || bl != 255 {
		t.Errorf("under LodBelow: R %d B %d, want the substitute", rd, bl)
	}
	if rd, bl := red(0.5 * 1.15); rd == 0 || rd == 255 || bl == 0 {
		t.Errorf("in the band: R %d B %d, want a mix", rd, bl)
	}
	if rd, bl := red(0.5 * 1.4); rd != 255 || bl != 0 {
		t.Errorf("above the band: R %d B %d, want the full picture", rd, bl)
	}
}
