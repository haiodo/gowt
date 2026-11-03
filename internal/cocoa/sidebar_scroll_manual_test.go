//go:build darwin

package cocoa

import (
	"os"
	"testing"
)

func TestSidebarScrollUnderToolbar(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	onMain(func() {
		w := NewNSWindowOverload1(int64(msg(class("NSWindow"), "alloc"))).InitWithContentRect(NSRect{Width: 400, Height: 360}, 1|styleMaskFullSize, 2, false)
		defer w.Release()
		root := NewNSViewOverload1(int64(msg(uintptr(w.Id), "contentView")))
		sv := NewNSScrollViewOverload1(int64(msg(msg(class("NSScrollView"), "alloc"), "init")))
		defer sv.Release()
		sv.SetFrame(root.Bounds())
		msg(uintptr(root.Id), "addSubview:", uintptr(sv.Id))
		tv := NewNSTableViewOverload1(int64(msg(msg(class("NSTableView"), "alloc"), "init")))
		defer tv.Release()
		msg(uintptr(sv.Id), "setDocumentView:", uintptr(tv.Id))
		msg(uintptr(sv.Id), "setAutomaticallyAdjustsContentInsets:", 0)
		original := NSRect{X: 7, Y: 3, Width: 5, Height: 2}
		msgRectOnly(uintptr(sv.Id), sel("setContentInsets:"), original)
		SetSidebarScrollUnderToolbar(root, true)
		state, ok := sidebarScrolls[uintptr(sv.Id)]
		if !ok {
			fatalf(t, "top scroll view has no material")
		}
		if msg(uintptr(sv.Id), "automaticallyAdjustsContentInsets")&255 == 0 {
			t.Error("automatic content insets disabled")
		}
		SetSourceList(&tv.NSView)
		if msg(uintptr(sv.Id), "automaticallyAdjustsContentInsets")&255 == 0 {
			t.Error("source-list refresh removed automatic insets")
		}
		if msg(state.edge, "isAccessibilityElement") != 0 || msg(state.edge, "accessibilityIsIgnored") == 0 {
			t.Error("material exposed to accessibility")
		}
		sv.SetFrame(NSRect{Width: 320, Height: root.Bounds().Height})
		SetSidebarScrollUnderToolbar(root, true)
		edge := NewNSViewOverload1(int64(state.edge)).Frame()
		var mask NSRect
		layer := msg(msg(state.edge, "layer"), "mask")
		OSObjc_msgSend_stretOverload9(&mask, int64(layer), int64(sel("frame")))
		if edge.Width != 320 || mask.Width != 320 || mask.Height != edge.Height {
			t.Errorf("edge=%+v mask=%+v", edge, mask)
		}
		if msg(msg(layer, "animationKeys"), "count") != 0 {
			t.Error("mask resize animated")
		}
		msg(uintptr(sv.Id), "removeFromSuperview")
		SetSidebarScrollUnderToolbar(root, true)
		if _, ok := sidebarScrolls[uintptr(sv.Id)]; ok {
			t.Error("detached scroll view retained its sidebar state")
		}
		var detachedInsets NSRect
		OSObjc_msgSend_stretOverload9(&detachedInsets, int64(sv.Id), int64(sel("contentInsets")))
		if detachedInsets != original || msg(uintptr(sv.Id), "automaticallyAdjustsContentInsets")&255 != 0 {
			t.Errorf("detached scroll insets restored as %+v", detachedInsets)
		}
		msg(uintptr(root.Id), "addSubview:", uintptr(sv.Id))
		SetSidebarScrollUnderToolbar(root, true)
		if _, ok := sidebarScrolls[uintptr(sv.Id)]; !ok {
			t.Error("reattached scroll view has no sidebar state")
		}
		SetSidebarScrollUnderToolbar(root, false)
		var restored NSRect
		OSObjc_msgSend_stretOverload9(&restored, int64(sv.Id), int64(sel("contentInsets")))
		if restored != original || msg(uintptr(sv.Id), "automaticallyAdjustsContentInsets")&255 != 0 {
			t.Errorf("insets restored as %+v", restored)
		}
		if _, ok := sidebarScrolls[uintptr(sv.Id)]; ok {
			t.Error("scroll state retained after disabling")
		}
	})
}
