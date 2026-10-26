//go:build linux

package cairo

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestScreenShadow(t *testing.T) {
	r := newRenderer(t, Options{})
	for _, box := range []bool{false, true} {
		rc := gfx.NewRecorderWith(r)
		rc.Scale(3, 3) // the node scale must not reach the shadow
		sh := gfx.Shadow{DX: 15, Color: shBlack, Screen: true}
		if box {
			rc.FillShadowRRect(2, 2, 5, 5, 0, shRed, sh)
		} else {
			rc.FillShadowPath(gfx.RectPath(gfx.R(2, 2, 5, 5)), shRed, sh)
		}
		tg := r.NewTarget(100, 40)
		tg.(gfx.ScreenScaler).SetScreenScale(2)
		tg.Begin(1, gfx.Point{}, gfx.Color{})
		tg.Draw([]gfx.Item{{Pic: rc.Finish(), M: gfx.Identity(), Alpha: 1}})
		if p := pixel(tg, 45, 12); p != [4]byte{0, 0, 0, 255} { // 6..21 moved by 15*2
			t.Errorf("box=%v: screen shadow: %v", box, p)
		}
		if p := pixel(tg, 25, 12); p[3] != 0 {
			t.Errorf("box=%v: gap: %v", box, p)
		}
		if p := pixel(tg, 90, 12); p[3] != 0 {
			t.Errorf("box=%v: the shadow scaled with the node: %v", box, p)
		}
		tg.Close()
	}
}
