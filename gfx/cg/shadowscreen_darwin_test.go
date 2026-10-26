//go:build darwin

package cg

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestScreenShadow(t *testing.T) {
	r := NewRenderer(Options{})
	for _, box := range []bool{false, true} {
		s := r.NewSurface(100, 40)
		s.SetScreenScale(2)
		rc := gfx.NewRecorder()
		rc.Scale(3, 3) // the node scale must not reach the shadow
		sh := gfx.Shadow{DX: 15, Color: shBlack, Screen: true}
		if box {
			rc.FillShadowRRect(2, 2, 5, 5, 0, shRed, sh)
		} else {
			rc.FillShadowPath(gfx.RectPath(gfx.R(2, 2, 5, 5)), shRed, sh)
		}
		s.Begin(1, 0, 0, gfx.Color{})
		s.Replay(rc.Finish(), 0, 0)
		if p := pixel(s, 45, 12); p != [4]byte{0, 0, 0, 255} { // 6..21 moved by 15*2
			t.Errorf("box=%v: screen shadow: %v", box, p)
		}
		if p := pixel(s, 25, 12); p[3] != 0 {
			t.Errorf("box=%v: gap: %v", box, p)
		}
		if p := pixel(s, 90, 12); p[3] != 0 {
			t.Errorf("box=%v: the shadow scaled with the node: %v", box, p)
		}
		s.Close()
	}
}
