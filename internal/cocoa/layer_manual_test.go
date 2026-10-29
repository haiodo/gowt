//go:build darwin

package cocoa

import "testing"

// LayerBacked must turn wantsLayer on and give the view a backing layer; AppKit leaves an NSBox without one.
func TestLayerBacked(t *testing.T) {
	v := NewNSView()
	v.Alloc()
	v.Init()
	if got := msg(uintptr(v.Id), "wantsLayer"); got != 0 {
		t.Skipf("the view already wants a layer: %d", got)
	}
	LayerBacked(v.Id)
	if got := msg(uintptr(v.Id), "wantsLayer"); got != 1 {
		t.Fatalf("wantsLayer = %d, want 1", got)
	}
	if v.Layer() == nil {
		t.Fatal("no backing layer after LayerBacked")
	}
}
