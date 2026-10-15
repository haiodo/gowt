//go:build darwin

// Hand-written: the grouped-section colour, sidebar source lists and the toolbar spacer that keeps toolbar items past a sidebar.
package cocoa

import (
	"fmt"
	"math"

	"github.com/ebitengine/purego"
)

// cardFill is the grouped background of System Settings: quaternarySystemFillColor (macOS 14),
// quaternaryLabelColor before it.
func cardFill() uintptr {
	if c := class("NSColor"); respondsTo(c, "quaternarySystemFillColor") {
		return msg(c, "quaternarySystemFillColor")
	}
	return msg(class("NSColor"), "quaternaryLabelColor")
}

// GroupedFillRGB is the System Settings card colour as 0..255 sRGB: cardFill over windowBackgroundColor,
// resolved for the app's current appearance. SWT paints controls in opaque colours, so the translucent
// system fill is flattened here.
func GroupedFillRGB() (r, g, b int32) {
	appearance := class("NSAppearance")
	old := msg(appearance, "currentAppearance")
	msg(old, "retain")
	msg(appearance, "setCurrentAppearance:", msg(msg(class("NSApplication"), "sharedApplication"), "effectiveAppearance"))
	defer func() {
		msg(appearance, "setCurrentAppearance:", old)
		msg(old, "release")
	}()
	srgb := msg(class("NSColorSpace"), "sRGBColorSpace")
	bg := NewNSColorOverload1(int64(msg(msg(class("NSColor"), "windowBackgroundColor"), "colorUsingColorSpace:", srgb)))
	fill := NewNSColorOverload1(int64(msg(cardFill(), "colorUsingColorSpace:", srgb)))
	if bg.Id == 0 || fill.Id == 0 {
		return 242, 242, 242
	}
	a := fill.AlphaComponent()
	mix := func(b, f float64) int32 { return int32(math.Round(255 * (b*(1-a) + f*a))) }
	return mix(bg.RedComponent(), fill.RedComponent()), mix(bg.GreenComponent(), fill.GreenComponent()), mix(bg.BlueComponent(), fill.BlueComponent())
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
	// AppKit draws its own darker source-list selection outside drawRect:; GowtSidebarRowView draws the pill.
	msg(v, "setSelectionHighlightStyle:", ^uintptr(0)) // NSTableViewSelectionHighlightStyleNone
	if sv := msg(v, "enclosingScrollView"); sv != 0 {
		msg(sv, "setDrawsBackground:", 0)
		msg(msg(sv, "contentView"), "setDrawsBackground:", 0)
		// The list sits below the title bar already; the automatic inset added the title bar height again.
		msg(sv, "setAutomaticallyAdjustsContentInsets:", 0)
		msgRectOnly(sv, sel("setContentInsets:"), NSRect{}) // NSEdgeInsets has the layout of four doubles
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
		// The dirty rect arrives in FP registers on arm64 and in memory on amd64: only self is read.
		draw := purego.NewCallback(func(self, _ uintptr) {
			if msg(self, "isSelected")&0xff != 0 {
				drawPill(self)
			}
		})
		var superSelected func(self, sel uintptr, on bool)
		purego.RegisterFunc(&superSelected, uintptr(OSClass_getMethodImplementation(int64(class("NSTableRowView")), int64(sel("setSelected:")))))
		selected := purego.NewCallback(func(self, cmd, on uintptr) {
			superSelected(self, cmd, on&0xff != 0)
			// The cell views draw the text, in the accent colour while selected.
			NeedsDisplayTree(NewNSViewOverload1(int64(self)))
		})
		rowViewClass = allocClass(class("NSTableRowView"), "GowtSidebarRowView", 0)
		addMethod(rowViewClass, sel("isEmphasized"), no, "c@:")
		addMethod(rowViewClass, sel("drawRect:"), draw, "v@:{CGRect={CGPoint=dd}{CGSize=dd}}")
		addMethod(rowViewClass, sel("setSelected:"), selected, "v@:c")
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
	width := max(1, edge+spacerGap-x0)
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
	// Both sizes fixed and never zero: the toolbar warns about an item it measures as ambiguous or empty.
	width = msgF64Ret(msg(view, "widthAnchor"), sel("constraintEqualToConstant:"), 1)
	msg(width, "setActive:", 1)
	msg(msgF64Ret(msg(view, "heightAnchor"), sel("constraintEqualToConstant:"), 1), "setActive:", 1)
	return view, width
}

var roundedRect func(cls, sel uintptr, r NSRect, rx, ry float64) uintptr

// drawPill draws the Finder sidebar selection: a pill in tertiarySystemFillColor (macOS 14, else
// quaternaryLabelColor; black or white at 4.7%, #EBEBEB on Finder's #F7F7F7 glass), 8 points in from the
// sides of the row and 1 point from top and bottom.
func drawPill(row uintptr) {
	if roundedRect == nil {
		purego.RegisterFunc(&roundedRect, objcMsgSend)
	}
	r := NewNSViewOverload1(int64(row)).Bounds()
	r.X += 8
	r.Width -= 16
	r.Y++
	r.Height -= 2
	c := class("NSColor")
	fill := msg(c, "quaternaryLabelColor")
	if respondsTo(c, "tertiarySystemFillColor") {
		fill = msg(c, "tertiarySystemFillColor")
	}
	msg(fill, "setFill")
	msg(roundedRect(class("NSBezierPath"), sel("bezierPathWithRoundedRect:xRadius:yRadius:"), r, 6, 6), "fill")
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

// DescribeViewChain lists view and its superviews up to the window's content view, with the AppKit
// state that decides whether a view paints a background (for GOWT_SIDEBAR_DEBUG).
func DescribeViewChain(view *NSView) string {
	var out string
	content := msg(msg(uintptr(view.Id), "window"), "contentView")
	describe := func(v uintptr, indent string) {
		line := fmt.Sprintf("%s%s opaque=%d", indent, goString(msg(v, "className")), msg(v, "isOpaque")&0xff)
		if respondsTo(v, "drawsBackground") {
			line += fmt.Sprintf(" drawsBackground=%d", msg(v, "drawsBackground")&0xff)
		}
		if respondsTo(v, "backgroundColor") {
			line += " backgroundColor=" + goString(msg(msg(v, "backgroundColor"), "description"))
		}
		if l := msg(v, "layer"); l != 0 {
			line += fmt.Sprintf(" layer.opaque=%d layer.bg=%x", msg(l, "isOpaque")&0xff, msg(l, "backgroundColor"))
		}
		out += line + "\n"
	}
	for v := uintptr(view.Id); v != 0; v = msg(v, "superview") {
		describe(v, "  view ")
		if v == content {
			break
		}
	}
	if sv := msg(uintptr(view.Id), "enclosingScrollView"); sv != 0 {
		subs := msg(sv, "subviews")
		for i, n := uintptr(0), msg(subs, "count"); i < n; i++ {
			describe(msg(subs, "objectAtIndex:", i), "  scroll subview ")
		}
	}
	return out
}
