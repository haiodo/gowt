//go:build darwin

// Hand-written: grouped-section cards and the split view that a sidebar's toolbar separator tracks.
package cocoa

import "github.com/ebitengine/purego"

var (
	cardClass   uintptr
	msgF64Index func(self, sel uintptr, v float64, i int)
)

func cardKey() uintptr  { return sel("gowtCard") }
func splitKey() uintptr { return sel("gowtSidebarSplit") }

// cardFill is the grouped background of System Settings: quaternarySystemFillColor (macOS 14),
// quaternaryLabelColor before it.
func cardFill() uintptr {
	if c := class("NSColor"); respondsTo(c, "quaternarySystemFillColor") {
		return msg(c, "quaternarySystemFillColor")
	}
	return msg(class("NSColor"), "quaternaryLabelColor")
}

// GowtCardView sets its layer colour in updateLayer, where AppKit resolves dynamic colours for the view's
// appearance and which a light/dark switch calls again; without drawRect the layer has no backing bitmap.
func registerCardClass() {
	if cardClass != 0 {
		return
	}
	yes := purego.NewCallback(func(self, _ uintptr) uintptr { return 1 })
	update := purego.NewCallback(func(self, _ uintptr) {
		msg(msg(self, "layer"), "setBackgroundColor:", msg(cardFill(), "CGColor"))
	})
	noHit := purego.NewCallback(func(self, _ uintptr, x, y float64) uintptr { return 0 })
	cardClass = allocClass(class("NSView"), "GowtCardView", 0)
	addMethod(cardClass, sel("wantsUpdateLayer"), yes, "c@:")
	addMethod(cardClass, sel("updateLayer"), update, "v@:")
	addMethod(cardClass, sel("hitTest:"), noHit, "@@:{CGPoint=dd}")
	registerClass(cardClass)
}

// InstallCard puts a rounded fill in the grouped background colour right behind view, a sibling as
// with InstallBackdrop; on false it removes the fill. Keep the frame in step with SyncCard.
func InstallCard(view *NSView, radius float64, on bool) {
	initAssociated()
	v := uintptr(view.Id)
	if old := getAssociated(v, cardKey()); old != 0 {
		msg(old, "removeFromSuperview")
		setAssociated(v, cardKey(), 0, 1)
	}
	host := msg(v, "superview")
	if !on || host == 0 {
		return
	}
	registerCardClass()
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	c := msg(msg(cardClass, "alloc"), "init")
	msg(c, "setWantsLayer:", 1)
	layer := msg(c, "layer")
	msgF64(layer, sel("setCornerRadius:"), radius)
	if respondsTo(layer, "setCornerCurve:") {
		msg(layer, "setCornerCurve:", nsString("continuous"))
	}
	msgRectOnly(c, sel("setFrame:"), view.Frame())
	msg(c, "setHidden:", msg(v, "isHidden")&0xff)
	msg(host, "addSubview:positioned:relativeTo:", c, windowBelowArg, v)
	setAssociated(v, cardKey(), c, 1)
	msg(c, "release")
}

// SyncCard moves the fill of InstallCard to the frame of view.
func SyncCard(view *NSView) {
	initAssociated()
	if c := getAssociated(uintptr(view.Id), cardKey()); c != 0 {
		msgRectOnly(c, sel("setFrame:"), view.Frame())
	}
}

// SetCardHidden hides or shows the fill of InstallCard; SWT reports Show before it unhides view.
func SetCardHidden(view *NSView, hidden bool) {
	initAssociated()
	if c := getAssociated(uintptr(view.Id), cardKey()); c != 0 {
		h := uintptr(0)
		if hidden {
			h = 1
		}
		msg(c, "setHidden:", h)
	}
}

// HasCard reports whether view has a fill from InstallCard (for tests).
func HasCard(view *NSView) bool {
	initAssociated()
	return getAssociated(uintptr(view.Id), cardKey()) != 0
}

// SetSourceList gives a table or outline view the sidebar look: the source-list style (macOS 11) and no
// background of its own or of its scroll view, so the glass behind shows.
func SetSourceList(view *NSView) {
	v := uintptr(view.Id)
	if respondsTo(v, "setStyle:") {
		msg(v, "setStyle:", 3) // NSTableViewStyleSourceList
	}
	msg(v, "setBackgroundColor:", msg(class("NSColor"), "clearColor"))
	if sv := msg(v, "enclosingScrollView"); sv != 0 {
		msg(sv, "setDrawsBackground:", 0)
	}
}

// SetBackdropFrame moves the material of InstallBackdrop to a frame of its own, in the coordinates of
// the superview of view, for a backdrop larger than view.
func SetBackdropFrame(view *NSView, x, y, w, h float64) {
	initAssociated()
	if bd := getAssociated(uintptr(view.Id), backdropKey()); bd != 0 {
		msgRectOnly(bd, sel("setFrame:"), NSRect{X: x, Y: y, Width: w, Height: h})
	}
}

// InstallSidebarTracker gives the window of view an empty vertical NSSplitView for an
// NSTrackingSeparatorToolbarItem to follow, since AppKit aligns toolbar items with a split view divider
// only. It lies zero points high under the content view, never seen or clicked; an earlier toolbar is rebuilt.
func InstallSidebarTracker(view *NSView) bool {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 || class("NSTrackingSeparatorToolbarItem") == 0 {
		return false
	}
	if getAssociated(w, splitKey()) != 0 {
		return true
	}
	content := msg(w, "contentView")
	frame := msg(content, "superview")
	if frame == 0 {
		return false
	}
	sv := msg(msg(class("NSSplitView"), "alloc"), "init")
	msg(sv, "setVertical:", 1)
	msg(sv, "setDividerStyle:", 2) // NSSplitViewDividerStyleThin
	for range 2 {
		p := msg(msg(class("NSView"), "alloc"), "init")
		msg(sv, "addSubview:", p)
		msg(p, "release")
	}
	r := NewNSViewOverload1(int64(frame)).Bounds()
	r.Height = 0
	msgRectOnly(sv, sel("setFrame:"), r)
	msg(sv, "setAutoresizingMask:", 2) // NSViewWidthSizable
	msg(frame, "addSubview:positioned:relativeTo:", sv, windowBelowArg, content)
	setAssociated(w, splitKey(), sv, 1)
	msg(sv, "release")
	if d := getAssociated(w, toolbarKey()); d != 0 && toolbarDelegates[d] != nil {
		InstallToolbar(view, toolbarDelegates[d].items)
	}
	return true
}

// SetSidebarDivider puts the divider of the split view of InstallSidebarTracker at x points from the
// left edge of the window.
func SetSidebarDivider(view *NSView, x float64) {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return
	}
	sv := getAssociated(w, splitKey())
	if sv == 0 {
		return
	}
	if msgF64Index == nil {
		purego.RegisterFunc(&msgF64Index, objcMsgSend)
	}
	msgF64Index(sv, sel("setPosition:ofDividerAtIndex:"), x, 0)
}
