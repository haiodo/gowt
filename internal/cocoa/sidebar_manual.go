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
	if r, g, b, ok := overWindow(cardFill()); ok {
		return r, g, b
	}
	return 242, 242, 242
}

// RoundCorners clips view and its subviews to a continuous-curve rounded rectangle through its layer, so
// the corners show what lies behind the view; false where the view has no layer.
func RoundCorners(view *NSView, r float64) bool {
	v := uintptr(view.Id)
	msg(v, "setWantsLayer:", 1)
	l := msg(v, "layer")
	if l == 0 {
		return false
	}
	if msgF64 == nil {
		purego.RegisterFunc(&msgF64, objcMsgSend)
	}
	msgF64(l, sel("setCornerRadius:"), r)
	msg(l, "setMasksToBounds:", 1)
	if respondsTo(l, "setCornerCurve:") {
		msg(l, "setCornerCurve:", nsString("continuous")) // kCACornerCurveContinuous
	}
	return true
}

// ClipBackdrop puts the backdrop of view in a container that clips to the backdrop frame: a 30-point
// gradient left of the inspector glass, most likely its shadow, darkened the toolbar band beside it. The
// container stands for the backdrop afterwards, so SetBackdropFrame moves it and the glass follows.
func ClipBackdrop(view *NSView) {
	initAssociated()
	v := uintptr(view.Id)
	bd := getAssociated(v, backdropKey())
	host := msg(bd, "superview")
	if bd == 0 || host == 0 {
		return
	}
	box := msg(msg(class("NSView"), "alloc"), "init")
	msg(box, "setWantsLayer:", 1)
	msg(msg(box, "layer"), "setMasksToBounds:", 1)
	r := NewNSViewOverload1(int64(bd)).Frame()
	msgRectOnly(box, sel("setFrame:"), r)
	msg(host, "addSubview:positioned:relativeTo:", box, 1, bd) // NSWindowAbove, then bd leaves that slot
	msg(bd, "retain")
	msg(bd, "removeFromSuperview")
	msgRectOnly(bd, sel("setFrame:"), NSRect{Width: r.Width, Height: r.Height})
	msg(bd, "setAutoresizingMask:", viewWidthHeightSizable)
	msg(box, "addSubview:", bd)
	msg(bd, "release")
	setAssociated(v, backdropKey(), box, 1)
	msg(box, "release")
}

// SetEdgeLine draws a separator line down the leading edge of the backdrop of view, its full height: the
// divider of a full-height NSSplitViewController inspector runs through the title bar too (Apple developer
// forums thread 818282). The line lies between the glass and view, so it takes no clicks; it hides while
// the backdrop is 0 wide.
func SetEdgeLine(view *NSView) {
	initAssociated()
	v := uintptr(view.Id)
	bd := getAssociated(v, backdropKey())
	if bd == 0 || msg(bd, "superview") == 0 {
		return
	}
	key := sel("gowtEdgeLine")
	line := getAssociated(v, key)
	if line == 0 {
		// NSBoxSeparator, vertical in a frame taller than wide, put right above the glass (NSWindowAbove)
		line = msg(msg(class("NSBox"), "alloc"), "init")
		msg(line, "setBoxType:", 2)
		msg(msg(bd, "superview"), "addSubview:positioned:relativeTo:", line, 1, bd)
		setAssociated(v, key, line, 1)
		msg(line, "release")
	}
	r := NewNSViewOverload1(int64(bd)).Frame()
	msgRectOnly(line, sel("setFrame:"), NSRect{X: r.X, Y: r.Y, Width: 1, Height: r.Height})
	msg(line, "setHidden:", boolArg(r.Width == 0))
}

// ClearFirstResponder makes the window of view its own first responder, so no control has the focus.
func ClearFirstResponder(view *NSView) {
	if w := msg(uintptr(view.Id), "window"); w != 0 {
		msg(w, "makeFirstResponder:", 0)
	}
}

