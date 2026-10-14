//go:build darwin

// Hand-written: grouped-section cards, sidebar source lists and the toolbar spacer that keeps toolbar items past a sidebar.
package cocoa

import (
	"math"

	"github.com/ebitengine/purego"
)

var cardClass uintptr

func cardKey() uintptr { return sel("gowtCard") }

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

const tableStyleSourceList = 3

// SetSourceList gives a table or outline view the sidebar look: the source-list style (macOS 11), no
// background of its own or of its scroll view so the glass behind shows, and the Finder selection, a
// grey pill (row views never emphasized) under text in the accent colour (drawn by the toolkit's cell).
func SetSourceList(view *NSView) {
	v := uintptr(view.Id)
	if !respondsTo(v, "setStyle:") {
		return
	}
	sourceListHooks(msg(v, "class"))
	// NSTableView caches which delegate methods exist when the delegate is set.
	d := msg(v, "delegate")
	msg(v, "setDelegate:", 0)
	msg(v, "setDelegate:", d)
	msg(v, "setStyle:", tableStyleSourceList)
	// The source-list style sizes rows from the sidebar icon size setting, apart from rowHeight, which SWT
	// reads as the item height. A custom size style keeps rowHeight in charge, raised to the measured height.
	tv := NewNSTableViewOverload1(int64(v))
	tv.Tile()
	h := tv.RowHeight()
	if tv.NumberOfRows() > 0 {
		h = max(h, tv.RectOfRow(0).Height-tv.IntercellSpacing().Height)
	}
	if respondsTo(v, "setRowSizeStyle:") {
		msg(v, "setRowSizeStyle:", 0) // NSTableViewRowSizeStyleCustom
	}
	tv.SetRowHeight(h)
	msg(v, "setBackgroundColor:", 0) // the hook below turns any colour into clear
	if sv := msg(v, "enclosingScrollView"); sv != 0 {
		msg(sv, "setDrawsBackground:", 0)
		msg(msg(sv, "contentView"), "setDrawsBackground:", 0)
	}
}

var (
	hookedTables = map[uintptr]bool{}
	rowViewClass uintptr
)

const rowViewIDName = "GowtSidebarRow"

// sourceListHooks gives the table class cls two methods that act only on its source-list instances:
// setBackgroundColor: stores clear, since SWT sets its default opaque colour again whenever the
// background is refreshed (an alpha-0 parent background does not count as one there), and the
// delegate method tableView:rowViewForRow: returns row views that are never emphasized. SWT tables are
// their own delegate, and other tables keep the default row view (nil).
func sourceListHooks(cls uintptr) {
	if hookedTables[cls] {
		return
	}
	hookedTables[cls] = true
	setBG := sel("setBackgroundColor:")
	var orig func(self, sel, color uintptr)
	purego.RegisterFunc(&orig, uintptr(OSClass_getMethodImplementation(int64(cls), int64(setBG))))
	imp := purego.NewCallback(func(self, cmd, color uintptr) {
		if msg(self, "style") == tableStyleSourceList {
			color = msg(class("NSColor"), "clearColor")
		}
		orig(self, cmd, color)
	})
	if !OSClass_addMethod(int64(cls), int64(setBG), int64(imp), "v@:@") {
		OSMethod_setImplementation(OSClass_getInstanceMethod(int64(cls), int64(setBG)), int64(imp))
	}
	if rowViewClass == 0 {
		no := purego.NewCallback(func(self, _ uintptr) uintptr { return 0 })
		rowViewClass = allocClass(class("NSTableRowView"), "GowtSidebarRowView", 0)
		addMethod(rowViewClass, sel("isEmphasized"), no, "c@:")
		registerClass(rowViewClass)
	}
	rowView := purego.NewCallback(func(self, _, table uintptr, row int) uintptr {
		if msg(table, "style") != tableStyleSourceList {
			return 0
		}
		id := nsString(rowViewIDName)
		if rv := msg(table, "makeViewWithIdentifier:owner:", id, table); rv != 0 {
			return rv
		}
		rv := msg(msg(msg(rowViewClass, "alloc"), "init"), "autorelease")
		msg(rv, "setIdentifier:", id)
		return rv
	})
	OSClass_addMethod(int64(cls), int64(sel("tableView:rowViewForRow:")), int64(rowView), "@@:@q")
}

// SetBackdropFrame moves the material of InstallBackdrop to a frame of its own, in the coordinates of
// the superview of view, for a backdrop larger than view.
func SetBackdropFrame(view *NSView, x, y, w, h float64) {
	initAssociated()
	if bd := getAssociated(uintptr(view.Id), backdropKey()); bd != 0 {
		msgRectOnly(bd, sel("setFrame:"), NSRect{X: x, Y: y, Width: w, Height: h})
	}
}

