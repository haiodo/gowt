//go:build darwin

package cocoa

// LayerBacked gives an NSView its own backing layer. Without one an SWT Label's NSBox makes the
// window server keep ~225 MB of graphics buffers on every text change (.work/monitor-mem/report.md).
func LayerBacked(view int64) {
	msg(uintptr(view), "setWantsLayer:", 1)
}

// SetLayerBacked turns the backing layer of view on or off. look uses it on the window content view
// while panes float over the content: a layer-backed gfx canvas inside a tree that draws into the
// window backing store composites its CALayer above the panes whatever the NSView order.
func SetLayerBacked(view int64, on bool) {
	if on {
		LayerBacked(view)
		return
	}
	if msg(uintptr(view), "wantsLayer") != 0 {
		msg(uintptr(view), "setWantsLayer:", 0)
	}
}

// SubviewIndex is the index of view among the subviews of host, -1 where it is not one. The
// subviews array runs back to front: a higher index draws above a lower one.
func SubviewIndex(host, view int64) int {
	subs := msg(uintptr(host), "subviews")
	for i := uintptr(0); i < msg(subs, "count"); i++ {
		if msg(subs, "objectAtIndex:", i) == uintptr(view) {
			return int(i)
		}
	}
	return -1
}

// WantsLayer reports whether view currently has a backing layer.
func WantsLayer(view int64) bool { return msg(uintptr(view), "wantsLayer") != 0 }
