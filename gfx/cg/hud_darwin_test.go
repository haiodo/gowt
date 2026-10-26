//go:build darwin

package cg

import (
	"testing"

	"github.com/haiodo/gowt/gfx"
)

func TestHostHUDGroup(t *testing.T) {
	initHost()
	root := send0(send0(clsLayer, sels.alloc), sels.init)
	h := NewHost(root, 1)
	defer h.Close()
	h.Begin()
	hud := h.NewLayer(gfx.GroupHUD).(*layer)
	hud.Set(gfx.R(1, 2, 30, 40), gfx.Point{}, gfx.Identity())
	h.Commit()
	if n := len(h.layers[gfx.GroupHUD]); n != 1 {
		t.Fatalf("%d HUD layers", n)
	}
	if got := msgRectRet(hud.ref, sels.frame); got != (cgRect{1, 2, 30, 40}) {
		t.Errorf("frame %v", got)
	}
	hud.Remove()
	if n := len(h.layers[gfx.GroupHUD]); n != 0 {
		t.Errorf("%d HUD layers after Remove", n)
	}
}
