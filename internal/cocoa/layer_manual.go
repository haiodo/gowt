//go:build darwin

package cocoa

// LayerBacked gives an NSView its own backing layer. Without one an SWT Label's NSBox makes the
// window server keep ~225 MB of graphics buffers on every text change (.work/monitor-mem/report.md).
func LayerBacked(view int64) {
	msg(uintptr(view), "setWantsLayer:", 1)
}