// WindowIsFirstResponder reports whether the window of view is its own first responder.
func WindowIsFirstResponder(view *NSView) bool {
	w := msg(uintptr(view.Id), "window")
	return w != 0 && msg(w, "firstResponder") == w
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
	sourceListHooks(msg(v, "class"), msg(v, "isKindOfClass:", class("NSOutlineView"))&0xff != 0)
	// setStyle: and setSelectionHighlightStyle: reselect the selected rows, which the delegate (the SWT widget) reports
	// as a Selection. Setting the delegate again also refreshes NSTableView's cache of which delegate methods exist.
	d := msg(v, "delegate")
	quiet := func(selector string, arg uintptr) {
		msg(v, "setDelegate:", 0)
		msg(v, selector, arg)
		msg(v, "setDelegate:", d)
	}
	// Columns that fill the table exactly are fitted again by setStyle: (_sizeTableColumnsToFitWidth:), whatever
	// the column autoresizing style: the padding comes off a column with the autoresizing mask. SWT's widths stay.
	tv := NewNSTableViewOverload1(int64(v))
	cols := tv.TableColumns()
	widths := make([]float64, cols.Count())
	for i := range widths {
		widths[i] = NewNSTableColumnOverload1(cols.ObjectAtIndex(int64(i)).Id).Width()
	}
	quiet("setStyle:", tableStyleSourceList)
	for i, w := range widths {
		NewNSTableColumnOverload1(cols.ObjectAtIndex(int64(i)).Id).SetWidth(w)
	}
	// The source-list style sizes rows from the sidebar icon size setting, apart from rowHeight, which SWT
	// reads as the item height. A custom size style keeps rowHeight in charge, raised to the measured height.
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
	quiet("setSelectionHighlightStyle:", ^uintptr(0)) // NSTableViewSelectionHighlightStyleNone
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
// delegate method tableView:rowViewForRow: (outlineView:rowViewForItem: for an outline view) returns
// row views that are never emphasized. SWT tables are their own delegate, and other tables keep the
// default row view (nil).
func sourceListHooks(cls uintptr, outline bool) {
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
	rowView := func(table uintptr) uintptr {
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
	}
	// An outline view asks its delegate for outlineView:rowViewForItem: and never for tableView:rowViewForRow:.
	if outline {
		forItem := purego.NewCallback(func(self, _, outline, item uintptr) uintptr { return rowView(outline) })
		OSClass_addMethod(int64(cls), int64(sel("outlineView:rowViewForItem:")), int64(forItem), "@@:@@")
		return
	}
	forRow := purego.NewCallback(func(self, _, table uintptr, row int) uintptr { return rowView(table) })
	OSClass_addMethod(int64(cls), int64(sel("tableView:rowViewForRow:")), int64(forRow), "@@:@q")
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

// ReleaseSidebar forgets the sidebar edge and the inspector split view of the window of view; call it
// when the window is disposed.
func ReleaseSidebar(view *NSView) {
	w := msg(uintptr(view.Id), "window")
	delete(sidebarEdges, w)
	delete(sidebarHidden, w)
	delete(inspectorSplits, w)
}

// inspectorSplits holds, per window, the split view of InstallInspectorTracker, owned by the window.
var inspectorSplits = map[uintptr]uintptr{}

// InstallInspectorTracker gives the window of view an empty vertical NSSplitView whose divider
// SetInspectorDivider keeps on the inspector edge: AppKit aligns toolbar items, through an
// NSTrackingSeparatorToolbarItem, only with a split view divider, and the SWT window has none. It lies
// zero points high under the content view, never seen or clicked. The first call rebuilds an earlier
// toolbar with the separator.
func InstallInspectorTracker(view *NSView) {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 || inspectorSplits[w] != 0 || class("NSTrackingSeparatorToolbarItem") == 0 {
		return
	}
	content := msg(w, "contentView")
	theme := msg(content, "superview")
	if theme == 0 {
		return
	}
	sv := msg(msg(class("NSSplitView"), "alloc"), "init")
	msg(sv, "setVertical:", 1)
	msg(sv, "setDividerStyle:", 2) // NSSplitViewDividerStyleThin
	for range 2 {
		p := msg(msg(class("NSView"), "alloc"), "init")
		msg(sv, "addSubview:", p)
		msg(p, "release")
	}
	r := NewNSViewOverload1(int64(theme)).Bounds()
	r.Height = 0
	msgRectOnly(sv, sel("setFrame:"), r)
	msg(sv, "setAutoresizingMask:", 2) // NSViewWidthSizable
	msg(theme, "addSubview:positioned:relativeTo:", sv, windowBelowArg, content)
	msg(sv, "release")
	inspectorSplits[w] = sv
	if d := getAssociated(w, toolbarKey()); d != 0 && toolbarDelegates[d] != nil {
		InstallToolbar(view, toolbarDelegates[d].items)
	}
}

var msgF64Index func(self, sel uintptr, v float64, i uintptr)

// SetInspectorDivider puts the divider of the split view of InstallInspectorTracker x points from the
// left edge of the window. A hidden inspector hides the second subview, which collapses it, as
// NSSplitViewController collapses an inspector.
func SetInspectorDivider(view *NSView, x float64, hidden bool) {
	sv := inspectorSplits[msg(uintptr(view.Id), "window")]
	if sv == 0 {
		return
	}
	pane := msg(msg(sv, "subviews"), "objectAtIndex:", 1)
	if hidden != (msg(pane, "isHidden")&0xff != 0) {
		msg(pane, "setHidden:", boolArg(hidden))
		msg(sv, "adjustSubviews")
	}
	if hidden {
		return
	}
	if msgF64Index == nil {
		purego.RegisterFunc(&msgF64Index, objcMsgSend)
	}
	msgF64Index(sv, sel("setPosition:ofDividerAtIndex:"), x, 0)
}

// sidebarHidden holds the windows whose sidebar is hidden.
var sidebarHidden = map[uintptr]bool{}

// SetSidebarHidden takes the sidebar spacers out of the toolbar or puts them back; without them
// the items and title follow the traffic lights, as in Finder. Search text and the rest stay.
func SetSidebarHidden(view *NSView, hidden bool) {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return
	}
	if hidden {
		sidebarHidden[w] = true
	} else {
		delete(sidebarHidden, w)
	}
	s := toolbarDelegates[getAssociated(w, toolbarKey())]
	tb := msg(w, "toolbar")
	if s == nil || tb == 0 || !s.sidebar || s.collapsed == hidden {
		return
	}
	s.collapsed = hidden
	if hidden {
		items := msg(tb, "items")
		for i := int(msg(items, "count")) - 1; i >= 0; i-- {
			id := goString(msg(msg(items, "objectAtIndex:", uintptr(i)), "itemIdentifier"))
			if id == sidebarSpacerID || id == sidebarEndID {
				msg(tb, "removeItemAtIndex:", uintptr(i))
			}
		}
		s.spacer, s.width, s.end = 0, 0, 0
		return
	}
	// The spacer goes in first: the end spacer's index in ids counts it.
	all := s.ids()
	for _, id := range []string{sidebarSpacerID, sidebarEndID} {
		if i := msg(all, "indexOfObject:", nsString(id)); i < msg(all, "count") {
			msg(tb, "insertItemWithItemIdentifier:atIndex:", nsString(id), i)
		}
	}
}

// spacerGap is the room between the sidebar edge and the end of the spacer; AppKit adds its item spacing.
// sidebarEndWidth: the spacer after the Sidebar items. AppKit gives it an item viewer at least 44
// points wide, so the gap between the Sidebar items and the next item straddles the sidebar edge.
const (
	spacerGap       = 4
	sidebarEndWidth = 4
)

// fitSpacer sizes the spacer of the toolbar of w so that it ends spacerGap points past the sidebar edge.
// Its own left edge is measured once it is in the window, before that taken as the zoom button's right
// edge plus 8 points. With Sidebar items the spacer after them ends there instead.
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
	width := edge + spacerGap - x0
	if s.end != 0 {
		if msg(s.spacer, "window") == w && msg(s.end, "window") == w {
			// Both frames come from the same toolbar layout, also when the constraint changed since.
			v := NewNSViewOverload1(int64(s.end))
			r := v.ConvertRect_toView_(v.Bounds(), nil)
			cur := NewNSViewOverload1(int64(s.spacer)).Bounds().Width
			width = min(width, cur+edge+spacerGap-(r.X+r.Width))
		} else {
			width -= 40 * float64(s.overCount())
		}
	}
	width = max(1, width)
	if math.Abs(width-s.spacerWidth) < 0.5 {
		return
	}
	s.spacerWidth = width
	msgF64(s.width, sel("setConstant:"), width)
}

