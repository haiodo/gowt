//go:build windows

package d2d

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestScreenShadow(t *testing.T) {
	for _, box := range []bool{false, true} {
		rc := gfx.NewRecorderWith(shared)
		rc.Scale(3, 3) // the node scale must not reach the shadow
		sh := gfx.Shadow{DX: 15, Color: black, Screen: true}
		if box {
			rc.FillShadowRRect(2, 2, 5, 5, 0, red, sh)
		} else {
			rc.FillShadowPath(gfx.RectPath(gfx.R(2, 2, 5, 5)), red, sh)
		}
		tg := shared.NewTarget(100, 40)
		tg.(gfx.ScreenScaler).SetScreenScale(2)
		tg.Begin(1, gfx.Point{}, gfx.Color{})
		tg.Draw([]gfx.Item{{Pic: rc.Finish(), M: gfx.Identity(), Alpha: 1}})
		img := gfx.FromBGRA(tg.Pixels(), 100, 40)
		tg.Close()
		if p := px(img, 45, 12); !near(p, [4]byte{0, 0, 0, 255}, 4) { // 6..21 moved by 15*2
			t.Errorf("box=%v: screen shadow: %v", box, p)
		}
		if p := px(img, 25, 12); p[3] != 0 {
			t.Errorf("box=%v: gap: %v", box, p)
		}
		if p := px(img, 90, 12); p[3] != 0 {
			t.Errorf("box=%v: the shadow scaled with the node: %v", box, p)
		}
	}
}