// sidebarEdges holds, per window, the x of the sidebar's right edge in window points.
var sidebarEdges = map[uintptr]float64{}

// SetSidebarEdge makes the toolbar of the window of view start its navigational items and title x points
// from the window's left edge, past a sidebar. AppKit puts navigational items and the title at the leading
// edge, right after the traffic lights, so an empty navigational item of fitted width goes first. The
// first call rebuilds an earlier toolbar with that item.
func SetSidebarEdge(view *NSView, x float64) {
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return
	}
	_, had := sidebarEdges[w]
	sidebarEdges[w] = x
	if d := getAssociated(w, toolbarKey()); !had && d != 0 && toolbarDelegates[d] != nil {
		InstallToolbar(view, toolbarDelegates[d].items)
	}
	fitSpacer(w)
}

// ReleaseSidebar forgets the sidebar edge of the window of view; call it when the window is disposed.
func ReleaseSidebar(view *NSView) { delete(sidebarEdges, msg(uintptr(view.Id), "window")) }

// spacerGap is the room between the sidebar edge and the end of the spacer; AppKit adds its item spacing.
const spacerGap = 4

// fitSpacer sizes the spacer of the toolbar of w so that it ends spacerGap points past the sidebar edge.
// Its own left edge is measured once it is in the window, before that taken as the zoom button's right
// edge plus 8 points.
func fitSpacer(w uintptr) {
	initAssociated()
	edge, ok := sidebarEdges[w]
	s := toolbarDelegates[getAssociated(w, toolbarKey())]
	if !ok || s == nil || s.width == 0 {
		return
	}
	var x0 float64
	if msg(s.spacer, "window") == w {
		v := NewNSViewOverload1(int64(s.spacer))
		x0 = v.ConvertRect_toView_(v.Bounds(), nil).X
	} else if b := msg(w, "standardWindowButton:", 2); b != 0 { // NSWindowZoomButton
		v := NewNSViewOverload1(int64(b))
		r := v.ConvertRect_toView_(v.Bounds(), nil)
		x0 = r.X + r.Width + 8
	}
	width := max(0, edge+spacerGap-x0)
	if math.Abs(width-s.spacerWidth) < 0.5 {
		return
	}
	s.spacerWidth = width
	msgF64(s.width, sel("setConstant:"), width)
}

var (
	spacerClass uintptr
	superLayout func(self, sel uintptr)
	msgF64Ret   func(self, sel uintptr, v float64) uintptr
)

// newSpacer returns the view of the spacer item, retained, and its width constraint. Its layout refits
// the width, since the toolbar places the item after the window layout ran; the refit is queued on the
// run loop, because a constraint changed inside the toolbar's layout pass asks for another pass.
func newSpacer() (view, width uintptr) {
	if spacerClass == 0 {
		purego.RegisterFunc(&superLayout, uintptr(OSClass_getMethodImplementation(int64(class("NSView")), int64(sel("layout")))))
		layout := purego.NewCallback(func(self, cmd uintptr) {
			superLayout(self, cmd)
			msg(self, "performSelectorOnMainThread:withObject:waitUntilDone:", sel("gowtFit"), 0, 0)
		})
		fit := purego.NewCallback(func(self, _ uintptr) {
			if w := msg(self, "window"); w != 0 {
				fitSpacer(w)
			}
		})
		spacerClass = allocClass(class("NSView"), "GowtToolbarSpacer", 0)
		addMethod(spacerClass, sel("layout"), layout, "v@:")
		addMethod(spacerClass, sel("gowtFit"), fit, "v@:")
		registerClass(spacerClass)
	}
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	if msgF64Ret == nil {
		purego.RegisterFunc(&msgF64Ret, objcMsgSend)
	}
	view = msg(msg(spacerClass, "alloc"), "init")
	msg(view, "setTranslatesAutoresizingMaskIntoConstraints:", 0)
	width = msgF64Ret(msg(view, "widthAnchor"), sel("constraintEqualToConstant:"), 0)
	msg(width, "setActive:", 1)
	return view, width
}

// NeedsDisplayTree marks view and every view under it for display.
func NeedsDisplayTree(view *NSView) {
	var walk func(v uintptr)
	walk = func(v uintptr) {
		msg(v, "setNeedsDisplay:", 1)
		subs := msg(v, "subviews")
		for i, n := uintptr(0), msg(subs, "count"); i < n; i++ {
			walk(msg(subs, "objectAtIndex:", i))
		}
	}
	walk(uintptr(view.Id))
}