// minViewerWidth is the narrowest item viewer AppKit puts a toolbar item view in, even a 1-point
// spacer: measured on macOS 26 (docs/theme-look.md), AppKit does not report it.
const minViewerWidth = 44

// SidebarItemsMinEdge is the sidebar edge, in window points, below which the Sidebar items of the
// toolbar of the window of view no longer end before it: the first spacer at its narrowest viewer.
// ok is false without Sidebar items or before the toolbar has placed both spacers.
func SidebarItemsMinEdge(view *NSView) (edge float64, ok bool) {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return 0, false
	}
	s := toolbarDelegates[getAssociated(w, toolbarKey())]
	if s == nil || s.spacer == 0 || s.end == 0 || msg(s.spacer, "window") != w || msg(s.end, "window") != w {
		return 0, false
	}
	// Both frames come from one toolbar layout; the end is fitted to edge+spacerGap, and the items
	// move left with the spacer only while its viewer is wider than minViewerWidth.
	v := NewNSViewOverload1(int64(s.end))
	r := v.ConvertRect_toView_(v.Bounds(), nil)
	cur := NewNSViewOverload1(int64(s.spacer)).Bounds().Width
	return r.X + r.Width - max(cur, minViewerWidth) + minViewerWidth - spacerGap, true
}

// DescribeSidebarToolbar lists, in window points, the zoom button's right edge and the frames of the
// two sidebar spacers and of their superviews (the item viewers), for GOWT_SIDEBAR_DEBUG.
func DescribeSidebarToolbar(view *NSView) string {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return "no window\n"
	}
	frame := func(v uintptr) string {
		if v == 0 || msg(v, "window") != w {
			return "not in the window"
		}
		nv := NewNSViewOverload1(int64(v))
		r := nv.ConvertRect_toView_(nv.Bounds(), nil)
		return fmt.Sprintf("x %.1f width %.1f right %.1f", r.X, r.Width, r.X+r.Width)
	}
	out := ""
	if b := msg(w, "standardWindowButton:", 2); b != 0 {
		out += "zoom button " + frame(b) + "\n"
	}
	s := toolbarDelegates[getAssociated(w, toolbarKey())]
	if s == nil {
		return out + "no toolbar\n"
	}
	for _, v := range []struct {
		name string
		v    uintptr
	}{{"spacer", s.spacer}, {"end spacer", s.end}} {
		out += v.name + " " + frame(v.v) + "\n"
		if v.v != 0 {
			out += v.name + " superview " + goString(msg(msg(v.v, "superview"), "className")) + " " + frame(msg(v.v, "superview")) + "\n"
		}
	}
	return out
}

// overCount is the number of items over the sidebar.
func (s *toolbarState) overCount() int {
	n := 0
	for _, it := range s.items {
		if s.overSidebar(it) {
			n++
		}
	}
	return n
}

var (
	spacerClass uintptr
	superLayout func(self, sel uintptr)
	msgF64Ret   func(self, sel uintptr, v float64) uintptr
)

// SidebarToolbarLaidOut, when set, runs with the NSWindow after its toolbar laid the sidebar
// spacers out, which is when SidebarItemsMinEdge has a new measurement.
var SidebarToolbarLaidOut func(window uintptr)

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
			defer enterCallback()()
			if w := msg(self, "window"); w != 0 {
				fitSpacer(w)
				if SidebarToolbarLaidOut != nil {
					SidebarToolbarLaidOut(w)
				}
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
