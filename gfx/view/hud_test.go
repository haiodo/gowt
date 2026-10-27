package view

import (
	"testing"

	g "github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

func TestPointerHUDBeforeScene(t *testing.T) {
	rg := newRig(t)
	badge := gfx.NewNode(2)
	rg.r.hud = func(p gfx.Point) *gfx.Node {
		if p.X < 20 {
			return badge
		}
		return nil
	}
	// view (10, 10) is over the scene node, and over the HUD badge: the HUD wins
	rg.r.pointer(PointerMove, gfx.Point{X: 10, Y: 10}, 0, 0, g.Mods{})
	if p := rg.got[0]; p.HUD != badge || p.Node != nil {
		t.Errorf("over the HUD: HUD %v Node %v", p.HUD, p.Node)
	}
	// view (30, 10) is over the scene node only
	rg.r.pointer(PointerMove, gfx.Point{X: 30, Y: 10}, 0, 0, g.Mods{})
	if p := rg.got[1]; p.HUD != nil || p.Node != rg.node {
		t.Errorf("beside the HUD: HUD %v Node %v", p.HUD, p.Node)
	}
}
